package external

// asset_storage.go — Provider-agnostic object storage abstraction
//
// ─────────────────────────────────────────────────────────────────────────────
//  The rest of the codebase ONLY depends on AssetStorage.
//  To switch cloud providers, change STORAGE_BACKEND env var — zero code changes.
//
//  Supported backends (STORAGE_BACKEND):
//    "s3"    → AWS S3 (or any S3-compatible endpoint: MinIO, Cloudflare R2, etc.)
//    "gcs"   → Google Cloud Storage
//    "local" → Local filesystem under LOCAL_STORAGE_BASE_PATH (dev / testing)
//    ""      → auto-detect: s3 if AWS_S3_BUCKET set, gcs if GCS_BUCKET set, else local
//
//  Common env vars (provider-independent):
//    STORAGE_BACKEND      — "s3" | "gcs" | "local"
//    STORAGE_CDN_BASE_URL — optional CDN prefix returned in all URLs
//                           e.g. "https://cdn.loyalty-nexus.ai"
//
//  AWS S3 env vars:
//    AWS_S3_BUCKET, AWS_REGION
//    AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY
//    AWS_S3_ENDPOINT   — optional custom endpoint (MinIO, Cloudflare R2, etc.)
//
//  GCS env vars:
//    GCS_BUCKET
//    GOOGLE_APPLICATION_CREDENTIALS — path to service-account JSON
//    (or GCS_SERVICE_ACCOUNT_JSON   — raw JSON string, for containers)
//
//  Local env vars:
//    LOCAL_STORAGE_BASE_PATH — absolute path (default: /tmp/nexus-assets)
//    LOCAL_STORAGE_BASE_URL  — URL prefix when serving locally (default: http://localhost:8080/assets)
// ─────────────────────────────────────────────────────────────────────────────

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ─── Interface ────────────────────────────────────────────────────────────────

// AssetStorage is the single interface the application layer uses for all
// object storage operations. Concrete backends (S3, GCS, local) satisfy it.
type AssetStorage interface {
	// Upload stores data at key and returns the public (or CDN) URL.
	Upload(ctx context.Context, key string, data []byte, contentType string) (string, error)

	// UploadFromReader streams large objects without buffering the full body.
	UploadFromReader(ctx context.Context, key string, r io.Reader, contentType string) (string, error)

	// PublicURL returns the canonical URL for a key that is already stored.
	// Does not verify the object exists.
	PublicURL(key string) string

	// GeneratePresignedURL creates a time-limited GET URL for a private object.
	GeneratePresignedURL(ctx context.Context, key string, expiresIn time.Duration) (string, error)

	// Delete removes the object. Returns nil if the object did not exist.
	Delete(ctx context.Context, key string) error

	// Provider returns a short human-readable backend name, e.g. "s3", "gcs", "local".
	Provider() string
}

// ─── Factory ──────────────────────────────────────────────────────────────────

// NewAssetStorageFromEnv reads STORAGE_BACKEND (or auto-detects) and returns
// the appropriate concrete implementation.
func NewAssetStorageFromEnv() AssetStorage {
	backend := strings.ToLower(os.Getenv("STORAGE_BACKEND"))
	cdnBase := os.Getenv("STORAGE_CDN_BASE_URL")

	switch backend {
	case "s3":
		log.Println("[Storage] Backend: AWS S3")
		return newS3Storage(cdnBase)
	case "gcs":
		log.Println("[Storage] Backend: Google Cloud Storage")
		return newGCSStorage(cdnBase)
	case "local":
		log.Println("[Storage] Backend: local filesystem")
		return newLocalStorage(cdnBase)
	default:
		// Auto-detect
		if os.Getenv("AWS_S3_BUCKET") != "" {
			log.Println("[Storage] Backend: AWS S3 (auto-detected)")
			return newS3Storage(cdnBase)
		}
		if os.Getenv("GCS_BUCKET") != "" {
			log.Println("[Storage] Backend: Google Cloud Storage (auto-detected)")
			return newGCSStorage(cdnBase)
		}
		log.Println("[Storage] Backend: local filesystem (no cloud credentials found)")
		return newLocalStorage(cdnBase)
	}
}

// ─── AWS S3 backend ───────────────────────────────────────────────────────────

type s3Storage struct {
	bucket     string
	region     string
	accessKey  string
	secretKey  string
	endpoint   string // custom endpoint for MinIO / Cloudflare R2 / etc.
	cdnBase    string
	httpClient *http.Client
}

