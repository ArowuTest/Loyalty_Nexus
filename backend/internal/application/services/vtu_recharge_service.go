package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"loyalty-nexus/internal/infrastructure/external"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── VTU Recharge Service ──────────────────────────────────────────────────────
// Handles platform-initiated recharges (user pays via Paystack → VTPass tops up).
// Public endpoints — no authentication required to recharge.
//
// Payment flow (matches RechargeMax):
//   1. Frontend calls POST /api/v1/recharge/initiate → gets Paystack URL
//   2. Paystack callback_url = backend GET /api/v1/recharge/callback?ref=REF
//   3. Backend verifies payment, fires VTPass async, redirects browser to
//      /recharge?payment=success&reference=REF
//   4. Frontend polls GET /api/v1/recharge/status/{ref} (adaptive intervals)
//      until status=SUCCESS or FAILED, then shows in-page banner + reward details.
//
// Double-points design: Paystack path + MTN CDR both award points (intentional).

type VTURechargeService struct {
	db          *gorm.DB
	vtpass      *external.VTPassHTTPClient
	bundleSvc   *external.NetworkBundleService
	rechargeSvc *RechargeService
	notifySvc   *NotificationService
}

// ── DTOs ──────────────────────────────────────────────────────────────────────

type InitiateRechargeRequest struct {
	MSISDN        string     `json:"msisdn"`
	Network       string     `json:"network"`
	RechargeType  string     `json:"recharge_type"`
	AmountKobo    int64      `json:"amount_kobo"`
	VariationCode string     `json:"variation_code"`
	Email         string     `json:"email"`
	UserID        *uuid.UUID `json:"user_id,omitempty"`
}

type InitiateRechargeResponse struct {
	RechargeID   uuid.UUID `json:"recharge_id"`
	PaymentRef   string    `json:"payment_ref"`
	PaymentURL   string    `json:"payment_url"`
	AmountKobo   int64     `json:"amount_kobo"`
	Network      string    `json:"network"`
	RechargeType string    `json:"recharge_type"`
}

// ── VTURecharge DB entity ─────────────────────────────────────────────────────

