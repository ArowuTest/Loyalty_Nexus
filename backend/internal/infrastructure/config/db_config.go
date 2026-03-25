package config

import (
	"context"
	"encoding/json"
	"sync"
	"time"
	"gorm.io/gorm"
)

type ConfigManager struct {
	db         *gorm.DB
	cache      map[string]any
	mu         sync.RWMutex
	lastUpdate time.Time
}

func NewConfigManager(db *gorm.DB) *ConfigManager {
	return &ConfigManager{
		db:    db,
		cache: make(map[string]any),
	}
}

func (m *ConfigManager) Refresh(ctx context.Context) error {
	type configRow struct {
		ConfigKey   string
		ConfigValue json.RawMessage
	}
	var rows []configRow
	if err := m.db.WithContext(ctx).Table("program_configs").Select("config_key, config_value").Find(&rows).Error; err != nil {
		return err
	}

	newCache := make(map[string]any)
	for _, r := range rows {
		var val any
		json.Unmarshal(r.ConfigValue, &val)
		newCache[r.ConfigKey] = val
	}

	m.mu.Lock()
	m.cache = newCache
	m.lastUpdate = time.Now()
	m.mu.Unlock()
	return nil
}

func (m *ConfigManager) GetInt(key string, fallback int) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if v, ok := m.cache[key]; ok {
		if f, ok := v.(float64); ok {
			return int(f)
		}
	}
	return fallback
}

func (m *ConfigManager) GetFloat(key string, fallback float64) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if v, ok := m.cache[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return fallback
}
