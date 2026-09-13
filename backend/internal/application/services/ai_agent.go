package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type agentDirective struct {
	Action string `json:"action"`
	Query  string `json:"query,omitempty"`
	URL    string `json:"url,omitempty"`
	Answer string `json:"answer,omitempty"`
	Reason string `json:"reason,omitempty"`
}

func parseAgentDirective(raw string) (agentDirective, error) {
	var out agentDirective
	s := strings.TrimSpace(raw)
	if first := strings.Index(s, "{"); first >= 0 {
		if last := strings.LastIndex(s, "}"); last > first {
			s = s[first : last+1]
		}
	}
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return out, fmt.Errorf("agent directive JSON: %w", err)
	}
	out.Action = strings.ToLower(strings.TrimSpace(out.Action))
	switch out.Action {
	case "web_search", "read_url", "final":
		return out, nil
	default:
		return out, fmt.Errorf("unsupported agent action %q", out.Action)
	}
}

func addAgentProvider(trace []string, provider string) []string {
	if provider == "" {
		return trace
	}
	if len(trace) == 0 || trace[len(trace)-1] != provider {
		return append(trace, provider)
	}
	return trace
}

func (o *AIStudioOrchestrator) dispatchNexusAgent(ctx context.Context, env promptEnvelope) (*studioProviderResult, error) {
	if strings.TrimSpace(env.Prompt) == "" {
		return nil, fmt.Errorf("nexus-agent: task description is required")
	}
	waitSeconds := 8
	if o.cfg != nil {
		waitSeconds = o.cfg.GetInt("nexus_agent_queue_wait_seconds", 8)
	}
	if waitSeconds < 1 {
		waitSeconds = 1
	}
	if waitSeconds > 60 {
		waitSeconds = 60
	}

	acquireCtx, cancel := context.WithTimeout(ctx, time.Duration(waitSeconds)*time.Second)
	defer cancel()
	select {
	case <-o.agentSemaphore:
		defer func() { o.agentSemaphore <- struct{}{} }()
	case <-acquireCtx.Done():
		return nil, fmt.Errorf("nexus-agent is at capacity — please try again in a moment")
	}

	output, iterations, provider, cost, err := o.runReActLoop(ctx, env)
	if err == nil {
		return &studioProviderResult{
			OutputText: output,
			Provider:   "route/" + provider,
			CostMicros: cost,
		}, nil
	}

	// Governed final fallback: same Admin route, but no more tools or action protocol.
	fallbackSys := "You are Nexus Agent. Complete the user's task directly and thoroughly. " +
		"Use only information already available in the request. Do not claim fresh web access."
	_, text, fallbackCost, used, fallbackErr := o.runToolStageChain(ctx, nil, "nexus-agent", "main", providerInput{
		SystemPrompt: fallbackSys,
		UserPrompt:   env.Prompt,
	})
	if fallbackErr != nil {
		return nil, fmt.Errorf("nexus-agent failed after %d iterations: react=%v fallback=%v", iterations, err, fallbackErr)
	}
	return &studioProviderResult{
		OutputText: text,
		Provider:   "route/" + used,
		CostMicros: cost + fallbackCost,
	}, nil
}

func (o *AIStudioOrchestrator) runReActLoop(ctx context.Context, env promptEnvelope) (string, int, string, int, error) {
	maxIterations := 8
	if o.cfg != nil {
		maxIterations = o.cfg.GetInt("nexus_agent_max_iterations", 8)
	}
	if maxIterations < 1 {
		maxIterations = 1
	}
	if maxIterations > 12 {
		maxIterations = 12
	}

	today := time.Now().UTC().Format("Monday, 2 January 2006")
	systemPrompt := "You are Nexus Agent, an autonomous AI assistant with real tools. Today is " + today + ". " +
		"On every turn return exactly ONE JSON object and no markdown. " +
		"Allowed actions: " +
		"{\"action\":\"web_search\",\"query\":\"specific query\",\"reason\":\"why\"}, " +
		"{\"action\":\"read_url\",\"url\":\"https://...\",\"reason\":\"why\"}, or " +
		"{\"action\":\"final\",\"answer\":\"complete answer\"}. " +
		"Use web_search for current facts. Read useful URLs when deeper evidence is needed. " +
		"Never invent a tool result. Finish with final when enough evidence exists."

	task := env.Prompt
	if env.DocumentURL != "" {
		task += "\nAttached document URL: " + env.DocumentURL + ". Use read_url when useful."
	}
	var observations []string
	var providers []string
	totalCost := 0

	for iteration := 1; iteration <= maxIterations; iteration++ {
		var user strings.Builder
		user.WriteString("TASK:\n")
		user.WriteString(task)
		if len(observations) > 0 {
			user.WriteString("\n\nOBSERVATIONS:\n")
			user.WriteString(strings.Join(observations, "\n\n"))
		}
		if iteration == maxIterations {
			user.WriteString("\n\nThis is the final reasoning turn. Return action=final with the best supported answer.")
		}

		_, raw, cost, used, err := o.runToolStageChain(ctx, nil, "nexus-agent", "main", providerInput{
			SystemPrompt: systemPrompt,
			UserPrompt:   user.String(),
		})
		totalCost += cost
		providers = addAgentProvider(providers, used)
		if err != nil {
			return "", iteration, strings.Join(providers, "→"), totalCost, err
		}

		directive, parseErr := parseAgentDirective(raw)
		if parseErr != nil {
			observations = append(observations,
				"FORMAT_ERROR: previous model output was not valid agent JSON. Output excerpt: "+truncateStr(raw, 600))
			continue
		}

		switch directive.Action {
		case "final":
			if strings.TrimSpace(directive.Answer) == "" {
				observations = append(observations, "FORMAT_ERROR: final action had an empty answer.")
				continue
			}
			return directive.Answer, iteration, strings.Join(providers, "→"), totalCost, nil

		case "web_search":
			query := strings.TrimSpace(directive.Query)
			if query == "" {
				observations = append(observations, "TOOL_ERROR web_search: query is required.")
				continue
			}
			_, result, searchCost, searchProvider, searchErr := o.runToolStageChain(ctx, nil, "nexus-agent", "search", providerInput{
				UserPrompt: query,
			})
			totalCost += searchCost
			providers = addAgentProvider(providers, searchProvider)
			if searchErr != nil {
				observations = append(observations, "TOOL_ERROR web_search: "+truncateStr(searchErr.Error(), 500))
			} else {
				observations = append(observations, "WEB_SEARCH "+query+":\n"+truncateStr(result, 9000))
			}

		case "read_url":
			target := strings.TrimSpace(directive.URL)
			if target == "" {
				observations = append(observations, "TOOL_ERROR read_url: url is required.")
				continue
			}
			result := o.agentReadURL(ctx, target)
			observations = append(observations, "READ_URL "+target+":\n"+truncateStr(result, 7000))
		}
	}

	return "", maxIterations, strings.Join(providers, "→"), totalCost,
		fmt.Errorf("nexus-agent exceeded maximum iterations (%d)", maxIterations)
}
