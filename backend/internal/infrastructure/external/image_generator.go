package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

// ─── ProductionImageGenerator ─────────────────────────────────────────────

// ProductionImageGenerator implements ImageGenerator using HuggingFace
// Inference API, FAL.AI, and an optional rembg background-removal service.
// All HTTP calls propagate the caller's context.
type ProductionImageGenerator struct {
	FALAPIKey         string
	HuggingFaceAPIKey string
	RembgServiceURL   string // e.g. http://rembg-service:5000  (empty → Photoroom)
	S3Uploader        S3Uploader
	client            *http.Client
}

// NewProductionImageGenerator returns a ready-to-use generator.
func NewProductionImageGenerator(falKey, hfKey, rembgURL string, uploader S3Uploader) *ProductionImageGenerator {
	return &ProductionImageGenerator{
		FALAPIKey:         falKey,
		HuggingFaceAPIKey: hfKey,
		RembgServiceURL:   rembgURL,
		S3Uploader:        uploader,
		client:            &http.Client{Timeout: 90 * time.Second},
	}
}

// GenerateImage dispatches to HuggingFace (hf-flux-schnell) or FAL.AI
// (fal-flux-dev / default). On HuggingFace failure it retries via FAL.AI.
func (g *ProductionImageGenerator) GenerateImage(ctx context.Context, prompt, model string) (string, error) {
	if model == "hf-flux-schnell" {
		url, err := g.generateViaHuggingFace(ctx, prompt)
		if err == nil {
			return url, nil
		}
		// fallback to FAL.AI
	}
	return g.generateViaFAL(ctx, prompt)
}

func (g *ProductionImageGenerator) generateViaHuggingFace(ctx context.Context, prompt string) (string, error) {
	body, _ := json.Marshal(map[string]string{"inputs": prompt})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api-inference.huggingface.co/models/black-forest-labs/FLUX.1-schnell",
		bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+g.HuggingFaceAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("huggingface request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("huggingface returned %d", resp.StatusCode)
	}
	imgData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read HF response: %w", err)
	}
	if g.S3Uploader == nil {
		return "", fmt.Errorf("S3 uploader not configured")
	}
	key := fmt.Sprintf("generated/%s.png", uuid.New().String())
	return g.S3Uploader.Upload(ctx, key, imgData, "image/png")
}

func (g *ProductionImageGenerator) generateViaFAL(ctx context.Context, prompt string) (string, error) {
	payload := map[string]interface{}{
		"prompt":        prompt,
		"image_size":    "square_hd",
		"num_images":    1,
		"output_format": "jpeg",
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://fal.run/fal-ai/flux/dev", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Key "+g.FALAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fal.ai request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("fal.ai returned %d: %s", resp.StatusCode, string(raw))
	}

	var result struct {
		Images []struct {
			URL string `json:"url"`
		} `json:"images"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parse FAL response: %w", err)
	}
	if len(result.Images) == 0 || result.Images[0].URL == "" {
		return "", fmt.Errorf("fal.ai: no images in response")
	}
	return result.Images[0].URL, nil
}

// RemoveBackground removes the background from an image. Uses the self-hosted
// rembg service when RembgServiceURL is set; otherwise falls back to Photoroom.
func (g *ProductionImageGenerator) RemoveBackground(ctx context.Context, imageURL string) (string, error) {
	if g.RembgServiceURL != "" {
		return g.removeViaRembg(ctx, imageURL)
	}
	return g.removeViaPhotoroom(ctx, imageURL)
}

func (g *ProductionImageGenerator) removeViaRembg(ctx context.Context, imageURL string) (string, error) {
	body, _ := json.Marshal(map[string]string{"url": imageURL})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		g.RembgServiceURL+"/remove", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("rembg request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("rembg returned %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read rembg body: %w", err)
	}

	// Try JSON {"result_url": "..."}
	var jsonResp struct {
		ResultURL string `json:"result_url"`
	}
	if jsonErr := json.Unmarshal(data, &jsonResp); jsonErr == nil && jsonResp.ResultURL != "" {
		return jsonResp.ResultURL, nil
	}

	// Raw binary PNG — upload to S3
	if g.S3Uploader == nil {
		return "", fmt.Errorf("rembg: raw binary returned but S3 not configured")
	}
	key := fmt.Sprintf("nobg/%s.png", uuid.New().String())
	return g.S3Uploader.Upload(ctx, key, data, "image/png")
}

func (g *ProductionImageGenerator) removeViaPhotoroom(ctx context.Context, imageURL string) (string, error) {
	var bodyBuf bytes.Buffer
	mw := multipart.NewWriter(&bodyBuf)

	fw, err := mw.CreateFormField("image_url")
	if err != nil {
		return "", err
	}
	if _, err := fmt.Fprint(fw, imageURL); err != nil {
		return "", err
	}
	if err := mw.Close(); err != nil {
		return "", fmt.Errorf("photoroom multipart close: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://sdk.photoroom.com/v1/segment", &bodyBuf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("x-api-key", os.Getenv("PHOTOROOM_API_KEY"))

	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("photoroom request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("photoroom returned %d", resp.StatusCode)
	}

	imgData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if g.S3Uploader == nil {
		return "", fmt.Errorf("photoroom: S3 not configured")
	}
	key := fmt.Sprintf("nobg/%s.png", uuid.New().String())
	return g.S3Uploader.Upload(ctx, key, imgData, "image/png")
}

// AnimateImage converts a still image to a short video using FAL.AI Kling.
// Falls back to LTX-Video on failure.
func (g *ProductionImageGenerator) AnimateImage(ctx context.Context, imageURL string) (string, error) {
	videoURL, err := g.falVideoRequest(ctx,
		"https://fal.run/fal-ai/kling-video/v1.5/standard/image-to-video", imageURL)
	if err == nil {
		return videoURL, nil
	}
	return g.falVideoRequest(ctx, "https://fal.run/fal-ai/ltx-video", imageURL)
}

func (g *ProductionImageGenerator) falVideoRequest(ctx context.Context, endpoint, imageURL string) (string, error) {
	payload := map[string]interface{}{
		"image_url": imageURL,
		"prompt":    "animate this photo naturally",
		"duration":  "5",
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Key "+g.FALAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fal video request (%s): %w", endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("fal video (%s) returned %d: %s", endpoint, resp.StatusCode, string(raw))
	}

	var result struct {
		Video struct {
			URL string `json:"url"`
		} `json:"video"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parse FAL video response: %w", err)
	}
	if result.Video.URL == "" {
		return "", fmt.Errorf("fal video: empty url in response")
	}
	return result.Video.URL, nil
}

