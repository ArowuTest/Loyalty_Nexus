package config

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"time"
)

type ConfigManager struct {
	db         *sql.DB
	cache      map[string]any
	mu         sync.RWMutex
	lastUpdate time.Time
}

func NewConfigManager(db *sql.DB) *ConfigManager {
	return &ConfigManager{
		db:    db,
		cache: make(map[string]any),
	}
}

func (m *ConfigManager) Refresh(ctx context.Context) error {
	rows, err := m.db.QueryContext(ctx, "SELECT config_key, config_value FROM program_configs")
	if err != nil {
		return err
	}
	defer rows.Close()

	newCache := make(map[string]any)
	for rows.Next() {
		var key string
		var valRaw []byte
		if err := rows.Scan(&key, &valRaw); err != nil {
			return err
		}
		var val any
		json.Unmarshal(valRaw, &val)
		newCache[key] = val
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
