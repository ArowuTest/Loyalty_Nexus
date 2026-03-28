package entities

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"os"
	"strings"
)

// ResolveKey returns the usable API key for a provider config.
// Resolution order:
//  1. Decrypt api_key_enc (AES-GCM or b64) if non-empty.
//  2. Fall back to the named env var (env_key).
//
// Returns "" if no key is available.
func (p *AIProviderConfig) ResolveKey() string {
	if p.APIKeyEnc != "" {
		if key, err := decryptKey(p.APIKeyEnc); err == nil && key != "" {
			return key
		}
	}
	if p.EnvKey != "" {
		return os.Getenv(p.EnvKey)
	}
	return ""
}

// decryptKey reverses the encryption applied by the admin handler.
// Supports two formats:
//   - "aes:<base64>"  → AES-256-GCM, key from PROVIDER_ENCRYPTION_KEY env var
//   - "b64:<base64>"  → plain base64 (no encryption key configured at store time)
//   - anything else   → treated as legacy plaintext
func decryptKey(enc string) (string, error) {
	switch {
	case strings.HasPrefix(enc, "b64:"):
		raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(enc, "b64:"))
		return string(raw), err

	case strings.HasPrefix(enc, "aes:"):
		encKey := os.Getenv("PROVIDER_ENCRYPTION_KEY")
		if encKey == "" {
			return "", nil // key not available in this environment
		}
		data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(enc, "aes:"))
		if err != nil {
			return "", err
		}
		block, err := aes.NewCipher([]byte(encKey))
		if err != nil {
			return "", err
		}
		aesGCM, err := cipher.NewGCM(block)
		if err != nil {
			return "", err
		}
		ns := aesGCM.NonceSize()
		if len(data) < ns {
			return "", nil
		}
		plain, err := aesGCM.Open(nil, data[:ns], data[ns:], nil)
		return string(plain), err

	default:
		return enc, nil // legacy plaintext stored directly
	}
}
