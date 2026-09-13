package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"loyalty-nexus/internal/domain/entities"
)

const (
	openRouterCatalogSource   = "openrouter"
	modelIntelligenceInterval = 24 * time.Hour
)

type modelIntelligenceStore interface {
	RefreshCatalog(context.Context, string, []entities.AIModelCatalog) error
	ListAvailable(context.Context, string, bool) ([]entities.AIModelCatalog, error)
	UpsertScores(context.Context, []entities.AIModelScore) error
	ListScoutTargets(context.Context) ([]entities.AIModelScoutTarget, error)
	ReplacePendingRecommendations(context.Context, []entities.AIModelRecommendation) error
}
type modelProviderStore interface {
	ListAll(context.Context) ([]entities.AIProviderConfig, error)
}

type AIModelIntelligenceWorker struct {
	store      modelIntelligenceStore
	providers  modelProviderStore
	router     *AIStudioOrchestrator
	httpClient *http.Client
	interval   time.Duration
}

func NewAIModelIntelligenceWorker(store modelIntelligenceStore, providers modelProviderStore, router *AIStudioOrchestrator) *AIModelIntelligenceWorker {
	return &AIModelIntelligenceWorker{
		store:      store,
		providers:  providers,
		router:     router,
		interval:   modelIntelligenceInterval,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (w *AIModelIntelligenceWorker) Start(ctx context.Context) {
	if w == nil || w.store == nil || w.providers == nil {
		return
	}
	go func() {
		w.runOnce(ctx)
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.runOnce(ctx)
			}
		}
	}()
}

type openRouterModelResponse struct {
	Data []openRouterModel `json:"data"`
}

type openRouterModel struct {
	ID                  string                 `json:"id"`
	CanonicalSlug       string                 `json:"canonical_slug"`
	Name                string                 `json:"name"`
	Description         string                 `json:"description"`
	ContextLength       int64                  `json:"context_length"`
	Created             int64                  `json:"created"`
	Pricing             map[string]string      `json:"pricing"`
	SupportedParameters []string               `json:"supported_parameters"`
	ExpirationDate      *string                `json:"expiration_date"`
	Benchmarks          map[string]interface{} `json:"benchmarks"`
	Architecture        struct {
		InputModalities  []string `json:"input_modalities"`
		OutputModalities []string `json:"output_modalities"`
		Tokenizer        string   `json:"tokenizer"`
		InstructType     *string  `json:"instruct_type"`
	} `json:"architecture"`
}

func (w *AIModelIntelligenceWorker) runOnce(ctx context.Context) {
	provider, err := w.findOpenRouterProvider(ctx)
	if err != nil {
		log.Printf("[AIModelIntel] skipped: %v", err)
		return
	}
	key := provider.ResolveKey()
	if key == "" {
		log.Printf("[AIModelIntel] skipped: OpenRouter credential unavailable")
		return
	}
	catalog, err := w.fetchModels(ctx, key, "")
	if err != nil {
		log.Printf("[AIModelIntel] catalog refresh failed: %v", err)
		return
	}
	models := make([]entities.AIModelCatalog, 0, len(catalog))
	for _, m := range catalog {
		models = append(models, mapOpenRouterModel(m))
	}
	if err := w.store.RefreshCatalog(ctx, openRouterCatalogSource, models); err != nil {
		log.Printf("[AIModelIntel] persist failed: %v", err)
		return
	}
	w.refreshPublicRanks(ctx, key)
	w.refreshScoutRecommendations(ctx)
	log.Printf("[AIModelIntel] refreshed models=%d", len(models))
}

func (w *AIModelIntelligenceWorker) findOpenRouterProvider(ctx context.Context) (*entities.AIProviderConfig, error) {
	rows, err := w.providers.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		p := &rows[i]
		source, _ := p.ExtraConfig["catalog_source"].(string)
		base, _ := p.ExtraConfig["base_url"].(string)
		if strings.EqualFold(source, openRouterCatalogSource) || strings.Contains(strings.ToLower(base), "openrouter.ai") {
			if !p.IsActive {
				continue
			}
			return p, nil
		}
	}
	return nil, fmt.Errorf("no active OpenRouter provider marked with catalog_source=openrouter")
}

