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
	"os"
	"strings"
	"time"

	"loyalty-nexus/internal/infrastructure/external"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ─── VTU Recharge Service ────────────────────────────────────────────────────
// Handles platform-initiated recharges (user pays via Paystack → VTPass tops up).
// Public endpoints — no authentication required to recharge.
//
// Double-points design: when a user recharges on the platform AND MTN sends the
// CDR event, they earn points from both sources. This is intentional — "earn
// double points when you recharge on Loyalty Nexus" is the marketing hook.

type VTURechargeService struct {
	db          *gorm.DB
	vtpass      *external.VTPassHTTPClient
	bundleSvc   *external.NetworkBundleService
	rechargeSvc *RechargeService // for processAwardTransaction (double points)
	notifySvc   *NotificationService
}

// ── DTOs ──────────────────────────────────────────────────────────────────────

type InitiateRechargeRequest struct {
	MSISDN        string     `json:"msisdn"`
	Network       string     `json:"network"`
	RechargeType  string     `json:"recharge_type"`  // "airtime" | "data"
	AmountKobo    int64      `json:"amount_kobo"`
	VariationCode string     `json:"variation_code"` // data only; VTPass variation_code
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

// ── Recharge DB entity ────────────────────────────────────────────────────────

type VTURecharge struct {
	ID                uuid.UUID  `gorm:"column:id;primaryKey"`
	UserID            *uuid.UUID `gorm:"column:user_id"`
	MSISDN            string     `gorm:"column:msisdn"`
	Network           string     `gorm:"column:network"`
	RechargeType      string     `gorm:"column:recharge_type"`
	AmountKobo        int64      `gorm:"column:amount_kobo"`
	DataVariationCode string     `gorm:"column:data_variation_code"` // GAP fix: actual VTPass variation_code
	PaymentReference  string     `gorm:"column:payment_reference"`
	VTPassRequestID   string     `gorm:"column:vtpass_request_id"`
	VTPassProviderRef string     `gorm:"column:vtpass_provider_ref"`
	PaystackEventID   string     `gorm:"column:paystack_event_id"`
	Status            string     `gorm:"column:status"`
	FailureReason     string     `gorm:"column:failure_reason"`
	Email             string     `gorm:"column:email"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at"`
	CompletedAt       *time.Time `gorm:"column:completed_at"`
}

func (VTURecharge) TableName() string { return "recharges" }

// WebhookEventDedup — GAP-2 idempotency guard
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
		db:          db,
		vtpass:      vtpass,
		bundleSvc:   bundleSvc,
		rechargeSvc: rechargeSvc,
		notifySvc:   notifySvc,
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

// GetAllNetworksAdmin returns all networks including inactive ones (for admin panel).
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

// UpdateNetworkConfig updates a network's toggles (admin only).
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

	// Validate network is active
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
		DataVariationCode: req.VariationCode, // GAP fix: store the actual variation code
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

// ── ProcessVTUPaystackWebhook ─────────────────────────────────────────────────
// Handles Paystack charge.success for VTU recharges only (prefix NX_).
// Called from the existing PaystackWebhook handler after signature verification.
// GAP-2: idempotent via webhook_events dedup table.

func (s *VTURechargeService) ProcessVTUPaystackWebhook(ctx context.Context, body []byte, signature string) error {
	// 1. Verify HMAC-SHA512 signature
	if !s.verifySignature(body, signature) {
		return fmt.Errorf("invalid webhook signature")
	}

	// 2. Parse
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
		return nil // not a VTU recharge
	}

	// 3. GAP-2: idempotency guard
	eventID := fmt.Sprintf("paystack_%d", event.Data.ID)
	dedup := &WebhookEventDedup{
		ID: uuid.New(), Source: "paystack", EventID: eventID,
		EventType: event.Event, ProcessedAt: time.Now(),
	}
	if err := s.db.WithContext(ctx).Create(dedup).Error; err != nil {
		log.Printf("[VTU] duplicate paystack event %s ignored", eventID)
		return nil
	}

	// 4. Find recharge
	var recharge VTURecharge
	if err := s.db.WithContext(ctx).Where("payment_reference = ?", ref).First(&recharge).Error; err != nil {
		log.Printf("[VTU] recharge not found: %s", ref)
		return nil
	}

	// 5. Atomic claim
	res := s.db.WithContext(ctx).Model(&VTURecharge{}).
		Where("payment_reference = ? AND status = 'PENDING'", ref).
		Updates(map[string]interface{}{"status": "PROCESSING", "paystack_event_id": eventID})
	if res.Error != nil || res.RowsAffected == 0 {
		return res.Error
	}

	// 6. Fulfil async
	go s.fulfil(context.Background(), &recharge)
	return nil
}

