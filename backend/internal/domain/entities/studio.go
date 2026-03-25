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
	ID           uuid.UUID    `json:"id"`
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	Category     ToolCategory `json:"category"`
	PointCost    int64        `json:"point_cost"`
	Provider     string       `json:"-"` // Hidden from frontend
	ProviderTool string       `json:"-"` // e.g., "flux-schnell"
	IsActive     bool         `json:"is_active"`
	Icon         string       `json:"icon"`
}

type AIGeneration struct {
	ID            uuid.UUID `json:"id"`
	UserID        uuid.UUID `json:"user_id"`
	ToolID        uuid.UUID `json:"tool_id"`
	Prompt        string    `json:"prompt"`
	Status        string    `json:"status"` // pending, completed, failed
	OutputURL     string    `json:"output_url,omitempty"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	PointsDeducted int64     `json:"points_deducted"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
}
