package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/infrastructure/persistence"
	"loyalty-nexus/internal/presentation/http/middleware"
)

type AIRoutingAdminHandler struct {
	routing   *persistence.AIRoutingRepository
	providers *persistence.AIProviderRepository
}

func NewAIRoutingAdminHandler(r *persistence.AIRoutingRepository, p *persistence.AIProviderRepository) *AIRoutingAdminHandler {
	return &AIRoutingAdminHandler{routing: r, providers: p}
}

type aiRouteProviderView struct {
	Binding  entities.AIToolProviderBinding `json:"binding"`
	Provider *entities.AIProviderConfig     `json:"provider"`
}

type aiRouteStageView struct {
	Stage      entities.AIToolStage  `json:"stage"`
	Candidates []aiRouteProviderView `json:"candidates"`
}

func (h *AIRoutingAdminHandler) GetToolRoute(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	stages, err := h.routing.ListStagesForTool(r.Context(), slug)
	if err != nil {
		jsonError(w, "failed to load route: "+err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]aiRouteStageView, 0, len(stages))
	for _, stage := range stages {
		bindings, err := h.routing.ListBindingsForStage(r.Context(), stage.ID)
		if err != nil {
			jsonError(w, "failed to load bindings", http.StatusInternalServerError)
			return
		}
		view := aiRouteStageView{Stage: stage, Candidates: []aiRouteProviderView{}}
		for _, b := range bindings {
			p, err := h.providers.GetByID(r.Context(), b.ProviderID.String())
			if err != nil {
				continue
			}
			view.Candidates = append(view.Candidates, aiRouteProviderView{Binding: b, Provider: p})
		}
		out = append(out, view)
	}
	jsonOK(w, map[string]interface{}{"tool_slug": slug, "stages": out})
}

func routingChangedBy(r *http.Request) string {
	if v, ok := r.Context().Value(middleware.ContextUserID).(string); ok {
		return v
	}
	return "admin"
}

func requireRoutingAdmin(w http.ResponseWriter, r *http.Request) bool {
	return middleware.RequireRole(w, r, entities.RoleSuperAdmin)
}

func (h *AIRoutingAdminHandler) UpdateStage(w http.ResponseWriter, r *http.Request) {
	if !requireRoutingAdmin(w, r) {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		jsonError(w, "invalid stage id", http.StatusBadRequest)
		return
	}
	before, err := h.routing.GetStage(r.Context(), id)
	if err != nil {
		jsonError(w, "stage not found", http.StatusNotFound)
		return
	}
	var body struct {
		RoutingPolicy          *string `json:"routing_policy"`
		QueueClass             *string `json:"queue_class"`
		MaxQueueSeconds        *int    `json:"max_queue_seconds"`
		PaidHourlyBudgetMicros *int64  `json:"paid_hourly_budget_micros"`
		PaidDailyBudgetMicros  *int64  `json:"paid_daily_budget_micros"`
		IsRequired             *bool   `json:"is_required"`
		IsActive               *bool   `json:"is_active"`
	}
	if json.NewDecoder(r.Body).Decode(&body) != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	fields := map[string]interface{}{}
	if body.RoutingPolicy != nil {
		if !validRoutingPolicy(*body.RoutingPolicy) {
			jsonError(w, "invalid routing_policy", http.StatusBadRequest)
			return
		}
		fields["routing_policy"] = *body.RoutingPolicy
	}
	if body.QueueClass != nil {
		if !validQueueClass(*body.QueueClass) {
			jsonError(w, "invalid queue_class", http.StatusBadRequest)
			return
		}
		fields["queue_class"] = *body.QueueClass
	}
	if body.MaxQueueSeconds != nil {
		if *body.MaxQueueSeconds < 0 || *body.MaxQueueSeconds > 3600 {
			jsonError(w, "max_queue_seconds must be between 0 and 3600", 400)
			return
		}
		fields["max_queue_seconds"] = *body.MaxQueueSeconds
	}
	if body.PaidHourlyBudgetMicros != nil {
		if *body.PaidHourlyBudgetMicros < 0 {
			jsonError(w, "paid_hourly_budget_micros must be >= 0", 400)
			return
		}
		fields["paid_hourly_budget_micros"] = *body.PaidHourlyBudgetMicros
	}
	if body.PaidDailyBudgetMicros != nil {
		if *body.PaidDailyBudgetMicros < 0 {
			jsonError(w, "paid_daily_budget_micros must be >= 0", 400)
			return
		}
		fields["paid_daily_budget_micros"] = *body.PaidDailyBudgetMicros
	}
	if body.IsRequired != nil {
		fields["is_required"] = *body.IsRequired
	}
	if body.IsActive != nil {
		fields["is_active"] = *body.IsActive
	}
	if len(fields) == 0 {
		jsonError(w, "no fields to update", 400)
		return
	}
	if err := h.routing.UpdateStage(r.Context(), id, fields); err != nil {
		jsonError(w, "update failed", 500)
		return
	}
	after, _ := h.routing.GetStage(r.Context(), id)
	_ = h.routing.RecordRoutingChange(r.Context(), "stage", id, "update", before, after, routingChangedBy(r))
	jsonOK(w, after)
}