// ── fulfil ────────────────────────────────────────────────────────────────────

func (s *VTURechargeService) fulfil(ctx context.Context, recharge *VTURecharge) {
	amountNaira := int(recharge.AmountKobo / 100)
	var vtResult *external.VTPassPurchaseResult
	var err error

	if recharge.RechargeType == "AIRTIME" {
		vtResult, err = s.vtpass.PurchaseAirtime(ctx, recharge.Network, recharge.MSISDN, amountNaira)
	} else {
		// GAP fix: DataVariationCode is the real VTPass variation_code (e.g. "mtn-10mb-100")
		vtResult, err = s.vtpass.PurchaseData(ctx, recharge.Network, recharge.MSISDN, recharge.DataVariationCode, amountNaira)
	}

	if err != nil {
		s.markFailed(ctx, recharge, err.Error(), true)
		return
	}

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

func (s *VTURechargeService) markSuccess(ctx context.Context, recharge *VTURecharge) {
	now := time.Now()
	s.db.WithContext(ctx).Model(&VTURecharge{}).Where("id = ?", recharge.ID).
		Updates(map[string]interface{}{"status": "SUCCESS", "completed_at": now, "updated_at": now})

	// Award points (double-points design: Paystack path + MTN CDR path both award)
	if s.rechargeSvc != nil {
		user, err := s.rechargeSvc.userRepo.FindByPhoneNumber(ctx, recharge.MSISDN)
		if err == nil {
			payRef := "vtu_" + recharge.PaymentReference
			if awErr := s.rechargeSvc.processAwardTransaction(ctx, user, recharge.AmountKobo, payRef, false); awErr != nil {
				log.Printf("[VTU] point award error for %s: %v", recharge.MSISDN, awErr)
			}
		}
	}

	if s.notifySvc != nil {
		msg := fmt.Sprintf("Your ₦%d %s recharge is complete! 🎉 You've earned double Pulse Points. Ref: %s",
			recharge.AmountKobo/100, recharge.RechargeType, recharge.PaymentReference)
		go func() { _ = s.notifySvc.SendSMS(ctx, recharge.MSISDN, msg) }()
	}
	log.Printf("[VTU] ✓ success %s %s ₦%d", recharge.Network, recharge.MSISDN, recharge.AmountKobo/100)
}

func (s *VTURechargeService) markFailed(ctx context.Context, recharge *VTURecharge, reason string, doRefund bool) {
	s.db.WithContext(ctx).Model(&VTURecharge{}).Where("id = ?", recharge.ID).
		Updates(map[string]interface{}{"status": "FAILED", "failure_reason": reason, "updated_at": time.Now()})
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
	defer resp.Body.Close()
	s.db.WithContext(ctx).Model(&VTURecharge{}).Where("id = ?", recharge.ID).
		Update("status", "CANCELLED")
	log.Printf("[VTU] refund issued for %s", recharge.PaymentReference)
}

// ── Paystack init ─────────────────────────────────────────────────────────────

func (s *VTURechargeService) initPaystack(ctx context.Context, ref, email string, amountKobo int64) (string, error) {
	secret := os.Getenv("PAYSTACK_SECRET_KEY")
	if secret == "" {
		return "", fmt.Errorf("PAYSTACK_SECRET_KEY not set")
	}
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "https://loyalty-nexus.vercel.app"
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"reference":    ref,
		"email":        email,
		"amount":       amountKobo,
		"callback_url": frontendURL + "/recharge/success?ref=" + ref,
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
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Status bool   `json:"status"`
		Data   struct{ AuthorizationURL string `json:"authorization_url"` } `json:"data"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil || !result.Status {
		return "", fmt.Errorf("paystack: %s", result.Message)
	}
	return result.Data.AuthorizationURL, nil
}

func (s *VTURechargeService) verifySignature(body []byte, sig string) bool {
	secret := os.Getenv("PAYSTACK_SECRET_KEY")
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal([]byte(hex.EncodeToString(mac.Sum(nil))), []byte(sig))
}

// normaliseMSISDN converts 080XXXXXXXX → 234XXXXXXXXXX
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