type VTURecharge struct {
	ID                uuid.UUID  `gorm:"column:id;primaryKey"`
	UserID            *uuid.UUID `gorm:"column:user_id"`
	MSISDN            string     `gorm:"column:msisdn"`
	Network           string     `gorm:"column:network"`
	RechargeType      string     `gorm:"column:recharge_type"`
	AmountKobo        int64      `gorm:"column:amount_kobo"`
	DataVariationCode string     `gorm:"column:data_variation_code"`
	PaymentReference  string     `gorm:"column:payment_reference"`
	VTPassRequestID   string     `gorm:"column:vtpass_request_id"`
	VTPassProviderRef string     `gorm:"column:vtpass_provider_ref"`
	PaystackEventID   string     `gorm:"column:paystack_event_id"`
	Status            string     `gorm:"column:status"`
	FailureReason     string     `gorm:"column:failure_reason"`
	Email             string     `gorm:"column:email"`
	// Reward fields — populated by markSuccess, returned by GetRechargeStatus
	PointsEarned  int64  `gorm:"column:points_earned"`
	DrawEntries   int    `gorm:"column:draw_entries"`
	SpinEligible  bool   `gorm:"column:spin_eligible"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
	CompletedAt   *time.Time `gorm:"column:completed_at"`
}

func (VTURecharge) TableName() string { return "recharges" }

// WebhookEventDedup — idempotency guard
type WebhookEventDedup struct {
	ID          uuid.UUID `gorm:"column:id;primaryKey"`
	Source      string    `gorm:"column:source"`
	EventID     string    `gorm:"column:event_id"`
	EventType   string    `gorm:"column:event_type"`
	ProcessedAt time.Time `gorm:"column:processed_at"`
}

func (WebhookEventDedup) TableName() string { return "webhook_events" }

// ── Constructor ───────────────────────────────────────────────────────────────

func NewVTURechargeService(
	db *gorm.DB,
	vtpass *external.VTPassHTTPClient,
	bundleSvc *external.NetworkBundleService,
	rechargeSvc *RechargeService,
	notifySvc *NotificationService,
) *VTURechargeService {
	return &VTURechargeService{
		db: db, vtpass: vtpass, bundleSvc: bundleSvc,
		rechargeSvc: rechargeSvc, notifySvc: notifySvc,
	}
}

// ── GetActiveNetworks ─────────────────────────────────────────────────────────

func (s *VTURechargeService) GetActiveNetworks(ctx context.Context) ([]external.NetworkResponse, error) {
	var rows []struct {
		NetworkCode    string `gorm:"column:network_code"`
		NetworkName    string `gorm:"column:network_name"`
		LogoURL        string `gorm:"column:logo_url"`
		BrandColor     string `gorm:"column:brand_color"`
		IsActive       bool   `gorm:"column:is_active"`
		AirtimeEnabled bool   `gorm:"column:airtime_enabled"`
		DataEnabled    bool   `gorm:"column:data_enabled"`
		SortOrder      int    `gorm:"column:sort_order"`
	}
	if err := s.db.WithContext(ctx).
		Table("network_operator_configs").
		Where("is_active = true").
		Order("sort_order ASC").
		Find(&rows).Error; err != nil {
		return []external.NetworkResponse{{
			Code: "MTN", Name: "MTN Nigeria", Logo: "/networks/mtn.png",
			BrandColor: "#FFCC00", IsActive: true, AirtimeEnabled: true, DataEnabled: true,
		}}, nil
	}
	out := make([]external.NetworkResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, external.NetworkResponse{
			Code: r.NetworkCode, Name: r.NetworkName, Logo: r.LogoURL,
			BrandColor: r.BrandColor, IsActive: r.IsActive,
			AirtimeEnabled: r.AirtimeEnabled, DataEnabled: r.DataEnabled, SortOrder: r.SortOrder,
		})
	}
	return out, nil
}

func (s *VTURechargeService) GetAllNetworksAdmin(ctx context.Context) ([]map[string]interface{}, error) {
	var rows []struct {
		ID             uuid.UUID `gorm:"column:id"`
		NetworkCode    string    `gorm:"column:network_code"`
		NetworkName    string    `gorm:"column:network_name"`
		LogoURL        string    `gorm:"column:logo_url"`
		BrandColor     string    `gorm:"column:brand_color"`
		IsActive       bool      `gorm:"column:is_active"`
		AirtimeEnabled bool      `gorm:"column:airtime_enabled"`
		DataEnabled    bool      `gorm:"column:data_enabled"`
		MinAmount      int64     `gorm:"column:min_amount"`
		MaxAmount      int64     `gorm:"column:max_amount"`
		SortOrder      int       `gorm:"column:sort_order"`
	}
	if err := s.db.WithContext(ctx).Table("network_operator_configs").Order("sort_order ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]interface{}{
			"id": r.ID, "network_code": r.NetworkCode, "network_name": r.NetworkName,
			"logo_url": r.LogoURL, "brand_color": r.BrandColor,
			"is_active": r.IsActive, "airtime_enabled": r.AirtimeEnabled,
			"data_enabled": r.DataEnabled, "min_amount_kobo": r.MinAmount,
			"max_amount_kobo": r.MaxAmount, "sort_order": r.SortOrder,
		})
	}
	return out, nil
}

func (s *VTURechargeService) UpdateNetworkConfig(ctx context.Context, networkCode string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	return s.db.WithContext(ctx).
		Table("network_operator_configs").
		Where("network_code = ?", strings.ToUpper(networkCode)).
		Updates(updates).Error
}

// ── GetBundles ────────────────────────────────────────────────────────────────

func (s *VTURechargeService) GetBundles(ctx context.Context, networkCode string) ([]external.DataBundleResponse, error) {
	return s.bundleSvc.GetBundles(ctx, strings.ToUpper(networkCode))
}

// ── InitiateRecharge ──────────────────────────────────────────────────────────

func (s *VTURechargeService) InitiateRecharge(ctx context.Context, req InitiateRechargeRequest) (*InitiateRechargeResponse, error) {
	req.Network = strings.ToUpper(req.Network)
	req.RechargeType = strings.ToUpper(req.RechargeType)
	msisdn := normaliseMSISDN(req.MSISDN)

	var activeCount int64
	s.db.WithContext(ctx).Table("network_operator_configs").
		Where("network_code = ? AND is_active = true", req.Network).Count(&activeCount)
	if activeCount == 0 {
		return nil, fmt.Errorf("network %s is not currently available", req.Network)
	}
	if req.AmountKobo < 10000 {
		return nil, fmt.Errorf("minimum recharge amount is ₦100")
	}
	if req.RechargeType == "DATA" && req.VariationCode == "" {
		return nil, fmt.Errorf("variation_code is required for data recharges")
	}

	payRef := fmt.Sprintf("NX_%s_%s", uuid.New().String()[:8], time.Now().Format("20060102150405"))
	email := req.Email
	if email == "" {
		email = "guest@loyaltynexus.ng"
	}

	recharge := &VTURecharge{
		ID: uuid.New(), UserID: req.UserID,
		MSISDN: msisdn, Network: req.Network,
		RechargeType:      req.RechargeType,
		AmountKobo:        req.AmountKobo,
		DataVariationCode: req.VariationCode,
		PaymentReference:  payRef,
		Status:            "PENDING",
		Email:             email,
		CreatedAt:         time.Now(), UpdatedAt: time.Now(),
	}
	if err := s.db.WithContext(ctx).Create(recharge).Error; err != nil {
		return nil, fmt.Errorf("create recharge record: %w", err)
	}

	paystackURL, err := s.initPaystack(ctx, payRef, email, req.AmountKobo)
	if err != nil {
		return nil, fmt.Errorf("paystack init: %w", err)
	}

	return &InitiateRechargeResponse{
		RechargeID: recharge.ID, PaymentRef: payRef, PaymentURL: paystackURL,
		AmountKobo: req.AmountKobo, Network: req.Network, RechargeType: req.RechargeType,
	}, nil
}

// ── HandlePaystackCallback ────────────────────────────────────────────────────
// GET /api/v1/recharge/callback?reference=REF
// Called by Paystack after the user completes (or cancels) payment.
// Verifies payment with Paystack, fires VTPass fulfillment in background,
// then redirects the browser to /recharge?payment=success&reference=REF
// (or /recharge?payment=failed on failure).
// The frontend polls GET /api/v1/recharge/status/{ref} for the final result.

func (s *VTURechargeService) HandlePaystackCallback(ctx context.Context, reference string) (string, error) {
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "https://loyalty-nexus.vercel.app"
	}

	if reference == "" {
		return frontendURL + "/recharge?payment=failed&reason=missing+reference", nil
	}

	// Check DB first — if webhook already processed this we skip the Paystack API call
	var recharge VTURecharge
	dbErr := s.db.WithContext(ctx).Where("payment_reference = ?", reference).First(&recharge).Error
	if dbErr != nil {
		return frontendURL + "/recharge?payment=failed&reference=" + url.QueryEscape(reference), nil
	}

	// If already SUCCESS (webhook beat us) — redirect with full result embedded
	if recharge.Status == "SUCCESS" {
		q := url.Values{}
		q.Set("payment", "success")
		q.Set("reference", reference)
		q.Set("txn_status", "SUCCESS")
		q.Set("amount", fmt.Sprintf("%d", recharge.AmountKobo/100))
		q.Set("points", fmt.Sprintf("%d", recharge.PointsEarned))
		q.Set("draw_entries", fmt.Sprintf("%d", recharge.DrawEntries))
		if recharge.SpinEligible { q.Set("spin_eligible", "true") }
		q.Set("msisdn", recharge.MSISDN)
		q.Set("network", recharge.Network)
		return frontendURL + "/recharge?" + q.Encode(), nil
	}

	// Verify payment with Paystack (only if not already confirmed by webhook)
	verified := s.verifyPaystackPayment(ctx, reference)
	if !verified {
		return frontendURL + "/recharge?payment=failed&reference=" + url.QueryEscape(reference), nil
	}

	// Atomically claim: PENDING → PROCESSING
	log.Printf("[VTU] callback: Paystack verified ref=%s — attempting atomic claim", reference)
	claim := s.db.WithContext(ctx).Model(&VTURecharge{}).
		Where("payment_reference = ? AND status = 'PENDING'", reference).
		Updates(map[string]interface{}{"status": "PROCESSING", "updated_at": time.Now()})
	if claim.Error != nil {
		log.Printf("[VTU] callback: DB claim error ref=%s: %v", reference, claim.Error)
	} else if claim.RowsAffected > 0 {
		log.Printf("[VTU] callback: claimed ref=%s — launching fulfil goroutine", reference)
		// Won the race — fire VTPass fulfillment in background
		go s.fulfil(context.Background(), &recharge)
	} else {
		log.Printf("[VTU] callback: ref=%s already claimed (RowsAffected=0) — skipping", reference)
	}

	// Redirect to frontend — frontend will poll for the result
	q := url.Values{}
	q.Set("payment", "success")
	q.Set("reference", reference)
	return frontendURL + "/recharge?" + q.Encode(), nil
}

// verifyPaystackPayment calls Paystack verify API to confirm a payment
func (s *VTURechargeService) verifyPaystackPayment(ctx context.Context, reference string) bool {
	secret := os.Getenv("PAYSTACK_SECRET_KEY")
	if secret == "" {
		log.Printf("[VTU] verifyPaystackPayment: PAYSTACK_SECRET_KEY not set — cannot verify")
		return false
	}
	log.Printf("[VTU] verifyPaystackPayment: checking ref=%s", reference)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.paystack.co/transaction/verify/"+url.PathEscape(reference), nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+secret)
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Status bool `json:"status"`
		Data   struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return false
	}
	return result.Status && result.Data.Status == "success"
}

// ── ProcessVTUPaystackWebhook ─────────────────────────────────────────────────

func (s *VTURechargeService) ProcessVTUPaystackWebhook(ctx context.Context, body []byte, signature string) error {
	if !s.verifySignature(body, signature) {
		return fmt.Errorf("invalid webhook signature")
	}

	var event struct {
		Event string `json:"event"`
		Data  struct {
			ID        int64  `json:"id"`
			Reference string `json:"reference"`
			Amount    int64  `json:"amount"`
			Status    string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &event); err != nil || event.Event != "charge.success" || event.Data.Status != "success" {
		return nil
	}
	ref := event.Data.Reference
	if !strings.HasPrefix(ref, "NX_") {
		return nil
	}

	eventID := fmt.Sprintf("paystack_%d", event.Data.ID)
	dedup := &WebhookEventDedup{
		ID: uuid.New(), Source: "paystack", EventID: eventID,
		EventType: event.Event, ProcessedAt: time.Now(),
	}
	if err := s.db.WithContext(ctx).Create(dedup).Error; err != nil {
		log.Printf("[VTU] duplicate paystack event %s ignored", eventID)
		return nil
	}

	var recharge VTURecharge
	if err := s.db.WithContext(ctx).Where("payment_reference = ?", ref).First(&recharge).Error; err != nil {
		log.Printf("[VTU] recharge not found: %s", ref)
		return nil
	}

	res := s.db.WithContext(ctx).Model(&VTURecharge{}).
		Where("payment_reference = ? AND status = 'PENDING'", ref).
		Updates(map[string]interface{}{"status": "PROCESSING", "paystack_event_id": eventID, "updated_at": time.Now()})
	if res.Error != nil || res.RowsAffected == 0 {
		return res.Error
	}

	go s.fulfil(context.Background(), &recharge)
	return nil
}

// ── fulfil ────────────────────────────────────────────────────────────────────

func (s *VTURechargeService) fulfil(ctx context.Context, recharge *VTURecharge) {
	amountNaira := int(recharge.AmountKobo / 100)
	log.Printf("[VTU] fulfil: ref=%s type=%s network=%s msisdn=%s amount=%dNGN",
		recharge.PaymentReference, recharge.RechargeType, recharge.Network, recharge.MSISDN, amountNaira)

	var vtResult *external.VTPassPurchaseResult
	var err error

	if recharge.RechargeType == "AIRTIME" {
		vtResult, err = s.vtpass.PurchaseAirtime(ctx, recharge.Network, recharge.MSISDN, amountNaira)
	} else {
		vtResult, err = s.vtpass.PurchaseData(ctx, recharge.Network, recharge.MSISDN, recharge.DataVariationCode, amountNaira)
	}

	if err != nil {
		log.Printf("[VTU] fulfil error for ref=%s: %v", recharge.PaymentReference, err)
		s.markFailed(ctx, recharge, err.Error(), true)
		return
	}

	log.Printf("[VTU] fulfil VTPass result: ref=%s reqID=%s success=%v pending=%v failed=%v desc=%q",
		recharge.PaymentReference, vtResult.RequestID, vtResult.Success, vtResult.Pending, vtResult.Failed, vtResult.Description)

	s.db.WithContext(ctx).Model(&VTURecharge{}).Where("id = ?", recharge.ID).
		Updates(map[string]interface{}{"vtpass_request_id": vtResult.RequestID, "vtpass_provider_ref": vtResult.ProviderRef})

	if vtResult.Pending {
		go s.requeryLoop(ctx, recharge, vtResult.RequestID)
		return
	}
	if vtResult.Failed {
		s.markFailed(ctx, recharge, vtResult.Description, true)
		return
	}
	s.markSuccess(ctx, recharge)
}

func (s *VTURechargeService) requeryLoop(ctx context.Context, recharge *VTURecharge, reqID string) {
	for i := 1; i <= 30; i++ {
		time.Sleep(30 * time.Second)
		var latest VTURecharge
		if s.db.WithContext(ctx).Where("id = ?", recharge.ID).First(&latest).Error != nil {
			return
		}
		if latest.Status != "PROCESSING" {
			return
		}
		vtResult, err := s.vtpass.RequeryTransaction(ctx, reqID)
		if err != nil {
			continue
		}
		if vtResult.Success {
			s.markSuccess(ctx, &latest)
			return
		}
		if vtResult.Failed {
			s.markFailed(ctx, &latest, vtResult.Description, true)
			return
		}
	}
	s.markFailed(ctx, recharge, "VTPass requery exhausted after 15 minutes — refund issued", true)
}

// ── markSuccess ───────────────────────────────────────────────────────────────
// Awards points, calculates draw entries and spin eligibility, persists all
// reward fields so GetRechargeStatus can return them to the frontend.

func (s *VTURechargeService) markSuccess(ctx context.Context, recharge *VTURecharge) {
	now := time.Now()
	amountNaira := recharge.AmountKobo / 100

	// Calculate rewards (₦250 = 1 Pulse Point, ₦200 = 1 draw entry, ₦1000+ = spin)
	// Double-points for direct platform recharges — per product spec.
	pointsEarned := amountNaira / 250 * 2 // 2× for platform recharge (vs. CDR path)
	drawEntries  := int(amountNaira / 200) // ₦200 = 1 draw entry
	spinEligible := amountNaira >= 1000    // ₦1,000+ unlocks spin wheel

	log.Printf("[VTU] markSuccess: ref=%s msisdn=%s amount=₦%d pts=%d drawEntries=%d spin=%v",
		recharge.PaymentReference, recharge.MSISDN, amountNaira, pointsEarned, drawEntries, spinEligible)

	// Persist reward fields + SUCCESS status on the recharge record
	if dbErr := s.db.WithContext(ctx).Model(&VTURecharge{}).Where("id = ?", recharge.ID).
		Updates(map[string]interface{}{
			"status":        "SUCCESS",
			"completed_at":  now,
			"updated_at":    now,
			"points_earned": pointsEarned,
			"draw_entries":  drawEntries,
			"spin_eligible": spinEligible,
		}).Error; dbErr != nil {
		log.Printf("[VTU] markSuccess: DB update failed ref=%s: %v", recharge.PaymentReference, dbErr)
	}

	// ── Award points in the platform wallet ledger ─────────────────────────
	// Uses FindByPhoneNumber which tries both "2348XXXXXXX" and "+2348XXXXXXX".
	// If rechargeSvc is not wired (should never happen in prod) we skip gracefully.
	if s.rechargeSvc == nil {
		log.Printf("[VTU] markSuccess: rechargeSvc is nil — skipping wallet award for ref=%s", recharge.PaymentReference)
	} else {
		user, findErr := s.rechargeSvc.userRepo.FindByPhoneNumber(ctx, recharge.MSISDN)
		if findErr != nil {
			log.Printf("[VTU] markSuccess: user not found for msisdn=%s (err=%v) — no wallet award for ref=%s",
				recharge.MSISDN, findErr, recharge.PaymentReference)
		} else {
			log.Printf("[VTU] markSuccess: found user id=%s for msisdn=%s — awarding %d pts",
				user.ID, recharge.MSISDN, pointsEarned)
			payRef := "vtu_" + recharge.PaymentReference
			if awErr := s.rechargeSvc.processAwardTransaction(ctx, user, recharge.AmountKobo, payRef, false); awErr != nil {
				log.Printf("[VTU] markSuccess: processAwardTransaction FAILED for msisdn=%s ref=%s: %v",
					recharge.MSISDN, recharge.PaymentReference, awErr)
			} else {
				log.Printf("[VTU] markSuccess: wallet award committed — user=%s pts=%d spin=%v",
					user.ID, pointsEarned, spinEligible)
				// Back-fill user_id on the recharge record if it was a guest session
				if recharge.UserID == nil {
					s.db.WithContext(ctx).Model(&VTURecharge{}).Where("id = ?", recharge.ID).
						Update("user_id", user.ID)
				}
			}
		}
	}

	// ── SMS notification ───────────────────────────────────────────────────
	if s.notifySvc != nil {
		displayPhone := recharge.MSISDN
		if len(displayPhone) == 13 && displayPhone[:3] == "234" {
			displayPhone = "0" + displayPhone[3:]
		}
		msg := fmt.Sprintf("✅ Your ₦%d %s recharge to %s is complete! You earned %d Pulse Points. Ref: %s",
			amountNaira, strings.ToUpper(recharge.RechargeType[:1])+strings.ToLower(recharge.RechargeType[1:]),
			displayPhone, pointsEarned, recharge.PaymentReference)
		go func() { _ = s.notifySvc.SendSMS(ctx, recharge.MSISDN, msg) }()
	}
	log.Printf("[VTU] ✓ SUCCESS ref=%s %s %s ₦%d pts=%d entries=%d spin=%v",
		recharge.PaymentReference, recharge.Network, recharge.MSISDN, amountNaira, pointsEarned, drawEntries, spinEligible)
}

func (s *VTURechargeService) markFailed(ctx context.Context, recharge *VTURecharge, reason string, doRefund bool) {
	s.db.WithContext(ctx).Model(&VTURecharge{}).Where("id = ?", recharge.ID).
		Updates(map[string]interface{}{
			"status":         "FAILED",
			"failure_reason": reason,
			"updated_at":     time.Now(),
		})
	if doRefund {
		go s.issueRefund(context.Background(), recharge, reason)
	}
}

func (s *VTURechargeService) issueRefund(ctx context.Context, recharge *VTURecharge, reason string) {
	secret := os.Getenv("PAYSTACK_SECRET_KEY")
	if secret == "" {
		return
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"transaction":   recharge.PaymentReference,
		"merchant_note": "VTU recharge failed: " + reason,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.paystack.co/refund", bytes.NewBuffer(payload))
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+secret)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil || resp.StatusCode >= 400 {
		log.Printf("[VTU] CRITICAL refund failed for %s — manual action required", recharge.PaymentReference)
		s.db.WithContext(ctx).Exec(`INSERT INTO audit_logs (id,entity_type,entity_id,action,description,created_at)
			VALUES (gen_random_uuid(),'recharge',$1,'REFUND_FAILED',$2,NOW())`,
			recharge.ID, "Manual refund required: "+recharge.PaymentReference)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	s.db.WithContext(ctx).Model(&VTURecharge{}).Where("id = ?", recharge.ID).Update("status", "CANCELLED")
	log.Printf("[VTU] refund issued for %s", recharge.PaymentReference)
}

// ── initPaystack ──────────────────────────────────────────────────────────────
// callback_url points to the BACKEND callback endpoint, which verifies payment,
// fires VTPass, and redirects the browser to the frontend.

func (s *VTURechargeService) initPaystack(ctx context.Context, ref, email string, amountKobo int64) (string, error) {
	secret := os.Getenv("PAYSTACK_SECRET_KEY")
	if secret == "" {
		return "", fmt.Errorf("PAYSTACK_SECRET_KEY not set")
	}
	backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		backendURL = "https://loyalty-nexus-api.onrender.com"
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"reference":    ref,
		"email":        email,
		"amount":       amountKobo,
		"callback_url": backendURL + "/api/v1/recharge/callback?reference=" + url.QueryEscape(ref),
		"metadata": map[string]interface{}{
			"custom_fields": []map[string]string{
				{"display_name": "Platform", "variable_name": "platform", "value": "LoyaltyNexus VTU"},
			},
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.paystack.co/transaction/initialize", bytes.NewBuffer(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+secret)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Status  bool   `json:"status"`
		Data    struct{ AuthorizationURL string `json:"authorization_url"` } `json:"data"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil || !result.Status {
		return "", fmt.Errorf("paystack: %s", result.Message)
	}
	return result.Data.AuthorizationURL, nil
}

