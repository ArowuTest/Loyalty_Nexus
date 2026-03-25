package external

// asset_storage_crypto.go — RSA signing helper used by the GCS backend
// for OAuth2 service-account JWT assertions.
// Kept in a separate file to avoid polluting asset_storage.go with crypto imports.

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

// rsaSignPKCS1v15SHA256 signs msg with a PKCS#8 or PKCS#1 PEM-encoded RSA
// private key and returns the raw signature bytes.
func rsaSignPKCS1v15SHA256(pemKey, msg []byte) ([]byte, error) {
	block, _ := pem.Decode(pemKey)
	if block == nil {
		return nil, fmt.Errorf("GCS: failed to decode PEM block from private key")
	}

	// Try PKCS#8 first (service account keys are PKCS#8)
	var rsaKey *rsa.PrivateKey
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// Fall back to PKCS#1
		rsaKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("GCS: parse private key: %w", err)
		}
	} else {
		var ok bool
		rsaKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("GCS: private key is not RSA")
		}
	}

	h := sha256.Sum256(msg)
	return rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, h[:])
}
