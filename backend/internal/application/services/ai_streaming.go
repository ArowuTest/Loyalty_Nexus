package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"loyalty-nexus/internal/domain/entities"
)

var errStreamingUnsupported = errors.New("provider template does not support streaming")

func (o *AIStudioOrchestrator) ChatRouteStream(
	ctx context.Context,
	toolSlug, systemPrompt, userPrompt string,
	onChunk func(string),
) (text, provider string, err error) {
	if toolSlug == "" || toolSlug == "general" || toolSlug == "ask-nexus" {
		toolSlug = "nexus-chat"
	}
	text, used, err := o.runToolStageStreamChain(ctx, toolSlug, "main", providerInput{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
	}, onChunk)
	if err != nil {
		return "", "", err
	}
	return text, "route/" + used, nil
}
func (o *AIStudioOrchestrator) runToolStageStreamChain(
	ctx context.Context,
	toolSlug, stageKey string,
	in providerInput,
	onChunk func(string),
) (string, string, error) {
	if o.routingDB == nil {
		return "", "", ErrNoConfiguredAIRoute
	}
	candidates, err := o.routingDB.ListCandidates(ctx, toolSlug, stageKey)
	if err != nil {
		return "", "", err
	}
	if len(candidates) == 0 {
		return "", "", ErrNoConfiguredAIRoute
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
			o.recordRoutingSkip(ctx, nil, c, attemptNo, "SKIPPED_CIRCUIT", "CIRCUIT", circuitReason)
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
				o.recordRoutingSkip(ctx, nil, c, attemptNo, "SKIPPED_BUDGET", "BUDGET", budgetReason)
				lastErr = fmt.Errorf("%s: %s", c.Provider.Slug, budgetReason)
				continue
			}
		}

		release, capacityReason, capacityOK := o.capacity.Reserve(ctx, c.Binding)
		if !capacityOK {
			o.capacity.SettlePaidBudget(ctx, budgetRes, 0, false)
			o.capacity.CircuitNeutral(ctx, c.Binding, halfOpen)
			o.recordRoutingSkip(ctx, nil, c, attemptNo, "SKIPPED_CAPACITY", "CAPACITY", capacityReason)
			lastErr = fmt.Errorf("%s: %s", c.Provider.Slug, capacityReason)
			continue
		}

		started := time.Now()
		attempt := &entities.AIGenerationAttempt{
			ID: uuid.New(), ToolID: &c.Stage.ToolID, StageID: &c.Stage.ID, BindingID: &c.Binding.ID,
			ProviderID: &c.Provider.ID, StageKey: c.Stage.StageKey, AttemptNo: attemptNo,
			ProviderSlug: c.Provider.Slug, ModelID: c.Provider.ModelID,
			Outcome: "STARTED", StartedAt: started,
		}
		o.ledgerRecord(ctx, "stream-start", attempt)

		provider := c.Provider
		provider.ExtraConfig = mergeProviderRequestConfig(c.Provider.ExtraConfig, c.Binding.RequestConfig)
		timeout := time.Duration(c.Binding.TimeoutMS) * time.Millisecond
		if timeout <= 0 {
			timeout = 120 * time.Second
		}

		emitted := false
		wrappedChunk := func(chunk string) {
			if chunk == "" {
				return
			}
			emitted = true
			onChunk(chunk)
		}
		// The in-flight slot and the timer are released from defers, so a panicking
		// adapter (recovered by the HTTP server) cannot leak either (review M6).
		text, callErr := func() (string, error) {
			callCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			defer release()
			return o.callByTemplateStream(callCtx, provider, in, wrappedChunk)
		}()
		completed := time.Now()
		duration := int(completed.Sub(started).Milliseconds())

		if callErr == nil {
			o.capacity.CircuitSuccess(ctx, c.Binding)
			o.capacity.SettlePaidBudget(ctx, budgetRes, int64(provider.CostMicros), true)
			o.ledgerFinalize(ctx, attempt, map[string]interface{}{
				"outcome": "SUCCEEDED", "duration_ms": duration,
				"cost_micros": provider.CostMicros, "completed_at": completed,
			})
			return text, provider.Slug, nil
		}

		class, mayFailover := classifyRoutingError(callErr)
		if errors.Is(callErr, errStreamingUnsupported) {
			class, mayFailover = "STREAM_UNSUPPORTED", true
			o.capacity.CircuitNeutral(ctx, c.Binding, halfOpen)
		} else if mayFailover {
			o.capacity.CircuitFailure(ctx, c.Binding, halfOpen)
		} else {
			o.capacity.CircuitNeutral(ctx, c.Binding, halfOpen)
		}
		o.capacity.SettlePaidBudget(ctx, budgetRes, 0, false)

		msg := callErr.Error()
		if len(msg) > 500 {
			msg = msg[:500]
		}
		o.ledgerFinalize(ctx, attempt, map[string]interface{}{
			"outcome": "FAILED", "error_class": class, "error_message": msg,
			"duration_ms": duration, "completed_at": completed,
		})
		lastErr = fmt.Errorf("%s: %w", provider.Slug, callErr)
		if emitted || !mayFailover {
			return "", "", lastErr
		}
	}
	if lastErr == nil {
		lastErr = ErrNoConfiguredAIRoute
	}

	_, text, _, used, syncErr := o.runToolStageChain(ctx, nil, toolSlug, stageKey, in)
	if syncErr != nil {
		return "", "", fmt.Errorf("stream and sync routes exhausted: %w", lastErr)
	}
	if text != "" {
		onChunk(text)
	}
	return text, used, nil
}

