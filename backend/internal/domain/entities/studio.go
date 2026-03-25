package entities

import (
	"time"
	"github.com/google/uuid"
)

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
	SortOrder    int          `json:"sort_order"   gorm:"column:sort_order;default:0"`
	CreatedAt    time.Time    `json:"created_at"   gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time    `json:"updated_at"   gorm:"column:updated_at;autoUpdateTime"`
}

func (StudioTool) TableName() string { return "studio_tools" }

type AIGeneration struct {
	ID             uuid.UUID `json:"id"            gorm:"column:id;primaryKey"`
	UserID         uuid.UUID `json:"user_id"       gorm:"column:user_id;index"`
	ToolID         uuid.UUID `json:"tool_id"       gorm:"column:tool_id"`
	ToolSlug       string    `json:"tool_slug"     gorm:"column:tool_slug;default:''"`
	Prompt         string    `json:"prompt"        gorm:"column:prompt"`
	Status         string    `json:"status"        gorm:"column:status"` // pending | processing | completed | failed
	OutputURL      string    `json:"output_url,omitempty"    gorm:"column:output_url;default:''"`
	OutputText     string    `json:"output_text,omitempty"   gorm:"column:output_text;default:''"`
	ErrorMessage   string    `json:"error_message,omitempty" gorm:"column:error_message;default:''"`
	Provider       string    `json:"provider,omitempty"      gorm:"column:provider;default:''"`
	CostMicros     int       `json:"cost_micros"   gorm:"column:cost_micros;default:0"`
	DurationMs     int       `json:"duration_ms"   gorm:"column:duration_ms;default:0"`
	PointsDeducted int64     `json:"points_deducted" gorm:"column:points_deducted"`
	CreatedAt      time.Time `json:"created_at"    gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at"    gorm:"column:updated_at;autoUpdateTime"`
	ExpiresAt      time.Time `json:"expires_at"    gorm:"column:expires_at"`
}

func (AIGeneration) TableName() string { return "ai_generations" }
