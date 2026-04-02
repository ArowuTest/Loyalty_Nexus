package external

// text_extractor.go — Extracts readable text from uploaded files and URLs
//
// Supported sources:
//   - PDF files (via pdfcpu)
//   - Plain text / Markdown files (direct read)
//   - Web URLs (HTTP fetch + HTML strip)
//   - Google Drive share links (converted to export URL)
//   - Google Docs / Sheets / Slides share links (export as plain text)
//
// Usage:
//   extractor := NewTextExtractor()
//   text, err := extractor.ExtractFromURL(ctx, "https://drive.google.com/file/d/xxx/view")
//   text, err := extractor.ExtractFromURL(ctx, "https://s3.amazonaws.com/bucket/uploads/file.pdf")

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

const (
	maxExtractBytes  = 500_000  // 500 KB of raw content max
	maxOutputChars   = 12_000   // ~12K chars injected into prompt (fits in context window)
	httpTimeout      = 20 * time.Second
)

// TextExtractor extracts readable text from various file and URL sources.
type TextExtractor struct {
	client *http.Client
}

// NewTextExtractor creates a TextExtractor with a sensible HTTP timeout.
func NewTextExtractor() *TextExtractor {
	return &TextExtractor{
		client: &http.Client{
			Timeout: httpTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

// ExtractFromURL fetches and extracts text from a URL.
// It handles:
//   - Google Drive file share links  → converted to direct download
//   - Google Docs/Sheets/Slides      → exported as plain text
//   - S3/CDN URLs ending in .pdf     → parsed with pdfcpu
//   - S3/CDN URLs ending in .txt/.md → read directly
//   - Generic web pages              → HTML stripped to plain text
func (e *TextExtractor) ExtractFromURL(ctx context.Context, rawURL string) (string, error) {
	if rawURL == "" {
		return "", fmt.Errorf("empty URL")
	}

	// Normalise Google Drive / Docs URLs first
	normalised, driveType := normaliseGoogleURL(rawURL)

	switch driveType {
	case "drive_file":
		// Google Drive file → download directly
		return e.fetchAndExtract(ctx, normalised)
	case "docs", "sheets", "slides":
		// Google Workspace docs → export as plain text
		return e.fetchText(ctx, normalised)
	default:
		return e.fetchAndExtract(ctx, normalised)
	}
}

// ExtractFromBytes extracts text from raw file bytes given a content type.
func (e *TextExtractor) ExtractFromBytes(ctx context.Context, data []byte, contentType string) (string, error) {
	ct := strings.ToLower(contentType)
	switch {
	case strings.Contains(ct, "pdf"):
		return extractPDFBytes(data)
	case strings.Contains(ct, "text"):
		return truncate(string(data)), nil
	default:
		// Try treating as text
		if utf8.Valid(data) {
			return truncate(string(data)), nil
		}
		return "", fmt.Errorf("unsupported binary content type: %s", contentType)
	}
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

func (e *TextExtractor) fetchAndExtract(ctx context.Context, targetURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "NexusAI/1.0 (document reader)")

	resp, err := e.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d fetching URL", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, maxExtractBytes)
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	ct := resp.Header.Get("Content-Type")
	return e.ExtractFromBytes(ctx, data, ct)
}

func (e *TextExtractor) fetchText(ctx context.Context, targetURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "NexusAI/1.0 (document reader)")

	resp, err := e.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d fetching document", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, maxExtractBytes)
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	ct := resp.Header.Get("Content-Type")
	if strings.Contains(strings.ToLower(ct), "html") {
		return truncate(stripHTML(string(data))), nil
	}
	return truncate(string(data)), nil
}

// normaliseGoogleURL converts Google Drive/Docs share URLs to direct download URLs.
// Returns (normalisedURL, type) where type is one of:
//   "drive_file" — Google Drive file (PDF, DOCX, etc.)
//   "docs"       — Google Docs → export as text
//   "sheets"     — Google Sheets → export as CSV
//   "slides"     — Google Slides → export as text
//   ""           — not a Google URL, return as-is
func normaliseGoogleURL(rawURL string) (string, string) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL, ""
	}

	host := strings.ToLower(u.Host)

	// ── Google Drive file share: https://drive.google.com/file/d/{ID}/view
	if strings.Contains(host, "drive.google.com") {
		// Extract file ID from /file/d/{ID}/...
		re := regexp.MustCompile(`/file/d/([a-zA-Z0-9_-]+)`)
		if m := re.FindStringSubmatch(u.Path); len(m) > 1 {
			fileID := m[1]
			downloadURL := fmt.Sprintf("https://drive.google.com/uc?export=download&id=%s", fileID)
			return downloadURL, "drive_file"
		}
		// Shared folder or other Drive URL — return as-is
		return rawURL, ""
	}

	// ── Google Docs: https://docs.google.com/document/d/{ID}/edit
	if strings.Contains(host, "docs.google.com") {
		re := regexp.MustCompile(`/document/d/([a-zA-Z0-9_-]+)`)
		if m := re.FindStringSubmatch(u.Path); len(m) > 1 {
			docID := m[1]
			exportURL := fmt.Sprintf("https://docs.google.com/document/d/%s/export?format=txt", docID)
			return exportURL, "docs"
		}

		re = regexp.MustCompile(`/spreadsheets/d/([a-zA-Z0-9_-]+)`)
		if m := re.FindStringSubmatch(u.Path); len(m) > 1 {
			sheetID := m[1]
			exportURL := fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s/export?format=csv", sheetID)
			return exportURL, "sheets"
		}

		re = regexp.MustCompile(`/presentation/d/([a-zA-Z0-9_-]+)`)
		if m := re.FindStringSubmatch(u.Path); len(m) > 1 {
			slideID := m[1]
			exportURL := fmt.Sprintf("https://docs.google.com/presentation/d/%s/export/txt", slideID)
			return exportURL, "slides"
		}
	}

	return rawURL, ""
}

