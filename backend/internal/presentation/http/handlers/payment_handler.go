package handlers

import (
	"encoding/json"
	"net/http"
	"log"
	"loyalty-nexus/internal/application/services"
)

type PaymentHandler struct {
	rechargeService *services.RechargeService // assuming this exists
}

func NewPaymentHandler(rs *services.RechargeService) *PaymentHandler {
	return &PaymentHandler{rechargeService: rs}
}

func (h *PaymentHandler) PaystackWebhook(w http.ResponseWriter, r *http.Request) {
	var event struct {
		Event string `json:"event"`
		Data  struct {
			Reference string `json:"reference"`
			Amount    int64  `json:"amount"`
			Customer  struct {
				Email string `json:"email"`
			} `json:"customer"`
			Metadata struct {
				MSISDN string `json:"msisdn"`
				Network string `json:"network"`
				Type string `json:"type"` // airtime, data
			} `json:"metadata"`
		} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "Invalid payload", 400)
		return
	}

	if event.Event == "charge.success" {
		log.Printf("[Paystack] Success: Ref %s | MSISDN %s", event.Data.Reference, event.Data.Metadata.MSISDN)
		// Trigger Provisioning and Ledger Updates
		// h.rechargeService.ProcessSuccessfulPayment(r.Context(), ...)
	}

	w.WriteHeader(200)
}
