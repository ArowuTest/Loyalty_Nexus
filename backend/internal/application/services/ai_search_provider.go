package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (o *AIStudioOrchestrator) callTavilySearch(ctx context.Context, apiKey, query string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("Tavily API key not configured")
	}
	if strings.TrimSpace(query) == "" {
		return "", fmt.Errorf("Tavily query is required")
	}
	payload := map[string]interface{}{
		"api_key":             apiKey,
		"query":               query,
		"search_depth":        "advanced",
		"include_answer":      true,
		"include_raw_content": false,
		"max_results":         5,
		"include_domains":     []string{},
		"exclude_domains":     []string{},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.tavily.com/search", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Tavily HTTP: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Tavily HTTP %d: %s", resp.StatusCode, truncateStr(string(raw), 300))
	}
	var result struct {
		Answer  string `json:"answer"`
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("Tavily parse: %w", err)
	}
	var sb strings.Builder
	if result.Answer != "" {
		sb.WriteString("Direct answer: ")
		sb.WriteString(result.Answer)
		sb.WriteString("\n\n")
	}
	sb.WriteString(fmt.Sprintf("Search results for %q:\n", query))
	for i, row := range result.Results {
		sb.WriteString(fmt.Sprintf("%d. %s — %s\n%s\n", i+1, row.Title, row.URL, truncateStr(row.Content, 500)))
	}
	if len(result.Results) == 0 && result.Answer == "" {
		return "[web_search: no results found]", nil
	}
	return sb.String(), nil
}
