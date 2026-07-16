package services

// studio_service.go — Application-layer orchestration for Nexus Studio.
//
// Responsibilities:
//   1. Gate-keep point deduction (PulsePoints only — zero-hardcoding via DB)
//   2. Create AIGeneration job record inside an atomic GORM transaction
//   3. Delegate provider dispatch to AIStudioOrchestrator (async)
//   4. Compensate on failure (TxTypeStudioRefund)
//   5. Notify user when asset is ready
//
// Financial rule: PointCost is read from studio_tools.point_cost (DB), never
// hardcoded.  Admin can change it via UpdateToolCost without a code deploy.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
)

// StudioService is injected into HTTP handlers and the async worker.
type StudioService struct {
	studioRepo  repositories.StudioRepository
	userRepo    repositories.UserRepository
	txRepo      repositories.TransactionRepository
	notifySvc   *NotificationService
	settingsSvc *SettingsService
	db          *gorm.DB
}

func NewStudioService(
	sr repositories.StudioRepository,
	ur repositories.UserRepository,
	tr repositories.TransactionRepository,
	ns *NotificationService,
	_ interface{}, // kept for legacy call-site compatibility (was monetSvc)
	db *gorm.DB,
) *StudioService {
	return &StudioService{
		studioRepo: sr,
		userRepo:   ur,
		txRepo:     tr,
		notifySvc:  ns,
		db:         db,
	}
}

// SetSettingsService injects the SettingsService for admin-configurable TTL.
func (s *StudioService) SetSettingsService(ss *SettingsService) { s.settingsSvc = ss }

// storageTTL returns the admin-configured asset TTL for the given tier.
// Falls back to 48h if SettingsService is not wired or DB is unavailable.
func (s *StudioService) storageTTL(ctx context.Context, tier string) time.Duration {
	if s.settingsSvc != nil {
		return s.settingsSvc.StorageTTL(ctx, tier)
	}
	return 48 * time.Hour // safe default
}

// ─── Read-only queries ────────────────────────────────────────────────────────

func (s *StudioService) ListActiveTools(ctx context.Context) ([]entities.StudioTool, error) {
	return s.studioRepo.ListActiveTools(ctx)
}

func (s *StudioService) FindToolByID(ctx context.Context, id uuid.UUID) (*entities.StudioTool, error) {
	return s.studioRepo.FindToolByID(ctx, id)
}

func (s *StudioService) FindToolBySlug(ctx context.Context, slug string) (*entities.StudioTool, error) {
	return s.studioRepo.FindToolBySlug(ctx, slug)
}

func (s *StudioService) FindGenerationByID(ctx context.Context, id uuid.UUID) (*entities.AIGeneration, error) {
	return s.studioRepo.FindGenerationByID(ctx, id)
}

func (s *StudioService) FindGenerationBySlug(ctx context.Context, slug string) (*entities.AIGeneration, error) {
	return s.studioRepo.FindGenerationBySlug(ctx, slug)
}

func (s *StudioService) SlugExists(ctx context.Context, slug string) (bool, error) {
	return s.studioRepo.SlugExists(ctx, slug)
}

func (s *StudioService) SetVanitySlug(ctx context.Context, id uuid.UUID, slug string) error {
	return s.studioRepo.SetVanitySlug(ctx, id, slug)
}

func (s *StudioService) GetUserGallery(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entities.AIGeneration, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.studioRepo.GetUserGallery(ctx, userID, limit, offset)
}

func (s *StudioService) CountUserGenerationsToday(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.studioRepo.CountUserGenerationsToday(ctx, userID)
}

// ─── Job creation (atomic) ────────────────────────────────────────────────────