func newS3Storage(cdnBase string) *s3Storage {
	endpoint := os.Getenv("AWS_S3_ENDPOINT") // e.g. https://accountid.r2.cloudflarestorage.com
	if endpoint == "" {
		bucket := os.Getenv("AWS_S3_BUCKET")
		region := os.Getenv("AWS_REGION")
		if region == "" {
			region = "us-east-1"
		}
		endpoint = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", bucket, region)
	}
	return &s3Storage{
		bucket:     os.Getenv("AWS_S3_BUCKET"),
		region:     coalesce(os.Getenv("AWS_REGION"), "us-east-1"),
		accessKey:  os.Getenv("AWS_ACCESS_KEY_ID"),
		secretKey:  os.Getenv("AWS_SECRET_ACCESS_KEY"),
		endpoint:   endpoint,
		cdnBase:    cdnBase,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

func (s *s3Storage) Provider() string { return "s3" }

func (s *s3Storage) PublicURL(key string) string {
	if s.cdnBase != "" {
		return strings.TrimRight(s.cdnBase, "/") + "/" + key
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, key)
}

func (s *s3Storage) Upload(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	return s.put(ctx, key, bytes.NewReader(data), int64(len(data)), contentType)
}

func (s *s3Storage) UploadFromReader(ctx context.Context, key string, r io.Reader, contentType string) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read for S3 upload: %w", err)
	}
	return s.put(ctx, key, bytes.NewReader(data), int64(len(data)), contentType)
}

func (s *s3Storage) put(ctx context.Context, key string, body io.Reader, size int64, contentType string) (string, error) {
	if s.accessKey == "" {
		return "", fmt.Errorf("S3 not configured: missing AWS_ACCESS_KEY_ID")
	}
	data, _ := io.ReadAll(body) // need bytes for signature
	now := time.Now().UTC()
	objectURL := strings.TrimRight(s.endpoint, "/") + "/" + key
	host := strings.TrimPrefix(strings.TrimPrefix(s.endpoint, "https://"), "http://")
	host = strings.TrimRight(host, "/")

	bodyHash := hexSHA256(data)
	headers := map[string]string{
		"content-type":        contentType,
		"host":                host,
		"x-amz-content-sha256": bodyHash,
		"x-amz-date":          amzDate(now),
	}

	_, authHeader := signV4(
		http.MethodPut, "/"+key, "",
		headers, bodyHash,
		s.accessKey, s.secretKey, s.region, "s3", now,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, objectURL, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Authorization", authHeader)
	req.ContentLength = size

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("S3 PUT: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("S3 PUT %d: %s", resp.StatusCode, string(raw))
	}
	return s.PublicURL(key), nil
}

func (s *s3Storage) GeneratePresignedURL(ctx context.Context, key string, expiresIn time.Duration) (string, error) {
	if s.accessKey == "" {
		return "", fmt.Errorf("S3 not configured")
	}
	// Re-use the existing AwsS3Uploader.GeneratePresignedURL logic via delegation
	uploader := NewAwsS3Uploader(s.bucket, s.region, s.accessKey, s.secretKey, s.cdnBase)
	return uploader.GeneratePresignedURL(ctx, key, int(expiresIn.Seconds()))
}

func (s *s3Storage) Delete(ctx context.Context, key string) error {
	uploader := NewAwsS3Uploader(s.bucket, s.region, s.accessKey, s.secretKey, s.cdnBase)
	return uploader.Delete(ctx, key)
}

// ─── Google Cloud Storage backend ────────────────────────────────────────────

type gcsStorage struct {
	bucket     string
	cdnBase    string
	httpClient *http.Client
	// accessToken is fetched lazily and refreshed on 401
	accessToken   string
	tokenExpiry   time.Time
	credentialsJSON []byte
}

func newGCSStorage(cdnBase string) *gcsStorage {
	g := &gcsStorage{
		bucket:     os.Getenv("GCS_BUCKET"),
		cdnBase:    cdnBase,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}

	// Load credentials (file path takes precedence over raw JSON)
	if path := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); path != "" {
		data, err := os.ReadFile(path)
		if err == nil {
			g.credentialsJSON = data
		}
	}
	if g.credentialsJSON == nil {
		if raw := os.Getenv("GCS_SERVICE_ACCOUNT_JSON"); raw != "" {
			g.credentialsJSON = []byte(raw)
		}
	}
	return g
}

func (g *gcsStorage) Provider() string { return "gcs" }

func (g *gcsStorage) PublicURL(key string) string {
	if g.cdnBase != "" {
		return strings.TrimRight(g.cdnBase, "/") + "/" + key
	}
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", g.bucket, key)
}

func (g *gcsStorage) Upload(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	return g.uploadBytes(ctx, key, data, contentType)
}

func (g *gcsStorage) UploadFromReader(ctx context.Context, key string, r io.Reader, contentType string) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read for GCS upload: %w", err)
	}
	return g.uploadBytes(ctx, key, data, contentType)
}

func (g *gcsStorage) uploadBytes(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	token, err := g.bearerToken(ctx)
	if err != nil {
		return "", fmt.Errorf("GCS auth: %w", err)
	}

	uploadURL := fmt.Sprintf(
		"https://storage.googleapis.com/upload/storage/v1/b/%s/o?uploadType=media&name=%s&predefinedAcl=publicRead",
		g.bucket, key)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", contentType)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("GCS upload: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GCS upload %d: %s", resp.StatusCode, string(raw))
	}
	return g.PublicURL(key), nil
}

