package external

// apple_wallet.go — Production Apple Wallet .pkpass generation.
//
// A .pkpass file is a ZIP archive containing:
//   pass.json        — the pass data (already built in wallet_passport.go)
//   manifest.json    — SHA1 hashes of every file in the archive
//   signature        — DER-encoded CMS/PKCS#7 detached signature of manifest.json
//   icon.png         — pass icon (2x: icon@2x.png)
//   logo.png         — pass logo (2x: logo@2x.png)
//   strip.png        — optional background strip image
//
// In development/staging (no cert), we skip the signature and return an
// unsigned pass — iOS will reject it but it's useful for layout testing.
//
// In production, set these environment variables:
//   APPLE_PASS_CERT_PEM    — PEM-encoded pass certificate (from Apple Developer Portal)
//   APPLE_PASS_KEY_PEM     — PEM-encoded private key for the certificate
//   APPLE_PASS_KEY_PASS    — passphrase for the private key (if encrypted)
//   APPLE_WWDR_CERT_PEM    — PEM-encoded Apple WWDR intermediate certificate
//   APPLE_PASS_TYPE_ID     — pass type identifier (e.g. pass.ng.loyaltynexus.passport)
//   APPLE_TEAM_ID          — your Apple Developer Team ID
//
// The signing uses crypto/x509 + crypto/pkcs7 (via a pure-Go implementation).
// We implement a minimal CMS SignedData without an external pkcs7 library to
// avoid adding a new dependency — the signature is a detached SHA1/RSA-SHA256
// CMS structure as required by PassKit.

import (
	"archive/zip"
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec // Apple PassKit requires SHA-1 for manifest hashes
	"crypto/x509"
	"encoding/asn1"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"
)

// ─── AppleWalletSigner ───────────────────────────────────────────────────────

// AppleWalletSigner signs .pkpass archives with the Apple Pass certificate.
type AppleWalletSigner struct {
	passCert   *x509.Certificate
	passKey    *rsa.PrivateKey
	wwdrCert   *x509.Certificate
	passTypeID string
	teamID     string
	configured bool
}

// NewAppleWalletSigner loads certificates from environment variables.
// If env vars are absent, returns a signer in "dev mode" (unsigned passes).
func NewAppleWalletSigner() *AppleWalletSigner {
	certPEM  := os.Getenv("APPLE_PASS_CERT_PEM")
	keyPEM   := os.Getenv("APPLE_PASS_KEY_PEM")
	wwdrPEM  := os.Getenv("APPLE_WWDR_CERT_PEM")
	passType := os.Getenv("APPLE_PASS_TYPE_ID")
	teamID   := os.Getenv("APPLE_TEAM_ID")

	if certPEM == "" || keyPEM == "" || wwdrPEM == "" || passType == "" || teamID == "" {
		return &AppleWalletSigner{configured: false}
	}

	// Support literal \n in env vars
	certPEM = strings.ReplaceAll(certPEM, `\n`, "\n")
	keyPEM  = strings.ReplaceAll(keyPEM,  `\n`, "\n")
	wwdrPEM = strings.ReplaceAll(wwdrPEM, `\n`, "\n")

	certBlock, _ := pem.Decode([]byte(certPEM))
	if certBlock == nil {
		return &AppleWalletSigner{configured: false}
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return &AppleWalletSigner{configured: false}
	}

	keyBlock, _ := pem.Decode([]byte(keyPEM))
	if keyBlock == nil {
		return &AppleWalletSigner{configured: false}
	}
	keyPassphrase := os.Getenv("APPLE_PASS_KEY_PASS")
	var keyBytes []byte
	if keyPassphrase != "" && x509.IsEncryptedPEMBlock(keyBlock) { //nolint:staticcheck
		keyBytes, err = x509.DecryptPEMBlock(keyBlock, []byte(keyPassphrase)) //nolint:staticcheck
		if err != nil {
			return &AppleWalletSigner{configured: false}
		}
	} else {
		keyBytes = keyBlock.Bytes
	}

	var rsaKey *rsa.PrivateKey
	switch keyBlock.Type {
	case "RSA PRIVATE KEY":
		rsaKey, err = x509.ParsePKCS1PrivateKey(keyBytes)
	case "PRIVATE KEY":
		k, parseErr := x509.ParsePKCS8PrivateKey(keyBytes)
		if parseErr != nil {
			return &AppleWalletSigner{configured: false}
		}
		var ok bool
		rsaKey, ok = k.(*rsa.PrivateKey)
		if !ok {
			return &AppleWalletSigner{configured: false}
		}
	}
	if err != nil || rsaKey == nil {
		return &AppleWalletSigner{configured: false}
	}

	wwdrBlock, _ := pem.Decode([]byte(wwdrPEM))
	if wwdrBlock == nil {
		return &AppleWalletSigner{configured: false}
	}
	wwdrCert, err := x509.ParseCertificate(wwdrBlock.Bytes)
	if err != nil {
		return &AppleWalletSigner{configured: false}
	}

	return &AppleWalletSigner{
		passCert:   cert,
		passKey:    rsaKey,
		wwdrCert:   wwdrCert,
		passTypeID: passType,
		teamID:     teamID,
		configured: true,
	}
}