func (o *AIStudioOrchestrator) callByTemplateStream(
	ctx context.Context,
	p entities.AIProviderConfig,
	in providerInput,
	onChunk func(string),
) (string, error) {
	switch p.Template {
	case entities.TemplatePollText:
		return o.streamOpenAICompatible(ctx, p, in, onChunk)
	case entities.TemplateGemini:
		return o.streamGeminiConfigured(ctx, p, in, onChunk)
	default:
		return "", errStreamingUnsupported
	}
}
func (o *AIStudioOrchestrator) streamOpenAICompatible(
	ctx context.Context,
	p entities.AIProviderConfig,
	in providerInput,
	onChunk func(string),
) (string, error) {
	key := p.ResolveKey()
	baseURL := resolveBaseURLForProvider(p)
	payload := map[string]interface{}{
		"model":  p.ModelID,
		"stream": true,
		"messages": []map[string]interface{}{
			{"role": "system", "content": in.SystemPrompt},
			{"role": "user", "content": in.UserPrompt},
		},
	}
	if webSearch, _ := p.ExtraConfig["web_search"].(bool); webSearch {
		if strings.Contains(strings.ToLower(baseURL), "openrouter.ai") {
			payload["tools"] = []map[string]interface{}{{"type": "openrouter:web_search"}}
		} else {
			payload["search"] = true
		}
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("stream HTTP: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		if refusal, ok := openAIErrorRefusal(raw); ok {
			return "", refusal
		}
		return "", fmt.Errorf("API %d: %s", resp.StatusCode, truncateStr(string(raw), 300))
	}
	return consumeOpenAIStream(resp.Body, onChunk)
}

// consumeOpenAIStream reads an OpenAI-compatible SSE body, forwarding each
// delta to onChunk. A finish_reason of content_filter, or a policy error
// event, ends the stream as a ProviderRefusalError.
func consumeOpenAIStream(body io.Reader, onChunk func(string)) (string, error) {
	var full strings.Builder
	scanner := bufio.NewScanner(body)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		var event struct {
			Choices []struct {
				FinishReason string `json:"finish_reason"`
				Delta        struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		if event.Error != nil {
			if refusal, ok := openAIErrorRefusal([]byte(data)); ok {
				return "", refusal
			}
			return full.String(), fmt.Errorf("API stream error: %s", event.Error.Message)
		}
		for _, choice := range event.Choices {
			if choice.FinishReason == "content_filter" {
				return "", &ProviderRefusalError{Provider: "openai-compatible", Reason: "finish_reason=content_filter"}
			}
			if choice.Delta.Content == "" {
				continue
			}
			full.WriteString(choice.Delta.Content)
			onChunk(choice.Delta.Content)
		}
	}
	if err := scanner.Err(); err != nil {
		return full.String(), fmt.Errorf("stream read: %w", err)
	}
	if full.Len() == 0 {
		return "", fmt.Errorf("stream returned no content")
	}
	return full.String(), nil
}
func (o *AIStudioOrchestrator) streamGeminiConfigured(
	ctx context.Context,
	p entities.AIProviderConfig,
	in providerInput,
	onChunk func(string),
) (string, error) {
	key := p.ResolveKey()
	model := p.ModelID
	if model == "" {
		return "", fmt.Errorf("Gemini model_id is required")
	}
	endpoint := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse",
		model,
	)
	payload := map[string]interface{}{
		"system_instruction": map[string]interface{}{"parts": []map[string]string{{"text": in.SystemPrompt}}},
		"contents":           []map[string]interface{}{{"parts": []map[string]string{{"text": in.UserPrompt}}}},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("x-goog-api-key", key) // header, not query string: URLs leak into logs
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini stream HTTP: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("gemini stream %d: %s", resp.StatusCode, truncateStr(string(raw), 300))
	}
	return consumeGeminiStream(resp.Body, onChunk)
}

// consumeGeminiStream reads a streamGenerateContent SSE body, forwarding each
// visible text part to onChunk. A prompt-level block or a content-grounded
// finishReason ends the stream as a ProviderRefusalError; thought parts of
// reasoning models are never forwarded.
func consumeGeminiStream(body io.Reader, onChunk func(string)) (string, error) {
	var full strings.Builder
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}
		var event geminiGenerateContentResponse
		if json.Unmarshal([]byte(data), &event) != nil {
			continue
		}
		if event.Error != nil {
			return full.String(), fmt.Errorf("gemini stream error %d (%s): %s", event.Error.Code, event.Error.Status, event.Error.Message)
		}
		if event.PromptFeedback != nil && event.PromptFeedback.BlockReason != "" {
			return "", &ProviderRefusalError{Provider: "gemini", Reason: "prompt blocked: " + event.PromptFeedback.BlockReason}
		}
		for _, c := range event.Candidates {
			if geminiRefusalFinishReasons[c.FinishReason] {
				return "", &ProviderRefusalError{Provider: "gemini", Reason: "candidate finished: " + c.FinishReason}
			}
			if text := geminiCandidateText(c); text != "" {
				full.WriteString(text)
				onChunk(text)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return full.String(), err
	}
	if full.Len() == 0 {
		return "", fmt.Errorf("gemini stream returned no content")
	}
	return full.String(), nil
}
