package entities

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ProviderExtraConfig is a flexible JSON bag for template-specific params.
// e.g. { "voice_id": "EXAVITQu4vr4xnSDxMaL", "language": "en" }
type ProviderExtraConfig map[string]interface{}

func (c ProviderExtraConfig) Value() (driver.Value, error) {
	if c == nil {
		return "{}", nil
	}
	b, err := json.Marshal(c)
	return string(b), err
}

func (c *ProviderExtraConfig) Scan(src interface{}) error {
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, c)
	case string:
		return json.Unmarshal([]byte(v), c)
	default:
		return fmt.Errorf("ProviderExtraConfig.Scan: unsupported type %T", src)
	}
}

// AIProviderConfig represents a single AI provider registered by an admin.
// Multiple providers can share the same category — they form a priority-ordered
// fallback chain that the dispatcher tries top-to-bottom automatically.
type AIProviderConfig struct {
	ID           uuid.UUID           `json:"id"             gorm:"column:id;primaryKey"`
	Name         string              `json:"name"           gorm:"column:name"`
	Slug         string              `json:"slug"           gorm:"column:slug;uniqueIndex"`
	Category     string              `json:"category"       gorm:"column:category;index"` // text|image|video|tts|transcribe|translate|music|bg-remove|vision
	Template     string              `json:"template"       gorm:"column:template"`        // driver key
	EnvKey       string              `json:"env_key"        gorm:"column:env_key"`         // env var name (never expose value)
	APIKeyEnc    string              `json:"-"              gorm:"column:api_key_enc"`     // AES-GCM encrypted, never sent to frontend
	ModelID      string              `json:"model_id"       gorm:"column:model_id"`
	ExtraConfig  ProviderExtraConfig `json:"extra_config"   gorm:"column:extra_config;serializer:json;type:jsonb"`
	Priority     int                 `json:"priority"       gorm:"column:priority"`        // 1=primary, higher=backup
	IsPrimary    bool                `json:"is_primary"     gorm:"column:is_primary"`
	IsActive     bool                `json:"is_active"      gorm:"column:is_active"`
	CostMicros   int                 `json:"cost_micros"    gorm:"column:cost_micros"`
	PulsePts     int                 `json:"pulse_pts"      gorm:"column:pulse_pts"`
	Notes        string              `json:"notes"          gorm:"column:notes"`
	LastTestedAt *time.Time          `json:"last_tested_at" gorm:"column:last_tested_at"`
	LastTestOK   *bool               `json:"last_test_ok"   gorm:"column:last_test_ok"`
	LastTestMsg  string              `json:"last_test_msg"  gorm:"column:last_test_msg"`
	CreatedAt    time.Time           `json:"created_at"     gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time           `json:"updated_at"     gorm:"column:updated_at;autoUpdateTime"`

	// HasKey is set by the repo (true if env_key is set in env OR api_key_enc is non-empty).
	// Never persisted — computed at read time.
	HasKey bool `json:"has_key" gorm:"-"`
}

func (AIProviderConfig) TableName() string { return "ai_provider_configs" }

// ── Category constants ────────────────────────────────────────────────────────
const (
	ProviderCategoryText       = "text"
	ProviderCategoryImage      = "image"
	ProviderCategoryVideo      = "video"
	ProviderCategoryTTS        = "tts"
	ProviderCategoryTranscribe = "transcribe"
	ProviderCategoryTranslate  = "translate"
	ProviderCategoryMusic      = "music"
	ProviderCategoryBGRemove   = "bg-remove"
	ProviderCategoryVision     = "vision"
)

// ── Template constants ────────────────────────────────────────────────────────
// Templates define WHICH driver function to call.
const (
	TemplatePollText        = "openai-compatible"   // generic OpenAI-compat POST /v1/chat/completions
	TemplatePollImage       = "pollinations-image"
	TemplatePollTTS         = "pollinations-tts"
	TemplatePollVideo       = "pollinations-video"
	TemplatePollMusic       = "pollinations-music"
	TemplateGemini          = "gemini"
	TemplateDeepSeek        = "deepseek"
	TemplateGroqWhisper     = "groq-whisper"
	TemplateAssemblyAI      = "assemblyai"
	TemplateGoogleTTS       = "google-tts"
	TemplateGoogleTranslate = "google-translate"
	TemplateHFImage         = "hf-image"
	TemplateFALImage        = "fal-image"
	TemplateFALVideo        = "fal-video"
	TemplateFALBGRemove     = "fal-bg-remove"
	TemplateElevenLabsTTS   = "elevenlabs-tts"
	TemplateElevenLabsMusic = "elevenlabs-music"
	TemplateMubert          = "mubert"
	TemplateRemoveBG        = "remove-bg"
	TemplateRembg           = "rembg"
)

// ValidCategories and ValidTemplates for frontend dropdowns.
var ValidCategories = []string{
	ProviderCategoryText, ProviderCategoryImage, ProviderCategoryVideo,
	ProviderCategoryTTS, ProviderCategoryTranscribe, ProviderCategoryTranslate,
	ProviderCategoryMusic, ProviderCategoryBGRemove, ProviderCategoryVision,
}

var ValidTemplates = []string{
	TemplatePollText, TemplatePollImage, TemplatePollTTS, TemplatePollVideo, TemplatePollMusic,
	TemplateGemini, TemplateDeepSeek, TemplateGroqWhisper, TemplateAssemblyAI,
	TemplateGoogleTTS, TemplateGoogleTranslate,
	TemplateHFImage, TemplateFALImage, TemplateFALVideo, TemplateFALBGRemove,
	TemplateElevenLabsTTS, TemplateElevenLabsMusic,
	TemplateMubert, TemplateRemoveBG, TemplateRembg,
}
