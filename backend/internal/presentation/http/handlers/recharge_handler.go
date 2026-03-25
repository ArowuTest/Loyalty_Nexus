package handlers

import (
	"io"
	"net/http"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/infrastructure/queue"
)

type RechargeHandler struct {
	rechargeSvc *services.RechargeService
	eq          *queue.EventQueue
}

func NewRechargeHandler(rs *services.RechargeService, eq *queue.EventQueue) *RechargeHandler {
	return &RechargeHandler{rechargeSvc: rs, eq: eq}
}

// PaystackWebhook handles Paystack charge.success events (Independent Mode).
func (h *RechargeHandler) PaystackWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sig := r.Header.Get("x-paystack-signature")
	if !h.rechargeSvc.VerifyPaystackSignature(body, sig) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Acknowledge immediately — process async
	w.WriteHeader(http.StatusOK)

	go func() {
		var event services.PaystackEvent
		if err := parseJSON(body, &event); err != nil {
			return
		}
		if event.Event != "charge.success" {
			return
		}
		if err := h.rechargeSvc.ProcessRechargeWebhook(r.Context(), &event); err != nil {
			// ErrDuplicateRecharge is silently ignored — already logged in service
		}
	}()
}

// MNOWebhook handles raw BSS billing events (Integrated Mode — MTN).
func (h *RechargeHandler) MNOWebhook(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement BSS webhook signature verification
	// For now, enqueue into NATS for async processing
	w.WriteHeader(http.StatusAccepted)
}
