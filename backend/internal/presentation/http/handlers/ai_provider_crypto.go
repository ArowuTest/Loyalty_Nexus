package handlers

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"loyalty-nexus/internal/domain/entities"
)

// encryptProviderKey encrypts a raw API key using AES-256-GCM.
// The encryption key is read from PROVIDER_ENCRYPTION_KEY env var (32-byte hex).
// If the env var is not set, the key is stored as base64-encoded plaintext
// (still better than raw in case of accidental log exposure).
func encryptProviderKey(raw string) (string, error) {
	encKey := os.Getenv("PROVIDER_ENCRYPTION_KEY")
	if encKey == "" {
		// Fail CLOSED by default: the reversible base64 fallback needs an explicit
		// local-dev opt-in and is never allowed in production — so a prod deploy
		// that forgets ENVIRONMENT cannot silently persist a plaintext-equivalent key.
		isProd := strings.EqualFold(os.Getenv("ENVIRONMENT"), "production") || strings.EqualFold(os.Getenv("GO_ENV"), "production")
		if isProd || os.Getenv("PROVIDER_KEY_ALLOW_PLAINTEXT_DEV") != "1" {
			return "", fmt.Errorf("PROVIDER_ENCRYPTION_KEY is required to store provider credentials (local development only: set PROVIDER_KEY_ALLOW_PLAINTEXT_DEV=1 to accept reversible storage)")
		}
		return "b64:" + base64.StdEncoding.EncodeToString([]byte(raw)), nil
	}

	keyBytes, err := decodeProviderMasterKey(encKey)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("aes.NewCipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("cipher.NewGCM: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("rand nonce: %w", err)
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(raw), nil)
	return "aes:" + base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decodeProviderMasterKey(value string) ([]byte, error) {
	if len(value) == 64 {
		decoded, err := hex.DecodeString(value)
		if err == nil && len(decoded) == 32 {
			return decoded, nil
		}
	}
	if len(value) == 32 {
		return []byte(value), nil
	}
	return nil, fmt.Errorf("provider master key must be 32 raw bytes or 64 hex characters")
}

// decryptProviderKey reverses encryptProviderKey.
func decryptProviderKey(enc string) (string, error) { //nolint:unused
	if strings.HasPrefix(enc, "b64:") {
		raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(enc, "b64:"))
		return string(raw), err
	}
	if !strings.HasPrefix(enc, "aes:") {
		return enc, nil // legacy plain value
	}

	encKey := os.Getenv("PROVIDER_ENCRYPTION_KEY")
	if encKey == "" {
		return "", fmt.Errorf("PROVIDER_ENCRYPTION_KEY not set — cannot decrypt")
	}

	keyBytes, err := decodeProviderMasterKey(encKey)
	if err != nil {
		return "", err
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(enc, "aes:"))
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	return string(plaintext), err
}

// resolveProviderKey returns the usable API key for a provider.
// Delegates to the entity method so key resolution logic lives in one place.
func resolveProviderKey(p *entities.AIProviderConfig) string {
	return p.ResolveKey()
}

// ── Provider ping ─────────────────────────────────────────────────────────────

// pingProvider fires a minimal request against the provider to check credentials
// and returns (ok, humanMessage). The message is scrubbed of the credential so a
// transport error (url.Error echoes the full request URL) can never surface the
// key in the API response or the persisted test result (B1).
func pingProvider(ctx context.Context, p *entities.AIProviderConfig) (bool, string) {
	key := resolveProviderKey(p)
	ok, msg := pingProviderRaw(ctx, p, key)
	return ok, redactSecret(msg, key)
}

// redactSecret replaces every occurrence of secret in msg. Very short values are
// left alone: there is nothing meaningful to hide and it avoids masking "".
func redactSecret(msg, secret string) string {
	if len(secret) < 8 {
		return msg
	}
	return strings.ReplaceAll(msg, secret, "***")
}

