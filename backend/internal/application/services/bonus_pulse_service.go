package services

// bonus_pulse_service.go
//
// Super-admins can award bonus Pulse Points to individual users as part of
// campaigns or incentive programmes.
//
// Every award atomically:
//   1. Resolves the recipient by phone number (normalised).
//   2. Acquires a SELECT FOR UPDATE lock on the wallet row.
//   3. Credits pulse_points and lifetime_points.
//   4. Writes an immutable TxTypeBonus ledger entry to transactions.
//   5. Writes an audit row to pulse_point_awards.
//
// All five steps happen inside a single DB transaction so the records are
// always consistent.

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
)

// ─── Domain types ────────────────────────────────────────────────────────────

// AwardBonusPulseRequest is the input to AwardBonusPulse.
type AwardBonusPulseRequest struct {
	// MSISDN of the recipient (any Nigerian format accepted).
	PhoneNumber string `json:"phone_number"`
	// Number of Pulse Points to award. Must be > 0.
	Points int64 `json:"points"`
	// Optional campaign name, e.g. "Ramadan 2025".
	Campaign string `json:"campaign"`
	// Optional free-text reason.
	Note string `json:"note"`
	// UUID of the admin making the award (from JWT claims).
	AwardedByID uuid.UUID `json:"-"`
	// Display name of the admin (from JWT claims or DB lookup).
	AwardedByName string `json:"-"`
}

// AwardBonusPulseResult is returned on success.
type AwardBonusPulseResult struct {
	AwardID       uuid.UUID `json:"award_id"`
	TransactionID uuid.UUID `json:"transaction_id"`
	UserID        uuid.UUID `json:"user_id"`
	PhoneNumber   string    `json:"phone_number"`
	PointsAwarded int64     `json:"points_awarded"`
	NewBalance    int64     `json:"new_balance"`
	Campaign      string    `json:"campaign"`
	AwardedAt     time.Time `json:"awarded_at"`
}

// PulseAwardRecord is a single row from the pulse_point_awards audit table.
type PulseAwardRecord struct {
	ID            uuid.UUID `gorm:"column:id"              json:"id"`
	UserID        uuid.UUID `gorm:"column:user_id"         json:"user_id"`
	PhoneNumber   string    `gorm:"column:phone_number"    json:"phone_number"`
	Points        int64     `gorm:"column:points"          json:"points"`
	Campaign      string    `gorm:"column:campaign"        json:"campaign"`
	Note          string    `gorm:"column:note"            json:"note"`
	AwardedBy     uuid.UUID `gorm:"column:awarded_by"      json:"awarded_by"`
	AwardedByName string    `gorm:"column:awarded_by_name" json:"awarded_by_name"`
	TransactionID uuid.UUID `gorm:"column:transaction_id"  json:"transaction_id"`
	CreatedAt     time.Time `gorm:"column:created_at"      json:"created_at"`
}

func (PulseAwardRecord) TableName() string { return "pulse_point_awards" }

// ─── Service ─────────────────────────────────────────────────────────────────

// BonusPulseService handles super-admin bonus Pulse Point awards.
type BonusPulseService struct {
	db       *gorm.DB
	userRepo repositories.UserRepository
}

// NewBonusPulseService constructs a BonusPulseService.
func NewBonusPulseService(db *gorm.DB, userRepo repositories.UserRepository) *BonusPulseService {
	return &BonusPulseService{db: db, userRepo: userRepo}
}

