package external

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

// ─── AwsS3Uploader ────────────────────────────────────────────────────────

// AwsS3Uploader implements S3Uploader using pure-Go AWS Signature V4 signing.
// It requires no third-party SDK — only the standard library.
type AwsS3Uploader struct {
	BucketName      string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	// CDNBaseURL, when non-empty, is prepended to the object key to form the
	// returned URL (e.g. "https://cdn.example.com").
	CDNBaseURL string
	client     *http.Client
}

// NewAwsS3Uploader constructs an uploader with the supplied credentials.
func NewAwsS3Uploader(bucket, region, accessKey, secretKey, cdnBase string) *AwsS3Uploader {
	return &AwsS3Uploader{
		BucketName:      bucket,
		Region:          region,
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
		CDNBaseURL:      cdnBase,
		client:          &http.Client{Timeout: 60 * time.Second},
	}
}

// NewS3UploaderFromEnv reads AWS credentials from well-known environment
// variables and returns a configured uploader.
func NewS3UploaderFromEnv() *AwsS3Uploader {
	return NewAwsS3Uploader(
		os.Getenv("AWS_S3_BUCKET"),
		os.Getenv("AWS_REGION"),
		os.Getenv("AWS_ACCESS_KEY_ID"),
		os.Getenv("AWS_SECRET_ACCESS_KEY"),
		os.Getenv("AWS_CDN_BASE_URL"),
	)
}

// Upload puts data at key in the configured S3 bucket and returns the public URL.
func (u *AwsS3Uploader) Upload(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	if u.AccessKeyID == "" {
		return "", fmt.Errorf("S3 not configured")
	}

	now := time.Now().UTC()
	endpoint := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", u.BucketName, u.Region, key)

	bodyHash := hexSHA256(data)

	headers := map[string]string{
		"content-type":        contentType,
		"host":                fmt.Sprintf("%s.s3.%s.amazonaws.com", u.BucketName, u.Region),
		"x-amz-content-sha256": bodyHash,
		"x-amz-date":          amzDate(now),
	}

	signature, authHeader := signV4(
		http.MethodPut, "/"+key, "",
		headers, bodyHash,
		u.AccessKeyID, u.SecretAccessKey, u.Region, "s3", now,
	)
	_ = signature

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Authorization", authHeader)

	resp, err := u.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("S3 upload: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated &&
		resp.StatusCode != http.StatusNoContent {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("S3 upload returned %d: %s", resp.StatusCode, string(raw))
	}

	return u.publicURL(key), nil
}

// GeneratePresignedURL creates a pre-signed GET URL valid for expiresInSeconds.
func (u *AwsS3Uploader) GeneratePresignedURL(ctx context.Context, key string, expiresInSeconds int) (string, error) {
	if u.AccessKeyID == "" {
		return "", fmt.Errorf("S3 not configured")
	}

	now := time.Now().UTC()
	datestamp := now.Format("20060102")
	amzDateVal := amzDate(now)
	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", datestamp, u.Region)
	credential := u.AccessKeyID + "/" + credentialScope

	queryParams := url.Values{}
	queryParams.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	queryParams.Set("X-Amz-Credential", credential)
	queryParams.Set("X-Amz-Date", amzDateVal)
	queryParams.Set("X-Amz-Expires", fmt.Sprintf("%d", expiresInSeconds))
	queryParams.Set("X-Amz-SignedHeaders", "host")

	host := fmt.Sprintf("%s.s3.%s.amazonaws.com", u.BucketName, u.Region)
	canonicalHeaders := "host:" + host + "\n"
	signedHeaders := "host"

	canonicalRequest := strings.Join([]string{
		http.MethodGet,
		"/" + key,
		queryParams.Encode(),
		canonicalHeaders,
		signedHeaders,
		"UNSIGNED-PAYLOAD",
	}, "\n")

	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDateVal,
		credentialScope,
		hexSHA256([]byte(canonicalRequest)),
	}, "\n")

	signingKey := deriveSigningKey(u.SecretAccessKey, datestamp, u.Region, "s3")
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	queryParams.Set("X-Amz-Signature", signature)

	return fmt.Sprintf("https://%s/%s?%s", host, key, queryParams.Encode()), nil
}

