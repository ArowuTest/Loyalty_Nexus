package persistence

// passport_repo_postgres.go — GORM/PostgreSQL implementations of all passport repository interfaces.
// Each struct holds a *gorm.DB and implements the interface defined in
// domain/repositories/passport_repository.go.
// Naming convention matches the rest of the project: postgres{Name}Repository.

import (
	"context"
	"time"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ─── postgresPassportRepository ───────────────────────────────────────────────

type postgresPassportRepository struct {
	db *gorm.DB
}

// NewPostgresPassportRepository creates a new GORM-backed passport repository.
func NewPostgresPassportRepository(db *gorm.DB) repositories.PassportRepository {
	return &postgresPassportRepository{db: db}
}

func (r *postgresPassportRepository) LogEvent(ctx context.Context, event *entities.PassportEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *postgresPassportRepository) GetEvents(ctx context.Context, userID uuid.UUID, limit int) ([]entities.PassportEvent, error) {
	var events []entities.PassportEvent
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

func (r *postgresPassportRepository) GetStats(ctx context.Context) (*repositories.PassportStats, error) {
	var stats repositories.PassportStats

	// Total passports (distinct users with at least one passport event)
	r.db.WithContext(ctx).Model(&entities.PassportEvent{}).
		Distinct("user_id").Count(&stats.TotalPassports)

	// Apple downloads
	r.db.WithContext(ctx).Model(&entities.PassportEvent{}).
		Where("event_type = ?", "apple_pass_issued").Count(&stats.AppleDownloads)

	// Google saves
	r.db.WithContext(ctx).Model(&entities.PassportEvent{}).
		Where("event_type = ?", "google_pass_issued").Count(&stats.GoogleSaves)

	// QR scans today
	today := time.Now().Truncate(24 * time.Hour)
	r.db.WithContext(ctx).Model(&entities.PassportEvent{}).
		Where("event_type = ? AND created_at >= ?", "qr_verified", today).
		Count(&stats.QRScansToday)

	// Tier breakdown
	type tierRow struct {
		Tier  string `gorm:"column:tier"`
		Count int64  `gorm:"column:count"`
	}
	var tiers []tierRow
	r.db.WithContext(ctx).Table("users").
		Select("tier, COUNT(*) as count").
		Group("tier").
		Order("count DESC").
		Scan(&tiers)
	for _, t := range tiers {
		stats.TierBreakdown = append(stats.TierBreakdown, repositories.TierCount{
			Tier:  t.Tier,
			Count: t.Count,
		})
	}

	// Top badge earners
	type earnerRow struct {
		UserID      uuid.UUID `gorm:"column:user_id"`
		PhoneNumber string    `gorm:"column:phone_number"`
		BadgeCount  int64     `gorm:"column:badge_count"`
	}
	var earners []earnerRow
	r.db.WithContext(ctx).Table("user_badges ub").
		Select("ub.user_id, u.phone_number, COUNT(*) as badge_count").
		Joins("JOIN users u ON u.id = ub.user_id").
		Group("ub.user_id, u.phone_number").
		Order("badge_count DESC").
		Limit(10).
		Scan(&earners)
	for _, e := range earners {
		stats.TopBadgeEarners = append(stats.TopBadgeEarners, repositories.BadgeEarner{
			UserID:      e.UserID,
			PhoneNumber: maskPhone(e.PhoneNumber),
			BadgeCount:  e.BadgeCount,
		})
	}

	return &stats, nil
}

// ─── postgresWalletRegistrationRepository ────────────────────────────────────

type postgresWalletRegistrationRepository struct {
	db *gorm.DB
}

// NewPostgresWalletRegistrationRepository creates a new GORM-backed wallet registration repository.
func NewPostgresWalletRegistrationRepository(db *gorm.DB) repositories.WalletRegistrationRepository {
	return &postgresWalletRegistrationRepository{db: db}
}

func (r *postgresWalletRegistrationRepository) Upsert(ctx context.Context, reg *entities.WalletRegistration) error {
	if reg.ID == uuid.Nil {
		reg.ID = uuid.New()
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "device_id"}, {Name: "serial_number"}},
			DoUpdates: clause.AssignmentColumns([]string{"push_token", "is_active", "updated_at", "push_token_updated_at"}),
		}).
		Create(reg).Error
}

func (r *postgresWalletRegistrationRepository) Deactivate(ctx context.Context, deviceID, passTypeID, serialNumber string) error {
	return r.db.WithContext(ctx).
		Model(&entities.WalletRegistration{}).
		Where("device_id = ? AND serial_number = ?", deviceID, serialNumber).
		Update("is_active", false).Error
}

func (r *postgresWalletRegistrationRepository) GetUpdatedSerials(ctx context.Context, deviceID, passTypeID string, since time.Time) ([]string, error) {
	var serials []string
	err := r.db.WithContext(ctx).
		Model(&entities.WalletRegistration{}).
		Where("device_id = ? AND is_active = true AND updated_at > ?", deviceID, since).
		Pluck("serial_number", &serials).Error
	return serials, err
}

func (r *postgresWalletRegistrationRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]entities.WalletRegistration, error) {
	var regs []entities.WalletRegistration
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_active = true", userID).
		Find(&regs).Error
	return regs, err
}

