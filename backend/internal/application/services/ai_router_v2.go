package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"loyalty-nexus/internal/domain/entities"
)

var ErrNoConfiguredAIRoute = errors.New("no configured AI route")

type aiGenerationContextKey struct{}

func withAIGenerationID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, aiGenerationContextKey{}, id)
}

func aiGenerationIDFromContext(ctx context.Context) *uuid.UUID {
	id, ok := ctx.Value(aiGenerationContextKey{}).(uuid.UUID)
	if !ok || id == uuid.Nil {
		return nil
	}
	return &id
}

type AIRoutingStore interface {
	ListCandidates(ctx context.Context, toolSlug, stageKey string) ([]entities.AIRouteCandidate, error)
	ListStagesForTool(ctx context.Context, toolSlug string) ([]entities.AIToolStage, error)
	RecordAttempt(ctx context.Context, attempt *entities.AIGenerationAttempt) error
	UpdateAttempt(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error
	ValidateToolRoute(ctx context.Context, toolSlug string) error
}

func (o *AIStudioOrchestrator) SetAIRouting(store AIRoutingStore, capacity *AICapacityController) {
	o.routingDB = store
	o.capacity = capacity
}

type AIStudioWorkerPoolConfig struct {
	Workers int
	Buffer  int
}

func (o *AIStudioOrchestrator) QueuePolicyForTool(ctx context.Context, toolSlug string) (string, int) {
	if o == nil || o.routingDB == nil {
		return entities.QueueAsync, 60
	}
	stages, err := o.routingDB.ListStagesForTool(ctx, toolSlug)
	if err != nil || len(stages) == 0 {
		return entities.QueueAsync, 60
	}
	rank := map[string]int{
		entities.QueueBackground:  1,
		entities.QueueRealtime:    2,
		entities.QueueInteractive: 3,
		entities.QueueAsync:       4,
		entities.QueueHeavyAsync:  5,
	}
	selectedClass := entities.QueueAsync
	selectedWait := 60
	selectedRank := 0
	for _, stage := range stages {
		if !stage.IsActive {
			continue
		}
		r := rank[stage.QueueClass]
		if r > selectedRank {
			selectedRank = r
			selectedClass = stage.QueueClass
			selectedWait = stage.MaxQueueSeconds
		} else if r == selectedRank && stage.MaxQueueSeconds > selectedWait {
			selectedWait = stage.MaxQueueSeconds
		}
	}
	return selectedClass, selectedWait
}

func (o *AIStudioOrchestrator) StudioWorkerPoolConfig(queueClass string) AIStudioWorkerPoolConfig {
	cfg := AIStudioWorkerPoolConfig{Workers: 8, Buffer: 1024}
	switch queueClass {
	case entities.QueueRealtime:
		cfg = AIStudioWorkerPoolConfig{Workers: 8, Buffer: 512}
	case entities.QueueInteractive:
		cfg = AIStudioWorkerPoolConfig{Workers: 16, Buffer: 2048}
	case entities.QueueHeavyAsync:
		cfg = AIStudioWorkerPoolConfig{Workers: 4, Buffer: 512}
	case entities.QueueBackground:
		cfg = AIStudioWorkerPoolConfig{Workers: 2, Buffer: 256}
	}
	if o != nil && o.cfg != nil {
		key := strings.ToLower(queueClass)
		cfg.Workers = o.cfg.GetInt("ai_studio_worker_"+key+"_workers", cfg.Workers)
		cfg.Buffer = o.cfg.GetInt("ai_studio_worker_"+key+"_buffer", cfg.Buffer)
	}
	if cfg.Workers < 1 {
		cfg.Workers = 1
	}
	if cfg.Workers > 128 {
		cfg.Workers = 128
	}
	if cfg.Buffer < 16 {
		cfg.Buffer = 16
	}
	if cfg.Buffer > 20000 {
		cfg.Buffer = 20000
	}
	return cfg
}

func classifyRoutingError(err error) (string, bool) {
	if err == nil {
		return "", false
	}
	s := strings.ToLower(err.Error())
	switch {
	case strings.Contains(s, "invalid input"), strings.Contains(s, "validation"), strings.Contains(s, "bad request"), strings.Contains(s, "http 400"):
		return "INVALID_INPUT", false
	case strings.Contains(s, "safety"), strings.Contains(s, "policy"), strings.Contains(s, "moderation"):
		return "POLICY_REJECTION", false
	case strings.Contains(s, "429"), strings.Contains(s, "rate limit"), strings.Contains(s, "quota"):
		return "RATE_LIMIT", true
	case strings.Contains(s, "timeout"), strings.Contains(s, "deadline exceeded"):
		return "TIMEOUT", true
	case strings.Contains(s, "401"), strings.Contains(s, "403"), strings.Contains(s, "unauthorized"), strings.Contains(s, "forbidden"):
		return "AUTH_CONFIG", true
	case strings.Contains(s, "500"), strings.Contains(s, "502"), strings.Contains(s, "503"), strings.Contains(s, "504"):
		return "PROVIDER_5XX", true
	default:
		return "PROVIDER_ERROR", true
	}
}

func estimateRouteCostMicros(c entities.AIRouteCandidate, in providerInput) int64 {
	estimate := int64(c.Provider.CostMicros)
	switch c.Provider.Template {
	case entities.TemplateGrokVideo:
		dur := in.DurationSecs
		if dur <= 0 {
			dur = 6
		}
		v := int64(50000 * dur)
		if v > estimate {
			estimate = v
		}
	case entities.TemplateFALImageUltra:
		n := 1
		if in.Extra != nil {
			if raw, ok := in.Extra["num_images"].(float64); ok && raw >= 1 && raw <= 4 {
				n = int(raw)
			}
		}
		v := int64(c.Provider.CostMicros * n)
		if v > estimate {
			estimate = v
		}
	}
	return estimate
}

func (o *AIStudioOrchestrator) recordRoutingSkip(
	ctx context.Context,
	generationID *uuid.UUID,
	c entities.AIRouteCandidate,
	attemptNo int,
	outcome, class, message string,
) {
	now := time.Now()
	_ = o.routingDB.RecordAttempt(ctx, &entities.AIGenerationAttempt{
		GenerationID: generationID, ToolID: &c.Stage.ToolID, StageID: &c.Stage.ID,
		BindingID: &c.Binding.ID, ProviderID: &c.Provider.ID, StageKey: c.Stage.StageKey,
		AttemptNo: attemptNo, ProviderSlug: c.Provider.Slug, ModelID: c.Provider.ModelID,
		Outcome: outcome, ErrorClass: class, ErrorMessage: message, StartedAt: now, CompletedAt: &now,
	})
}

func (o *AIStudioOrchestrator) runToolStageChain(ctx context.Context, generationID *uuid.UUID, toolSlug, stageKey string, in providerInput) (outputURL, outputText string, costMicros int, usedSlug string, err error) {
	if generationID == nil {
		generationID = aiGenerationIDFromContext(ctx)
	}
	if o.routingDB == nil {
		return "", "", 0, "", ErrNoConfiguredAIRoute
	}
	candidates, err := o.routingDB.ListCandidates(ctx, toolSlug, stageKey)
	if err != nil {
		return "", "", 0, "", fmt.Errorf("resolve AI route: %w", err)
	}
	if len(candidates) == 0 {
		return "", "", 0, "", ErrNoConfiguredAIRoute
	}

	var lastErr error
	attemptNo := 0
	for _, c := range candidates {
		if c.Stage.RoutingPolicy == entities.RoutingFreeOnly && c.Binding.CostTier != entities.CostTierFree {
			continue
		}
		if c.Stage.RoutingPolicy == entities.RoutingPremiumOnly && c.Binding.CostTier != entities.CostTierPremium {
			continue
		}
		if c.Binding.CostTier != entities.CostTierFree && !c.Binding.AllowPaidFallback {
			continue
		}
		attemptNo++

		halfOpen, circuitReason, circuitOK := o.capacity.CircuitPermit(ctx, c.Binding)
		if !circuitOK {
			o.recordRoutingSkip(ctx, generationID, c, attemptNo, "SKIPPED_CIRCUIT", "CIRCUIT", circuitReason)
			lastErr = fmt.Errorf("%s: %s", c.Provider.Slug, circuitReason)
			continue
		}

		var budgetRes AIBudgetReservation
		if c.Binding.CostTier != entities.CostTierFree {
			estimate := estimateRouteCostMicros(c, in)
			var budgetReason string
			var budgetOK bool
			budgetRes, budgetReason, budgetOK = o.capacity.ReservePaidBudget(ctx, c.Stage, estimate)
			if !budgetOK {
				o.capacity.CircuitNeutral(ctx, c.Binding, halfOpen)
				o.recordRoutingSkip(ctx, generationID, c, attemptNo, "SKIPPED_BUDGET", "BUDGET", budgetReason)
				lastErr = fmt.Errorf("%s: %s", c.Provider.Slug, budgetReason)
				continue
			}
		}

		release, capacityReason, capacityOK := o.capacity.Reserve(ctx, c.Binding)
		if !capacityOK {
			o.capacity.SettlePaidBudget(ctx, budgetRes, 0, false)
			o.capacity.CircuitNeutral(ctx, c.Binding, halfOpen)
			o.recordRoutingSkip(ctx, generationID, c, attemptNo, "SKIPPED_CAPACITY", "CAPACITY", capacityReason)
			lastErr = fmt.Errorf("%s: %s", c.Provider.Slug, capacityReason)
			continue
		}

		started := time.Now()
		attempt := &entities.AIGenerationAttempt{
			GenerationID: generationID, ToolID: &c.Stage.ToolID, StageID: &c.Stage.ID,
			BindingID: &c.Binding.ID, ProviderID: &c.Provider.ID, StageKey: c.Stage.StageKey,
			AttemptNo: attemptNo, ProviderSlug: c.Provider.Slug, ModelID: c.Provider.ModelID,
			Outcome: "STARTED", StartedAt: started,
		}
		_ = o.routingDB.RecordAttempt(ctx, attempt)

		timeout := time.Duration(c.Binding.TimeoutMS) * time.Millisecond
		if timeout <= 0 {
			timeout = 120 * time.Second
		}
		callCtx, cancel := context.WithTimeout(ctx, timeout)
		provider := c.Provider
		provider.ExtraConfig = mergeProviderRequestConfig(c.Provider.ExtraConfig, c.Binding.RequestConfig)
		url, text, cost, callErr := o.callByTemplate(callCtx, provider, in)
		cancel()
		release()

		completed := time.Now()
		duration := int(completed.Sub(started).Milliseconds())
		if callErr == nil {
			o.capacity.CircuitSuccess(ctx, c.Binding)
			o.capacity.SettlePaidBudget(ctx, budgetRes, int64(cost), true)
			_ = o.routingDB.UpdateAttempt(ctx, attempt.ID, map[string]interface{}{
				"outcome": "SUCCEEDED", "duration_ms": duration, "cost_micros": cost, "completed_at": completed,
			})
			return url, text, cost, c.Provider.Slug, nil
		}

		class, mayFailover := classifyRoutingError(callErr)
		if mayFailover {
			o.capacity.CircuitFailure(ctx, c.Binding, halfOpen)
		} else {
			o.capacity.CircuitNeutral(ctx, c.Binding, halfOpen)
		}
		o.capacity.SettlePaidBudget(ctx, budgetRes, 0, false)
		msg := callErr.Error()
		if len(msg) > 500 {
			msg = msg[:500]
		}
		_ = o.routingDB.UpdateAttempt(ctx, attempt.ID, map[string]interface{}{
			"outcome": "FAILED", "error_class": class, "error_message": msg,
			"duration_ms": duration, "completed_at": completed,
		})
		lastErr = fmt.Errorf("%s: %w", c.Provider.Slug, callErr)
		if !mayFailover {
			return "", "", 0, "", lastErr
		}
	}
	if lastErr == nil {
		lastErr = ErrNoConfiguredAIRoute
	}
	return "", "", 0, "", fmt.Errorf("all configured providers exhausted for %s/%s: %w", toolSlug, stageKey, lastErr)
}
func mergeProviderRequestConfig(base, override entities.ProviderExtraConfig) entities.ProviderExtraConfig {
	out := entities.ProviderExtraConfig{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range override {
		out[k] = v
	}
	return out
}

func (o *AIStudioOrchestrator) ChatRoute(
	ctx context.Context,
	toolSlug, systemPrompt, userPrompt string,
) (text, provider string, err error) {
	if toolSlug == "" || toolSlug == "general" || toolSlug == "ask-nexus" {
		toolSlug = "nexus-chat"
	}
	_, text, _, usedSlug, err := o.runToolStageChain(ctx, nil, toolSlug, "main", providerInput{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
	})
	if err != nil {
		return "", "", err
	}
	return text, "route/" + usedSlug, nil
}

func (o *AIStudioOrchestrator) GenerateWebsite(
	ctx context.Context,
	systemPrompt, userPrompt string,
	images []string,
) (html, provider string, costMicros int, err error) {
	_, html, costMicros, usedSlug, err := o.runToolStageChain(ctx, nil, "website-builder", "main", providerInput{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Images:       images,
	})
	if err != nil {
		return "", "", 0, err
	}
	return html, "route/" + usedSlug, costMicros, nil
}