// RequestGeneration creates an AIGeneration job, deducts PulsePoints from the
// user's wallet, and writes an immutable ledger transaction — all in one DB txn.
// Returns the pending job; caller must dispatch it to AIStudioOrchestrator async.
//
// dailyLimit is the caller's configured cap (from network_configs). Pass 0 to
// skip the in-transaction quota check (the handler-layer check still applies).
func (s *StudioService) RequestGeneration(
	ctx context.Context,
	userID uuid.UUID,
	toolID uuid.UUID,
	prompt string,
	dailyLimit int,
) (*entities.AIGeneration, error) {

	// 1. Resolve tool (reads point_cost from DB — never hardcoded)
	tool, err := s.studioRepo.FindToolByID(ctx, toolID)
	if err != nil {
		return nil, fmt.Errorf("tool not found: %w", err)
	}
	if !tool.IsActive {
		return nil, fmt.Errorf("tool %q is currently unavailable", tool.Name)
	}

	// 2. Resolve user + wallet
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// 3. Check IsFree — skip wallet checks for free tools but still enforce daily quota
	if tool.IsFree {
		// Build generation record with zero cost
		now := time.Now()
		gen := &entities.AIGeneration{
			ID:             uuid.New(),
			UserID:         userID,
			ToolID:         toolID,
			ToolSlug:       tool.Slug,
			Prompt:         prompt,
			Status:         "pending",
			PointsDeducted: 0,
			CreatedAt:      now,
			UpdatedAt:      now,
			ExpiresAt:      now.Add(s.storageTTL(ctx, user.Tier)), // admin-configurable per tier
		}
		err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// Enforce daily quota inside the transaction for free tools too
			if dailyLimit > 0 {
				var countNow int64
				today := time.Now().UTC().Truncate(24 * time.Hour)
				if cntErr := tx.Table("ai_generations").
					Where("user_id = ? AND created_at >= ?", userID, today).
					Count(&countNow).Error; cntErr != nil {
					return fmt.Errorf("quota check failed: %w", cntErr)
				}
				if int(countNow) >= dailyLimit {
					return fmt.Errorf("daily generation limit reached (%d/%d)", countNow, dailyLimit)
				}
			}
			if err := s.studioRepo.CreateGenerationTx(ctx, tx, gen); err != nil {
				return fmt.Errorf("create generation: %w", err)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		// Best-effort session tracking for free tools
		if sess, sessErr := s.studioRepo.GetOrCreateActiveSession(ctx, userID); sessErr == nil {
			_ = s.studioRepo.UpdateSession(ctx, sess.ID, 0)
		}
		return gen, nil
	}

	wallet, err := s.userRepo.GetWalletForUpdate(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("wallet not found: %w", err)
	}

	// 4. Entry threshold check (minimum balance to open the tool)
	if wallet.PulsePoints < tool.EntryPointCost {
		return nil, fmt.Errorf("insufficient PulsePoints to access %q: need %d to unlock, have %d",
			tool.Name, tool.EntryPointCost, wallet.PulsePoints)
	}

	// 5. Enforce PulsePoint balance for generation cost (Financial Rule: PulsePoints ≠ SpinCredits)
	if wallet.PulsePoints < tool.PointCost {
		return nil, fmt.Errorf("insufficient PulsePoints: need %d, have %d",
			tool.PointCost, wallet.PulsePoints)
	}

	// 6. Build generation record
	now := time.Now()
	gen := &entities.AIGeneration{
		ID:             uuid.New(),
		UserID:         userID,
		ToolID:         toolID,
		ToolSlug:       tool.Slug,
		Prompt:         prompt,
		Status:         "pending",
		PointsDeducted: tool.PointCost,
		CreatedAt:      now,
		UpdatedAt:      now,
		ExpiresAt:      now.Add(s.storageTTL(ctx, user.Tier)), // admin-configurable per tier
	}

	// 7. Atomic: deduct wallet + ledger entry + create job
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		if tool.PointCost > 0 {
			// Deduct PulsePoints with a floor guard.
			// IMPORTANT: we must NOT use tx.Save(wallet) here — that does a full-row
			// overwrite using the in-memory snapshot, which is stale if another goroutine
			// already deducted points between our GetWalletForUpdate call and now.
			// Two concurrent Generate requests both read PulsePoints=100, both compute 50,
			// and both Save(50) → user gets 2 generations for the price of 1 (Lost Update).
			//
			// The atomic UPDATE with WHERE pulse_points >= cost prevents this:
			//   - Row is modified in-place inside the DB engine
			//   - RowsAffected == 0 means another request already spent the balance
			deductResult := tx.Table("wallets").
				Where("user_id = ? AND pulse_points >= ?", userID, tool.PointCost).
				UpdateColumn("pulse_points", gorm.Expr("pulse_points - ?", tool.PointCost))
			if deductResult.Error != nil {
				return fmt.Errorf("wallet update: %w", deductResult.Error)
			}
			if deductResult.RowsAffected == 0 {
				return fmt.Errorf("insufficient PulsePoints: need %d (concurrent request may have spent them)",
					tool.PointCost)
			}

			// Immutable ledger entry
			ledgerTx := &entities.Transaction{
				ID:          uuid.New(),
				UserID:      userID,
				PhoneNumber: user.PhoneNumber,
				Type:        entities.TxTypeStudioSpend,
				PointsDelta: -tool.PointCost,
				Reference:   "studio_" + gen.ID.String()[:8],
				Metadata: func() json.RawMessage {
					b, _ := json.Marshal(map[string]any{
						"tool_id":   toolID.String(),
						"tool_slug": tool.Slug,
						"tool_name": tool.Name,
						"gen_id":    gen.ID.String(),
					})
					return b
				}(),
				CreatedAt: now,
			}
			if err := s.txRepo.SaveTx(ctx, tx, ledgerTx); err != nil {
				return fmt.Errorf("ledger write: %w", err)
			}
		}

		// Daily generation quota — enforced INSIDE the transaction so the count
		// and the insert are atomic.  The handler-level check above is a fast
		// pre-flight that avoids entering the transaction on the happy path;
		// this check is the authoritative one that prevents the race where two
		// concurrent requests both read count=9 (under a limit of 10) and both
		// create a generation, taking the user to 11.
		if dailyLimit > 0 {
			var countNow int64
			today := time.Now().UTC().Truncate(24 * time.Hour)
			if cntErr := tx.Table("ai_generations").
				Where("user_id = ? AND created_at >= ?", userID, today).
				Count(&countNow).Error; cntErr != nil {
				return fmt.Errorf("quota check failed: %w", cntErr)
			}
			if int(countNow) >= dailyLimit {
				return fmt.Errorf("daily generation limit reached (%d/%d)", countNow, dailyLimit)
			}
		}

		// Create job record
		if err := s.studioRepo.CreateGenerationTx(ctx, tx, gen); err != nil {
			return fmt.Errorf("create generation: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Update session usage (best-effort — don't fail the job if session update fails)
	if sess, sessErr := s.studioRepo.GetOrCreateActiveSession(ctx, userID); sessErr == nil {
		_ = s.studioRepo.UpdateSession(ctx, sess.ID, tool.PointCost)
	}

	return gen, nil
}

// ─── Completion & failure (called by AIStudioOrchestrator) ───────────────────

// CompleteGeneration persists all result fields and fires the SMS notification.
func (s *StudioService) CompleteGeneration(
	ctx context.Context,
	genID uuid.UUID,
	outputURL, outputURL2, outputText, provider string,
	costMicros, durationMs int,
) error {
	if err := s.studioRepo.CompleteGeneration(ctx, genID, "completed", outputURL, outputURL2, outputText, provider, costMicros, durationMs); err != nil {
		return err
	}

	// Notify user (best-effort — don't fail the job if SMS fails)
	gen, err := s.studioRepo.FindGenerationByID(ctx, genID)
	if err != nil {
		return nil // already committed above
	}
	user, _ := s.userRepo.FindByID(ctx, gen.UserID)
	tool, _ := s.studioRepo.FindToolByID(ctx, gen.ToolID)
	if user != nil && tool != nil && s.notifySvc != nil {
		s.notifySvc.NotifyAssetReady(ctx, user.PhoneNumber, tool.Name)
		// Also send push notification so the app banner fires
		go func() {
			_ = s.notifySvc.SendToUser(ctx, gen.UserID,
				"Studio Ready ✨", tool.Name+" is ready — tap to download before it expires.",
				map[string]string{"screen": "studio", "generation_id": gen.ID.String()},
			)
		}()
	}
	return nil
}

// FailGeneration marks the job failed and issues a compensating PulsePoints refund.
func (s *StudioService) FailGeneration(ctx context.Context, genID uuid.UUID, reason string) error {
	// Mark failed
	if err := s.studioRepo.UpdateStatus(ctx, genID, "failed", "", reason); err != nil {
		return err
	}

	// Compensating refund — immutable new ledger row
	gen, err := s.studioRepo.FindGenerationByID(ctx, genID)
	if err != nil {
		return nil // can't refund without the gen record
	}
	if gen.PointsDeducted == 0 {
		return nil // nothing to refund
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		// Guard against double-refund: mark refund_granted atomically via CAS.
		// If two goroutines both call FailGeneration for the same job (e.g. lifecycle
		// worker + async worker timing conflict), only the first one sets this flag;
		// the second sees RowsAffected == 0 and aborts without re-crediting the wallet.
		guard := tx.Table("ai_generations").
			Where("id = ? AND refund_granted = FALSE", gen.ID).
			Updates(map[string]interface{}{"refund_granted": true, "refund_pts": gen.PointsDeducted})
		if guard.Error != nil {
			return guard.Error
		}
		if guard.RowsAffected == 0 {
			return nil // refund already issued by a concurrent call — safe to skip
		}

		// Restore wallet atomically (no read-modify-write — prevents Lost Update)
		if err := tx.Table("wallets").
			Where("user_id = ?", gen.UserID).
			UpdateColumn("pulse_points", gorm.Expr("pulse_points + ?", gen.PointsDeducted)).
			Error; err != nil {
			return err
		}

		// Immutable compensating ledger entry
		user, _ := s.userRepo.FindByID(ctx, gen.UserID)
		phone := ""
		if user != nil {
			phone = user.PhoneNumber
		}
		refundTx := &entities.Transaction{
			ID:          uuid.New(),
			UserID:      gen.UserID,
			PhoneNumber: phone,
			Type:        entities.TxTypeStudioRefund,
			PointsDelta: gen.PointsDeducted, // positive — restoring points
			Reference:   "refund_" + gen.ID.String()[:8],
			Metadata: func() json.RawMessage {
				b, _ := json.Marshal(map[string]any{
					"reason": reason,
					"gen_id": gen.ID.String(),
				})
				return b
			}(),
			CreatedAt: time.Now(),
		}
		return tx.Create(refundTx).Error
	})
}

// ─── Admin helpers ────────────────────────────────────────────────────────────

// UpdateToolCost changes a tool's PulsePoint cost (zero-hardcoding rule).
func (s *StudioService) UpdateToolCost(ctx context.Context, toolID uuid.UUID, newCost int64) error {
	if newCost < 0 {
		return fmt.Errorf("point cost cannot be negative")
	}
	return s.studioRepo.UpdateToolCost(ctx, toolID, newCost)
}

// SetToolEnabled activates or deactivates a tool globally.
func (s *StudioService) SetToolEnabled(ctx context.Context, toolID uuid.UUID, enabled bool) error {
	return s.studioRepo.SetToolEnabled(ctx, toolID, enabled)
}

// UpsertTool creates or replaces a tool by slug (used by seed/admin).
func (s *StudioService) UpsertTool(ctx context.Context, tool *entities.StudioTool) error {
	if tool.ID == uuid.Nil {
		tool.ID = uuid.New()
	}
	if tool.Slug == "" {
		return fmt.Errorf("tool slug is required")
	}
	return s.studioRepo.UpsertTool(ctx, tool)
}

// ListStalePendingJobs returns jobs stuck in pending/processing state.
func (s *StudioService) ListStalePendingJobs(ctx context.Context, staleSeconds, limit int) ([]entities.AIGeneration, error) {
	return s.studioRepo.ListPendingGenerations(ctx, staleSeconds, limit)
}

// GetToolErrors returns recent failed ai_generations for a specific tool.
func (s *StudioService) GetToolErrors(ctx context.Context, toolID uuid.UUID, limit int) ([]entities.AIGeneration, error) {
	return s.studioRepo.GetToolErrors(ctx, toolID, limit)
}

// GetToolStats returns 30-day aggregated stats per tool.
func (s *StudioService) GetToolStats(ctx context.Context) ([]repositories.ToolStats, error) {
	return s.studioRepo.GetToolStats(ctx)
}

// ListGenerations returns paginated generations with optional filters.
func (s *StudioService) ListGenerations(ctx context.Context, filter repositories.GenerationFilter) ([]entities.AIGeneration, int, error) {
	return s.studioRepo.ListGenerations(ctx, filter)
}

// ─── Dispute flow ─────────────────────────────────────────────────────────────

// DisputeGeneration handles a user disputing an unsatisfactory generation output.
// Validates the dispute window, calculates refund, restores wallet, writes ledger.
func (s *StudioService) DisputeGeneration(ctx context.Context, genID uuid.UUID, userID uuid.UUID) error {
	gen, err := s.studioRepo.FindGenerationByID(ctx, genID)
	if err != nil {
		return fmt.Errorf("generation not found: %w", err)
	}
	if gen.UserID != userID {
		return fmt.Errorf("access denied")
	}
	if gen.Status != "completed" {
		return fmt.Errorf("can only dispute completed generations")
	}
	if gen.DisputedAt != nil {
		return fmt.Errorf("already disputed")
	}
	if gen.RefundGranted {
		return fmt.Errorf("refund already granted")
	}

	// Check tool's refund window
	tool, err := s.studioRepo.FindToolByID(ctx, gen.ToolID)
	if err != nil {
		return fmt.Errorf("tool not found: %w", err)
	}
	if tool.RefundWindowMins == 0 {
		return fmt.Errorf("this tool does not support refunds")
	}
	windowEnd := gen.CreatedAt.Add(time.Duration(tool.RefundWindowMins) * time.Minute)
	if time.Now().After(windowEnd) {
		return fmt.Errorf("refund window expired (%d minutes after generation)", tool.RefundWindowMins)
	}

	// Calculate refund amount
	refundPts := (gen.PointsDeducted * int64(tool.RefundPct)) / 100
	if refundPts == 0 {
		return fmt.Errorf("no refund applicable for this tool")
	}

	// Atomic: restore wallet + ledger + mark disputed
	user, _ := s.userRepo.FindByID(ctx, userID)
	now := time.Now()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// CAS: mark as disputed atomically so two concurrent dispute requests
		// cannot both succeed.  The outer DisputedAt check (above) is a fast
		// pre-flight; this UPDATE is the authoritative guard.
		// If another goroutine already committed the dispute, RowsAffected == 0.
		disputeGuard := tx.Table("ai_generations").
			Where("id = ? AND disputed_at IS NULL AND refund_granted = FALSE", gen.ID).
			Updates(map[string]interface{}{
				"disputed_at":    now,
				"refund_granted": true,
				"refund_pts":     refundPts,
			})
		if disputeGuard.Error != nil {
			return disputeGuard.Error
		}
		if disputeGuard.RowsAffected == 0 {
			return fmt.Errorf("refund already processed for this generation")
		}

		// Restore wallet atomically — no read-modify-write to prevent Lost Update
		if err := tx.Table("wallets").
			Where("user_id = ?", userID).
			UpdateColumn("pulse_points", gorm.Expr("pulse_points + ?", refundPts)).
			Error; err != nil {
			return err
		}

		// Ledger entry
		phone := ""
		if user != nil {
			phone = user.PhoneNumber
		}
		refundTx := &entities.Transaction{
			ID:          uuid.New(),
			UserID:      userID,
			PhoneNumber: phone,
			Type:        entities.TxTypeStudioRefund,
			PointsDelta: refundPts,
			Reference:   "dispute_" + gen.ID.String()[:8],
			Metadata: func() json.RawMessage {
				b, _ := json.Marshal(map[string]any{
					"gen_id":     gen.ID.String(),
					"tool_slug":  gen.ToolSlug,
					"refund_pct": tool.RefundPct,
					"reason":     "user_dispute",
				})
				return b
			}(),
			CreatedAt: now,
		}
		return tx.Create(refundTx).Error
	})
}