func (s *VTURechargeService) verifySignature(body []byte, sig string) bool {
	secret := os.Getenv("PAYSTACK_WEBHOOK_SECRET")
	if secret == "" {
		log.Printf("[VTU] PAYSTACK_WEBHOOK_SECRET not set — skipping signature check (sandbox mode)")
		return true
	}
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal([]byte(hex.EncodeToString(mac.Sum(nil))), []byte(sig))
}

// ── GetRechargeStatus ─────────────────────────────────────────────────────────
// Returns full recharge status including reward fields.
// Used by the frontend polling loop after payment.

func (s *VTURechargeService) GetRechargeStatus(ctx context.Context, ref string) (map[string]interface{}, error) {
	var recharge VTURecharge
	if err := s.db.WithContext(ctx).Where("payment_reference = ?", ref).First(&recharge).Error; err != nil {
		return nil, fmt.Errorf("recharge not found")
	}
	// Convert 234XXXXXXXXXX → 0XXXXXXXXXX for display
	displayPhone := recharge.MSISDN
	if len(displayPhone) == 13 && displayPhone[:3] == "234" {
		displayPhone = "0" + displayPhone[3:]
	}
	return map[string]interface{}{
		"reference":      recharge.PaymentReference,
		"status":         recharge.Status,
		"network":        recharge.Network,
		"msisdn":         displayPhone,
		"amount_kobo":    recharge.AmountKobo,
		"type":           recharge.RechargeType,
		"failure_reason": recharge.FailureReason,
		"points_earned":  recharge.PointsEarned,
		"draw_entries":   recharge.DrawEntries,
		"spin_eligible":  recharge.SpinEligible,
	}, nil
}

// ── normaliseMSISDN ───────────────────────────────────────────────────────────

func normaliseMSISDN(phone string) string {
	digits := ""
	for _, ch := range phone {
		if ch >= '0' && ch <= '9' {
			digits += string(ch)
		}
	}
	if len(digits) == 11 && digits[0] == '0' {
		return "234" + digits[1:]
	}
	if len(digits) == 13 && digits[:3] == "234" {
		return digits
	}
	return digits
}
