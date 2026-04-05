package services

// winner_service.go — Full prize lifecycle and MoMo disbursement flow
// NEW — does not exist in RechargeMax. Built from spec §8 (MoMo Prize Fulfillment).
//
// Happy path (spec §8.1):
//   1. User wins "₦500 MoMo Cash" on the spin wheel
//   2. If MoMo not linked → FulfillPendingMoMo, hold 48h
//   3. Once linked & verified → MTN MoMo Disbursement API POST /v1_0/transfer
//   4. Poll until SUCCESSFUL → mark completed, send SMS
//
// Edge cases (spec §8.2):
//   - No MoMo account: hold prize for 48h, instruct user to dial *671#
//   - API timeout: exponential backoff, max 3 retries
//   - Duplicate prevention: X-Reference-Id = spin_results.id (idempotent MoMo API)
//   - Admin alert when all retries fail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
)

// ─── MoMo API config ──────────────────────────────────────────────────────

type MoMoDisbursement struct {
	BaseURL         string
	SubscriptionKey string
	APIUser         string
	APIKey          string
	Environment     string // sandbox | production
}

func NewMoMoDisbursement() *MoMoDisbursement {
	env := os.Getenv("MOMO_ENVIRONMENT")
	if env == "" {
		env = "sandbox"
	}
	base := "https://sandbox.momodeveloper.mtn.com"
	if env == "production" {
		base = "https://proxy.momoapi.mtn.com"
	}
	return &MoMoDisbursement{
		BaseURL:         base,
		SubscriptionKey: os.Getenv("MOMO_SUBSCRIPTION_KEY"),
		APIUser:         os.Getenv("MOMO_API_USER"),
		APIKey:          os.Getenv("MOMO_API_KEY"),
		Environment:     env,
	}
}

// ─── WinnerService ────────────────────────────────────────────────────────

type WinnerService struct {
	db        *gorm.DB
	userRepo  repositories.UserRepository
	prizeRepo repositories.PrizeRepository
	notifySvc *NotificationService
	momo      *MoMoDisbursement
}

func NewWinnerService(
	db *gorm.DB,
	userRepo repositories.UserRepository,
	prizeRepo repositories.PrizeRepository,
	notifySvc *NotificationService,
) *WinnerService {
	return &WinnerService{
		db:        db,
		userRepo:  userRepo,
		prizeRepo: prizeRepo,
		notifySvc: notifySvc,
		momo:      NewMoMoDisbursement(),
	}
}

// ─── MoMo Token ──────────────────────────────────────────────────────────

type momoToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func (w *WinnerService) getMoMoToken(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/disbursement/token/", w.momo.BaseURL)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	req.SetBasicAuth(w.momo.APIUser, w.momo.APIKey)
	req.Header.Set("Ocp-Apim-Subscription-Key", w.momo.SubscriptionKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("momo token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("momo token failed (%d): %s", resp.StatusCode, string(body))
	}
	var tok momoToken
	if err := json.Unmarshal(body, &tok); err != nil {
		return "", fmt.Errorf("momo token decode: %w", err)
	}
	return tok.AccessToken, nil
}

// ─── MoMo Account Verification ───────────────────────────────────────────

// VerifyMoMoAccount calls MTN MoMo GET /accountholder to check if a number is active.
func (w *WinnerService) VerifyMoMoAccount(ctx context.Context, msisdn string) (bool, error) {
	token, err := w.getMoMoToken(ctx)
	if err != nil {
		return false, err
	}

	url := fmt.Sprintf("%s/disbursement/v1_0/accountholder/msisdn/%s/active", w.momo.BaseURL, msisdn)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Ocp-Apim-Subscription-Key", w.momo.SubscriptionKey)
	req.Header.Set("X-Target-Environment", w.momo.Environment)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("momo account check: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)

	// Sandbox returns 200 with {"result": true} for active accounts
	var result struct {
		Result bool `json:"result"`
	}
	if resp.StatusCode == http.StatusOK {
		if err := json.Unmarshal(body, &result); err == nil {
			return result.Result, nil
		}
		// Some environments return bare `true`
		return string(bytes.TrimSpace(body)) == "true", nil
	}
	return false, fmt.Errorf("account check failed (%d): %s", resp.StatusCode, string(body))
}

// ─── MoMo Transfer ────────────────────────────────────────────────────────

type momoTransferRequest struct {
	Amount      string `json:"amount"`
	Currency    string `json:"currency"`
	ExternalID  string `json:"externalId"`
	Payee       struct {
		PartyIDType string `json:"partyIdType"`
		PartyID     string `json:"partyId"`
	} `json:"payee"`
	PayerMessage string `json:"payerMessage"`
	PayeeNote    string `json:"payeeNote"`
}

type momoTransferStatus struct {
	Amount               string `json:"amount"`
	Currency             string `json:"currency"`
	FinancialTransactionID string `json:"financialTransactionId,omitempty"`
	ExternalID           string `json:"externalId"`
	Status               string `json:"status"` // PENDING | SUCCESSFUL | FAILED
	Reason               *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"reason,omitempty"`
}

// DisburseToMoMo initiates a MoMo cash transfer (spec §8.1 step 7).
// Uses spin_results.id as X-Reference-Id (idempotent — prevents double payment on retry).
func (w *WinnerService) DisburseToMoMo(ctx context.Context, spinResultID uuid.UUID, amountNaira float64, msisdn string) error {
	token, err := w.getMoMoToken(ctx)
	if err != nil {
		return err
	}

	body := momoTransferRequest{
		Amount:      fmt.Sprintf("%.2f", amountNaira),
		Currency:    "NGN",
		ExternalID:  spinResultID.String(), // idempotency key
		PayerMessage: "Loyalty Nexus Spin Prize",
		PayeeNote:   "Congratulations on your win!",
	}
	body.Payee.PartyIDType = "MSISDN"
	body.Payee.PartyID = msisdn

	bodyBytes, _ := json.Marshal(body)
	url := fmt.Sprintf("%s/disbursement/v1_0/transfer", w.momo.BaseURL)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Reference-Id", spinResultID.String())
	req.Header.Set("X-Target-Environment", w.momo.Environment)
	req.Header.Set("Ocp-Apim-Subscription-Key", w.momo.SubscriptionKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("momo transfer initiation: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(resp.Body)

	// 202 Accepted = transfer initiated successfully
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("momo transfer failed (%d): %s", resp.StatusCode, string(respBody))
	}

	// Poll for completion with exponential backoff (spec §8.2 — max 3 retries)
	return w.pollMoMoTransfer(ctx, token, spinResultID.String())
}

// pollMoMoTransfer polls GET /v1_0/transfer/{referenceId} until status is SUCCESSFUL or FAILED.
func (w *WinnerService) pollMoMoTransfer(ctx context.Context, token, referenceID string) error {
	maxAttempts := 3
	baseDelay := 2 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		url := fmt.Sprintf("%s/disbursement/v1_0/transfer/%s", w.momo.BaseURL, referenceID)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-Target-Environment", w.momo.Environment)
		req.Header.Set("Ocp-Apim-Subscription-Key", w.momo.SubscriptionKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("[MOMO] Poll attempt %d error: %v", attempt, err)
			time.Sleep(baseDelay * time.Duration(attempt*attempt)) // exponential backoff
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		var status momoTransferStatus
		if err := json.Unmarshal(body, &status); err != nil {
			time.Sleep(baseDelay * time.Duration(attempt*attempt))
			continue
		}

		switch status.Status {
		case "SUCCESSFUL":
			log.Printf("[MOMO] Transfer %s SUCCESSFUL (txId: %s)", referenceID, status.FinancialTransactionID)
			return nil
		case "FAILED":
			reason := "UNKNOWN"
			if status.Reason != nil {
				reason = status.Reason.Message
			}
			return fmt.Errorf("momo transfer FAILED: %s", reason)
		default:
			// PENDING — wait and retry
			log.Printf("[MOMO] Transfer %s still PENDING (attempt %d/%d)", referenceID, attempt, maxAttempts)
			time.Sleep(baseDelay * time.Duration(attempt*attempt))
		}
	}
	return fmt.Errorf("momo transfer polling timed out after %d attempts", maxAttempts)
}

// ─── Full Prize Fulfillment Flow ─────────────────────────────────────────

// FulfillCashPrize orchestrates the full MoMo disbursement lifecycle (spec §8).
// Called by PrizeFulfillmentService when prize_type = momo_cash.
func (w *WinnerService) FulfillCashPrize(ctx context.Context, spinResult *entities.SpinResult) error {
	user, err := w.userRepo.FindByID(ctx, spinResult.UserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Case 1: User has no MoMo linked → hold prize 48h
	if !user.MoMoVerified || user.MoMoNumber == "" {
		heldUntil := time.Now().Add(48 * time.Hour)
		_ = w.db.Table("spin_results").Where("id = ?", spinResult.ID).Updates(map[string]interface{}{
			"fulfillment_status": entities.FulfillPendingMoMo,
			"error_message":      fmt.Sprintf("Held until %s. User must set up MoMo.", heldUntil.Format(time.RFC3339)),
			"updated_at":         time.Now(),
		}).Error

		// Send SMS with instructions (spec §8.2)
		_ = w.notifySvc.SendSMS(ctx, user.PhoneNumber, fmt.Sprintf(
			"You won ₦%.0f on Loyalty Nexus! Dial *671# on your MTN line to open a MoMo account. Your prize is held for 48 hours.",
			spinResult.PrizeValue,
		))
		return nil // Not an error — prize is legitimately held
	}

	// Case 2: MoMo is linked — verify the account is still active
	active, err := w.VerifyMoMoAccount(ctx, user.MoMoNumber)
	if err != nil {
		log.Printf("[WINNER] MoMo verification failed for user %s: %v", user.ID, err)
		// Treat as transient — retry on next worker run
		return fmt.Errorf("momo verification failed (will retry): %w", err)
	}
	if !active {
		_ = w.db.Table("spin_results").Where("id = ?", spinResult.ID).Updates(map[string]interface{}{
			"fulfillment_status": entities.FulfillPendingMoMo,
			"error_message":      "MoMo account not active",
			"updated_at":         time.Now(),
		}).Error
		_ = w.notifySvc.SendSMS(ctx, user.PhoneNumber,
			"Your MoMo account is not active. Please activate it to receive your prize.")
		return nil
	}

	// Case 3: Disburse (with retry logic built into DisburseToMoMo)
	_ = w.db.Table("spin_results").Where("id = ?", spinResult.ID).Updates(map[string]interface{}{
		"fulfillment_status": entities.FulfillProcessing,
		"momo_number":        user.MoMoNumber,
		"updated_at":         time.Now(),
	}).Error

	if err := w.DisburseToMoMo(ctx, spinResult.ID, spinResult.PrizeValue, user.MoMoNumber); err != nil {
		log.Printf("[WINNER] Disbursement failed for spin %s: %v", spinResult.ID, err)

		// Mark as failed + admin alert
		_ = w.db.Table("spin_results").Where("id = ?", spinResult.ID).Updates(map[string]interface{}{
			"fulfillment_status": entities.FulfillFailed,
			"error_message":      err.Error(),
			"retry_count":        gorm.Expr("retry_count + 1"),
			"updated_at":         time.Now(),
		}).Error

		// Admin alert (insert into admin_alerts table if exists)
		_ = w.db.Exec(`
			INSERT INTO fraud_events (id, user_id, event_type, severity, details, created_at)
			VALUES (?, ?, 'momo_disbursement_failed', 'high', ?, NOW())
		`, uuid.New(), spinResult.UserID, fmt.Sprintf("MoMo disbursement failed for spin %s: %s", spinResult.ID, err.Error())).Error

		return fmt.Errorf("disbursement failed: %w", err)
	}

	// Success
	now := time.Now()
	_ = w.db.Table("spin_results").Where("id = ?", spinResult.ID).Updates(map[string]interface{}{
		"fulfillment_status": entities.FulfillCompleted,
		"error_message":      "",
		"fulfilled_at":       now,
		"updated_at":         now,
	}).Error

	// SMS confirmation (spec §8.1 step 9)
	_ = w.notifySvc.SendSMS(ctx, user.PhoneNumber, fmt.Sprintf(
		"₦%.0f has been sent to your MoMo wallet (%s). Congratulations from Loyalty Nexus!",
		spinResult.PrizeValue, user.MoMoNumber,
	))

	return nil
}

// ─── Draw Winner Fulfillment ──────────────────────────────────────────────

// FulfillDrawWinner fulfills a prize for a monthly draw winner.
func (w *WinnerService) FulfillDrawWinner(ctx context.Context, winnerID uuid.UUID) error {
	var winner DrawWinner
	if err := w.db.WithContext(ctx).Table("draw_winners").Where("id = ?", winnerID).First(&winner).Error; err != nil {
		return fmt.Errorf("draw winner not found: %w", err)
	}
	if winner.ClaimStatus == "FULFILLED" {
		return fmt.Errorf("already fulfilled")
	}

	user, err := w.userRepo.FindByID(ctx, winner.UserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	amountNaira := float64(winner.PrizeValue) / 100 // kobo → naira
	if err := w.DisburseToMoMo(ctx, winner.ID, amountNaira, user.MoMoNumber); err != nil {
		_ = w.db.Table("draw_winners").Where("id = ?", winnerID).Update("claim_status", "FAILED")
		return err
	}

	_ = w.db.Table("draw_winners").Where("id = ?", winnerID).Update("claim_status", "FULFILLED")

	_ = w.notifySvc.SendSMS(ctx, user.PhoneNumber, fmt.Sprintf(
		"Congratulations! ₦%.0f draw prize sent to your MoMo wallet. Thank you for using Loyalty Nexus!",
		amountNaira,
	))
	return nil
}

// ─── Held Prize Recovery ──────────────────────────────────────────────────

// ProcessHeldPrizes retries disbursement for prizes held due to MoMo issues.
// Called by the lifecycle worker every hour.
func (w *WinnerService) ProcessHeldPrizes(ctx context.Context) error {
	var held []entities.SpinResult
	err := w.db.WithContext(ctx).
		Table("spin_results").
		Where("fulfillment_status = ? AND prize_type = ? AND created_at > ?",
			string(entities.FulfillPendingMoMo),
			string(entities.PrizeMoMoCash),
			time.Now().Add(-48*time.Hour),
		).
		Limit(50).
		Find(&held).Error
	if err != nil {
		return err
	}

	for _, spinResult := range held {
		if err := w.FulfillCashPrize(ctx, &spinResult); err != nil {
			log.Printf("[WINNER] Held prize retry failed for spin %s: %v", spinResult.ID, err)
		}
	}
	return nil
}

// ExpireHeldPrizes marks prizes held >48h as expired and notifies the user.
func (w *WinnerService) ExpireHeldPrizes(ctx context.Context) error {
	var expired []entities.SpinResult
	err := w.db.WithContext(ctx).
		Table("spin_results").
		Where("fulfillment_status = ? AND prize_type = ? AND created_at < ?",
			string(entities.FulfillPendingMoMo),
			string(entities.PrizeMoMoCash),
			time.Now().Add(-48*time.Hour),
		).
		Find(&expired).Error
	if err != nil {
		return err
	}

	for _, s := range expired {
		_ = w.db.Table("spin_results").Where("id = ?", s.ID).Updates(map[string]interface{}{
			"fulfillment_status": entities.FulfillFailed,
			"error_message":      "Prize expired: MoMo not set up within 48 hours",
			"updated_at":         time.Now(),
		})
		user, err := w.userRepo.FindByID(ctx, s.UserID)
		if err == nil {
			_ = w.notifySvc.SendSMS(ctx, user.PhoneNumber,
				"Your unclaimed prize has expired. Keep recharging for more chances to win!")
		}
	}
	return nil
}
