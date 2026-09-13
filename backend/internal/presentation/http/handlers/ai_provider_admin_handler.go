package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/infrastructure/persistence"
	"loyalty-nexus/internal/presentation/http/middleware"
)

// AIProviderAdminHandler handles CRUD + test for ai_provider_configs.
// Routes (all require adminAuth middleware):
//
//	GET    /api/v1/admin/ai-providers            → ListProviders
//	POST   /api/v1/admin/ai-providers            → CreateProvider
//	PUT    /api/v1/admin/ai-providers/{id}       → UpdateProvider
//	DELETE /api/v1/admin/ai-providers/{id}       → DeleteProvider
//	POST   /api/v1/admin/ai-providers/{id}/activate   → ActivateProvider
//	POST   /api/v1/admin/ai-providers/{id}/deactivate → DeactivateProvider
//	POST   /api/v1/admin/ai-providers/{id}/test  → TestProvider
//	GET    /api/v1/admin/ai-providers/meta       → GetProviderMeta (categories, templates)
type AIProviderAdminHandler struct {
	repo *persistence.AIProviderRepository
}

func NewAIProviderAdminHandler(repo *persistence.AIProviderRepository) *AIProviderAdminHandler {
	return &AIProviderAdminHandler{repo: repo}
}

// requireProviderAdmin mirrors requireRoutingAdmin: provider credentials are the
// most sensitive surface in AI Routing V2, so every mutation and the live Test
// endpoint are super_admin-only. Before this, any admin role (finance/ops/content)
// could rewrite a provider's base_url and Test it — exfiltrating the decrypted
// key (review B1). Read endpoints stay open to any admin: they never serialize
// key material (APIKeyEnc is json:"-").
func requireProviderAdmin(w http.ResponseWriter, r *http.Request) bool {
	return middleware.RequireRole(w, r, entities.RoleSuperAdmin)
}

// providerBaseURL extracts extra_config.base_url ("" when unset).
func providerBaseURL(cfg entities.ProviderExtraConfig) string {
	if cfg == nil {
		return ""
	}
	if u, ok := cfg["base_url"].(string); ok {
		return strings.TrimSpace(u)
	}
	return ""
}

// validateProviderBaseURL accepts only an https:// URL with a host. The stored
// API key is sent as a Bearer token to this host on every live request AND on
// Test, so a plain-http, loopback, or cloud-metadata endpoint would leak it (B1).
func validateProviderBaseURL(u string) error {
	if u == "" {
		return nil
	}
	parsed, err := url.Parse(u)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("extra_config.base_url must be an https:// URL with a host")
	}
	return nil
}

