package entities

import (
	"time"

	"github.com/google/uuid"
)

const (
	RoutingFreeFirst    = "FREE_FIRST"
	RoutingBalanced     = "BALANCED"
	RoutingQualityFirst = "QUALITY_FIRST"
	RoutingFreeOnly     = "FREE_ONLY"
	RoutingPremiumOnly  = "PREMIUM_ONLY"
)

const (
	QueueRealtime    = "REALTIME"
	QueueInteractive = "INTERACTIVE"
	QueueAsync       = "ASYNC"
	QueueHeavyAsync  = "HEAVY_ASYNC"
	QueueBackground  = "BACKGROUND"
)

const (
	CostTierFree    = "FREE"
	CostTierLowCost = "LOW_COST"
	CostTierPremium = "PREMIUM"
)

type AIToolStage struct {
	ID                     uuid.UUID `json:"id" gorm:"column:id;primaryKey"`
	ToolID                 uuid.UUID `json:"tool_id" gorm:"column:tool_id;index"`
	StageKey               string    `json:"stage_key" gorm:"column:stage_key"`
	Capability             string    `json:"capability" gorm:"column:capability;index"`
	RoutingPolicy          string    `json:"routing_policy" gorm:"column:routing_policy"`
	QueueClass             string    `json:"queue_class" gorm:"column:queue_class"`
	MaxQueueSeconds        int       `json:"max_queue_seconds" gorm:"column:max_queue_seconds"`
	PaidHourlyBudgetMicros int64     `json:"paid_hourly_budget_micros" gorm:"column:paid_hourly_budget_micros"`
	PaidDailyBudgetMicros  int64     `json:"paid_daily_budget_micros" gorm:"column:paid_daily_budget_micros"`
	IsRequired             bool      `json:"is_required" gorm:"column:is_required"`
	IsActive               bool      `json:"is_active" gorm:"column:is_active"`
	SortOrder              int       `json:"sort_order" gorm:"column:sort_order"`
	CreatedAt              time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt              time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (AIToolStage) TableName() string { return "ai_tool_stages" }

type AIToolProviderBinding struct {
	ID                      uuid.UUID           `json:"id" gorm:"column:id;primaryKey"`
	StageID                 uuid.UUID           `json:"stage_id" gorm:"column:stage_id;index"`
	ProviderID              uuid.UUID           `json:"provider_id" gorm:"column:provider_id;index"`
	Priority                int                 `json:"priority" gorm:"column:priority"`
	IsActive                bool                `json:"is_active" gorm:"column:is_active"`
	CostTier                string              `json:"cost_tier" gorm:"column:cost_tier"`
	MaxConcurrent           int                 `json:"max_concurrent" gorm:"column:max_concurrent"`
	RequestsPerMinute       int                 `json:"requests_per_minute" gorm:"column:requests_per_minute"`
	TimeoutMS               int                 `json:"timeout_ms" gorm:"column:timeout_ms"`
	MaxRetries              int                 `json:"max_retries" gorm:"column:max_retries"`
	AllowPaidFallback       bool                `json:"allow_paid_fallback" gorm:"column:allow_paid_fallback"`
	CircuitFailureThreshold int                 `json:"circuit_failure_threshold" gorm:"column:circuit_failure_threshold"`
	CircuitOpenSeconds      int                 `json:"circuit_open_seconds" gorm:"column:circuit_open_seconds"`
	RequestConfig           ProviderExtraConfig `json:"request_config" gorm:"column:request_config;serializer:json;type:jsonb"`
	ConfigVersion           int64               `json:"config_version" gorm:"column:config_version"`
	Notes                   string              `json:"notes" gorm:"column:notes"`
	CreatedAt               time.Time           `json:"created_at" gorm:"column:created_at"`
	UpdatedAt               time.Time           `json:"updated_at" gorm:"column:updated_at"`
}

func (AIToolProviderBinding) TableName() string { return "ai_tool_provider_bindings" }

type AIRouteCandidate struct {
	Stage    AIToolStage
	Binding  AIToolProviderBinding
	Provider AIProviderConfig
}

type AIGenerationAttempt struct {
	ID           uuid.UUID  `json:"id" gorm:"column:id;primaryKey"`
	GenerationID *uuid.UUID `json:"generation_id,omitempty" gorm:"column:generation_id;index"`
	ToolID       *uuid.UUID `json:"tool_id,omitempty" gorm:"column:tool_id"`
	StageID      *uuid.UUID `json:"stage_id,omitempty" gorm:"column:stage_id"`
	BindingID    *uuid.UUID `json:"binding_id,omitempty" gorm:"column:binding_id"`
	ProviderID   *uuid.UUID `json:"provider_id,omitempty" gorm:"column:provider_id"`
	StageKey     string     `json:"stage_key" gorm:"column:stage_key"`
	AttemptNo    int        `json:"attempt_no" gorm:"column:attempt_no"`
	ProviderSlug string     `json:"provider_slug" gorm:"column:provider_slug"`
	ModelID      string     `json:"model_id" gorm:"column:model_id"`
	Outcome      string     `json:"outcome" gorm:"column:outcome"`
	ErrorClass   string     `json:"error_class" gorm:"column:error_class"`
	ErrorMessage string     `json:"error_message" gorm:"column:error_message"`
	HTTPStatus   int        `json:"http_status" gorm:"column:http_status"`
	DurationMS   int        `json:"duration_ms" gorm:"column:duration_ms"`
	CostMicros   int        `json:"cost_micros" gorm:"column:cost_micros"`
	StartedAt    time.Time  `json:"started_at" gorm:"column:started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty" gorm:"column:completed_at"`
}

func (AIGenerationAttempt) TableName() string { return "ai_generation_attempts" }

type AIModelCatalog struct {
	ID            uuid.UUID           `json:"id" gorm:"column:id;primaryKey"`
	Source        string              `json:"source" gorm:"column:source"`
	ModelID       string              `json:"model_id" gorm:"column:model_id"`
	Provider      string              `json:"provider" gorm:"column:provider"`
	DisplayName   string              `json:"display_name" gorm:"column:display_name"`
	IsFree        bool                `json:"is_free" gorm:"column:is_free"`
	IsAvailable   bool                `json:"is_available" gorm:"column:is_available"`
	ContextWindow int64               `json:"context_window" gorm:"column:context_window"`
	Capabilities  ProviderExtraConfig `json:"capabilities" gorm:"column:capabilities;serializer:json;type:jsonb"`
	Pricing       ProviderExtraConfig `json:"pricing" gorm:"column:pricing;serializer:json;type:jsonb"`
	Metadata      ProviderExtraConfig `json:"metadata" gorm:"column:metadata;serializer:json;type:jsonb"`
	DiscoveredAt  time.Time           `json:"discovered_at" gorm:"column:discovered_at"`
	LastSeenAt    time.Time           `json:"last_seen_at" gorm:"column:last_seen_at"`
}

func (AIModelCatalog) TableName() string { return "ai_model_catalog" }

type AIModelScore struct {
	ID             uuid.UUID           `json:"id" gorm:"column:id;primaryKey"`
	ModelCatalogID uuid.UUID           `json:"model_catalog_id" gorm:"column:model_catalog_id;index"`
	Capability     string              `json:"capability" gorm:"column:capability;index"`
	ScoreSource    string              `json:"score_source" gorm:"column:score_source"`
	Score          float64             `json:"score" gorm:"column:score"`
	Metadata       ProviderExtraConfig `json:"metadata" gorm:"column:metadata;serializer:json;type:jsonb"`
	EvaluatedAt    time.Time           `json:"evaluated_at" gorm:"column:evaluated_at"`
}

func (AIModelScore) TableName() string { return "ai_model_scores" }

type AIModelRecommendation struct {
	ID             uuid.UUID  `json:"id" gorm:"column:id;primaryKey"`
	ToolSlug       string     `json:"tool_slug" gorm:"column:tool_slug;index"`
	StageKey       string     `json:"stage_key" gorm:"column:stage_key"`
	ModelCatalogID uuid.UUID  `json:"model_catalog_id" gorm:"column:model_catalog_id;index"`
	Recommendation string     `json:"recommendation" gorm:"column:recommendation"`
	Reason         string     `json:"reason" gorm:"column:reason"`
	Score          float64    `json:"score" gorm:"column:score"`
	Status         string     `json:"status" gorm:"column:status"`
	CreatedAt      time.Time  `json:"created_at" gorm:"column:created_at"`
	ReviewedAt     *time.Time `json:"reviewed_at,omitempty" gorm:"column:reviewed_at"`
}

func (AIModelRecommendation) TableName() string { return "ai_model_recommendations" }

type AIModelScoutTarget struct {
	ToolSlug      string `json:"tool_slug" gorm:"column:tool_slug"`
	StageKey      string `json:"stage_key" gorm:"column:stage_key"`
	Capability    string `json:"capability" gorm:"column:capability"`
	RoutingPolicy string `json:"routing_policy" gorm:"column:routing_policy"`
}
