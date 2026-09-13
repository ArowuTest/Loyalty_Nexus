package services

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"loyalty-nexus/internal/domain/entities"
)

type routingStoreStub struct {
	stages []entities.AIToolStage
}

func (s *routingStoreStub) ListCandidates(context.Context, string, string) ([]entities.AIRouteCandidate, error) {
	return nil, nil
}
func (s *routingStoreStub) ListStagesForTool(context.Context, string) ([]entities.AIToolStage, error) {
	return s.stages, nil
}
func (s *routingStoreStub) RecordAttempt(context.Context, *entities.AIGenerationAttempt) error {
	return nil
}
func (s *routingStoreStub) UpdateAttempt(context.Context, uuid.UUID, map[string]interface{}) error {
	return nil
}
func (s *routingStoreStub) ValidateToolRoute(context.Context, string) error { return nil }

func TestQueuePolicyChoosesHeaviestActiveStage(t *testing.T) {
	o := &AIStudioOrchestrator{routingDB: &routingStoreStub{stages: []entities.AIToolStage{
		{IsActive: true, QueueClass: entities.QueueInteractive, MaxQueueSeconds: 5},
		{IsActive: true, QueueClass: entities.QueueHeavyAsync, MaxQueueSeconds: 300},
		{IsActive: true, QueueClass: entities.QueueAsync, MaxQueueSeconds: 120},
	}}}
	class, wait := o.QueuePolicyForTool(context.Background(), "video-jingle")
	if class != entities.QueueHeavyAsync || wait != 300 {
		t.Fatalf("got class=%s wait=%d", class, wait)
	}
}

func TestEstimateRouteCostCoversVariableVideoDuration(t *testing.T) {
	c := entities.AIRouteCandidate{
		Provider: entities.AIProviderConfig{Template: entities.TemplateGrokVideo, CostMicros: 300000},
	}
	if got := estimateRouteCostMicros(c, providerInput{DurationSecs: 10}); got != 500000 {
		t.Fatalf("got %d want 500000", got)
	}
}

func TestEstimateRouteCostCoversMultiImageCount(t *testing.T) {
	c := entities.AIRouteCandidate{
		Provider: entities.AIProviderConfig{Template: entities.TemplateFALImageUltra, CostMicros: 40000},
	}
	got := estimateRouteCostMicros(c, providerInput{Extra: map[string]interface{}{"num_images": float64(4)}})
	if got != 160000 {
		t.Fatalf("got %d want 160000", got)
	}
}

func functionBody(src, name string) string {
	start := strings.Index(src, "func (o *AIStudioOrchestrator) "+name)
	if start < 0 {
		return ""
	}
	rest := src[start:]
	if next := strings.Index(rest[5:], "\nfunc "); next >= 0 {
		return rest[:next+5]
	}
	return rest
}

func TestCustomerFacingDispatchersDoNotSelectProviderCredentialsDirectly(t *testing.T) {
	files := []string{"ai_studio_service.go", "ai_agent.go"}
	var source strings.Builder
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		source.Write(raw)
		source.WriteByte('\n')
	}
	src := source.String()
	names := []string{
		"dispatchText", "dispatchImage", "dispatchVideo", "dispatchVoiceOrTranslate",
		"dispatchMusic", "dispatchComposite", "dispatchVision", "dispatchAvatar",
		"dispatchNexusAgent",
	}
	for _, name := range names {
		body := functionBody(src, name)
		if body == "" {
			t.Fatalf("dispatcher %s not found", name)
		}
		for _, forbidden := range []string{"os.Getenv(", ".grokClient", "GEMINI_API_KEY", "GROQ_API_KEY", "DEEPSEEK_API_KEY", "FAL_API_KEY", "POLLINATIONS_SECRET_KEY"} {
			if strings.Contains(body, forbidden) {
				t.Fatalf("dispatcher %s contains forbidden provider authority %q", name, forbidden)
			}
		}
	}
}
