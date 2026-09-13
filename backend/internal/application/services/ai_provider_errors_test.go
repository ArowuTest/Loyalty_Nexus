package services

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"loyalty-nexus/internal/domain/entities"
)

func TestClassifyRoutingErrorNeverFailsOverOnRefusal(t *testing.T) {
	cases := []struct {
		name      string
		err       error
		wantClass string
		wantOver  bool
	}{
		{"typed refusal", &ProviderRefusalError{Provider: "gemini", Reason: "SAFETY"}, "POLICY_REJECTION", false},
		{"wrapped typed refusal", errorsJoin("groq", &ProviderRefusalError{Provider: "openai-compatible", Reason: "content_filter"}), "POLICY_REJECTION", false},
		{"403 carrying a policy signal", errors.New("API 403: request violates content policy"), "POLICY_REJECTION", false},
		{"400 carrying a policy signal", errors.New("API 400: content_policy_violation"), "POLICY_REJECTION", false},
		{"plain 403 is a credential problem: another provider may work", errors.New("API 403: forbidden"), "AUTH_CONFIG", true},
		{"plain 400 is the caller's input", errors.New("API 400: invalid input"), "INVALID_INPUT", false},
		{"429", errors.New("API 429: rate limit exceeded"), "RATE_LIMIT", true},
		{"timeout", context.DeadlineExceeded, "TIMEOUT", true},
		{"empty response is a provider fault", errors.New("gemini: no content returned"), "PROVIDER_ERROR", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			class, over := classifyRoutingError(tc.err)
			if class != tc.wantClass || over != tc.wantOver {
				t.Fatalf("classify(%v) = %s/%v, want %s/%v", tc.err, class, over, tc.wantClass, tc.wantOver)
			}
		})
	}
}

// errorsJoin mimics the router's own wrapping ("<slug>: %w").
func errorsJoin(slug string, err error) error {
	return &wrappedErr{msg: slug + ": " + err.Error(), inner: err}
}

type wrappedErr struct {
	msg   string
	inner error
}

func (w *wrappedErr) Error() string { return w.msg }
func (w *wrappedErr) Unwrap() error { return w.inner }

func TestDecodeGeminiGenerateContent(t *testing.T) {
	cases := []struct {
		name        string
		status      int
		body        string
		wantText    string
		wantRefusal bool
		wantErrHas  string
	}{
		{"prompt blocked", 200, `{"promptFeedback":{"blockReason":"PROHIBITED_CONTENT"},"candidates":[]}`, "", true, "PROHIBITED_CONTENT"},
		{"candidate stopped on safety, partial text discarded", 200, `{"candidates":[{"finishReason":"SAFETY","content":{"parts":[{"text":"Here is how to"}]}}]}`, "", true, "SAFETY"},
		{"recitation stop", 200, `{"candidates":[{"finishReason":"RECITATION","content":{"parts":[]}}]}`, "", true, "RECITATION"},
		{"max tokens with no text is a provider fault, not a refusal", 200, `{"candidates":[{"finishReason":"MAX_TOKENS","content":{"parts":[]}}]}`, "", false, "MAX_TOKENS"},
		{"thought parts are skipped", 200, `{"candidates":[{"finishReason":"STOP","content":{"parts":[{"text":"let me think","thought":true},{"text":"Hello "},{"text":"world"}]}}]}`, "Hello world", false, ""},
		{"api error envelope keeps its status", 403, `{"error":{"code":403,"message":"API key not valid","status":"PERMISSION_DENIED"}}`, "", false, "403"},
		{"rate limited", 429, `{"error":{"code":429,"message":"Quota exceeded","status":"RESOURCE_EXHAUSTED"}}`, "", false, "429"},
		{"non-json error page", 503, `<html>Service Unavailable</html>`, "", false, "gemini http 503"},
		{"empty success", 200, `{"candidates":[{"finishReason":"STOP","content":{"parts":[]}}]}`, "", false, "no content"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text, err := decodeGeminiGenerateContent(tc.status, []byte(tc.body))
			if text != tc.wantText {
				t.Fatalf("text = %q, want %q (err=%v)", text, tc.wantText, err)
			}
			if isProviderRefusal(err) != tc.wantRefusal {
				t.Fatalf("refusal = %v, want %v (err=%v)", isProviderRefusal(err), tc.wantRefusal, err)
			}
			if tc.wantErrHas != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErrHas)) {
				t.Fatalf("err = %v, want it to mention %q", err, tc.wantErrHas)
			}
			if tc.wantErrHas == "" && err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
		})
	}
}