// ─── postgresGoogleWalletObjectRepository ────────────────────────────────────

type postgresGoogleWalletObjectRepository struct {
	db *gorm.DB
}

// NewPostgresGoogleWalletObjectRepository creates a new GORM-backed Google Wallet object repository.
func NewPostgresGoogleWalletObjectRepository(db *gorm.DB) repositories.GoogleWalletObjectRepository {
	return &postgresGoogleWalletObjectRepository{db: db}
}

func (r *postgresGoogleWalletObjectRepository) Upsert(ctx context.Context, obj *entities.GoogleWalletObject) error {
	if obj.ID == uuid.Nil {
		obj.ID = uuid.New()
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"object_id", "class_id", "last_synced_at", "last_sync_status", "updated_at"}),
		}).
		Create(obj).Error
}

func (r *postgresGoogleWalletObjectRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.GoogleWalletObject, error) {
	var obj entities.GoogleWalletObject
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&obj).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &obj, err
}

func (r *postgresGoogleWalletObjectRepository) GetSyncCandidates(ctx context.Context, syncOlderThan time.Time, limit int) ([]entities.GoogleWalletObject, error) {
	var objs []entities.GoogleWalletObject
	err := r.db.WithContext(ctx).
		Where("last_synced_at < ?", syncOlderThan).
		Order("last_synced_at ASC").
		Limit(limit).
		Find(&objs).Error
	return objs, err
}

// ─── postgresGhostNudgeRepository ─────────────────────────────────────────────

type postgresGhostNudgeRepository struct {
	db *gorm.DB
}

// NewPostgresGhostNudgeRepository creates a new GORM-backed ghost nudge repository.
func NewPostgresGhostNudgeRepository(db *gorm.DB) repositories.GhostNudgeRepository {
	return &postgresGhostNudgeRepository{db: db}
}

func (r *postgresGhostNudgeRepository) Log(ctx context.Context, entry *entities.GhostNudgeLog) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(entry).Error
}

func (r *postgresGhostNudgeRepository) WasNudgedSince(ctx context.Context, userID uuid.UUID, since time.Time) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entities.GhostNudgeLog{}).
		Where("user_id = ? AND nudged_at > ?", userID, since).
		Count(&count).Error
	return count > 0, err
}

func (r *postgresGhostNudgeRepository) GetRecentLog(ctx context.Context, limit int) ([]repositories.GhostNudgeLogRow, error) {
	type nudgeRow struct {
		UserID      uuid.UUID `gorm:"column:user_id"`
		PhoneNumber string    `gorm:"column:phone_number"`
		NudgedAt    time.Time `gorm:"column:nudged_at"`
		Message     string    `gorm:"column:message"`
		Status      string    `gorm:"column:status"`
	}
	var rows []nudgeRow
	err := r.db.WithContext(ctx).
		Table("ghost_nudge_log gn").
		Select("gn.user_id, u.phone_number, gn.nudged_at, gn.message, gn.status").
		Joins("JOIN users u ON u.id = gn.user_id").
		Order("gn.nudged_at DESC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]repositories.GhostNudgeLogRow, len(rows))
	for i, row := range rows {
		result[i] = repositories.GhostNudgeLogRow{
			UserID:      row.UserID,
			PhoneNumber: maskPhone(row.PhoneNumber),
			NudgedAt:    row.NudgedAt,
			Message:     row.Message,
			Status:      row.Status,
		}
	}
	return result, nil
}

// ─── postgresUSSDSessionRepository ───────────────────────────────────────────

type postgresUSSDSessionRepository struct {
	db *gorm.DB
}

// NewPostgresUSSDSessionRepository creates a new GORM-backed USSD session repository.
func NewPostgresUSSDSessionRepository(db *gorm.DB) repositories.USSDSessionRepository {
	return &postgresUSSDSessionRepository{db: db}
}

func (r *postgresUSSDSessionRepository) Upsert(ctx context.Context, session *entities.USSDSession) error {
	if session.ID == uuid.Nil {
		session.ID = uuid.New()
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "session_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"menu_state", "input_buffer", "expires_at", "updated_at"}),
		}).
		Create(session).Error
}

func (r *postgresUSSDSessionRepository) GetBySessionID(ctx context.Context, sessionID string) (*entities.USSDSession, error) {
	var session entities.USSDSession
	err := r.db.WithContext(ctx).
		Where("session_id = ? AND expires_at > ?", sessionID, time.Now()).
		First(&session).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &session, err
}

func (r *postgresUSSDSessionRepository) DeleteExpired(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&entities.USSDSession{}).Error
}

func (r *postgresUSSDSessionRepository) GetActiveSessions(ctx context.Context, limit int) ([]entities.USSDSession, error) {
	var sessions []entities.USSDSession
	err := r.db.WithContext(ctx).
		Where("expires_at > ?", time.Now()).
		Order("created_at DESC").
		Limit(limit).
		Find(&sessions).Error
	return sessions, err
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// maskPhone masks a phone number to show only the last 4 digits.
func maskPhone(phone string) string {
	if len(phone) <= 4 {
		return phone
	}
	return "****" + phone[len(phone)-4:]
}
