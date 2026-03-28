package services

import (
	"context"
	"fmt"
	"time"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/infrastructure/config"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PassportService manages the Digital Passport — streaks, tiers, badges,
// milestones, and achievement history. (Master Spec §4 / SRS §3.4)
type PassportService struct {
	db  *gorm.DB
	cfg *config.ConfigManager
}

func NewPassportService(db *gorm.DB, cfg *config.ConfigManager) *PassportService {
	return &PassportService{db: db, cfg: cfg}
}

// ── Tier thresholds (lifetime Pulse Points) ────────────────────────────────
var tierThresholds = []struct {
	Name   string
	Points int64
}{
	{"PLATINUM", 50_000},
	{"GOLD", 10_000},
	{"SILVER", 2_000},
	{"BRONZE", 0},
}

// BadgeDefinition is a single achievement a user can earn.
type BadgeDefinition struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

// ── Badge catalogue ────────────────────────────────────────────────────────
var badgeCatalogue = []BadgeDefinition{
	{"first_recharge",   "First Steps",       "Complete your first recharge",                "⚡"},
	{"streak_7",         "Week Warrior",       "7-day recharge streak",                       "🔥"},
	{"streak_30",        "Month Master",       "30-day recharge streak",                      "🏆"},
	{"streak_90",        "Quarter King",       "90-day recharge streak",                      "👑"},
	{"spin_first",       "Lucky Spin",         "Play your first spin",                        "🎡"},
	{"spin_100",         "Spin Centurion",     "100 total spins",                             "💯"},
	{"studio_first",     "AI Explorer",        "Use AI Studio for the first time",            "🧠"},
	{"studio_50",        "Studio Pro",         "50 AI generations",                           "🎨"},
	{"wars_top3",        "State Champion",     "State ranked top 3 in Regional Wars",         "🌍"},
	{"silver_tier",      "Silver Member",      "Reached SILVER tier",                         "🥈"},
	{"gold_tier",        "Gold Member",        "Reached GOLD tier",                           "🥇"},
	{"platinum_tier",    "Platinum Elite",     "Reached PLATINUM tier",                       "💎"},
	{"referral_5",       "Connector",          "Referred 5 users",                            "🤝"},
	{"big_winner",       "Big Winner",         "Won a prize worth ₦5,000+",                   "🎁"},
}

// UserPassport is the full profile returned to the client.
type UserPassport struct {
	UserID           uuid.UUID         `json:"user_id"`
	Tier             string            `json:"tier"`
	StreakCount      int               `json:"streak_count"`
	LifetimePoints   int64             `json:"lifetime_points"`
	PulsePoints      int64             `json:"pulse_points"`
	SpinCredits      int               `json:"spin_credits"`
	Badges           []BadgeDefinition `json:"badges"`
	NextTier         string            `json:"next_tier"`
	PointsToNext     int64             `json:"points_to_next_tier"`
	MemberSince      time.Time         `json:"member_since"`
	// AmountToNextSpin is the naira amount still needed to earn the next spin credit.
	// Computed as: spin_trigger_naira - (recharge_counter % spin_trigger_naira).
	AmountToNextSpin int64             `json:"amount_to_next_spin_naira"`
}

// GetPassport returns the full passport for a user.
func (svc *PassportService) GetPassport(ctx context.Context, userID uuid.UUID) (*UserPassport, error) {
	type userRow struct {
		Tier           string    `gorm:"column:tier"`
		StreakCount    int       `gorm:"column:streak_count"`
		LifetimePoints int64     `gorm:"column:lifetime_points"`
		CreatedAt      time.Time `gorm:"column:created_at"`
	}
	var u userRow
	if err := svc.db.WithContext(ctx).Table("users").
		Select("tier, streak_count, lifetime_points, created_at").
		Where("id = ?", userID).First(&u).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Load wallet for pulse_points and spin_credits (REQ-4.1, REQ-4.2)
	var wallet entities.Wallet
	_ = svc.db.WithContext(ctx).Where("user_id = ?", userID).First(&wallet).Error

	// Compute amount to next spin credit.
	// spin_trigger_naira is stored in network_configs; default 1000.
	var spinTriggerNaira int64 = 1000
	var cfgRow struct{ Value string `gorm:"column:value"` }
	if err := svc.db.WithContext(ctx).Table("network_configs").
		Select("value").Where("key = ?", "spin_trigger_naira").First(&cfgRow).Error; err == nil {
		if v, parseErr := fmt.Sscan(cfgRow.Value, &spinTriggerNaira); v == 0 || parseErr != nil {
			spinTriggerNaira = 1000
		}
	}
	var amountToNextSpin int64
	if spinTriggerNaira > 0 {
		recharged := wallet.RechargeCounter // cumulative naira recharged
		mod := recharged % spinTriggerNaira
		if mod == 0 && recharged > 0 {
			amountToNextSpin = 0
		} else {
			amountToNextSpin = spinTriggerNaira - mod
		}
	}

	// Load earned badge keys
	var earnedKeys []string
	svc.db.WithContext(ctx).Table("user_badges").
		Where("user_id = ?", userID).Pluck("badge_key", &earnedKeys)
	earnedSet := make(map[string]bool, len(earnedKeys))
	for _, k := range earnedKeys {
		earnedSet[k] = true
	}
	var badges []BadgeDefinition
	for _, b := range badgeCatalogue {
		if earnedSet[b.Key] {
			badges = append(badges, b)
		}
	}

	// Compute next tier
	nextTier := ""
	pointsToNext := int64(0)
	for _, th := range tierThresholds {
		if u.LifetimePoints < th.Points {
			nextTier = th.Name
			pointsToNext = th.Points - u.LifetimePoints
		}
	}

	return &UserPassport{
		UserID:           userID,
		Tier:             u.Tier,
		StreakCount:      u.StreakCount,
		LifetimePoints:   u.LifetimePoints,
		PulsePoints:      wallet.PulsePoints,
		SpinCredits:      wallet.SpinCredits,
		Badges:           badges,
		NextTier:         nextTier,
		PointsToNext:     pointsToNext,
		MemberSince:      u.CreatedAt,
		AmountToNextSpin: amountToNextSpin,
	}, nil
}

// EvaluateBadges checks and awards any newly earned badges after an action.
// actionType: "recharge" | "spin" | "studio" | "wars_top3" | "prize_won"
// meta: extra numeric context (e.g. prize_value_kobo, spin_count)
func (svc *PassportService) EvaluateBadges(ctx context.Context, userID uuid.UUID, actionType string, meta map[string]int64) error {
	type userRow struct {
		StreakCount      int   `gorm:"column:streak_count"`
		TotalSpins       int   `gorm:"column:total_spins"`
		StudioUseCount   int   `gorm:"column:studio_use_count"`
		TotalReferrals   int   `gorm:"column:total_referrals"`
		LifetimePoints   int64 `gorm:"column:lifetime_points"`
		Tier             string `gorm:"column:tier"`
	}
	var u userRow
	if err := svc.db.WithContext(ctx).Table("users").
		Select("streak_count, total_spins, studio_use_count, total_referrals, lifetime_points, tier").
		Where("id = ?", userID).First(&u).Error; err != nil {
		return err
	}

	candidates := []string{}
	switch actionType {
	case "recharge":
		candidates = append(candidates, "first_recharge")
		if u.StreakCount >= 7  { candidates = append(candidates, "streak_7") }
		if u.StreakCount >= 30 { candidates = append(candidates, "streak_30") }
		if u.StreakCount >= 90 { candidates = append(candidates, "streak_90") }
	case "spin":
		candidates = append(candidates, "spin_first")
		if u.TotalSpins >= 100 { candidates = append(candidates, "spin_100") }
	case "studio":
		candidates = append(candidates, "studio_first")
		if u.StudioUseCount >= 50 { candidates = append(candidates, "studio_50") }
	case "wars_top3":
		candidates = append(candidates, "wars_top3")
	case "prize_won":
		if v, ok := meta["prize_value_kobo"]; ok && v >= 500_000 {
			candidates = append(candidates, "big_winner")
		}
	case "referral":
		if u.TotalReferrals >= 5 { candidates = append(candidates, "referral_5") }
	}

	// Tier badges
	switch u.Tier {
	case "SILVER":   candidates = append(candidates, "silver_tier")
	case "GOLD":     candidates = append(candidates, "silver_tier", "gold_tier")
	case "PLATINUM": candidates = append(candidates, "silver_tier", "gold_tier", "platinum_tier")
	}

	// Existing badges
	var earnedKeys []string
	svc.db.WithContext(ctx).Table("user_badges").
		Where("user_id = ?", userID).Pluck("badge_key", &earnedKeys)
	earnedSet := make(map[string]bool)
	for _, k := range earnedKeys { earnedSet[k] = true }

	now := time.Now()
	for _, key := range candidates {
		if earnedSet[key] { continue }
		row := map[string]interface{}{
			"id":         uuid.New(),
			"user_id":    userID,
			"badge_key":  key,
			"earned_at":  now,
			"created_at": now,
		}
		svc.db.WithContext(ctx).Table("user_badges").
			Clauses(clause.OnConflict{DoNothing: true}).Create(row)
	}
	return nil
}

// UpdateTier re-evaluates and updates the user's tier based on lifetime points.
func (svc *PassportService) UpdateTier(ctx context.Context, userID uuid.UUID) (string, error) {
	var lifetimePts int64
	if err := svc.db.WithContext(ctx).Table("users").
		Where("id = ?", userID).Pluck("lifetime_points", &lifetimePts).Error; err != nil {
		return "", err
	}
	newTier := "BRONZE"
	for _, th := range tierThresholds {
		if lifetimePts >= th.Points {
			newTier = th.Name
			break
		}
	}
	if err := svc.db.WithContext(ctx).Table("users").
		Where("id = ?", userID).
		Update("tier", newTier).Error; err != nil {
		return "", err
	}
	return newTier, nil
}