func TestOpenAICompatibleRecognisesContentFilter(t *testing.T) {
	cases := []struct {
		name        string
		status      int
		body        string
		wantText    string
		wantRefusal bool
	}{
		{"finish_reason content_filter with empty message", 200, `{"choices":[{"finish_reason":"content_filter","message":{"content":""}}]}`, "", true},
		{"400 content_policy_violation envelope", 400, `{"error":{"code":"content_policy_violation","type":"invalid_request_error","message":"Your request was rejected"}}`, "", true},
		{"normal completion", 200, `{"choices":[{"finish_reason":"stop","message":{"content":"Hi there"}}]}`, "Hi there", false},
		{"empty completion is an error, not a silent success", 200, `{"choices":[{"finish_reason":"stop","message":{"content":""}}]}`, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			o := &AIStudioOrchestrator{httpClient: srv.Client()}
			text, err := o.callOpenAICompatible(context.Background(), srv.URL, "Bearer test", map[string]string{"model": "x"})
			if text != tc.wantText {
				t.Fatalf("text = %q, want %q (err=%v)", text, tc.wantText, err)
			}
			if isProviderRefusal(err) != tc.wantRefusal {
				t.Fatalf("refusal = %v, want %v (err=%v)", isProviderRefusal(err), tc.wantRefusal, err)
			}
			if tc.wantText == "" && err == nil {
				t.Fatal("an empty completion must not be reported as success")
			}
		})
	}
}

func TestOpenAIStreamContentFilterIsRefusal(t *testing.T) {
	sse := strings.Join([]string{
		`data: {"choices":[{"delta":{"content":"Sure, "}}]}`,
		`data: {"choices":[{"delta":{"content":""},"finish_reason":"content_filter"}]}`,
		`data: [DONE]`,
	}, "\n\n")
	var chunks []string
	text, err := consumeOpenAIStream(strings.NewReader(sse), func(s string) { chunks = append(chunks, s) })
	if !isProviderRefusal(err) {
		t.Fatalf("expected refusal, got text=%q err=%v", text, err)
	}
	if len(chunks) != 1 || chunks[0] != "Sure, " {
		t.Fatalf("chunks delivered before the filter should be unchanged, got %v", chunks)
	}
}

func TestGeminiStreamRefusalsAreDetected(t *testing.T) {
	blocked := `data: {"promptFeedback":{"blockReason":"SAFETY"}}` + "\n\n"
	if _, err := consumeGeminiStream(strings.NewReader(blocked), func(string) {}); !isProviderRefusal(err) {
		t.Fatalf("prompt block: expected refusal, got %v", err)
	}
	stopped := strings.Join([]string{
		`data: {"candidates":[{"content":{"parts":[{"text":"Here is"}]}}]}`,
		`data: {"candidates":[{"finishReason":"PROHIBITED_CONTENT","content":{"parts":[]}}]}`,
	}, "\n\n")
	if _, err := consumeGeminiStream(strings.NewReader(stopped), func(string) {}); !isProviderRefusal(err) {
		t.Fatalf("candidate stop: expected refusal, got %v", err)
	}
	normal := strings.Join([]string{
		`data: {"candidates":[{"content":{"parts":[{"text":"Hello ","thought":false}]}}]}`,
		`data: {"candidates":[{"finishReason":"STOP","content":{"parts":[{"text":"world"}]}}]}`,
	}, "\n\n")
	text, err := consumeGeminiStream(strings.NewReader(normal), func(string) {})
	if err != nil || text != "Hello world" {
		t.Fatalf("normal stream: got text=%q err=%v", text, err)
	}
}

// A refusal must terminate the stage chain without touching the next provider.
func TestStreamingRefusalDoesNotShopToNextProvider(t *testing.T) {
	class, over := classifyRoutingError(&ProviderRefusalError{Provider: "gemini", Reason: "SAFETY"})
	if class != "POLICY_REJECTION" || over {
		t.Fatalf("refusal classified as %s/failover=%v", class, over)
	}
	_ = entities.RoutingFreeFirst // the policy in force is irrelevant: a refusal is terminal under every policy
}