func defaultAIRouteLimits(queueClass string) (maxConcurrent, rpm int) {
	switch queueClass {
	case entities.QueueHeavyAsync:
		return 4, 30
	case entities.QueueAsync:
		return 12, 120
	case entities.QueueBackground:
		return 4, 60
	case entities.QueueRealtime, entities.QueueInteractive:
		return 32, 300
	default:
		return 12, 120
	}
}

func (h *AIRoutingAdminHandler) CreateBinding(w http.ResponseWriter, r *http.Request) {
	if !requireRoutingAdmin(w, r) {
		return
	}
	stageID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		jsonError(w, "invalid stage id", 400)
		return
	}
	stage, err := h.routing.GetStage(r.Context(), stageID)
	if err != nil {
		jsonError(w, "stage not found", 404)
		return
	}
	var body struct {
		ProviderID              string                       `json:"provider_id"`
		Priority                int                          `json:"priority"`
		CostTier                string                       `json:"cost_tier"`
		MaxConcurrent           *int                         `json:"max_concurrent"`
		RequestsPerMinute       *int                         `json:"requests_per_minute"`
		TimeoutMS               int                          `json:"timeout_ms"`
		MaxRetries              int                          `json:"max_retries"`
		AllowPaidFallback       *bool                        `json:"allow_paid_fallback"`
		CircuitFailureThreshold *int                         `json:"circuit_failure_threshold"`
		CircuitOpenSeconds      *int                         `json:"circuit_open_seconds"`
		RequestConfig           entities.ProviderExtraConfig `json:"request_config"`
		IsActive                *bool                        `json:"is_active"`
		Notes                   string                       `json:"notes"`
	}
	if json.NewDecoder(r.Body).Decode(&body) != nil {
		jsonError(w, "invalid body", 400)
		return
	}
	providerID, err := uuid.Parse(body.ProviderID)
	if err != nil {
		jsonError(w, "invalid provider_id", 400)
		return
	}
	provider, err := h.providers.GetByID(r.Context(), providerID.String())
	if err != nil {
		jsonError(w, "provider not found", 404)
		return
	}
	if body.Priority <= 0 {
		body.Priority = 100
	}
	if body.TimeoutMS == 0 {
		body.TimeoutMS = 120000
	}
	if body.TimeoutMS < 1000 || body.TimeoutMS > 600000 {
		jsonError(w, "timeout_ms out of range", 400)
		return
	}
	if body.MaxRetries < 0 || body.MaxRetries > 3 {
		jsonError(w, "max_retries must be between 0 and 3", 400)
		return
	}
	maxConcurrent, rpm := defaultAIRouteLimits(stage.QueueClass)
	if body.MaxConcurrent != nil {
		maxConcurrent = *body.MaxConcurrent
	}
	if body.RequestsPerMinute != nil {
		rpm = *body.RequestsPerMinute
	}
	if maxConcurrent < 0 || maxConcurrent > 10000 || rpm < 0 || rpm > 100000 {
		jsonError(w, "invalid capacity limits", 400)
		return
	}
	circuitThreshold := 5
	if body.CircuitFailureThreshold != nil {
		circuitThreshold = *body.CircuitFailureThreshold
	}
	circuitOpenSeconds := 60
	if body.CircuitOpenSeconds != nil {
		circuitOpenSeconds = *body.CircuitOpenSeconds
	}
	if circuitThreshold < 1 || circuitThreshold > 100 || circuitOpenSeconds < 1 || circuitOpenSeconds > 86400 {
		jsonError(w, "invalid circuit breaker limits", 400)
		return
	}
	if body.CostTier == "" {
		if provider.CostMicros == 0 {
			body.CostTier = entities.CostTierFree
		} else {
			body.CostTier = entities.CostTierLowCost
		}
	}
	if !validCostTier(body.CostTier) {
		jsonError(w, "invalid cost_tier", 400)
		return
	}
	allowPaid := true
	if body.AllowPaidFallback != nil {
		allowPaid = *body.AllowPaidFallback
	}
	requestConfig := body.RequestConfig
	if requestConfig == nil {
		requestConfig = entities.ProviderExtraConfig{}
	}
	if stage.Capability == "text.web_search" {
		requestConfig["web_search"] = true
	}
	active := true
	if body.IsActive != nil {
		active = *body.IsActive
	}
	b := &entities.AIToolProviderBinding{ID: uuid.New(), StageID: stageID, ProviderID: providerID, Priority: body.Priority, IsActive: active, CostTier: body.CostTier, MaxConcurrent: maxConcurrent, RequestsPerMinute: rpm, TimeoutMS: body.TimeoutMS, MaxRetries: body.MaxRetries, AllowPaidFallback: allowPaid, CircuitFailureThreshold: circuitThreshold, CircuitOpenSeconds: circuitOpenSeconds, RequestConfig: requestConfig, ConfigVersion: 1, Notes: body.Notes}
	if err := h.routing.CreateBinding(r.Context(), b); err != nil {
		jsonError(w, "create binding failed: "+err.Error(), 400)
		return
	}
	_ = h.routing.RecordRoutingChange(r.Context(), "binding", b.ID, "create", map[string]interface{}{}, b, routingChangedBy(r))
	jsonOK(w, b)
}

