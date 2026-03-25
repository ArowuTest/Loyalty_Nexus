package services

import (
	"context"
	"fmt"
	"time"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type StudioService struct {
	studioRepo repositories.StudioRepository
	userRepo   repositories.UserRepository
	txRepo     repositories.TransactionRepository
	notifySvc  *NotificationService
	monetSvc   *MonetizationService
	db         *gorm.DB
}

func NewStudioService(sr repositories.StudioRepository, ur repositories.UserRepository, tr repositories.TransactionRepository, ns *NotificationService, ms *MonetizationService, db *gorm.DB) *StudioService {
	return &StudioService{
		studioRepo: sr,
		userRepo:   ur,
		txRepo:     tr,
		notifySvc:  ns,
		monetSvc:   ms,
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

func (s *StudioService) CompleteGeneration(ctx context.Context, genID uuid.UUID, outputURL string, provider string, costMicros int) error {
	if err := s.studioRepo.UpdateStatus(ctx, genID, "completed", outputURL, ""); err != nil {
		return err
	}

	// Track GPU Usage (Innovation 6.4)
	s.monetSvc.TrackStudioUsage(ctx, genID, provider, costMicros)

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
	if err := s.studioRepo.UpdateStatus(ctx, genID, "failed", "", errMsg); err != nil {
		return err
	}

	// COMPENSATING TRANSACTION: Refund Points
	gen, err := s.studioRepo.FindGenerationByID(ctx, genID)
	if err != nil {
		return err
	}

	user, err := s.userRepo.FindByID(ctx, gen.UserID)
	if err != nil {
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		refundTx := &entities.Transaction{
			ID:          uuid.New(),
			UserID:      gen.UserID,
			MSISDN:      user.MSISDN,
			Type:        entities.TxTypeBonus,
			PointsDelta: gen.PointsDeducted,
			CreatedAt:   time.Now(),
			Metadata:    map[string]any{"reason": "Studio Refund", "gen_id": genID.String()},
		}

		if err := s.txRepo.SaveTx(ctx, tx, refundTx); err != nil {
			return err
		}
		return nil
	})
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

	err = s.db.Transaction(func(tx *gorm.DB) error {
		ledgerTx := &entities.Transaction{
			ID:          uuid.New(),
			UserID:      userID,
			MSISDN:      user.MSISDN,
			Type:        entities.TxTypeStudioSpend,
			PointsDelta: -tool.PointCost,
			CreatedAt:   time.Now(),
			Metadata:    map[string]any{"tool": tool.Name, "gen_id": gen.ID.String()},
		}

		if err := s.txRepo.SaveTx(ctx, tx, ledgerTx); err != nil {
			return err
		}

		if err := s.studioRepo.CreateGenerationTx(ctx, tx, gen); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return gen, nil
}

func (s *StudioService) FindGenerationByID(ctx context.Context, id uuid.UUID) (*entities.AIGeneration, error) {
	return s.studioRepo.FindGenerationByID(ctx, id)
}
