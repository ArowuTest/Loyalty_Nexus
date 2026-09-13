package services

import (
	"testing"

	"github.com/google/uuid"

	"loyalty-nexus/internal/domain/entities"
)

func TestOpenRouterModelIsFree(t *testing.T) {
	free := openRouterModel{
		ID:      "vendor/model:free",
		Name:    "Model (free)",
		Pricing: map[string]string{"prompt": "0", "completion": "0", "request": "0"},
	}
	if !openRouterModelIsFree(free) {
		t.Fatal("expected free model")
	}

	paid := openRouterModel{
		ID:      "vendor/model",
		Name:    "Model",
		Pricing: map[string]string{"prompt": "0.000001", "completion": "0.000002"},
	}
	if openRouterModelIsFree(paid) {
		t.Fatal("paid model classified as free")
	}
}
func TestParseScoutRecommendationsRejectsInventedModels(t *testing.T) {
	id := uuid.New()
	catalog := []entities.AIModelCatalog{{ID: id, ModelID: "vendor/free:free"}}
	raw := `[
	  {"tool_slug":"code-helper","stage_key":"main","model_id":"vendor/free:free","recommendation":"ADD_AS_BACKUP","reason":"fits","score":88},
	  {"tool_slug":"code-helper","stage_key":"main","model_id":"invented/model","recommendation":"ADD_AS_PRIMARY","reason":"bad","score":100}
	]`

	rows := parseScoutRecommendations(raw, catalog)
	if len(rows) != 1 {
		t.Fatalf("got %d recommendations, want 1", len(rows))
	}
	if rows[0].ModelCatalogID != id || rows[0].Recommendation != "ADD_AS_BACKUP" {
		t.Fatalf("unexpected parsed recommendation: %#v", rows[0])
	}
}

func TestMapOpenRouterModelCapabilities(t *testing.T) {
	m := openRouterModel{
		ID:                  "vendor/coder:free",
		Name:                "Coder (free)",
		ContextLength:       131072,
		Pricing:             map[string]string{"prompt": "0", "completion": "0"},
		SupportedParameters: []string{"tools", "structured_outputs"},
	}
	m.Architecture.InputModalities = []string{"text"}
	m.Architecture.OutputModalities = []string{"text"}
	got := mapOpenRouterModel(m)
	if !got.IsFree || got.ContextWindow != 131072 {
		t.Fatalf("unexpected mapped model: %#v", got)
	}
	if tc, _ := got.Capabilities["tool_calling"].(bool); !tc {
		t.Fatal("tool calling capability not mapped")
	}
}