func pingProviderRaw(ctx context.Context, p *entities.AIProviderConfig, key string) (bool, string) {
	client := &http.Client{Timeout: 15 * time.Second}

	switch p.Template {

	case entities.TemplatePollText, entities.TemplateDeepSeek:
		// OpenAI-compat: POST /v1/chat/completions with 1-token completion
		baseURL := resolveBaseURL(p)
		body := `{"model":"` + p.ModelID + `","messages":[{"role":"user","content":"hi"}],"max_tokens":1}`
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/chat/completions",
			strings.NewReader(body))
		if err != nil {
			return false, err.Error()
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+key)
		resp, err := client.Do(req)
		if err != nil {
			return false, err.Error()
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 || resp.StatusCode == 201 {
			return true, fmt.Sprintf("HTTP %d OK", resp.StatusCode)
		}
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)

	case entities.TemplateGemini:
		// Gemini: list models endpoint (cheap, no token cost). The key travels in
		// the x-goog-api-key header, never the URL — a URL-embedded key leaks via
		// url.Error messages, proxy/access logs and the persisted test result.
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
			"https://generativelanguage.googleapis.com/v1beta/models?pageSize=1", nil)
		req.Header.Set("x-goog-api-key", key)
		resp, err := client.Do(req)
		if err != nil {
			return false, err.Error()
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 {
			return true, "HTTP 200 OK"
		}
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)

	case entities.TemplatePollImage, entities.TemplatePollTTS, entities.TemplatePollVideo, entities.TemplatePollMusic:
		// Pollinations: list models (no cost)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://gen.pollinations.ai/image/models", nil)
		req.Header.Set("Authorization", "Bearer "+key)
		resp, err := client.Do(req)
		if err != nil {
			return false, err.Error()
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 {
			return true, "HTTP 200 OK"
		}
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)

	case entities.TemplateHFImage:
		// HuggingFace: whoami
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://huggingface.co/api/whoami-v2", nil)
		req.Header.Set("Authorization", "Bearer "+key)
		resp, err := client.Do(req)
		if err != nil {
			return false, err.Error()
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 {
			return true, "HTTP 200 OK (authenticated)"
		}
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)

	case entities.TemplateFALImage, entities.TemplateFALVideo, entities.TemplateFALBGRemove:
		// FAL: key validation via /v1/models (lightweight)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://fal.run/v1/models", nil)
		req.Header.Set("Authorization", "Key "+key)
		resp, err := client.Do(req)
		if err != nil {
			return false, err.Error()
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 || resp.StatusCode == 401 {
			if resp.StatusCode == 200 {
				return true, "HTTP 200 OK"
			}
			return false, "HTTP 401 — invalid FAL key"
		}
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)

	case entities.TemplateElevenLabsTTS, entities.TemplateElevenLabsMusic:
		// ElevenLabs: GET /v1/user/subscription
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.elevenlabs.io/v1/user/subscription", nil)
		req.Header.Set("xi-api-key", key)
		resp, err := client.Do(req)
		if err != nil {
			return false, err.Error()
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 {
			return true, "HTTP 200 OK"
		}
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)

	case entities.TemplateAssemblyAI:
		// AssemblyAI: GET /v2/account
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.assemblyai.com/v2/account", nil)
		req.Header.Set("Authorization", key)
		resp, err := client.Do(req)
		if err != nil {
			return false, err.Error()
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 {
			return true, "HTTP 200 OK"
		}
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)

	case entities.TemplateGoogleTTS:
		// Google Cloud TTS: list voices (1-result, no speech synthesised). Key in
		// header, not URL (see Gemini note).
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
			"https://texttospeech.googleapis.com/v1/voices?pageSize=1", nil)
		req.Header.Set("x-goog-api-key", key)
		resp, err := client.Do(req)
		if err != nil {
			return false, err.Error()
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 {
			return true, "HTTP 200 OK"
		}
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)

	case entities.TemplateGoogleTranslate:
		// Key in header, not URL (see Gemini note).
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
			"https://translation.googleapis.com/language/translate/v2/languages?target=en", nil)
		req.Header.Set("x-goog-api-key", key)
		resp, err := client.Do(req)
		if err != nil {
			return false, err.Error()
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 {
			return true, "HTTP 200 OK"
		}
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)

	case entities.TemplateGroqWhisper:
		// Groq: list models
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.groq.com/openai/v1/models", nil)
		req.Header.Set("Authorization", "Bearer "+key)
		resp, err := client.Do(req)
		if err != nil {
			return false, err.Error()
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 {
			return true, "HTTP 200 OK"
		}
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)

	case entities.TemplateRembg:
		// rembg self-hosted: GET /health
		svcURL := key // for rembg, env_key is REMBG_SERVICE_URL, key = URL
		if svcURL == "" {
			return false, "REMBG_SERVICE_URL not configured"
		}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, svcURL+"/health", nil)
		resp, err := client.Do(req)
		if err != nil {
			return false, err.Error()
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 {
			return true, "HTTP 200 OK"
		}
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)

	case entities.TemplateRemoveBG:
		// remove.bg: account info
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.remove.bg/v1.0/account", nil)
		req.Header.Set("X-Api-Key", key)
		resp, err := client.Do(req)
		if err != nil {
			return false, err.Error()
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 {
			return true, "HTTP 200 OK"
		}
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)

	default:
		return false, fmt.Sprintf("no ping defined for template %q — mark as tested manually", p.Template)
	}
}

// resolveBaseURL returns the base API URL for openai-compatible providers.
func resolveBaseURL(p *entities.AIProviderConfig) string {
	if u, ok := p.ExtraConfig["base_url"].(string); ok && u != "" {
		return u
	}
	switch {
	case strings.Contains(p.Slug, "pollinations"):
		return "https://gen.pollinations.ai"
	case strings.Contains(p.Slug, "deepseek"):
		return "https://api.deepseek.com"
	case strings.Contains(p.Slug, "groq"):
		return "https://api.groq.com/openai"
	default:
		return "https://gen.pollinations.ai"
	}
}
