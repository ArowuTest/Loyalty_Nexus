package handlers

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
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
		// No encryption key configured — store as base64 only (soft protection)
		return "b64:" + base64.StdEncoding.EncodeToString([]byte(raw)), nil
	}

	keyBytes := []byte(encKey)
	if len(keyBytes) != 32 {
		return "", fmt.Errorf("PROVIDER_ENCRYPTION_KEY must be exactly 32 bytes, got %d", len(keyBytes))
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

	keyBytes := []byte(encKey)
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

// pingProvider fires a minimal request against the provider to check credentials.
// Returns (ok bool, humanMessage string).
func pingProvider(ctx context.Context, p *entities.AIProviderConfig) (bool, string) {
	key := resolveProviderKey(p)

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
		if err != nil { return false, err.Error() }
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 || resp.StatusCode == 201 {
			return true, fmt.Sprintf("HTTP %d OK", resp.StatusCode)
		}
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)

	case entities.TemplateGemini:
		// Gemini: list models endpoint (cheap, no token cost)
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models?key=%s&pageSize=1", key)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := client.Do(req)
		if err != nil { return false, err.Error() }
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 { return true, "HTTP 200 OK" }
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)

	case entities.TemplatePollImage, entities.TemplatePollTTS, entities.TemplatePollVideo, entities.TemplatePollMusic:
		// Pollinations: list models (no cost)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://gen.pollinations.ai/image/models", nil)
		req.Header.Set("Authorization", "Bearer "+key)
		resp, err := client.Do(req)
		if err != nil { return false, err.Error() }
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 { return true, "HTTP 200 OK" }
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)

	case entities.TemplateHFImage:
		// HuggingFace: whoami
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://huggingface.co/api/whoami-v2", nil)
		req.Header.Set("Authorization", "Bearer "+key)
		resp, err := client.Do(req)
		if err != nil { return false, err.Error() }
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 { return true, "HTTP 200 OK (authenticated)" }
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)

	case entities.TemplateFALImage, entities.TemplateFALVideo, entities.TemplateFALBGRemove:
		// FAL: key validation via /v1/models (lightweight)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://fal.run/v1/models", nil)
		req.Header.Set("Authorization", "Key "+key)
		resp, err := client.Do(req)
		if err != nil { return false, err.Error() }
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 || resp.StatusCode == 401 {
			if resp.StatusCode == 200 { return true, "HTTP 200 OK" }
			return false, "HTTP 401 — invalid FAL key"
		}
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)

	case entities.TemplateElevenLabsTTS, entities.TemplateElevenLabsMusic:
		// ElevenLabs: GET /v1/user/subscription
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.elevenlabs.io/v1/user/subscription", nil)
		req.Header.Set("xi-api-key", key)
		resp, err := client.Do(req)
		if err != nil { return false, err.Error() }
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 { return true, "HTTP 200 OK" }
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)

	case entities.TemplateAssemblyAI:
		// AssemblyAI: GET /v2/account
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.assemblyai.com/v2/account", nil)
		req.Header.Set("Authorization", key)
		resp, err := client.Do(req)
		if err != nil { return false, err.Error() }
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 { return true, "HTTP 200 OK" }
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)

	case entities.TemplateGoogleTTS:
		// Google Cloud TTS: list voices (1-result, no speech synthesised)
		url := fmt.Sprintf("https://texttospeech.googleapis.com/v1/voices?key=%s&pageSize=1", key)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := client.Do(req)
		if err != nil { return false, err.Error() }
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 { return true, "HTTP 200 OK" }
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)

	case entities.TemplateGoogleTranslate:
		url := fmt.Sprintf("https://translation.googleapis.com/language/translate/v2/languages?key=%s&target=en", key)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := client.Do(req)
		if err != nil { return false, err.Error() }
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 { return true, "HTTP 200 OK" }
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)

	case entities.TemplateGroqWhisper:
		// Groq: list models
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.groq.com/openai/v1/models", nil)
		req.Header.Set("Authorization", "Bearer "+key)
		resp, err := client.Do(req)
		if err != nil { return false, err.Error() }
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 { return true, "HTTP 200 OK" }
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)

	case entities.TemplateRembg:
		// rembg self-hosted: GET /health
		svcURL := key // for rembg, env_key is REMBG_SERVICE_URL, key = URL
		if svcURL == "" { return false, "REMBG_SERVICE_URL not configured" }
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, svcURL+"/health", nil)
		resp, err := client.Do(req)
		if err != nil { return false, err.Error() }
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 { return true, "HTTP 200 OK" }
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)

	case entities.TemplateRemoveBG:
		// remove.bg: account info
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.remove.bg/v1.0/account", nil)
		req.Header.Set("X-Api-Key", key)
		resp, err := client.Do(req)
		if err != nil { return false, err.Error() }
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 { return true, "HTTP 200 OK" }
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