// ── GET /api/v1/admin/ai-providers ───────────────────────────────────────────
func (h *AIProviderAdminHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := h.repo.ListAll(r.Context())
	if err != nil {
		jsonError(w, "failed to list providers: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Group by category for the frontend
	grouped := map[string][]entities.AIProviderConfig{}
	for _, p := range providers {
		grouped[p.Category] = append(grouped[p.Category], p)
	}

	jsonOK(w, map[string]interface{}{
		"providers": providers,
		"grouped":   grouped,
		"total":     len(providers),
	})
}

// ── GET /api/v1/admin/ai-providers/meta ──────────────────────────────────────
func (h *AIProviderAdminHandler) GetProviderMeta(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, map[string]interface{}{
		"categories": entities.ValidCategories,
		"templates":  entities.ValidTemplates,
		"template_descriptions": map[string]string{
			"openai-compatible":  "OpenAI-compatible POST /v1/chat/completions (Groq, Pollinations, DeepSeek…)",
			"pollinations-image": "Pollinations image generation (GET /image/{prompt}?model=…)",
			"pollinations-tts":   "Pollinations TTS (POST /v1/audio/speech)",
			"pollinations-video": "Pollinations video (GET /image/{prompt}?model=seedance/wan-fast)",
			"pollinations-music": "Pollinations ElevenMusic (GET /audio/{prompt}?model=elevenmusic)",
			"gemini":             "Google Gemini (generateContent API)",
			"deepseek":           "DeepSeek Chat API",
			"groq-whisper":       "Groq Whisper transcription (multipart/form-data)",
			"assemblyai":         "AssemblyAI v2 transcription",
			"google-tts":         "Google Cloud Text-to-Speech v1",
			"google-translate":   "Google Translate API v2",
			"hf-image":           "HuggingFace serverless inference image",
			"fal-image":          "FAL.AI image generation",
			"fal-video":          "FAL.AI video generation (Kling, LTX…)",
			"fal-bg-remove":      "FAL.AI background removal (BiRefNet…)",
			"elevenlabs-tts":     "ElevenLabs Text-to-Speech",
			"elevenlabs-music":   "ElevenLabs Music/Sound generation",
			"mubert":             "Mubert royalty-free music (RecordTrackTTM)",
			"remove-bg":          "remove.bg background removal API",
			"rembg":              "Self-hosted rembg microservice",
		},
	})
}

// ── POST /api/v1/admin/ai-providers ──────────────────────────────────────────
func (h *AIProviderAdminHandler) CreateProvider(w http.ResponseWriter, r *http.Request) {
	if !requireProviderAdmin(w, r) {
		return
	}
	var body struct {
		Name        string                       `json:"name"`
		Slug        string                       `json:"slug"`
		Category    string                       `json:"category"`
		Template    string                       `json:"template"`
		EnvKey      string                       `json:"env_key"`
		APIKey      string                       `json:"api_key"` // raw — we encrypt before storing
		ModelID     string                       `json:"model_id"`
		ExtraConfig entities.ProviderExtraConfig `json:"extra_config"`
		Priority    int                          `json:"priority"`
		IsPrimary   bool                         `json:"is_primary"`
		IsActive    bool                         `json:"is_active"`
		CostMicros  int                          `json:"cost_micros"`
		PulsePts    int                          `json:"pulse_pts"`
		Notes       string                       `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if body.Name == "" || body.Category == "" || body.Template == "" {
		jsonError(w, "name, category, and template are required", http.StatusBadRequest)
		return
	}
	if !isValidCategory(body.Category) {
		jsonError(w, "unknown category: "+body.Category, http.StatusBadRequest)
		return
	}
	if !isValidTemplate(body.Template) {
		jsonError(w, "unknown template: "+body.Template, http.StatusBadRequest)
		return
	}
	if err := validateProviderBaseURL(providerBaseURL(body.ExtraConfig)); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Auto-generate slug from name if not provided
	if body.Slug == "" {
		body.Slug = slugifyProvider(body.Name)
	}
	if body.Priority == 0 {
		body.Priority = 99 // default to end of chain
	}

	// Encrypt raw API key if provided
	apiKeyEnc := ""
	if body.APIKey != "" {
		enc, err := encryptProviderKey(body.APIKey)
		if err != nil {
			log.Printf("[AIProviderAdmin] encrypt key error: %v", err)
			jsonError(w, "failed to encrypt API key", http.StatusInternalServerError)
			return
		}
		apiKeyEnc = enc
	}

	p := &entities.AIProviderConfig{
		ID:          uuid.New(),
		Name:        body.Name,
		Slug:        body.Slug,
		Category:    body.Category,
		Template:    body.Template,
		EnvKey:      body.EnvKey,
		APIKeyEnc:   apiKeyEnc,
		ModelID:     body.ModelID,
		ExtraConfig: body.ExtraConfig,
		Priority:    body.Priority,
		IsPrimary:   body.IsPrimary,
		IsActive:    body.IsActive,
		CostMicros:  body.CostMicros,
		PulsePts:    body.PulsePts,
		Notes:       body.Notes,
	}

	if err := h.repo.Create(r.Context(), p); err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			jsonError(w, "slug already exists: "+body.Slug, http.StatusConflict)
			return
		}
		jsonError(w, "failed to create provider: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return without encrypted key
	p.HasKey = body.APIKey != "" || body.EnvKey != ""
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(p); err != nil {
		log.Printf("[AIProviderAdmin] encode error: %v", err)
	}
}

// ── PUT /api/v1/admin/ai-providers/{id} ──────────────────────────────────────
func (h *AIProviderAdminHandler) UpdateProvider(w http.ResponseWriter, r *http.Request) {
	if !requireProviderAdmin(w, r) {
		return
	}
	id := r.PathValue("id")
	if id == "" {
		jsonError(w, "id required", http.StatusBadRequest)
		return
	}

	p, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		jsonError(w, "provider not found", http.StatusNotFound)
		return
	}

	var body struct {
		Name        *string                      `json:"name"`
		Category    *string                      `json:"category"`
		Template    *string                      `json:"template"`
		EnvKey      *string                      `json:"env_key"`
		APIKey      *string                      `json:"api_key"` // raw — encrypted before storing
		ModelID     *string                      `json:"model_id"`
		ExtraConfig entities.ProviderExtraConfig `json:"extra_config"`
		Priority    *int                         `json:"priority"`
		IsPrimary   *bool                        `json:"is_primary"`
		IsActive    *bool                        `json:"is_active"`
		CostMicros  *int                         `json:"cost_micros"`
		PulsePts    *int                         `json:"pulse_pts"`
		Notes       *string                      `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if body.Name != nil {
		p.Name = *body.Name
	}
	if body.Category != nil {
		if !isValidCategory(*body.Category) {
			jsonError(w, "unknown category", http.StatusBadRequest)
			return
		}
		p.Category = *body.Category
	}
	if body.Template != nil {
		if !isValidTemplate(*body.Template) {
			jsonError(w, "unknown template", http.StatusBadRequest)
			return
		}
		p.Template = *body.Template
	}
	if body.EnvKey != nil {
		p.EnvKey = *body.EnvKey
	}
	if body.ModelID != nil {
		p.ModelID = *body.ModelID
	}
	if body.ExtraConfig != nil {
		// B1: bind the stored credential to the endpoint it was provisioned for.
		// The key is sent as a Bearer token to base_url on every live request and
		// on Test, so if base_url changes WITHOUT a fresh api_key the old key is
		// cleared rather than silently pointed at a new host — otherwise an admin
		// could redirect an existing key to an attacker-controlled URL and harvest
		// it. A legitimate endpoint change re-supplies the key in the same call.
		newBase := providerBaseURL(body.ExtraConfig)
		if err := validateProviderBaseURL(newBase); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if newBase != providerBaseURL(p.ExtraConfig) && p.APIKeyEnc != "" &&
			(body.APIKey == nil || *body.APIKey == "") {
			p.APIKeyEnc = ""
			log.Printf("[AIProviderAdmin] base_url changed for %s without a new api_key — stored key cleared; re-supply the key for the new endpoint", p.Slug)
		}
		p.ExtraConfig = body.ExtraConfig
	}
	if body.Priority != nil {
		p.Priority = *body.Priority
	}
	if body.IsPrimary != nil {
		p.IsPrimary = *body.IsPrimary
	}
	if body.IsActive != nil {
		p.IsActive = *body.IsActive
	}
	if body.CostMicros != nil {
		p.CostMicros = *body.CostMicros
	}
	if body.PulsePts != nil {
		p.PulsePts = *body.PulsePts
	}
	if body.Notes != nil {
		p.Notes = *body.Notes
	}

	// Re-encrypt API key if a new one was provided
	if body.APIKey != nil && *body.APIKey != "" {
		enc, err := encryptProviderKey(*body.APIKey)
		if err != nil {
			jsonError(w, "failed to encrypt API key", http.StatusInternalServerError)
			return
		}
		p.APIKeyEnc = enc
	}

	if err := h.repo.Update(r.Context(), p); err != nil {
		jsonError(w, "update failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// Reflect credential presence so the UI can see when a base_url change
	// cleared the key and prompt for it.
	p.HasKey = p.APIKeyEnc != "" || p.EnvKey != ""
	jsonOK(w, p)
}

// ── DELETE /api/v1/admin/ai-providers/{id} ───────────────────────────────────
func (h *AIProviderAdminHandler) DeleteProvider(w http.ResponseWriter, r *http.Request) {
	if !requireProviderAdmin(w, r) {
		return
	}
	id := r.PathValue("id")
	if id == "" {
		jsonError(w, "id required", http.StatusBadRequest)
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		jsonError(w, "delete failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "deleted"})
}

// ── POST /api/v1/admin/ai-providers/{id}/activate ────────────────────────────
func (h *AIProviderAdminHandler) ActivateProvider(w http.ResponseWriter, r *http.Request) {
	if !requireProviderAdmin(w, r) {
		return
	}
	id := r.PathValue("id")
	if err := h.repo.SetActive(r.Context(), id, true); err != nil {
		jsonError(w, "activate failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "activated"})
}

// ── POST /api/v1/admin/ai-providers/{id}/deactivate ──────────────────────────
func (h *AIProviderAdminHandler) DeactivateProvider(w http.ResponseWriter, r *http.Request) {
	if !requireProviderAdmin(w, r) {
		return
	}
	id := r.PathValue("id")
	if err := h.repo.SetActive(r.Context(), id, false); err != nil {
		jsonError(w, "deactivate failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "deactivated"})
}

// ── POST /api/v1/admin/ai-providers/{id}/test ────────────────────────────────
// Performs a minimal live ping against the provider to verify credentials.
func (h *AIProviderAdminHandler) TestProvider(w http.ResponseWriter, r *http.Request) {
	if !requireProviderAdmin(w, r) {
		return
	}
	id := r.PathValue("id")
	p, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		jsonError(w, "provider not found", http.StatusNotFound)
		return
	}

	ok, msg := pingProvider(r.Context(), p)

	// Persist result
	if dbErr := h.repo.UpdateTestResult(r.Context(), id, ok, msg); dbErr != nil {
		log.Printf("[AIProviderAdmin] update test result: %v", dbErr)
	}

	status := "ok"
	if !ok {
		status = "failed"
	}
	jsonOK(w, map[string]interface{}{
		"status":         status,
		"message":        msg,
		"last_tested_at": time.Now().UTC(),
	})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func isValidCategory(c string) bool {
	for _, v := range entities.ValidCategories {
		if v == c {
			return true
		}
	}
	return false
}

func isValidTemplate(t string) bool {
	for _, v := range entities.ValidTemplates {
		if v == t {
			return true
		}
	}
	return false
}

func slugifyProvider(s string) string {
	s = strings.ToLower(s)
	var out strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
		} else {
			out.WriteRune('-')
		}
	}
	// Collapse repeated dashes
	result := strings.Join(strings.Fields(strings.ReplaceAll(out.String(), "-", " ")), "-")
	return strings.Trim(result, "-")
}
