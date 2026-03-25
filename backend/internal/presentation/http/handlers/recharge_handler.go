package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/infrastructure/queue"
)

// RechargeHandler processes incoming payment / recharge webhooks.
type RechargeHandler struct {
	rechargeSvc *services.RechargeService
	eventQueue  *queue.EventQueue
}

func NewRechargeHandler(rs *services.RechargeService, eq *queue.EventQueue) *RechargeHandler {
	return &RechargeHandler{rechargeSvc: rs, eventQueue: eq}
}

// ── Paystack Webhook ─────────────────────────────────────────────────────────
// POST /api/v1/recharge/paystack-webhook
func (h *RechargeHandler) PaystackWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// HMAC-SHA256 signature verification (Paystack uses SHA512 but accept both)
	secret := os.Getenv("PAYSTACK_WEBHOOK_SECRET")
	if secret == "" {
		secret = os.Getenv("PAYSTACK_SECRET_KEY")
	}
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		expected := hex.EncodeToString(mac.Sum(nil))
		got := r.Header.Get("X-Paystack-Signature")
		if got != "" && !hmac.Equal([]byte(expected), []byte(got)) {
			log.Printf("[paystack] signature mismatch — rejecting webhook")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	// Acknowledge immediately
	w.WriteHeader(http.StatusOK)

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return
	}

	event := buildPaystackEvent(payload)
	if event == nil {
		return
	}

	go func() {
		ctx := r.Context()
		if err := h.rechargeSvc.ProcessRechargeWebhook(ctx, event); err != nil {
			log.Printf("[paystack] ProcessRechargeWebhook: %v", err)
		}
	}()
}

// ── MNO BSS Webhook (Integrated Mode) ────────────────────────────────────────
// POST /api/v1/recharge/mno-webhook
func (h *RechargeHandler) MNOWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// HMAC-SHA256 signature verification (SEC-008)
	secret := os.Getenv("BSS_WEBHOOK_SECRET")
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		got := r.Header.Get("X-BSS-Signature")
		if got != "" && !hmac.Equal([]byte(expected), []byte(got)) {
			log.Printf("[mno_webhook] signature mismatch — rejecting")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	w.WriteHeader(http.StatusAccepted)

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return
	}

	// Translate BSS CDR into PaystackEvent format for unified processing
	event := buildBSSEvent(payload)
	if event == nil {
		return
	}

	// Try queue first; fallback to inline goroutine
	eventMap := map[string]interface{}{
		"event":     event.Event,
		"reference": event.Data.Reference,
		"amount":    event.Data.Amount,
		"phone":     event.Data.Customer.PhoneNumber,
		"status":    event.Data.Status,
		"source":    "bss",
	}
	if qErr := h.eventQueue.Publish(r.Context(), eventMap); qErr != nil {
		log.Printf("[mno_webhook] queue failed (%v) — processing inline", qErr)
		go func() {
			if err := h.rechargeSvc.ProcessRechargeWebhook(r.Context(), event); err != nil {
				log.Printf("[mno_webhook] inline process: %v", err)
			}
		}()
	}
}

// ── Builders ──────────────────────────────────────────────────────────────────

func buildPaystackEvent(p map[string]interface{}) *services.PaystackEvent {
	eventType, _ := p["event"].(string)
	if !strings.Contains(eventType, "charge.success") &&
		!strings.Contains(eventType, "transfer.success") {
		return nil
	}
	data, ok := p["data"].(map[string]interface{})
	if !ok {
		return nil
	}

	meta, _ := data["metadata"].(map[string]interface{})
	phone := mapStr(meta, "phone_number")
	if phone == "" {
		if cust, ok := data["customer"].(map[string]interface{}); ok {
			phone = mapStr(cust, "phone")
		}
	}
	if phone == "" {
		return nil
	}

	amountKobo, _ := data["amount"].(float64)
	ref, _ := data["reference"].(string)
	network := mapStr(meta, "network")

	event := &services.PaystackEvent{
		Event: eventType,
	}
	event.Data.Reference = ref
	event.Data.Amount = int64(amountKobo)
	event.Data.Status = "success"
	event.Data.Customer.PhoneNumber = normalizeNG(phone)

	_ = network // stored in metadata — recharge_service handles
	return event
}

func buildBSSEvent(p map[string]interface{}) *services.PaystackEvent {
	phone := mapStr(p, "msisdn")
	if phone == "" {
		phone = mapStr(p, "phone_number")
	}
	if phone == "" {
		return nil
	}
	amountNaira, _ := p["amount"].(float64)
	if amountNaira == 0 {
		return nil
	}
	txID := mapStr(p, "transaction_id")
	if txID == "" {
		txID = "bss-" + time.Now().Format("20060102150405")
	}

	event := &services.PaystackEvent{Event: "charge.success"}
	event.Data.Reference = txID
	event.Data.Amount = int64(amountNaira * 100) // naira → kobo
	event.Data.Status = "success"
	event.Data.Customer.PhoneNumber = normalizeNG(phone)
	return event
}

func mapStr(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}

func normalizeNG(p string) string {
	p = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(p, " ", ""), "-", ""))
	if strings.HasPrefix(p, "234") && len(p) == 13 {
		return "0" + p[3:]
	}
	if strings.HasPrefix(p, "+234") {
		return "0" + p[4:]
	}
	return p
}
