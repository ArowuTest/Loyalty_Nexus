// Package persistence implements the repository layer for Loyalty Nexus,
// providing database access via GORM for all domain entities.
package persistence

import (
	"context"
	"os"
	"time"

	"gorm.io/gorm"

	"loyalty-nexus/internal/domain/entities"
)

// AIProviderRepository handles CRUD for ai_provider_configs.
type AIProviderRepository struct {
	db *gorm.DB
}

func NewAIProviderRepository(db *gorm.DB) *AIProviderRepository {
	return &AIProviderRepository{db: db}
}

// ListAll returns all providers ordered by category, then priority.
func (r *AIProviderRepository) ListAll(ctx context.Context) ([]entities.AIProviderConfig, error) {
	var rows []entities.AIProviderConfig
	err := r.db.WithContext(ctx).
		Order("category, priority, created_at").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	r.populateHasKey(rows)
	return rows, nil
}

// ListByCategory returns active providers for a category sorted by priority (ascending).
func (r *AIProviderRepository) ListByCategory(ctx context.Context, category string) ([]entities.AIProviderConfig, error) {
	var rows []entities.AIProviderConfig
	err := r.db.WithContext(ctx).
		Where("category = ? AND is_active = true", category).
		Order("priority ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	r.populateHasKey(rows)
	return rows, nil
}

// GetBySlug returns a single provider by slug.
func (r *AIProviderRepository) GetBySlug(ctx context.Context, slug string) (*entities.AIProviderConfig, error) {
	var row entities.AIProviderConfig
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&row).Error
	if err != nil {
		return nil, err
	}
	row.HasKey = r.keyPresent(&row)
	return &row, nil
}

// GetByID returns a single provider by UUID.
func (r *AIProviderRepository) GetByID(ctx context.Context, id string) (*entities.AIProviderConfig, error) {
	var row entities.AIProviderConfig
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if err != nil {
		return nil, err
	}
	row.HasKey = r.keyPresent(&row)
	return &row, nil
}

// Create inserts a new provider config.
func (r *AIProviderRepository) Create(ctx context.Context, p *entities.AIProviderConfig) error {
	return r.db.WithContext(ctx).Create(p).Error
}

// Update saves changes to an existing provider.
func (r *AIProviderRepository) Update(ctx context.Context, p *entities.AIProviderConfig) error {
	return r.db.WithContext(ctx).Save(p).Error
}

// UpdateTestResult records the last health-check result.
func (r *AIProviderRepository) UpdateTestResult(ctx context.Context, id string, ok bool, msg string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Table("ai_provider_configs").
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_tested_at": now,
			"last_test_ok":   ok,
			"last_test_msg":  msg,
			"updated_at":     now,
		}).Error
}

// SetActive enables or disables a provider.
func (r *AIProviderRepository) SetActive(ctx context.Context, id string, active bool) error {
	return r.db.WithContext(ctx).
		Table("ai_provider_configs").
		Where("id = ?", id).
		Updates(map[string]interface{}{"is_active": active, "updated_at": time.Now()}).Error
}

// Delete permanently removes a provider config.
func (r *AIProviderRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&entities.AIProviderConfig{}).Error
}

// populateHasKey fills the transient HasKey field for a slice.
func (r *AIProviderRepository) populateHasKey(rows []entities.AIProviderConfig) {
	for i := range rows {
		rows[i].HasKey = r.keyPresent(&rows[i])
	}
}

// keyPresent returns true if the provider has a usable key
// (either the named env var is set, or an encrypted key is stored).
func (r *AIProviderRepository) keyPresent(p *entities.AIProviderConfig) bool {
	if p.APIKeyEnc != "" {
		return true
	}
	if p.EnvKey != "" && os.Getenv(p.EnvKey) != "" {
		return true
	}
	return false
}