// AwardBonusPulse credits bonus Pulse Points to a user atomically and records
// the full audit trail.
func (s *BonusPulseService) AwardBonusPulse(ctx context.Context, req AwardBonusPulseRequest) (*AwardBonusPulseResult, error) {
	if req.Points <= 0 {
		return nil, fmt.Errorf("points must be greater than zero")
	}

	phone := normaliseBonusPhone(req.PhoneNumber)
	if phone == "" {
		return nil, fmt.Errorf("invalid phone number: %q", req.PhoneNumber)
	}

	// Resolve recipient.
	user, err := s.userRepo.FindByPhoneNumber(ctx, phone)
	if err != nil {
		return nil, fmt.Errorf("user not found for phone %s: %w", phone, err)
	}

	var result *AwardBonusPulseResult

	txErr := s.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		// Lock the wallet row to prevent concurrent balance races.
		var wallet entities.Wallet
		if err := dbTx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", user.ID).
			First(&wallet).Error; err != nil {
			return fmt.Errorf("lock wallet: %w", err)
		}

		// Credit the wallet.
		newBalance := wallet.PulsePoints + req.Points
		if err := dbTx.Table("wallets").
			Where("user_id = ?", user.ID).
			Updates(map[string]interface{}{
				"pulse_points":    gorm.Expr("pulse_points + ?", req.Points),
				"lifetime_points": gorm.Expr("lifetime_points + ?", req.Points),
			}).Error; err != nil {
			return fmt.Errorf("credit wallet: %w", err)
		}

		// Write the immutable ledger entry.
		txID := uuid.New()
		ledgerEntry := &entities.Transaction{
			ID:           txID,
			UserID:       user.ID,
			PhoneNumber:  phone,
			Type:         entities.TxTypeBonus,
			PointsDelta:  req.Points,
			Amount:       0, // no monetary value — admin gift
			BalanceAfter: newBalance,
			Reference:    fmt.Sprintf("BONUS-%s", txID.String()[:8]),
		}
		if err := dbTx.Create(ledgerEntry).Error; err != nil {
			return fmt.Errorf("write ledger entry: %w", err)
		}

		// Write the audit row.
		awardID := uuid.New()
		now := time.Now().UTC()
		award := &PulseAwardRecord{
			ID:            awardID,
			UserID:        user.ID,
			PhoneNumber:   phone,
			Points:        req.Points,
			Campaign:      strings.TrimSpace(req.Campaign),
			Note:          strings.TrimSpace(req.Note),
			AwardedBy:     req.AwardedByID,
			AwardedByName: strings.TrimSpace(req.AwardedByName),
			TransactionID: txID,
			CreatedAt:     now,
		}
		if err := dbTx.Create(award).Error; err != nil {
			return fmt.Errorf("write audit record: %w", err)
		}

		result = &AwardBonusPulseResult{
			AwardID:       awardID,
			TransactionID: txID,
			UserID:        user.ID,
			PhoneNumber:   phone,
			PointsAwarded: req.Points,
			NewBalance:    newBalance,
			Campaign:      award.Campaign,
			AwardedAt:     now,
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	return result, nil
}

// ListAwards returns a paginated list of bonus pulse point awards, optionally
// filtered by phone number and/or campaign name.
func (s *BonusPulseService) ListAwards(
	ctx context.Context,
	phone, campaign string,
	limit, offset int,
) ([]PulseAwardRecord, int64, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	q := s.db.WithContext(ctx).Table("pulse_point_awards")
	if p := normaliseBonusPhone(phone); p != "" {
		q = q.Where("phone_number = ?", p)
	}
	if c := strings.TrimSpace(campaign); c != "" {
		q = q.Where("campaign ILIKE ?", "%"+c+"%")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count awards: %w", err)
	}

	var records []PulseAwardRecord
	if err := q.Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("list awards: %w", err)
	}
	return records, total, nil
}

// GetUserBonusTotal returns the total bonus Pulse Points awarded to a user
// (used by the user dashboard to show the bonus breakdown).
func (s *BonusPulseService) GetUserBonusTotal(ctx context.Context, userID uuid.UUID) (int64, error) {
	var total int64
	err := s.db.WithContext(ctx).
		Table("pulse_point_awards").
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(points), 0)").
		Scan(&total).Error
	if err != nil {
		return 0, fmt.Errorf("sum bonus points: %w", err)
	}
	return total, nil
}

// GetUserAwards returns the most recent bonus awards for a user (for the
// user-facing history panel).
func (s *BonusPulseService) GetUserAwards(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
) ([]PulseAwardRecord, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var records []PulseAwardRecord
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&records).Error
	if err != nil {
		return nil, fmt.Errorf("get user awards: %w", err)
	}
	return records, nil
}

// ─── Phone normalisation (package-local copy) ────────────────────────────────
// Duplicated here so this service has no dependency on mtn_push_service.
// If the canonical version is ever moved to a shared package, remove this.

func normaliseBonusPhone(raw string) string {
	// Strip all non-digit characters to get the raw digit string
	var digits strings.Builder
	for _, r := range raw {
		if unicode.IsDigit(r) {
			digits.WriteRune(r)
		}
	}
	d := digits.String()
	// Always return in +234XXXXXXXXXX format to match DB storage
	switch {
	case strings.HasPrefix(d, "234") && len(d) == 13:
		return "+" + d // 2348012345678 -> +2348012345678
	case strings.HasPrefix(d, "0") && len(d) == 11:
		return "+234" + d[1:] // 08012345678 -> +2348012345678
	case len(d) == 10:
		return "+234" + d // 8012345678 -> +2348012345678
	default:
		// Already in +234... format or unrecognised — return as-is
		return raw
	}
}