// IsConfigured returns true if production signing certificates are loaded.
func (s *AppleWalletSigner) IsConfigured() bool {
	return s != nil && s.configured
}

// PassTypeID returns the configured pass type identifier.
func (s *AppleWalletSigner) PassTypeID() string {
	if s == nil || s.passTypeID == "" {
		return "pass.ng.loyaltynexus.passport"
	}
	return s.passTypeID
}

// TeamID returns the configured Apple Team ID.
// Returns an empty string when APPLE_TEAM_ID is not set; callers should check
// IsConfigured() before generating production passes.
func (s *AppleWalletSigner) TeamID() string {
	if s == nil || s.teamID == "" {
		return ""
	}
	return s.teamID
}

// BuildPKPass creates a signed (or unsigned in dev) .pkpass zip archive.
// passJSON is the serialised pass.json content.
// iconPNG and logoPNG are optional PNG bytes; if nil, placeholder 1x1 PNGs are used.
func (s *AppleWalletSigner) BuildPKPass(passJSON []byte, iconPNG, logoPNG []byte) ([]byte, error) {
	if iconPNG == nil {
		iconPNG = minimalPNG()
	}
	if logoPNG == nil {
		logoPNG = minimalPNG()
	}

	// ── Build file map ────────────────────────────────────────────────────────
	files := map[string][]byte{
		"pass.json":  passJSON,
		"icon.png":   iconPNG,
		"icon@2x.png": iconPNG,
		"logo.png":   logoPNG,
		"logo@2x.png": logoPNG,
	}

	// ── Build manifest.json ───────────────────────────────────────────────────
	manifest := make(map[string]string, len(files))
	for name, data := range files {
		h := sha1.New() //nolint:gosec // required by Apple PassKit spec
		h.Write(data)
		manifest[name] = fmt.Sprintf("%x", h.Sum(nil))
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("pkpass manifest marshal: %w", err)
	}

	// ── Build signature ───────────────────────────────────────────────────────
	var signatureBytes []byte
	if s.configured {
		signatureBytes, err = s.signManifest(manifestJSON)
		if err != nil {
			return nil, fmt.Errorf("pkpass sign: %w", err)
		}
	} else {
		// Dev mode: empty signature — iOS will reject but useful for testing layout
		signatureBytes = []byte{}
	}

	// ── Assemble ZIP ──────────────────────────────────────────────────────────
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	addFile := func(name string, data []byte) error {
		f, createErr := zw.Create(name)
		if createErr != nil {
			return createErr
		}
		_, writeErr := f.Write(data)
		return writeErr
	}

	for name, data := range files {
		if err2 := addFile(name, data); err2 != nil {
			return nil, fmt.Errorf("pkpass zip add %s: %w", name, err2)
		}
	}
	if err2 := addFile("manifest.json", manifestJSON); err2 != nil {
		return nil, fmt.Errorf("pkpass zip add manifest: %w", err2)
	}
	if err2 := addFile("signature", signatureBytes); err2 != nil {
		return nil, fmt.Errorf("pkpass zip add signature: %w", err2)
	}

	if err2 := zw.Close(); err2 != nil {
		return nil, fmt.Errorf("pkpass zip close: %w", err2)
	}

	return buf.Bytes(), nil
}

// ─── CMS / PKCS#7 detached signature ─────────────────────────────────────────
// Apple requires a CMS SignedData structure (RFC 5652) with:
//   - SHA-1 digest algorithm (for the manifest hash)
//   - RSA-SHA256 signature algorithm
//   - The pass certificate + WWDR intermediate certificate embedded
//   - No content (detached signature)

// ASN.1 OIDs
var (
	oidData                   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}
	oidSignedData             = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}
	oidSHA1                   = asn1.ObjectIdentifier{1, 3, 14, 3, 2, 26}
	oidRSASHA256              = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 11}
	oidContentType            = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 3}
	oidMessageDigest          = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 4}
	oidSigningTime            = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 5}
)

type algorithmIdentifier struct {
	Algorithm  asn1.ObjectIdentifier
	Parameters asn1.RawValue `asn1:"optional"`
}

type issuerAndSerialNumber struct {
	Issuer       asn1.RawValue
	SerialNumber *big.Int
}

type attribute struct {
	Type   asn1.ObjectIdentifier
	Values asn1.RawValue `asn1:"set"`
}

