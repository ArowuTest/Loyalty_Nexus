package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"github.com/google/uuid"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/infrastructure/persistence"
)

type AIModelIntelligenceAdminHandler struct {
	intel     *persistence.AIModelIntelligenceRepository
	providers *persistence.AIProviderRepository
	routing   *persistence.AIRoutingRepository
}

func NewAIModelIntelligenceAdminHandler(
	intel *persistence.AIModelIntelligenceRepository,
	providers *persistence.AIProviderRepository,
	routing *persistence.AIRoutingRepository,
) *AIModelIntelligenceAdminHandler {
	return &AIModelIntelligenceAdminHandler{intel: intel, providers: providers, routing: routing}
}
func (h *AIModelIntelligenceAdminHandler) ListModels(w http.ResponseWriter, r *http.Request) {
	freeOnly := strings.EqualFold(r.URL.Query().Get("free_only"), "true")
	rows, err := h.intel.ListAvailable(r.Context(), "openrouter", freeOnly)
	if err != nil {
		jsonError(w, "failed to load model catalog", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]interface{}{"models": rows, "count": len(rows)})
}

func (h *AIModelIntelligenceAdminHandler) ListRecommendations(w http.ResponseWriter, r *http.Request) {
	status := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	rows, err := h.intel.ListRecommendations(r.Context(), status, limit)
	if err != nil {
		jsonError(w, "failed to load recommendations", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]interface{}{"recommendations": rows, "count": len(rows)})
}

func (h *AIModelIntelligenceAdminHandler) RejectRecommendation(w http.ResponseWriter, r *http.Request) {
	if !requireRoutingAdmin(w, r) {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		jsonError(w, "invalid recommendation id", 400)
		return
	}
	if err := h.intel.SetRecommendationStatus(r.Context(), id, "REJECTED"); err != nil {
		jsonError(w, "recommendation not found", 404)
		return
	}
	jsonOK(w, map[string]string{"status": "REJECTED"})
}
func (h *AIModelIntelligenceAdminHandler) ApproveRecommendation(w http.ResponseWriter, r *http.Request) {
	if !requireRoutingAdmin(w, r) {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		jsonError(w, "invalid recommendation id", 400)
		return
	}

	rec, err := h.intel.GetRecommendation(r.Context(), id)
	if err != nil || rec.Status != "PENDING" {
		jsonError(w, "pending recommendation not found", 404)
		return
	}
	model, err := h.intel.GetCatalogByID(r.Context(), rec.ModelCatalogID)
	if err != nil || !model.IsAvailable || !model.IsFree {
		jsonError(w, "recommended free model is no longer available", 409)
		return
	}
	stage, err := h.routing.ResolveStage(r.Context(), rec.ToolSlug, rec.StageKey)
	if err != nil {
		jsonError(w, "target tool stage no longer exists", 409)
		return
	}
	source, err := h.findOpenRouterSource(r)
	if err != nil {
		jsonError(w, err.Error(), 409)
		return
	}
	provider, err := h.ensureOpenRouterModelProvider(r, source, model, stage)
	if err != nil {
		jsonError(w, "failed to provision recommended model: "+err.Error(), 500)
		return
	}

	priority := 90
	if rec.Recommendation == "ADD_AS_PRIMARY" {
		priority = 5
	}
	bindings, _ := h.routing.ListBindingsForStage(r.Context(), stage.ID)
	for _, b := range bindings {
		if b.ProviderID != provider.ID {
			continue
		}
		err = h.routing.UpdateBinding(r.Context(), b.ID, map[string]interface{}{
			"is_active": true, "cost_tier": entities.CostTierFree, "priority": priority,
			"config_version": b.ConfigVersion + 1,
		})
		if err == nil {
			_ = h.intel.SetRecommendationStatus(r.Context(), id, "APPROVED")
			jsonOK(w, map[string]interface{}{"status": "APPROVED", "provider": provider, "binding_id": b.ID})
			return
		}
	}

	binding := &entities.AIToolProviderBinding{
		ID: uuid.New(), StageID: stage.ID, ProviderID: provider.ID, Priority: priority,
		IsActive: true, CostTier: entities.CostTierFree, MaxConcurrent: 0, RequestsPerMinute: 0,
		TimeoutMS: 120000, MaxRetries: 0, AllowPaidFallback: true, ConfigVersion: 1,
		Notes: "Created from approved AI Model Scout recommendation",
	}
	if err := h.routing.CreateBinding(r.Context(), binding); err != nil {
		jsonError(w, "failed to bind recommended model: "+err.Error(), 500)
		return
	}
	_ = h.routing.RecordRoutingChange(r.Context(), "binding", binding.ID, "scout-approve",
		map[string]interface{}{}, binding, routingChangedBy(r))
	_ = h.intel.SetRecommendationStatus(r.Context(), id, "APPROVED")
	jsonOK(w, map[string]interface{}{"status": "APPROVED", "provider": provider, "binding": binding})
}
func (h *AIModelIntelligenceAdminHandler) findOpenRouterSource(r *http.Request) (*entities.AIProviderConfig, error) {
	rows, err := h.providers.ListAll(r.Context())
	if err != nil {
		return nil, err
	}
	for i := range rows {
		p := &rows[i]
		source, _ := p.ExtraConfig["catalog_source"].(string)
		base, _ := p.ExtraConfig["base_url"].(string)
		if p.IsActive && (strings.EqualFold(source, "openrouter") || strings.Contains(strings.ToLower(base), "openrouter.ai")) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no active OpenRouter source provider is configured")
}

func (h *AIModelIntelligenceAdminHandler) ensureOpenRouterModelProvider(
	r *http.Request,
	source *entities.AIProviderConfig,
	model *entities.AIModelCatalog,
	stage *entities.AIToolStage,
) (*entities.AIProviderConfig, error) {
	slug := "openrouter-" + safeModelSlug(model.ModelID)
	if existing, err := h.providers.GetBySlug(r.Context(), slug); err == nil {
		return existing, nil
	}

	extra := entities.ProviderExtraConfig{}
	for k, v := range source.ExtraConfig {
		extra[k] = v
	}
	extra["base_url"] = "https://openrouter.ai/api"
	extra["catalog_source"] = "openrouter"
	category := entities.ProviderCategoryText
	if stage.Capability == "text.vision" {
		category = entities.ProviderCategoryVision
	}
	p := &entities.AIProviderConfig{
		ID: uuid.New(), Name: "OpenRouter · " + model.DisplayName, Slug: slug,
		Category: category, Template: entities.TemplatePollText,
		EnvKey: source.EnvKey, APIKeyEnc: source.APIKeyEnc, ModelID: model.ModelID,
		ExtraConfig: extra, Priority: 100, IsPrimary: false, IsActive: true,
		CostMicros: 0, PulsePts: 0, Notes: "Provisioned from AI Model Scout approval",
	}
	if err := h.providers.Create(r.Context(), p); err != nil {
		return nil, err
	}
	return p, nil
}

func safeModelSlug(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			continue
		}
		if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func extraInt(cfg entities.ProviderExtraConfig, key string, fallback int) int {
	v, ok := cfg[key]
	if !ok {
		return fallback
	}
	switch n := v.(type) {
	case float64:
		if n >= 0 {
			return int(n)
		}
	case int:
		if n >= 0 {
			return n
		}
	}
	return fallback
}
