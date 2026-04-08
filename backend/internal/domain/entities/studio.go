package entities

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// UIConfig is a flexible JSON bag that the frontend uses to render
// the correct purpose-built input form for each studio tool.
// The schema varies by ui_template (see migration 032).
type UIConfig map[string]interface{}

// Value implements driver.Valuer so GORM can serialise to JSONB.
func (c UIConfig) Value() (driver.Value, error) {
	if c == nil {
		return "{}", nil
	}
	b, err := json.Marshal(c)
	return string(b), err
}

// Scan implements sql.Scanner so GORM can deserialise from JSONB.
func (c *UIConfig) Scan(src interface{}) error {
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, c)
	case string:
		return json.Unmarshal([]byte(v), c)
	default:
		return fmt.Errorf("UIConfig.Scan: unsupported type %T", src)
	}
}

type ToolCategory string

const (
	CategoryChat   ToolCategory = "Chat"
	CategoryCreate ToolCategory = "Create"
	CategoryLearn  ToolCategory = "Learn"
	CategoryBuild  ToolCategory = "Build"
)

type StudioTool struct {
	ID           uuid.UUID    `json:"id"           gorm:"column:id;primaryKey"`
	Name         string       `json:"name"         gorm:"column:name"`
	Slug         string       `json:"slug"         gorm:"column:slug;uniqueIndex"`
	Description  string       `json:"description"  gorm:"column:description"`
	Category     ToolCategory `json:"category"     gorm:"column:category"`
	PointCost    int64        `json:"point_cost"   gorm:"column:point_cost"`
	Provider     string       `json:"-"            gorm:"column:provider;default:''"`
	ProviderTool string       `json:"-"            gorm:"column:provider_tool;default:''"`
	IsActive     bool         `json:"is_active"    gorm:"column:is_active;default:true"`
	Icon         string       `json:"icon"         gorm:"column:icon;default:''"`
	SortOrder        int          `json:"sort_order"         gorm:"column:sort_order;default:0"`
	EntryPointCost   int64        `json:"entry_point_cost"   gorm:"column:entry_point_cost;default:0"`
	RefundWindowMins int          `json:"refund_window_mins"  gorm:"column:refund_window_mins;default:5"`
	RefundPct        int          `json:"refund_pct"          gorm:"column:refund_pct;default:100"`
	IsFree           bool         `json:"is_free"             gorm:"column:is_free;default:false"`
	// UITemplate tells the frontend which input form to render (e.g. "music-composer", "image-creator").
	// UIConfig   carries the full parameter set for that template as a JSON bag.
	// Both are populated by migration 032 and can be updated via the admin panel.
	UITemplate string   `json:"ui_template" gorm:"column:ui_template;default:'knowledge-doc'"`
	UIConfig   UIConfig `json:"ui_config"   gorm:"column:ui_config;serializer:json;type:jsonb"`
	CreatedAt  time.Time `json:"created_at"  gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time `json:"updated_at"  gorm:"column:updated_at;autoUpdateTime"`
}

func (StudioTool) TableName() string { return "studio_tools" }

type AIGeneration struct {
	ID             uuid.UUID `json:"id"            gorm:"column:id;primaryKey"`
	UserID         uuid.UUID `json:"user_id"       gorm:"column:user_id;index"`
	ToolID         uuid.UUID `json:"tool_id"       gorm:"column:tool_id"`
	ToolSlug       string    `json:"tool_slug"     gorm:"column:tool_slug;default:''"`
	Prompt         string    `json:"prompt"        gorm:"column:prompt"`
	Status         string    `json:"status"        gorm:"column:status"` // pending | processing | completed | failed
	VanitySlug     string    `json:"vanity_slug,omitempty" db:"slug" gorm:"column:slug;default:null"` // e.g. "techvault-solutions"
	OutputURL      string    `json:"output_url,omitempty"     gorm:"column:output_url;default:''"` 
	OutputURL2     string    `json:"output_url_2,omitempty"   gorm:"column:output_url_2;default:''"` 
	OutputText     string    `json:"output_text,omitempty"    gorm:"column:output_text;default:''"` 
	ErrorMessage   string    `json:"error_message,omitempty" gorm:"column:error_message;default:''"`
	Provider       string    `json:"provider,omitempty"      gorm:"column:provider;default:''"`
	CostMicros     int       `json:"cost_micros"   gorm:"column:cost_micros;default:0"`
	DurationMs     int       `json:"duration_ms"   gorm:"column:duration_ms;default:0"`
	PointsDeducted int64      `json:"points_deducted" gorm:"column:points_deducted"`
	DisputedAt    *time.Time `json:"disputed_at,omitempty"  gorm:"column:disputed_at"`
	RefundGranted bool       `json:"refund_granted"         gorm:"column:refund_granted;default:false"`
	RefundPts     int64      `json:"refund_pts"             gorm:"column:refund_pts;default:0"`
	CreatedAt     time.Time  `json:"created_at"    gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time  `json:"updated_at"    gorm:"column:updated_at;autoUpdateTime"`
	ExpiresAt     time.Time  `json:"expires_at"    gorm:"column:expires_at"`
}

func (AIGeneration) TableName() string { return "ai_generations" }

// StudioSession tracks a user's activity window across multiple generations.
type StudioSession struct {
	ID              uuid.UUID  `json:"id"               gorm:"column:id;primaryKey"`
	UserID          uuid.UUID  `json:"user_id"          gorm:"column:user_id;index"`
	StartedAt       time.Time  `json:"started_at"       gorm:"column:started_at;autoCreateTime"`
	LastActiveAt    time.Time  `json:"last_active_at"   gorm:"column:last_active_at"`
	EndedAt         *time.Time `json:"ended_at,omitempty" gorm:"column:ended_at"`
	TotalPtsUsed    int64      `json:"total_pts_used"   gorm:"column:total_pts_used;default:0"`
	GenerationCount int        `json:"generation_count" gorm:"column:generation_count;default:0"`
}

func (StudioSession) TableName() string { return "studio_sessions" }
