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
	// ... (parse payload) ...

	// Idempotency: Use Paystack reference as unique key
	// In production: SELECT count(*) FROM transactions WHERE metadata->>'ref' = event.Data.Reference
	
	if event.Event == "charge.success" {
		log.Printf("[Paystack] Success: Ref %s | MSISDN %s", event.Data.Reference, event.Data.Metadata.MSISDN)
		
		// Map Paystack network string to our normalized names
		network := strings.ToUpper(event.Data.Metadata.Network)
		
		err := h.rechargeService.ProcessSuccessfulPayment(
			r.Context(), 
			event.Data.Metadata.MSISDN, 
			event.Data.Amount, 
			network, 
			event.Data.Reference,
		)
		if err != nil {
			log.Printf("Failed to process recharge: %v", err)
			http.Error(w, "Processing failed", 500)
			return
		}
	}

	w.WriteHeader(200)
}