// RegisterVoiceClone creates an ElevenLabs Instant Voice Clone from a user's
// recorded audio sample and stores the resulting voice_id on the user, so the
// Talking Avatar tool can speak scripts in their own voice. Any previous clone
// is best-effort deleted first to avoid orphaning voices on the ElevenLabs side.
// Returns the new voice_id.
func (s *StudioService) RegisterVoiceClone(ctx context.Context, userID uuid.UUID, audio []byte, filename string) (string, error) {
	apiKey := os.Getenv("ELEVENLABS_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("voice cloning unavailable: ELEVENLABS_API_KEY not configured")
	}
	if len(audio) < 2000 {
		return "", fmt.Errorf("recording too short — please record at least a few seconds of clear speech")
	}
	if filename == "" {
		filename = "sample.webm"
	}

	// Best-effort delete of the user's previous clone (keeps the ElevenLabs
	// voice library tidy; failure here is non-fatal).
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("user not found")
	}
	if user.ClonedVoiceID != "" {
		s.deleteElevenLabsVoice(ctx, apiKey, user.ClonedVoiceID)
	}

	// Build the multipart body: name + files (the audio sample).
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("name", "nexus-voice-"+userID.String()[:8])
	_ = mw.WriteField("remove_background_noise", "true")
	fw, err := mw.CreateFormFile("files", filename)
	if err != nil {
		return "", err
	}
	if _, err := fw.Write(audio); err != nil {
		return "", err
	}
	_ = mw.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.elevenlabs.io/v1/voices/add", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("xi-api-key", apiKey)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("elevenlabs add-voice: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		msg := string(raw)
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return "", fmt.Errorf("voice clone failed (%d): %s", resp.StatusCode, msg)
	}
	var parsed struct {
		VoiceID string `json:"voice_id"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || parsed.VoiceID == "" {
		return "", fmt.Errorf("voice clone: no voice_id returned")
	}

	// Persist on the user.
	if err := s.db.WithContext(ctx).Table("users").
		Where("id = ?", userID).
		Update("cloned_voice_id", parsed.VoiceID).Error; err != nil {
		return "", fmt.Errorf("failed to save voice: %w", err)
	}
	return parsed.VoiceID, nil
}

// deleteElevenLabsVoice best-effort removes a voice from the ElevenLabs library.
func (s *StudioService) deleteElevenLabsVoice(ctx context.Context, apiKey, voiceID string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, "https://api.elevenlabs.io/v1/voices/"+voiceID, nil)
	if err != nil {
		return
	}
	req.Header.Set("xi-api-key", apiKey)
	client := &http.Client{Timeout: 15 * time.Second}
	if resp, err := client.Do(req); err == nil {
		_ = resp.Body.Close()
	}
}

// GetSessionUsage returns the active session for a user (nil if none within 30 min).
func (s *StudioService) GetSessionUsage(ctx context.Context, userID uuid.UUID) (*entities.StudioSession, error) {
	return s.studioRepo.GetSessionUsage(ctx, userID)
}

// GetPromptHistory returns the N most-recent distinct prompts for a tool.
func (s *StudioService) GetPromptHistory(ctx context.Context, userID uuid.UUID, toolSlug string, limit int) ([]repositories.PromptHistoryItem, error) {
	return s.studioRepo.GetPromptHistory(ctx, userID, toolSlug, limit)
}

// CheckProviderHealthGate returns true when the primary active provider for
// the given tool slug's category was last tested as unhealthy within the past
// 30 minutes.  Returns false (allow through) when data is absent or stale.
// Called by the Generate handler BEFORE point deduction (BUG-050).
func (s *StudioService) CheckProviderHealthGate(ctx context.Context, toolSlug string) bool {
	cat, ok := slugCategory[toolSlug]
	if !ok {
		return false // unknown slug — allow through
	}
	var cfg struct {
		LastTestOK   *bool      `gorm:"column:last_test_ok"`
		LastTestedAt *time.Time `gorm:"column:last_tested_at"`
	}
	err := s.db.WithContext(ctx).
		Table("ai_provider_configs").
		Select("last_test_ok, last_tested_at").
		Where("category = ? AND is_active = true AND is_primary = true", string(cat)).
		Order("priority ASC").
		Limit(1).
		Scan(&cfg).Error
	if err != nil || cfg.LastTestOK == nil || cfg.LastTestedAt == nil {
		return false // no data → allow through
	}
	// Block only if the last test explicitly failed AND was recent (< 30 min)
	return !*cfg.LastTestOK && time.Since(*cfg.LastTestedAt) <= 30*time.Minute
}