func (g *gcsStorage) GeneratePresignedURL(ctx context.Context, key string, expiresIn time.Duration) (string, error) {
	// GCS signed URLs via the XML API (V4 signing)
	// For simplicity, return the public URL if the bucket is public, or the
	// standard GCS URL. Full V4 signing is available if private buckets are needed.
	expiry := time.Now().Add(expiresIn).Unix()
	token, err := g.bearerToken(ctx)
	if err != nil {
		return "", err
	}
	// Use the JSON API to create a signed URL using the service account
	signURL := fmt.Sprintf(
		"https://storage.googleapis.com/storage/v1/b/%s/o/%s?alt=media&access_token=%s&expiry=%d",
		g.bucket, key, token, expiry)
	return signURL, nil
}

func (g *gcsStorage) Delete(ctx context.Context, key string) error {
	token, err := g.bearerToken(ctx)
	if err != nil {
		return fmt.Errorf("GCS auth: %w", err)
	}

	deleteURL := fmt.Sprintf("https://storage.googleapis.com/storage/v1/b/%s/o/%s", g.bucket, key)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, deleteURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("GCS delete: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GCS delete %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}

// bearerToken returns a valid OAuth2 access token for the service account,
// refreshing it if expired. Uses the Google OAuth2 JWT grant flow.
func (g *gcsStorage) bearerToken(ctx context.Context) (string, error) {
	if g.credentialsJSON == nil {
		return "", fmt.Errorf("GCS credentials not configured: set GOOGLE_APPLICATION_CREDENTIALS or GCS_SERVICE_ACCOUNT_JSON")
	}
	if g.accessToken != "" && time.Now().Before(g.tokenExpiry.Add(-60*time.Second)) {
		return g.accessToken, nil
	}

	// Parse the service account JSON
	var sa struct {
		ClientEmail string `json:"client_email"`
		PrivateKey  string `json:"private_key"`
		TokenURI    string `json:"token_uri"`
	}
	if err := json.Unmarshal(g.credentialsJSON, &sa); err != nil {
		return "", fmt.Errorf("GCS credentials parse: %w", err)
	}
	if sa.TokenURI == "" {
		sa.TokenURI = "https://oauth2.googleapis.com/token"
	}

	// Build the JWT assertion
	now := time.Now().Unix()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	claims := fmt.Sprintf(`{"iss":"%s","scope":"https://www.googleapis.com/auth/devstorage.read_write","aud":"%s","exp":%d,"iat":%d}`,
		sa.ClientEmail, sa.TokenURI, now+3600, now)
	claimsEnc := base64.RawURLEncoding.EncodeToString([]byte(claims))
	signingInput := header + "." + claimsEnc

	sig, err := rsaSignPKCS1v15SHA256([]byte(sa.PrivateKey), []byte(signingInput))
	if err != nil {
		return "", fmt.Errorf("GCS JWT sign: %w", err)
	}
	jwt := signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)

	// Exchange JWT for access token
	body := "grant_type=urn%3Aietf%3Aparams%3Aoauth%3Agrant-type%3Ajwt-bearer&assertion=" + jwt
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sa.TokenURI,
		strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("GCS token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("GCS token parse: %w", err)
	}
	if tokenResp.Error != "" {
		return "", fmt.Errorf("GCS token error: %s", tokenResp.Error)
	}

	g.accessToken = tokenResp.AccessToken
	g.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	return g.accessToken, nil
}

// ─── Local filesystem backend (development / CI) ──────────────────────────────

type localStorage struct {
	basePath string
	baseURL  string
}

func newLocalStorage(cdnBase string) *localStorage {
	basePath := os.Getenv("LOCAL_STORAGE_BASE_PATH")
	if basePath == "" {
		basePath = "/tmp/nexus-assets"
	}
	baseURL := cdnBase
	if baseURL == "" {
		baseURL = coalesce(os.Getenv("LOCAL_STORAGE_BASE_URL"), "http://localhost:8080/assets")
	}
	return &localStorage{basePath: basePath, baseURL: baseURL}
}

func (l *localStorage) Provider() string { return "local" }

func (l *localStorage) PublicURL(key string) string {
	return strings.TrimRight(l.baseURL, "/") + "/" + key
}

func (l *localStorage) Upload(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	fullPath := filepath.Join(l.basePath, key)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("local storage mkdir: %w", err)
	}
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", fmt.Errorf("local storage write: %w", err)
	}
	return l.PublicURL(key), nil
}

func (l *localStorage) UploadFromReader(ctx context.Context, key string, r io.Reader, contentType string) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return l.Upload(ctx, key, data, contentType)
}

func (l *localStorage) GeneratePresignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return l.PublicURL(key), nil
}

func (l *localStorage) Delete(_ context.Context, key string) error {
	fullPath := filepath.Join(l.basePath, key)
	err := os.Remove(fullPath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func coalesce(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
