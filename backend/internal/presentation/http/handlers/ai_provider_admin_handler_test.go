package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/presentation/http/middleware"
)

func providerAdminRequest(body string, role entities.AdminRole) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/ai-providers/x", strings.NewReader(body))
	if role != "" {
		r = r.WithContext(context.WithValue(r.Context(), middleware.ContextAdminRole, string(role)))
	}
	r.SetPathValue("id", "00000000-0000-0000-0000-000000000001")
	return r
}

// Every mutation and the live Test endpoint are super_admin-only (review B1).
// The repository is nil on purpose: any handler that consulted it would panic,
// so a clean 403 proves the role guard runs before any data access.
func TestProviderAdminMutationsAreSuperAdminOnly(t *testing.T) {
	h := NewAIProviderAdminHandler(nil)
	endpoints := map[string]http.HandlerFunc{
		"CreateProvider":     h.CreateProvider,
		"UpdateProvider":     h.UpdateProvider,
		"DeleteProvider":     h.DeleteProvider,
		"ActivateProvider":   h.ActivateProvider,
		"DeactivateProvider": h.DeactivateProvider,
		"TestProvider":       h.TestProvider,
	}
	for _, role := range []entities.AdminRole{entities.RoleFinance, entities.RoleOperations, entities.RoleContent, ""} {
		for name, fn := range endpoints {
			rec := httptest.NewRecorder()
			fn(rec, providerAdminRequest(`{}`, role))
			if rec.Code != http.StatusForbidden {
				t.Fatalf("%s with role %q: got %d, want 403", name, role, rec.Code)
			}
		}
	}
}

func TestProviderAdminSuperAdminPassesTheGuard(t *testing.T) {
	h := NewAIProviderAdminHandler(nil)
	rec := httptest.NewRecorder()
	// Malformed JSON is rejected after the role guard and before the repository
	// is touched — so this is the furthest a nil-repo request can travel.
	h.CreateProvider(rec, providerAdminRequest(`{not json`, entities.RoleSuperAdmin))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("super_admin should pass the guard and fail on the body: got %d, want 400", rec.Code)
	}
}

// A stored credential is bound to the base_url it was entered for (review B1).
func TestStoredKeyStillBound(t *testing.T) {
	old := entities.ProviderExtraConfig{"base_url": "https://api.groq.com/openai"}
	sameHost := entities.ProviderExtraConfig{"base_url": "https://api.groq.com/openai", "web_search": true}
	moved := entities.ProviderExtraConfig{"base_url": "https://attacker.example"}
	fresh, empty := "sk-new", ""
	cases := []struct {
		name     string
		old, new entities.ProviderExtraConfig
		key      *string
		want     bool
	}{
		{"unchanged base_url keeps the key", old, sameHost, nil, true},
		{"no base_url before or after keeps the key", nil, entities.ProviderExtraConfig{"web_search": true}, nil, true},
		{"moved without a key: cleared", old, moved, nil, false},
		{"moved with an empty key: cleared", old, moved, &empty, false},
		{"moved with a fresh key: rebound", old, moved, &fresh, true},
		{"base_url dropped (falls back to the slug default host): cleared", old, entities.ProviderExtraConfig{}, nil, false},
		{"base_url introduced where there was none: cleared", nil, moved, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := storedKeyStillBound(tc.old, tc.new, tc.key); got != tc.want {
				t.Fatalf("storedKeyStillBound = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestValidateProviderBaseURL(t *testing.T) {
	cases := []struct {
		url    string
		wantOK bool
	}{
		{"", true},
		{"https://api.groq.com/openai", true},
		{"https://openrouter.ai/api", true},
		{"http://api.groq.com/openai", false},
		{"https://", false},
		{"api.groq.com", false},
		{"ftp://api.groq.com", false},
		{"javascript:alert(1)", false},
	}
	for _, tc := range cases {
		t.Run(tc.url, func(t *testing.T) {
			err := validateProviderBaseURL(tc.url)
			if (err == nil) != tc.wantOK {
				t.Fatalf("validateProviderBaseURL(%q) err=%v, wantOK=%v", tc.url, err, tc.wantOK)
			}
		})
	}
}