func (w *AIModelIntelligenceWorker) fetchModels(ctx context.Context, key, sortBy string) ([]openRouterModel, error) {
	endpoint := "https://openrouter.ai/api/v1/models"
	if sortBy != "" {
		endpoint += "?sort=" + url.QueryEscape(sortBy)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Accept", "application/json")
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenRouter models HTTP %d: %s", resp.StatusCode, truncateStr(string(raw), 300))
	}
	var parsed openRouterModelResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	return parsed.Data, nil
}

func mapOpenRouterModel(m openRouterModel) entities.AIModelCatalog {
	provider := m.ID
	if i := strings.Index(provider, "/"); i > 0 {
		provider = provider[:i]
	}
	caps := entities.ProviderExtraConfig{
		"input_modalities":     m.Architecture.InputModalities,
		"output_modalities":    m.Architecture.OutputModalities,
		"supported_parameters": m.SupportedParameters,
		"tool_calling":         containsString(m.SupportedParameters, "tools"),
		"structured_outputs":   containsString(m.SupportedParameters, "structured_outputs"),
	}
	pricing := entities.ProviderExtraConfig{}
	for k, v := range m.Pricing {
		pricing[k] = v
	}
	metadata := entities.ProviderExtraConfig{
		"canonical_slug":  m.CanonicalSlug,
		"description":     m.Description,
		"created":         m.Created,
		"expiration_date": m.ExpirationDate,
		"benchmarks":      m.Benchmarks,
	}
	return entities.AIModelCatalog{
		ModelID:       m.ID,
		Provider:      provider,
		DisplayName:   m.Name,
		IsFree:        openRouterModelIsFree(m),
		IsAvailable:   true,
		ContextWindow: m.ContextLength,
		Capabilities:  caps,
		Pricing:       pricing,
		Metadata:      metadata,
	}
}

func openRouterModelIsFree(m openRouterModel) bool {
	if strings.HasSuffix(strings.ToLower(m.ID), ":free") || strings.Contains(strings.ToLower(m.Name), "(free)") {
		return true
	}
	seen := false
	for _, key := range []string{"prompt", "completion", "request"} {
		v := strings.TrimSpace(m.Pricing[key])
		if v == "" {
			continue
		}
		seen = true
		n, err := strconv.ParseFloat(v, 64)
		if err != nil || n != 0 {
			return false
		}
	}
	return seen
}
func containsString(rows []string, wanted string) bool {
	for _, v := range rows {
		if v == wanted {
			return true
		}
	}
	return false
}

func (w *AIModelIntelligenceWorker) refreshPublicRanks(ctx context.Context, key string) {
	catalog, err := w.store.ListAvailable(ctx, openRouterCatalogSource, false)
	if err != nil {
		log.Printf("[AIModelIntel] rank catalog read failed: %v", err)
		return
	}
	byID := make(map[string]entities.AIModelCatalog, len(catalog))
	for _, m := range catalog {
		byID[m.ModelID] = m
	}

	sorts := []string{
		"intelligence-high-to-low",
		"most-popular",
		"throughput-high-to-low",
		"latency-low-to-high",
	}
	var scores []entities.AIModelScore
	for _, sortBy := range sorts {
		ranked, err := w.fetchModels(ctx, key, sortBy)
		if err != nil {
			log.Printf("[AIModelIntel] rank %s failed: %v", sortBy, err)
			continue
		}
		denom := float64(len(ranked))
		if denom < 1 {
			continue
		}
		for idx, model := range ranked {
			row, ok := byID[model.ID]
			if !ok {
				continue
			}
			score := 100.0 * (1.0 - float64(idx)/denom)
			scores = append(scores, entities.AIModelScore{
				ModelCatalogID: row.ID,
				Capability:     "text.generate",
				ScoreSource:    "openrouter:" + sortBy,
				Score:          score,
				Metadata: entities.ProviderExtraConfig{
					"rank":       idx + 1,
					"population": len(ranked),
				},
			})
		}
	}
	if err := w.store.UpsertScores(ctx, scores); err != nil {
		log.Printf("[AIModelIntel] score upsert failed: %v", err)
	}
}

type scoutCandidate struct {
	ModelID       string `json:"model_id"`
	Name          string `json:"name"`
	ContextWindow int64  `json:"context_window"`
	ToolCalling   bool   `json:"tool_calling"`
	Structured    bool   `json:"structured_outputs"`
}

