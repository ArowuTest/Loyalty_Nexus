package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/presentation/http/middleware"

	"github.com/google/uuid"
)

// VTURechargeHandler handles public VTU recharge endpoints.
// No authentication required — anyone can recharge.
type VTURechargeHandler struct {
	svc *services.VTURechargeService
}

func NewVTURechargeHandler(svc *services.VTURechargeService) *VTURechargeHandler {
	return &VTURechargeHandler{svc: svc}
}

// GET /api/v1/recharge/networks
func (h *VTURechargeHandler) GetNetworks(w http.ResponseWriter, r *http.Request) {
	networks, err := h.svc.GetActiveNetworks(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load networks"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"networks": networks})
}

// GET /api/v1/recharge/networks/{code}/bundles
func (h *VTURechargeHandler) GetBundles(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "network code required"})
		return
	}
	bundles, err := h.svc.GetBundles(r.Context(), strings.ToUpper(code))
	if err != nil {
		log.Printf("[VTU] GetBundles %s: %v", code, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load bundles"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"bundles": bundles})
}

// POST /api/v1/recharge/initiate
// Public — no auth. Logged-in users pass their user_id in the JWT (optional).
func (h *VTURechargeHandler) Initiate(w http.ResponseWriter, r *http.Request) {
	var req services.InitiateRechargeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	// Optionally attach authenticated user ID if JWT is present
	if uid := userIDFromContext(r); uid != nil {
		req.UserID = uid
	}

	resp, err := h.svc.InitiateRecharge(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// POST /api/v1/recharge/vtu-webhook
// Paystack webhook for VTU recharges (NX_ prefix refs).
func (h *VTURechargeHandler) PaystackWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Ack immediately — Paystack requires fast 200
	w.WriteHeader(http.StatusOK)

	sig := r.Header.Get("X-Paystack-Signature")
	go func() {
		if err := h.svc.ProcessVTUPaystackWebhook(r.Context(), body, sig); err != nil {
			log.Printf("[VTU] ProcessVTUPaystackWebhook: %v", err)
		}
	}()
}

// ── Admin handlers ────────────────────────────────────────────────────────────

// GET /api/v1/admin/networks
func (h *VTURechargeHandler) AdminGetNetworks(w http.ResponseWriter, r *http.Request) {
	networks, err := h.svc.GetAllNetworksAdmin(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"networks": networks})
}

// PATCH /api/v1/admin/networks/{code}
func (h *VTURechargeHandler) AdminUpdateNetwork(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "network code required"})
		return
	}
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if err := h.svc.UpdateNetworkConfig(r.Context(), code, updates); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// ── helpers ─────────────────────────────────────────────────────────────────

// userIDFromContext returns the authenticated user's UUID if a valid JWT is
// present on the request. Returns nil for unauthenticated (guest) requests.
func userIDFromContext(r *http.Request) *uuid.UUID {
	val := r.Context().Value(middleware.ContextUserID)
	if val == nil {
		return nil
	}
	str, ok := val.(string)
	if !ok || str == "" {
		return nil
	}
	uid, err := uuid.Parse(str)
	if err != nil {
		return nil
	}
	return &uid
}