// extractPDFBytes uses pdfcpu to extract text from PDF bytes.
// It writes content to a temp directory and reads back the extracted text files.
func extractPDFBytes(data []byte) (string, error) {
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	// Write PDF to a temp file
	tmpDir, err := os.MkdirTemp("", "nexus-pdf-*")
	if err != nil {
		return "[PDF received — could not create temp directory for extraction.]", nil
	}
	defer os.RemoveAll(tmpDir)

	pdfPath := filepath.Join(tmpDir, "input.pdf")
	if writeErr := os.WriteFile(pdfPath, data, 0600); writeErr != nil {
		return "[PDF received — could not write temp file for extraction.]", nil
	}

	outDir := filepath.Join(tmpDir, "out")
	if mkErr := os.MkdirAll(outDir, 0700); mkErr != nil {
		return "[PDF received — could not create output directory.]", nil
	}

	// Extract content streams to outDir
	if extractErr := api.ExtractContentFile(pdfPath, outDir, nil, conf); extractErr != nil {
		// Fallback: return a message indicating PDF was received but text couldn't be extracted
		return "[PDF received — text extraction encountered an issue. The document may be image-based or encrypted.]", nil
	}

	// Read all extracted content files
	var sb strings.Builder
	entries, readErr := os.ReadDir(outDir)
	if readErr != nil {
		return "[PDF received — could not read extracted content.]", nil
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, fileErr := os.ReadFile(filepath.Join(outDir, entry.Name()))
		if fileErr != nil {
			continue
		}
		sb.Write(content)
		sb.WriteString("\n")
	}

	result := sb.String()
	if strings.TrimSpace(result) == "" {
		return "[PDF received — this appears to be an image-based PDF. Text extraction is not available for scanned documents.]", nil
	}
	return truncate(result), nil
}

// stripHTML removes HTML tags and decodes common entities.
func stripHTML(html string) string {
	// Remove script and style blocks entirely
	re := regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	html = re.ReplaceAllString(html, " ")

	// Remove all remaining tags
	re2 := regexp.MustCompile(`<[^>]+>`)
	html = re2.ReplaceAllString(html, " ")

	// Decode common HTML entities
	replacer := strings.NewReplacer(
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&quot;", `"`,
		"&#39;", "'",
		"&nbsp;", " ",
		"&mdash;", "—",
		"&ndash;", "–",
		"&hellip;", "…",
	)
	html = replacer.Replace(html)

	// Collapse whitespace
	re3 := regexp.MustCompile(`\s{2,}`)
	html = re3.ReplaceAllString(html, "\n")

	return strings.TrimSpace(html)
}

// truncate limits output to maxOutputChars to avoid blowing up the LLM context window.
func truncate(s string) string {
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) <= maxOutputChars {
		return s
	}
	return string(runes[:maxOutputChars]) + "\n\n[... content truncated to fit context window ...]"
}