func (w *AIModelIntelligenceWorker) refreshScoutRecommendations(ctx context.Context) {
	freeModels, err := w.store.ListAvailable(ctx, openRouterCatalogSource, true)
	if err != nil || len(freeModels) == 0 || w.router == nil {
		return
	}

	sort.SliceStable(freeModels, func(i, j int) bool {
		return freeModels[i].ContextWindow > freeModels[j].ContextWindow
	})
	if len(freeModels) > 20 {
		freeModels = freeModels[:20]
	}

	candidates := make([]scoutCandidate, 0, len(freeModels))
	for _, m := range freeModels {
		tc, _ := m.Capabilities["tool_calling"].(bool)
		so, _ := m.Capabilities["structured_outputs"].(bool)
		candidates = append(candidates, scoutCandidate{
			ModelID:       m.ModelID,
			Name:          m.DisplayName,
			ContextWindow: m.ContextWindow,
			ToolCalling:   tc,
			Structured:    so,
		})
	}

	targets, err := w.store.ListScoutTargets(ctx)
	if err != nil || len(targets) == 0 {
		return
	}
	prompt := w.buildScoutPrompt(candidates, targets)
	_, text, _, _, err := w.router.runToolStageChain(ctx, nil, "__ai-model-scout", "main", providerInput{
		SystemPrompt: "You are the Loyalty Nexus AI Model Scout. Recommend only from supplied candidates. Return strict JSON only.",
		UserPrompt:   prompt,
	})
	if err != nil {
		log.Printf("[AIModelScout] skipped: %v", err)
		return
	}
	rows := parseScoutRecommendations(text, freeModels)
	if len(rows) == 0 {
		log.Printf("[AIModelScout] no valid recommendations returned")
		return
	}
	if err := w.store.ReplacePendingRecommendations(ctx, rows); err != nil {
		log.Printf("[AIModelScout] persist failed: %v", err)
	}
}

func (w *AIModelIntelligenceWorker) buildScoutPrompt(candidates []scoutCandidate, targets []entities.AIModelScoutTarget) string {
	c, _ := json.Marshal(candidates)
	t, _ := json.Marshal(targets)
	return fmt.Sprintf(
		"Evaluate these FREE OpenRouter models for the listed Loyalty Nexus tool stages. "+
			"Candidates: %s Targets: %s "+
			"Return a JSON array only. Each item must contain tool_slug, stage_key, model_id, "+
			"recommendation (ADD_AS_PRIMARY, ADD_AS_BACKUP, or EVALUATE), reason, and score 0-100. "+
			"Prefer capability fit, context, structured output/tool support and robustness. Do not invent models.",
		string(c), string(t),
	)
}

type scoutRecommendationJSON struct {
	ToolSlug       string  `json:"tool_slug"`
	StageKey       string  `json:"stage_key"`
	ModelID        string  `json:"model_id"`
	Recommendation string  `json:"recommendation"`
	Reason         string  `json:"reason"`
	Score          float64 `json:"score"`
}

func parseScoutRecommendations(raw string, catalog []entities.AIModelCatalog) []entities.AIModelRecommendation {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var parsed []scoutRecommendationJSON
	if json.Unmarshal([]byte(raw), &parsed) != nil {
		return nil
	}

	ids := make(map[string]entities.AIModelCatalog, len(catalog))
	for _, m := range catalog {
		ids[m.ModelID] = m
	}
	out := make([]entities.AIModelRecommendation, 0, len(parsed))
	for _, p := range parsed {
		m, ok := ids[p.ModelID]
		if !ok || p.ToolSlug == "" || p.StageKey == "" {
			continue
		}
		rec := strings.ToUpper(p.Recommendation)
		if rec != "ADD_AS_PRIMARY" && rec != "ADD_AS_BACKUP" && rec != "EVALUATE" {
			rec = "EVALUATE"
		}
		score := p.Score
		if score < 0 {
			score = 0
		}
		if score > 100 {
			score = 100
		}
		out = append(out, entities.AIModelRecommendation{
			ToolSlug:       p.ToolSlug,
			StageKey:       p.StageKey,
			ModelCatalogID: m.ID,
			Recommendation: rec,
			Reason:         p.Reason,
			Score:          score,
			Status:         "PENDING",
		})
	}
	return out
}
