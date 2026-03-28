package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/domain/entities"
)

type contextKey string
const (
	ContextUserID    contextKey = "user_id"
	ContextPhone     contextKey = "phone"
	ContextIsAdmin   contextKey = "is_admin"
	ContextAdminRole contextKey = "admin_role"
	ContextAdminClaims contextKey = "admin_claims"
)

// AuthMiddleware validates the Bearer JWT token (user) and injects claims into context.
func AuthMiddleware(authSvc *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token == "" {
				writeError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}
			claims, err := authSvc.ValidateJWT(token)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}
			ctx := context.WithValue(r.Context(), ContextUserID, claims.UserID)
			ctx = context.WithValue(ctx, ContextPhone, claims.PhoneNumber)
			ctx = context.WithValue(ctx, ContextIsAdmin, false)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AdminAuthMiddleware validates admin JWT tokens (email+password issued).
// It injects the full JWTClaims so handlers can check RBAC roles.
func AdminAuthMiddleware(adminAuthSvc *services.AdminAuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token == "" {
				writeError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}
			claims, err := adminAuthSvc.ValidateAdminJWT(token)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid or expired admin token")
				return
			}
			if !claims.IsAdmin {
				writeError(w, http.StatusForbidden, "admin access required")
				return
			}
			ctx := context.WithValue(r.Context(), ContextUserID, claims.UserID)
			ctx = context.WithValue(ctx, ContextIsAdmin, true)
			ctx = context.WithValue(ctx, ContextAdminRole, string(claims.Role))
			ctx = context.WithValue(ctx, ContextAdminClaims, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns 403 if the caller's role is not in the allowed list.
// Use inside handlers: middleware.RequireRole(w, r, entities.RoleSuperAdmin, entities.RoleFinance)
func RequireRole(w http.ResponseWriter, r *http.Request, roles ...entities.AdminRole) bool {
	role, _ := r.Context().Value(ContextAdminRole).(string)
	for _, allowed := range roles {
		if string(allowed) == role {
			return true
		}
	}
	writeError(w, http.StatusForbidden, "insufficient role for this action")
	return false
}

// CORS middleware — restrict to known origins in production.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // Tighten in production
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequestLogger logs each HTTP request.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if encErr := json.NewEncoder(w).Encode(map[string]string{"error": msg}); encErr != nil {
		log.Printf("[Auth] writeError encode failure: %v", encErr)
	}
}
