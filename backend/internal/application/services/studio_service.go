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
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
)

// StudioService is injected into HTTP handlers and the async worker.
type StudioService struct {
	studioRepo repositories.StudioRepository
	userRepo   repositories.UserRepository
	txRepo     repositories.TransactionRepository
	notifySvc  *NotificationService
	db         *gorm.DB
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
func (s *StudioService) RequestGeneration(
	ctx context.Context,
	userID uuid.UUID,
	toolID uuid.UUID,
	prompt string,
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

	// 3. Check IsFree — skip all wallet checks for free tools
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
			ExpiresAt:      now.AddDate(0, 0, 30),
		}
		err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
		ExpiresAt:      now.AddDate(0, 0, 30), // 30-day asset retention
	}

	// 7. Atomic: deduct wallet + ledger entry + create job
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		if tool.PointCost > 0 {
			// Deduct PulsePoints
			wallet.PulsePoints -= tool.PointCost
			if err := tx.Save(wallet).Error; err != nil {
				return fmt.Errorf("wallet update: %w", err)
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
	outputURL, outputText, provider string,
	costMicros, durationMs int,
) error {
	if err := s.studioRepo.CompleteGeneration(ctx, genID, "completed", outputURL, outputText, provider, costMicros, durationMs); err != nil {
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

		// Restore wallet
		var wallet entities.Wallet
		if err := tx.Where("user_id = ?", gen.UserID).First(&wallet).Error; err != nil {
			return err
		}
		wallet.PulsePoints += gen.PointsDeducted
		if err := tx.Save(&wallet).Error; err != nil {
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
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Restore wallet
		var wallet entities.Wallet
		if err := tx.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
			return err
		}
		wallet.PulsePoints += refundPts
		if err := tx.Save(&wallet).Error; err != nil {
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
			CreatedAt: time.Now(),
		}
		if err := tx.Create(refundTx).Error; err != nil {
			return err
		}

		// Mark generation as disputed
		return s.studioRepo.DisputeGeneration(ctx, genID, refundPts)
	})
}

// GetSessionUsage returns the active session for a user (nil if none within 30 min).
func (s *StudioService) GetSessionUsage(ctx context.Context, userID uuid.UUID) (*entities.StudioSession, error) {
	return s.studioRepo.GetSessionUsage(ctx, userID)
}