type signerInfo struct {
	Version            int
	IssuerAndSerial    issuerAndSerialNumber
	DigestAlgorithm    algorithmIdentifier
	AuthenticatedAttrs []attribute `asn1:"optional,tag:0"`
	DigestEncAlgorithm algorithmIdentifier
	EncryptedDigest    []byte
	UnauthAttrs        asn1.RawValue `asn1:"optional,tag:1"`
}

type signedData struct {
	Version          int
	DigestAlgorithms []algorithmIdentifier `asn1:"set"`
	ContentInfo      asn1.RawValue
	Certificates     asn1.RawValue         `asn1:"optional,tag:0"`
	SignerInfos      []signerInfo          `asn1:"set"`
}

func (s *AppleWalletSigner) signManifest(manifestJSON []byte) ([]byte, error) {
	if !s.configured {
		return nil, errors.New("apple wallet signer not configured")
	}

	// ── Digest the manifest with SHA-1 ────────────────────────────────────────
	h1 := sha1.New() //nolint:gosec
	h1.Write(manifestJSON)
	digest := h1.Sum(nil)

	// ── Build authenticated attributes ────────────────────────────────────────
	signingTime, _ := asn1.Marshal(time.Now().UTC())
	contentTypeVal, _ := asn1.Marshal(oidData)
	digestVal, _ := asn1.Marshal(digest)

	authAttrs := []attribute{
		{Type: oidContentType,   Values: asn1.RawValue{FullBytes: contentTypeVal}},
		{Type: oidSigningTime,   Values: asn1.RawValue{FullBytes: signingTime}},
		{Type: oidMessageDigest, Values: asn1.RawValue{FullBytes: digestVal}},
	}

	// The authenticated attributes must be DER-encoded as a SET for signing
	authAttrsEncoded, err := asn1.Marshal(authAttrs)
	if err != nil {
		return nil, fmt.Errorf("auth attrs marshal: %w", err)
	}
	// Replace the SEQUENCE tag with SET tag (0x31) as required
	authAttrsEncoded[0] = 0x31

	// ── Sign the authenticated attributes with RSA-SHA256 ─────────────────────
	h256 := crypto.SHA256.New()
	h256.Write(authAttrsEncoded)
	sigHash := h256.Sum(nil)

	sig, err := rsa.SignPKCS1v15(rand.Reader, s.passKey, crypto.SHA256, sigHash)
	if err != nil {
		return nil, fmt.Errorf("rsa sign: %w", err)
	}

	// ── Build SignerInfo ──────────────────────────────────────────────────────
	si := signerInfo{
		Version: 1,
		IssuerAndSerial: issuerAndSerialNumber{
			Issuer:       asn1.RawValue{FullBytes: s.passCert.RawIssuer},
			SerialNumber: s.passCert.SerialNumber,
		},
		DigestAlgorithm:    algorithmIdentifier{Algorithm: oidSHA1},
		AuthenticatedAttrs: authAttrs,
		DigestEncAlgorithm: algorithmIdentifier{Algorithm: oidRSASHA256},
		EncryptedDigest:    sig,
	}

	// ── Embed certificates ────────────────────────────────────────────────────
	certBytes := append(s.passCert.Raw, s.wwdrCert.Raw...)
	certRaw := asn1.RawValue{
		Class:       asn1.ClassContextSpecific,
		Tag:         0,
		IsCompound:  true,
		Bytes:       certBytes,
	}

	// ── Build SignedData ──────────────────────────────────────────────────────
	sd := signedData{
		Version:          1,
		DigestAlgorithms: []algorithmIdentifier{{Algorithm: oidSHA1}},
		ContentInfo: asn1.RawValue{
			FullBytes: func() []byte {
				b, _ := asn1.Marshal(asn1.ObjectIdentifier(oidData))
				return b
			}(),
		},
		Certificates: certRaw,
		SignerInfos:  []signerInfo{si},
	}

	sdEncoded, err := asn1.Marshal(sd)
	if err != nil {
		return nil, fmt.Errorf("signed data marshal: %w", err)
	}

	// ── Wrap in ContentInfo (OID + EXPLICIT [0] SignedData) ───────────────────
	contentInfo, err := asn1.Marshal(struct {
		ContentType asn1.ObjectIdentifier
		Content     asn1.RawValue `asn1:"explicit,tag:0"`
	}{
		ContentType: oidSignedData,
		Content:     asn1.RawValue{FullBytes: sdEncoded},
	})
	if err != nil {
		return nil, fmt.Errorf("content info marshal: %w", err)
	}

	return contentInfo, nil
}

// ─── Minimal 1×1 transparent PNG ─────────────────────────────────────────────
// Used as a placeholder when no icon/logo assets are provided.
func minimalPNG() []byte {
	// A valid 1×1 transparent PNG (67 bytes)
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9c, 0x62, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
		0x42, 0x60, 0x82,
	}
}
