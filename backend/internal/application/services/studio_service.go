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
	db         *sql.DB
}

func NewStudioService(sr repositories.StudioRepository, ur repositories.UserRepository, tr repositories.TransactionRepository, db *sql.DB) *StudioService {
	return &StudioService{
		studioRepo: sr,
		userRepo:   ur,
		txRepo:     tr,
		db:         db,
	}
}

// RequestGeneration handles the atomic point deduction and creation of a generation record
func (s *StudioService) RequestGeneration(ctx context.Context, userID uuid.UUID, toolID uuid.UUID, prompt string) (*entities.AIGeneration, error) {
	tool, err := s.studioRepo.FindToolByID(ctx, toolID)
	if err != nil {
		return nil, fmt.Errorf("tool not found: %w", err)
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if user.PointBalance < tool.PointCost {
		return nil, fmt.Errorf("insufficient points: need %d, have %d", tool.PointCost, user.PointBalance)
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
