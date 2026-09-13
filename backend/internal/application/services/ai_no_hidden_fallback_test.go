package services

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"

	"loyalty-nexus/internal/domain/entities"
)

// v2DispatchFiles are the files on the AI Routing V2 request path. Everything a
// generation touches between the configured route and the provider must be
// Admin-configured: no provider key from a named environment variable, no
// model chosen in code, no credential in a URL, no chain outside the route.
var v2DispatchFiles = []string{
	"ai_router_v2.go", "ai_streaming.go", "ai_provider_dispatch.go",
	"ai_document_provider.go", "ai_capacity_controller.go", "ai_provider_errors.go",
}

func TestV2DispatchPathHasNoHiddenFallbacks(t *testing.T) {
	forbidden := []struct {
		name string
		re   *regexp.Regexp
	}{
		{"literal provider-key env read", regexp.MustCompile(`os\.Getenv\("[^"]*(API_KEY|SECRET|TOKEN)[^"]*"\)`)},
		{"credential in a query string", regexp.MustCompile(`\?key=|[?&]key=%s`)},
		{"category chain outside the configured route", regexp.MustCompile(`runProviderChain|hardcodedFallbackChain|errNoDBProviders`)},
		// Model IDs come from ai_provider_configs.model_id — an empty one is a
		// configuration error, never a default picked here.
		{"model defaulted in code", regexp.MustCompile(`\bmodel\s*=\s*"[^"]+"`)},
		{"model literal in code", regexp.MustCompile(`"(gemini-[0-9][^"]*|llama-[^"]*|deepseek-[^"]*|gpt-[0-9][^"]*|claude-[^"]*|mistral[^"]*|grok-[^"]*|fal-ai/[^"]*|wan-fast|seedance[^"]*)"`)},
	}
	for _, file := range v2DispatchFiles {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		src := stripGoComments(string(raw))
		for _, f := range forbidden {
			for _, loc := range f.re.FindAllStringIndex(src, -1) {
				line := 1 + strings.Count(src[:loc[0]], "\n")
				t.Errorf("%s:%d: %s: %q", file, line, f.name, src[loc[0]:loc[1]])
			}
		}
	}
}

// stripGoComments removes // and /* */ comments so the guard binds to code,
// not to prose that mentions a forbidden token.
func stripGoComments(src string) string {
	src = regexp.MustCompile(`(?s)/\*.*?\*/`).ReplaceAllString(src, "")
	return regexp.MustCompile(`(?m)//.*$`).ReplaceAllString(src, "")
}

func TestUnknownTemplateIsAnErrorNotAFallback(t *testing.T) {
	o := &AIStudioOrchestrator{} // no HTTP client: any network attempt would panic
	_, _, _, err := o.callByTemplate(context.Background(), entities.AIProviderConfig{
		Slug: "mystery", Template: "no-such-template",
	}, providerInput{UserPrompt: "hi"})
	if err == nil || !strings.Contains(err.Error(), "unknown template") {
		t.Fatalf("expected an unknown-template error, got %v", err)
	}
}

func TestGeminiWithoutModelIDIsAConfigurationError(t *testing.T) {
	o := &AIStudioOrchestrator{}
	_, err := o.callGeminiFlashWithModel(context.Background(), "", "key", "sys", "user")
	if err == nil || !strings.Contains(err.Error(), "model_id") {
		t.Fatalf("an empty model_id must not be defaulted in code, got %v", err)
	}
	_, _, _, err = o.callByTemplate(context.Background(), entities.AIProviderConfig{
		Slug: "gemini", Template: entities.TemplateGemini, ModelID: "",
	}, providerInput{UserPrompt: "hi"})
	if err == nil || !strings.Contains(err.Error(), "model_id") {
		t.Fatalf("dispatch with an empty model_id must fail loudly, got %v", err)
	}
}
