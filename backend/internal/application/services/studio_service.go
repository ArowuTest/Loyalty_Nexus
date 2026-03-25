package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"github.com/google/uuid"
)

type StudioService struct {
	studioRepo repositories.StudioRepository
	userRepo   repositories.UserRepository
	txRepo     repositories.TransactionRepository
	notifySvc  *NotificationService
	db         *sql.DB
}

func NewStudioService(sr repositories.StudioRepository, ur repositories.UserRepository, tr repositories.TransactionRepository, ns *NotificationService, db *sql.DB) *StudioService {
	return &StudioService{
		studioRepo: sr,
		userRepo:   ur,
		txRepo:     tr,
		notifySvc:  ns,
		db:         db,
	}
}

func (s *StudioService) ListActiveTools(ctx context.Context) ([]entities.StudioTool, error) {
	return s.studioRepo.ListActiveTools(ctx)
}

func (s *StudioService) FindToolByID(ctx context.Context, id uuid.UUID) (*entities.StudioTool, error) {
	return s.studioRepo.FindToolByID(ctx, id)
}

func (s *StudioService) GetUserGallery(ctx context.Context, userID uuid.UUID) ([]entities.AIGeneration, error) {
	return s.studioRepo.GetUserGallery(ctx, userID)
}

func (s *StudioService) CompleteGeneration(ctx context.Context, genID uuid.UUID, outputURL string) error {
	if err := s.studioRepo.UpdateStatus(ctx, genID, "completed", outputURL, ""); err != nil {
		return err
	}

	// Trigger SMS Notification
	gen, err := s.studioRepo.FindGenerationByID(ctx, genID)
	if err == nil {
		user, _ := s.userRepo.FindByID(ctx, gen.UserID)
		tool, _ := s.studioRepo.FindToolByID(ctx, gen.ToolID)
		if user != nil && tool != nil {
			s.notifySvc.NotifyAssetReady(ctx, user.MSISDN, tool.Name)
		}
	}

	return nil
}

func (s *StudioService) FailGeneration(ctx context.Context, genID uuid.UUID, errMsg string) error {
	// 1. Mark generation as failed
	if err := s.studioRepo.UpdateStatus(ctx, genID, "failed", "", errMsg); err != nil {
		return err
	}

	// 2. COMPENSATING TRANSACTION: Refund Points
	gen, err := s.studioRepo.FindGenerationByID(ctx, genID)
	if err != nil {
		return err
	}

	user, err := s.userRepo.FindByID(ctx, gen.UserID)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	refundTx := &entities.Transaction{
		ID:          uuid.New(),
		UserID:      gen.UserID,
		MSISDN:      user.MSISDN,
		Type:        entities.TxTypeBonus, // Refund type
		PointsDelta: gen.PointsDeducted,
		CreatedAt:   time.Now(),
		Metadata:    map[string]any{"reason": "Studio Refund", "gen_id": genID.String()},
	}

	if err := s.txRepo.SaveTx(ctx, tx, refundTx); err != nil {
		return err
	}

	return tx.Commit()
}
func (s *StudioService) RequestGeneration(ctx context.Context, userID uuid.UUID, toolID uuid.UUID, prompt string) (*entities.AIGeneration, error) {
	tool, err := s.studioRepo.FindToolByID(ctx, toolID)
	if err != nil {
		return nil, fmt.Errorf("tool not found: %w", err)
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if user.TotalPoints < tool.PointCost {
		return nil, fmt.Errorf("insufficient points: need %d, have %d", tool.PointCost, user.TotalPoints)
	}

	// Atomic Point Deduction + Generation Record
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	gen := &entities.AIGeneration{
		ID:             uuid.New(),
		UserID:         userID,
		ToolID:         toolID,
		Prompt:         prompt,
		Status:         "pending",
		PointsDeducted: tool.PointCost,
		CreatedAt:      time.Now(),
		ExpiresAt:      time.Now().AddDate(0, 0, 30), // 30-day retention
	}

	// 1. Create Transaction (Audit Log)
	ledgerTx := &entities.Transaction{
		ID:          uuid.New(),
		UserID:      userID,
		MSISDN:      user.MSISDN,
		Type:        entities.TxTypeStudioSpend, // Assuming spend type in entities
		PointsDelta: -tool.PointCost,
		CreatedAt:   time.Now(),
		Metadata:    map[string]any{"tool": tool.Name, "gen_id": gen.ID.String()},
	}

	if err := s.txRepo.SaveTx(ctx, tx, ledgerTx); err != nil {
		return nil, err
	}

	// 2. Create Generation Record
	if err := s.studioRepo.CreateGenerationTx(ctx, tx, gen); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return gen, nil
}
