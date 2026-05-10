package services

// settings_service.go — Admin-configurable platform settings.
//
// Settings are stored in the platform_settings table and cached in Redis
// for 5 minutes so every read is near-zero cost (no DB hit on hot paths).
//
// Usage:
//   svc := NewSettingsService(db, rdb)
//   ttl, err := svc.StorageTTL(ctx, "GOLD")  // returns time.Duration
//   hours, err := svc.GetInt(ctx, "storage_ttl_gold_hours", 72)

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const settingsCacheTTL = 5 * time.Minute

// PlatformSetting mirrors the platform_settings DB row.
type PlatformSetting struct {
	Key         string    `gorm:"column:key;primaryKey"  json:"key"`
	Value       string    `gorm:"column:value"           json:"value"`
	Label       string    `gorm:"column:label"           json:"label"`
	Description string    `gorm:"column:description"     json:"description"`
	Category    string    `gorm:"column:category"        json:"category"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	UpdatedBy   string    `gorm:"column:updated_by"      json:"updated_by"`
}

func (PlatformSetting) TableName() string { return "platform_settings" }

// SettingsService reads and writes platform_settings with Redis caching.
type SettingsService struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewSettingsService(db *gorm.DB, rdb *redis.Client) *SettingsService {
	return &SettingsService{db: db, rdb: rdb}
}

// cacheKey returns the Redis key for a setting.
func (s *SettingsService) cacheKey(key string) string {
	return "platform_setting:" + key
}

// GetString returns the string value for a key, falling back to defaultVal on any error.
func (s *SettingsService) GetString(ctx context.Context, key, defaultVal string) string {
	// 1. Try Redis cache first
	if s.rdb != nil {
		if val, err := s.rdb.Get(ctx, s.cacheKey(key)).Result(); err == nil {
			return val
		}
	}
	// 2. DB read
	var row PlatformSetting
	if err := s.db.WithContext(ctx).Where("key = ?", key).First(&row).Error; err != nil {
		return defaultVal
	}
	// 3. Repopulate cache
	if s.rdb != nil {
		_ = s.rdb.Set(ctx, s.cacheKey(key), row.Value, settingsCacheTTL).Err()
	}
	return row.Value
}

// GetInt returns the integer value for a key, falling back to defaultVal on any error or parse failure.
func (s *SettingsService) GetInt(ctx context.Context, key string, defaultVal int) int {
	raw := s.GetString(ctx, key, "")
	if raw == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVal
	}
	return n
}

// Set writes a setting to the DB and invalidates the Redis cache entry.
func (s *SettingsService) Set(ctx context.Context, key, value, updatedBy string) error {
	res := s.db.WithContext(ctx).Exec(
		`UPDATE platform_settings SET value = ?, updated_at = NOW(), updated_by = ? WHERE key = ?`,
		value, updatedBy, key,
	)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("setting key %q not found", key)
	}
	// Invalidate cache
	if s.rdb != nil {
		_ = s.rdb.Del(ctx, s.cacheKey(key)).Err()
	}
	return nil
}

// ListByCategory returns all settings for a given category, ordered by key.
func (s *SettingsService) ListByCategory(ctx context.Context, category string) ([]PlatformSetting, error) {
	var rows []PlatformSetting
	err := s.db.WithContext(ctx).Where("category = ?", category).Order("key").Find(&rows).Error
	return rows, err
}

// ListAll returns every setting row.
func (s *SettingsService) ListAll(ctx context.Context) ([]PlatformSetting, error) {
	var rows []PlatformSetting
	err := s.db.WithContext(ctx).Order("category, key").Find(&rows).Error
	return rows, err
}

// StorageTTL returns the configured TTL duration for the given membership tier.
// Tier values: "BRONZE", "SILVER", "GOLD", "PLATINUM", "" (free/unknown).
// Defaults match the migration seed values.
func (s *SettingsService) StorageTTL(ctx context.Context, tier string) time.Duration {
	var key string
	var defaultHours int
	switch tier {
	case "PLATINUM":
		key, defaultHours = "storage_ttl_platinum_hours", 168
	case "GOLD":
		key, defaultHours = "storage_ttl_gold_hours", 72
	case "SILVER":
		key, defaultHours = "storage_ttl_silver_hours", 48
	case "BRONZE":
		key, defaultHours = "storage_ttl_bronze_hours", 48
	default:
		key, defaultHours = "storage_ttl_free_hours", 24
	}
	hours := s.GetInt(ctx, key, defaultHours)
	if hours <= 0 {
		log.Printf("[SettingsService] StorageTTL: invalid value for %s (%d), using default %dh", key, hours, defaultHours)
		hours = defaultHours
	}
	return time.Duration(hours) * time.Hour
}

// ExpiryNotifyWindows returns the two pre-expiry notification lead times (first, second)
// as durations. E.g. (24h, 6h) means notify 24h before expiry, then again 6h before.
func (s *SettingsService) ExpiryNotifyWindows(ctx context.Context) (first, second time.Duration) {
	h1 := s.GetInt(ctx, "notify_expiry_first_hours", 24)
	h2 := s.GetInt(ctx, "notify_expiry_second_hours", 6)
	return time.Duration(h1) * time.Hour, time.Duration(h2) * time.Hour
}
