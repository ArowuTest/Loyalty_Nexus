package config

import (
	"context"
	"fmt"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"gorm.io/gorm"
)

// ConfigManager reads ALL business rules from the network_configs database table.
// ZERO values are ever hardcoded in application logic.
// The singleton is refresh-safe: a background goroutine calls Refresh() every 60s.
type ConfigManager struct {
	db    *gorm.DB
	mu    sync.RWMutex
	cache map[string]json.RawMessage
}

func NewConfigManager(db *gorm.DB) *ConfigManager {
	cm := &ConfigManager{
		db:    db,
		cache: make(map[string]json.RawMessage),
	}
	if err := cm.Refresh(context.Background()); err != nil {
		log.Printf("[CONFIG] Initial refresh failed (DB may be initialising): %v", err)
	}
	go cm.autoRefresh()
	return cm
}

// NewConfigManagerNoRefresh creates a ConfigManager without starting the background
// auto-refresh goroutine. Use this in integration tests to avoid lock contention
// between the test transaction and the background SELECT on network_configs.
func NewConfigManagerNoRefresh(db *gorm.DB) *ConfigManager {
	cm := &ConfigManager{
		db:    db,
		cache: make(map[string]json.RawMessage),
	}
	if err := cm.Refresh(context.Background()); err != nil {
		log.Printf("[CONFIG] Initial refresh failed: %v", err)
	}
	return cm
}

func (c *ConfigManager) autoRefresh() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if err := c.Refresh(context.Background()); err != nil {
			log.Printf("[CONFIG] Auto-refresh error: %v", err)
		}
	}
}

func (c *ConfigManager) Refresh(ctx context.Context) error {
	if c.db == nil {
		return nil // DB not yet connected, skip refresh
	}
	type row struct {
		Key   string `gorm:"column:key"`
		Value string `gorm:"column:value"`
	}
	var rows []row
	if err := c.db.WithContext(ctx).Table("network_configs").Select("key, value").Find(&rows).Error; err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, r := range rows {
		// Wrap plain values in JSON quotes if they are not already valid JSON.
		raw := json.RawMessage(r.Value)
		if len(r.Value) == 0 || (r.Value[0] != '{' && r.Value[0] != '[' && r.Value[0] != '"' &&
			!((r.Value[0] >= '0' && r.Value[0] <= '9') || r.Value[0] == '-') &&
			r.Value != "true" && r.Value != "false" && r.Value != "null") {
			raw = json.RawMessage(`"` + r.Value + `"`)
		}
		c.cache[r.Key] = raw
	}
	return nil
}

// raw returns the raw JSON for a key, checking env var override first.
func (c *ConfigManager) raw(key string) (string, bool) {
	// Env overrides take precedence (useful for Docker secrets)
	if v := os.Getenv("CFG_" + key); v != "" {
		return v, true
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if v, ok := c.cache[key]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err == nil {
			return s, true
		}
		return string(v), true
	}
	return "", false
}

func (c *ConfigManager) GetString(key, defaultVal string) string {
	if v, ok := c.raw(key); ok {
		// Strip surrounding quotes if JSON string
		if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
			return v[1 : len(v)-1]
		}
		return v
	}
	return defaultVal
}

func (c *ConfigManager) GetInt(key string, defaultVal int) int {
	if v, ok := c.raw(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}

func (c *ConfigManager) GetInt64(key string, defaultVal int64) int64 {
	if v, ok := c.raw(key); ok {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return defaultVal
}

func (c *ConfigManager) GetFloat(key string, defaultVal float64) float64 {
	if v, ok := c.raw(key); ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

func (c *ConfigManager) GetBool(key string, defaultVal bool) bool {
	if v, ok := c.raw(key); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return defaultVal
}

// Set writes a key-value pair to network_configs and refreshes the in-memory cache.
// This is used by admin endpoints to update business rules at runtime.
func (c *ConfigManager) Set(ctx context.Context, key, value string) error {
	if c.db == nil {
		return fmt.Errorf("database not connected")
	}
	err := c.db.WithContext(ctx).Exec(
		`INSERT INTO network_configs (key, value, updated_at)
		 VALUES (?, ?, NOW())
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
		key, value,
	).Error
	if err != nil {
		return err
	}
	// Immediately update the in-memory cache so the new value is visible
	// to the current process without waiting for the 60s auto-refresh.
	c.mu.Lock()
	c.cache[key] = json.RawMessage(value)
	c.mu.Unlock()
	return nil
}

// IsIndependentMode reads OPERATION_MODE — never hardcoded.
func (c *ConfigManager) IsIndependentMode() bool {
	return c.GetString("operation_mode", "independent") == "independent"
}
