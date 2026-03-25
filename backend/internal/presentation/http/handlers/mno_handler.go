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
	// REQ-2.1: Integrated Mode BSS Recharge Webhook
	if os.Getenv("OPERATION_MODE") != "integrated" {
		http.Error(w, "Integrated mode not enabled", 403)
		return
	}

	// 1. Verify MNO Signature (Strategic Trust)
	// In production: validate X-MNO-Signature header using shared secret
	
	var payload struct {
		MSISDN    string `json:"msisdn"`
		Amount    int64  `json:"amount_kobo"`
		Channel   string `json:"channel"` // e.g., 'ussd_555', 'momo_app'
		Reference string `json:"bss_ref"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid payload", 400)
		return
	}

	// 2. Produce identical internal RechargeEvent
	event := queue.RechargeEvent{
		MSISDN: payload.MSISDN,
		Amount: payload.Amount,
		Ref:    payload.Reference,
	}

	if err := h.eventQueue.PushRecharge(r.Context(), event); err != nil {
		http.Error(w, "Internal queue error", 500)
		return
	}

	log.Printf("[MNO Webhook] Accepted recharge for %s via %s", payload.MSISDN, payload.Channel)
	w.WriteHeader(202)
}
