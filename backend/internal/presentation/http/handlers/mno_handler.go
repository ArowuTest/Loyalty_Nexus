package handlers

import (
	"encoding/json"
	"net/http"
	"log"
	"os"
	"loyalty-nexus/internal/infrastructure/queue"
)

type MNOWebhookHandler struct {
	eventQueue *queue.EventQueue
}

func NewMNOWebhookHandler(eq *queue.EventQueue) *MNOWebhookHandler {
	return &MNOWebhookHandler{eventQueue: eq}
}

func (h *MNOWebhookHandler) BSSRechargeWebhook(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("OPERATION_MODE") != "integrated" {
		http.Error(w, "Integrated mode not enabled", http.StatusForbidden)
		return
	}

	// 1. Verify MNO Signature (Strategic Trust)
	// In production: validate X-MNO-Signature header
	
	var payload struct {
		MSISDN    string `json:"msisdn"`
		Amount    int64  `json:"amount_kobo"`
		Channel   string `json:"channel"`
		Reference string `json:"bss_ref"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// 2. Produce Identical internal RechargeEvent (REQ-2.1)
	event := queue.RechargeEvent{
		MSISDN: payload.MSISDN,
		Amount: payload.Amount,
		Ref:    payload.Reference,
	}

	if err := h.eventQueue.PushRecharge(r.Context(), event); err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	log.Printf("[MNO Webhook] Accepted recharge for %s via %s", payload.MSISDN, payload.Channel)
	w.WriteHeader(http.StatusAccepted)
}