// ─── FalAIAdapter (legacy — kept for call-site compatibility) ─────────────

// FalAIAdapter wraps the FAL.AI FLUX dev endpoint.
type FalAIAdapter struct {
	APIKey string
	client *http.Client
}

// NewFalAIAdapter returns an initialised adapter.
func NewFalAIAdapter(apiKey string) *FalAIAdapter {
	return &FalAIAdapter{APIKey: apiKey, client: &http.Client{Timeout: 90 * time.Second}}
}

// Generate calls FAL.AI flux/dev and returns the first image URL.
func (a *FalAIAdapter) Generate(ctx context.Context, prompt string) (string, error) {
	payload := map[string]interface{}{
		"prompt":        prompt,
		"image_size":    "square_hd",
		"num_images":    1,
		"output_format": "jpeg",
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://fal.run/fal-ai/flux/dev", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Key "+a.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fal.ai request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("fal.ai returned %d: %s", resp.StatusCode, string(raw))
	}

	var result struct {
		Images []struct {
			URL string `json:"url"`
		} `json:"images"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parse FAL response: %w", err)
	}
	if len(result.Images) == 0 || result.Images[0].URL == "" {
		return "", fmt.Errorf("fal.ai: no images in response")
	}
	return result.Images[0].URL, nil
}

// ─── HuggingFaceAdapter (legacy — kept for call-site compatibility) ───────

// HuggingFaceAdapter wraps the HuggingFace Inference API for FLUX.1-schnell.
type HuggingFaceAdapter struct {
	APIKey     string
	S3Uploader S3Uploader
	client     *http.Client
}

// NewHuggingFaceAdapter returns an initialised adapter.
func NewHuggingFaceAdapter(apiKey string, uploader S3Uploader) *HuggingFaceAdapter {
	return &HuggingFaceAdapter{
		APIKey:     apiKey,
		S3Uploader: uploader,
		client:     &http.Client{Timeout: 90 * time.Second},
	}
}

// Generate calls FLUX.1-schnell on HuggingFace, uploads raw PNG to S3, and
// returns the CDN URL.
func (a *HuggingFaceAdapter) Generate(ctx context.Context, prompt string) (string, error) {
	body, _ := json.Marshal(map[string]string{"inputs": prompt})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api-inference.huggingface.co/models/black-forest-labs/FLUX.1-schnell",
		bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+a.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("huggingface request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("huggingface returned %d", resp.StatusCode)
	}
	imgData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read HF response: %w", err)
	}
	if a.S3Uploader == nil {
		return "", fmt.Errorf("huggingface: S3 uploader not configured")
	}
	key := fmt.Sprintf("generated/%s.png", uuid.New().String())
	return a.S3Uploader.Upload(ctx, key, imgData, "image/png")
}