func (h *AIRoutingAdminHandler) UpdateBinding(w http.ResponseWriter, r *http.Request) {
	if !requireRoutingAdmin(w, r) {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		jsonError(w, "invalid binding id", 400)
		return
	}
	before, err := h.routing.GetBinding(r.Context(), id)
	if err != nil {
		jsonError(w, "binding not found", 404)
		return
	}

	var body struct {
		Priority                *int                          `json:"priority"`
		IsActive                *bool                         `json:"is_active"`
		CostTier                *string                       `json:"cost_tier"`
		MaxConcurrent           *int                          `json:"max_concurrent"`
		RequestsPerMinute       *int                          `json:"requests_per_minute"`
		TimeoutMS               *int                          `json:"timeout_ms"`
		MaxRetries              *int                          `json:"max_retries"`
		AllowPaidFallback       *bool                         `json:"allow_paid_fallback"`
		CircuitFailureThreshold *int                          `json:"circuit_failure_threshold"`
		CircuitOpenSeconds      *int                          `json:"circuit_open_seconds"`
		RequestConfig           *entities.ProviderExtraConfig `json:"request_config"`
		Notes                   *string                       `json:"notes"`
	}
	if json.NewDecoder(r.Body).Decode(&body) != nil {
		jsonError(w, "invalid body", 400)
		return
	}

	fields := map[string]interface{}{}
	if body.Priority != nil {
		if *body.Priority <= 0 || *body.Priority > 100000 {
			jsonError(w, "priority out of range", 400)
			return
		}
		fields["priority"] = *body.Priority
	}
	if body.IsActive != nil {
		fields["is_active"] = *body.IsActive
	}
	if body.CostTier != nil {
		if !validCostTier(*body.CostTier) {
			jsonError(w, "invalid cost_tier", 400)
			return
		}
		fields["cost_tier"] = *body.CostTier
	}
	if body.MaxConcurrent != nil {
		if *body.MaxConcurrent < 0 || *body.MaxConcurrent > 10000 {
			jsonError(w, "max_concurrent out of range", 400)
			return
		}
		fields["max_concurrent"] = *body.MaxConcurrent
	}
	if body.RequestsPerMinute != nil {
		if *body.RequestsPerMinute < 0 || *body.RequestsPerMinute > 100000 {
			jsonError(w, "requests_per_minute out of range", 400)
			return
		}
		fields["requests_per_minute"] = *body.RequestsPerMinute
	}
	if body.TimeoutMS != nil {
		if *body.TimeoutMS < 1000 || *body.TimeoutMS > 600000 {
			jsonError(w, "timeout_ms out of range", 400)
			return
		}
		fields["timeout_ms"] = *body.TimeoutMS
	}
	if body.MaxRetries != nil {
		if *body.MaxRetries < 0 || *body.MaxRetries > 3 {
			jsonError(w, "max_retries must be between 0 and 3", 400)
			return
		}
		fields["max_retries"] = *body.MaxRetries
	}
	if body.AllowPaidFallback != nil {
		fields["allow_paid_fallback"] = *body.AllowPaidFallback
	}
	if body.CircuitFailureThreshold != nil {
		if *body.CircuitFailureThreshold < 1 || *body.CircuitFailureThreshold > 100 {
			jsonError(w, "circuit_failure_threshold out of range", 400)
			return
		}
		fields["circuit_failure_threshold"] = *body.CircuitFailureThreshold
	}
	if body.CircuitOpenSeconds != nil {
		if *body.CircuitOpenSeconds < 1 || *body.CircuitOpenSeconds > 86400 {
			jsonError(w, "circuit_open_seconds out of range", 400)
			return
		}
		fields["circuit_open_seconds"] = *body.CircuitOpenSeconds
	}
	if body.RequestConfig != nil {
		fields["request_config"] = *body.RequestConfig
	}
	if body.Notes != nil {
		fields["notes"] = *body.Notes
	}
	if len(fields) == 0 {
		jsonError(w, "no supported fields", 400)
		return
	}

	fields["config_version"] = before.ConfigVersion + 1
	if err := h.routing.UpdateBinding(r.Context(), id, fields); err != nil {
		jsonError(w, "update failed: "+err.Error(), 400)
		return
	}
	after, _ := h.routing.GetBinding(r.Context(), id)
	_ = h.routing.RecordRoutingChange(r.Context(), "binding", id, "update", before, after, routingChangedBy(r))
	jsonOK(w, after)
}
func (h *AIRoutingAdminHandler) DeleteBinding(w http.ResponseWriter, r *http.Request) {
	if !requireRoutingAdmin(w, r) {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		jsonError(w, "invalid binding id", 400)
		return
	}
	before, err := h.routing.GetBinding(r.Context(), id)
	if err != nil {
		jsonError(w, "binding not found", 404)
		return
	}
	if err := h.routing.DeleteBinding(r.Context(), id); err != nil {
		jsonError(w, "delete failed", 500)
		return
	}
	_ = h.routing.RecordRoutingChange(r.Context(), "binding", id, "delete", before, map[string]interface{}{}, routingChangedBy(r))
	jsonOK(w, map[string]string{"status": "deleted"})
}

func (h *AIRoutingAdminHandler) ValidateToolRoute(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if err := h.routing.ValidateToolRoute(r.Context(), slug); err != nil {
		jsonError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	jsonOK(w, map[string]interface{}{"tool_slug": slug, "valid": true})
}

func validRoutingPolicy(v string) bool {
	switch v {
	case entities.RoutingFreeFirst, entities.RoutingBalanced, entities.RoutingQualityFirst, entities.RoutingFreeOnly, entities.RoutingPremiumOnly:
		return true
	}
	return false
}

func validQueueClass(v string) bool {
	switch v {
	case entities.QueueRealtime, entities.QueueInteractive, entities.QueueAsync, entities.QueueHeavyAsync, entities.QueueBackground:
		return true
	}
	return false
}

func validCostTier(v string) bool {
	switch v {
	case entities.CostTierFree, entities.CostTierLowCost, entities.CostTierPremium:
		return true
	}
	return false
}
