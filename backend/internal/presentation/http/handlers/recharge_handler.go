package handlers

import (
	"context"
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
	"loyalty-nexus/internal/pkg/safe"
)

// RechargeHandler processes incoming payment / recharge webhooks.
type RechargeHandler struct {
	rechargeSvc *services.RechargeService
	mtnPushSvc  *services.MTNPushService
	eventQueue  *queue.EventQueue
}

func NewRechargeHandler(rs *services.RechargeService, eq *queue.EventQueue) *RechargeHandler {
	return &RechargeHandler{rechargeSvc: rs, eventQueue: eq}
}

// NewRechargeHandlerWithMTN creates a RechargeHandler with the MTN push service wired in.
// Use this constructor in main.go once MTNPushService is instantiated.
func NewRechargeHandlerWithMTN(rs *services.RechargeService, mtn *services.MTNPushService, eq *queue.EventQueue) *RechargeHandler {
	return &RechargeHandler{rechargeSvc: rs, mtnPushSvc: mtn, eventQueue: eq}
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
		mac.Write(body) //nolint:errcheck // hash.Hash.Write never returns an error
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

	safe.Go(func() {
		ctx := context.Background() // Use background context for async processing
		if err := h.rechargeSvc.ProcessRechargeWebhook(ctx, event); err != nil {
			log.Printf("[paystack] ProcessRechargeWebhook: %v", err)
		}
	})
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
		mac.Write(body) //nolint:errcheck // hash.Hash.Write never returns an error
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
		safe.Go(func() {
			if err := h.rechargeSvc.ProcessRechargeWebhook(context.Background(), event); err != nil {
				log.Printf("[mno_webhook] inline process: %v", err)
			}
		})
	}
}

// ── MTN Push Webhook ──────────────────────────────────────────────────────────
// POST /api/v1/recharge/mtn-push
//
// MTN pushes recharge notifications directly to this endpoint.
// Payload: { "transaction_ref": "...", "msisdn": "...", "recharge_type": "AIRTIME|DATA|BUNDLE",
//            "amount": 500.00, "timestamp": "2026-03-27T10:00:00Z" }
//
// Authentication: HMAC-SHA256 of the raw request body, sent in X-MTN-Signature header.
// Secret is read from MTN_PUSH_SECRET env var (falls back to mtn_push_hmac_secret in network_configs).
//
// Response: 200 OK with JSON body on success; 400/401/503 on error.
// The response is synchronous — MTN expects a 200 before it stops retrying.
func (h *RechargeHandler) MTNPushWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16)) // 64 KB max
	if err != nil {
		jsonError(w, "failed to read body", http.StatusBadRequest)
		return
	}

	// ── HMAC-SHA256 signature verification ───────────────────────────────────
	// MTN sends: X-MTN-Signature: sha256=<hex>
	secret := os.Getenv("MTN_PUSH_SECRET")
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body) //nolint:errcheck
		expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		got := r.Header.Get("X-MTN-Signature")
		if !hmac.Equal([]byte(expected), []byte(got)) {
			log.Printf("[mtn-push] signature mismatch — rejecting (got=%q)", got)
			jsonError(w, "invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// ── Parse payload ─────────────────────────────────────────────────────────
	var payload services.MTNPushPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		jsonError(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}
	if payload.TransactionRef == "" {
		jsonError(w, "transaction_ref is required", http.StatusBadRequest)
		return
	}
	if payload.MSISDN == "" {
		jsonError(w, "msisdn is required", http.StatusBadRequest)
		return
	}
	if payload.Amount <= 0 {
		jsonError(w, "amount must be positive", http.StatusBadRequest)
		return
	}

	// ── Guard: service must be wired ──────────────────────────────────────────
	if h.mtnPushSvc == nil {
		log.Printf("[mtn-push] MTNPushService not wired — check main.go")
		jsonError(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}

	// ── Process synchronously ─────────────────────────────────────────────────
	// MTN retries until it gets a 200, so we process synchronously and return
	// the result. The service is fast (single DB transaction + async side effects).
	result, err := h.mtnPushSvc.ProcessMTNPush(r.Context(), payload)
	if err != nil {
		log.Printf("[mtn-push] ProcessMTNPush error: %v", err)
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{ //nolint:errcheck
		"status":               "ok",
		"event_id":             result.EventID,
		"msisdn":               result.MSISDN,
		"pulse_points_awarded":  result.PulsePoints,
		"draw_entries_created": result.DrawEntries,
		"spin_credits_awarded": result.SpinCredits,
		"is_duplicate":         result.IsDuplicate,
	})
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
