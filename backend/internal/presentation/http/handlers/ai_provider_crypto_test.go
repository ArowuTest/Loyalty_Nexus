package handlers

import "testing"

func TestProviderCryptoAccepts64HexMasterKey(t *testing.T) {
	t.Setenv("ENVIRONMENT", "test")
	t.Setenv("GO_ENV", "test")
	t.Setenv("PROVIDER_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	enc, err := encryptProviderKey("test-provider-key")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if len(enc) < 4 || enc[:4] != "aes:" {
		t.Fatalf("expected AES envelope")
	}
	got, err := decryptProviderKey(enc)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if got != "test-provider-key" {
		t.Fatalf("round trip mismatch")
	}
}

func TestProviderCryptoRejectsWeakStorageInProduction(t *testing.T) {
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("GO_ENV", "production")
	t.Setenv("PROVIDER_ENCRYPTION_KEY", "")
	if _, err := encryptProviderKey("test-provider-key"); err == nil {
		t.Fatal("production must reject provider key storage without strong encryption")
	}
}