// Delete removes an object from S3.
func (u *AwsS3Uploader) Delete(ctx context.Context, key string) error {
	if u.AccessKeyID == "" {
		return fmt.Errorf("S3 not configured")
	}

	now := time.Now().UTC()
	endpoint := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", u.BucketName, u.Region, key)

	bodyHash := hexSHA256([]byte{})
	headers := map[string]string{
		"host":                fmt.Sprintf("%s.s3.%s.amazonaws.com", u.BucketName, u.Region),
		"x-amz-content-sha256": bodyHash,
		"x-amz-date":          amzDate(now),
	}

	_, authHeader := signV4(
		http.MethodDelete, "/"+key, "",
		headers, bodyHash,
		u.AccessKeyID, u.SecretAccessKey, u.Region, "s3", now,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Authorization", authHeader)

	resp, err := u.client.Do(req)
	if err != nil {
		return fmt.Errorf("S3 delete: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("S3 delete returned %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}

// publicURL returns the CDN URL if configured, otherwise the S3 object URL.
func (u *AwsS3Uploader) publicURL(key string) string {
	if u.CDNBaseURL != "" {
		return strings.TrimRight(u.CDNBaseURL, "/") + "/" + key
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", u.BucketName, u.Region, key)
}

// ─── AWS Signature V4 helpers ─────────────────────────────────────────────

// signV4 computes the AWS Signature V4 Authorization header value.
// It returns (signature, authorizationHeaderValue).
func signV4(
	method, path, queryString string,
	headers map[string]string,
	payloadHash string,
	accessKeyID, secretAccessKey, region, service string,
	now time.Time,
) (string, string) {
	datestamp := now.Format("20060102")
	amzDateVal := amzDate(now)
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", datestamp, region, service)

	// Build sorted canonical headers
	headerKeys := make([]string, 0, len(headers))
	for k := range headers {
		headerKeys = append(headerKeys, strings.ToLower(k))
	}
	sort.Strings(headerKeys)

	// Rebuild with lower-cased keys
	lowerHeaders := make(map[string]string, len(headers))
	for k, v := range headers {
		lowerHeaders[strings.ToLower(k)] = v
	}

	var canonHeadersBuf strings.Builder
	for _, k := range headerKeys {
		canonHeadersBuf.WriteString(k)
		canonHeadersBuf.WriteString(":")
		canonHeadersBuf.WriteString(strings.TrimSpace(lowerHeaders[k]))
		canonHeadersBuf.WriteString("\n")
	}
	canonicalHeaders := canonHeadersBuf.String()
	signedHeaders := strings.Join(headerKeys, ";")

	canonicalRequest := strings.Join([]string{
		method,
		path,
		queryString,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDateVal,
		credentialScope,
		hexSHA256([]byte(canonicalRequest)),
	}, "\n")

	signingKey := deriveSigningKey(secretAccessKey, datestamp, region, service)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		accessKeyID, credentialScope, signedHeaders, signature)

	return signature, authHeader
}

// deriveSigningKey computes the HMAC signing key per the V4 derivation spec.
func deriveSigningKey(secretKey, datestamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretKey), []byte(datestamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	return hmacSHA256(kService, []byte("aws4_request"))
}

// hmacSHA256 returns the HMAC-SHA256 of msg using key.
func hmacSHA256(key, msg []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(msg)
	return h.Sum(nil)
}

// hexSHA256 returns the lowercase hex-encoded SHA-256 hash of data.
func hexSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// amzDate formats t as the ISO 8601 basic format required by SigV4.
func amzDate(t time.Time) string {
	return t.Format("20060102T150405Z")
}
