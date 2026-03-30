package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/presentation/http/middleware"
)

// AdminAuthHandler handles admin login, admin user management, and password changes.
type AdminAuthHandler struct {
	adminAuthSvc *services.AdminAuthService
}

func NewAdminAuthHandler(adminAuthSvc *services.AdminAuthService) *AdminAuthHandler {
	return &AdminAuthHandler{adminAuthSvc: adminAuthSvc}
}

// POST /api/v1/admin/auth/login — email + password → JWT
func (h *AdminAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email and password required"})
		return
	}

	token, admin, err := h.adminAuthSvc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token":     token,
		"admin_id":  admin.ID,
		"email":     admin.Email,
		"full_name": admin.FullName,
		"role":      admin.Role,
	})
}

// POST /api/v1/admin/auth/admins — create a new admin (super_admin only)
func (h *AdminAuthHandler) CreateAdmin(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r)
	if claims.Role != entities.RoleSuperAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "super_admin role required"})
		return
	}

	var req struct {
		Email    string             `json:"email"`
		Password string             `json:"password"`
		FullName string             `json:"full_name"`
		Role     entities.AdminRole `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email and password required"})
		return
	}
	if req.Role == "" {
		req.Role = entities.RoleOperations
	}

	admin, err := h.adminAuthSvc.CreateAdmin(r.Context(), req.Email, req.Password, req.FullName, req.Role)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, admin)
}

// GET /api/v1/admin/auth/admins — list all admins (super_admin only)
func (h *AdminAuthHandler) ListAdmins(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r)
	if claims.Role != entities.RoleSuperAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "super_admin role required"})
		return
	}
	admins, err := h.adminAuthSvc.ListAdmins(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list admins"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"admins": admins})
}

// DELETE /api/v1/admin/auth/admins/{id} — deactivate an admin (super_admin only)
func (h *AdminAuthHandler) DeactivateAdmin(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r)
	if claims.Role != entities.RoleSuperAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "super_admin role required"})
		return
	}
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid admin id"})
		return
	}
	if err := h.adminAuthSvc.DeactivateAdmin(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "admin deactivated"})
}

// POST /api/v1/admin/auth/change-password — change own password
func (h *AdminAuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r)
	adminID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid admin id in token"})
		return
	}
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if len(req.NewPassword) < 8 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "new password must be at least 8 characters"})
		return
	}
	if err := h.adminAuthSvc.UpdatePassword(r.Context(), adminID, req.OldPassword, req.NewPassword); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "password updated"})
}

// GET /api/v1/admin/auth/me — return current admin profile
func (h *AdminAuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"admin_id":  claims.UserID,
		"email":     claims.Email,
		"role":      claims.Role,
		"is_admin":  claims.IsAdmin,
	})
}

// claimsFromCtx extracts JWTClaims stored in context by AdminAuthMiddleware.
// Uses middleware.ContextAdminClaims key — must match what AdminAuthMiddleware stores.
func claimsFromCtx(r *http.Request) *entities.JWTClaims {
	if c, ok := r.Context().Value(middleware.ContextAdminClaims).(*entities.JWTClaims); ok {
		return c
	}
	return &entities.JWTClaims{}
}
