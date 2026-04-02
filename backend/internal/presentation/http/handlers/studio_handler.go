package handlers

// studio_handler.go — HTTP presentation layer for Nexus Studio (spec §9)
//
// Routes (all authenticated via JWT middleware):
//
//   GET    /api/v1/studio/tools            — list active tools with point costs
//   GET    /api/v1/studio/tools/{slug}     — single tool detail by slug
//   POST   /api/v1/studio/generate         — request a generation job (deducts PulsePoints)
//   GET    /api/v1/studio/generate/{id}    — poll job status
//   GET    /api/v1/studio/gallery          — user's completed gallery (paginated)
//   POST   /api/v1/studio/chat             — Nexus Chat (multi-provider LLM)
//   GET    /api/v1/studio/chat/usage       — daily chat usage count for user
//
// All DB mutations flow exclusively through StudioService (which owns the
// repository + wallet + ledger).  This handler never calls gorm.DB directly.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/infrastructure/config"
	"loyalty-nexus/internal/infrastructure/external"
	"loyalty-nexus/internal/presentation/http/middleware"
)

// StudioHandler handles all Nexus Studio HTTP endpoints.
type StudioHandler struct {
	studioSvc    *services.StudioService
	llmOrch      *external.LLMOrchestrator
	worker       *AsyncStudioWorker
	cfg          *config.ConfigManager
	assetStorage external.AssetStorage // optional: nil when STORAGE_BACKEND not set
}

func NewStudioHandler(
	ss *services.StudioService,
	lo *external.LLMOrchestrator,
	kb *AsyncStudioWorker,
	cfg *config.ConfigManager,
) *StudioHandler {
	return &StudioHandler{studioSvc: ss, llmOrch: lo, worker: kb, cfg: cfg}
}

// SetAssetStorage injects the storage backend (called from main.go after init).
func (h *StudioHandler) SetAssetStorage(s external.AssetStorage) { h.assetStorage = s }

// ─── POST /api/v1/studio/upload ──────────────────────────────────────────────
// Accepts a multipart form upload of an audio or image file (max 20 MB).
// Returns { url: "https://..." } pointing to the stored object so the
// frontend can pass it to the generate endpoint as image_url or prompt.
func (h *StudioHandler) UploadAsset(w http.ResponseWriter, r *http.Request) {
	if h.assetStorage == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "file upload not configured on this server",
		})
		return
	}

	if err := r.ParseMultipartForm(20 << 20); err != nil { // 20 MB
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file too large (max 20 MB)"})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing file field"})
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	// Allow audio, image, PDF, and plain text uploads
	allowedExts := map[string]string{
		"audio/mpeg":      ".mp3",
		"audio/wav":       ".wav",
		"audio/mp4":       ".m4a",
		"audio/x-m4a":    ".m4a",
		"audio/ogg":       ".ogg",
		"audio/webm":      ".webm",
		"image/jpeg":      ".jpg",
		"image/png":       ".png",
		"image/webp":      ".webp",
		"image/gif":       ".gif",
		"application/pdf": ".pdf",
		"text/plain":      ".txt",
		"text/markdown":   ".md",
	}
	ext, allowed := allowedExts[contentType]
	if !allowed {
		if strings.HasPrefix(contentType, "audio/") {
			parts := strings.SplitN(contentType, "/", 2)
			if len(parts) == 2 {
				ext = "." + strings.ToLower(strings.TrimPrefix(parts[1], "x-"))
				allowed = true
			}
		} else if strings.HasPrefix(contentType, "image/") {
			parts := strings.SplitN(contentType, "/", 2)
			if len(parts) == 2 {
				ext = "." + strings.ToLower(parts[1])
				allowed = true
			}
		}
	}
	if !allowed {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "unsupported file type. Allowed: images (JPG/PNG/WebP/GIF), audio (MP3/WAV/M4A), PDF, TXT",
		})
		return
	}
	if ext == "" {
		ext = ".bin"
	}

	key := "uploads/" + uuid.New().String() + ext
	pubURL, err := h.assetStorage.UploadFromReader(r.Context(), key, file, contentType)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "upload failed: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"url": pubURL, "key": key})
}

// ─── GET /api/v1/studio/tools ─────────────────────────────────────────────────

func (h *StudioHandler) ListTools(w http.ResponseWriter, r *http.Request) {
	tools, err := h.studioSvc.ListActiveTools(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load tools"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"tools": tools, "count": len(tools)})
}

// ─── GET /api/v1/studio/tools/{slug} ─────────────────────────────────────────

func (h *StudioHandler) GetTool(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "slug required"})
		return
	}
	tool, err := h.studioSvc.FindToolBySlug(r.Context(), slug)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "tool not found"})
		return
	}
	writeJSON(w, http.StatusOK, tool)
}

// ─── POST /api/v1/studio/generate ────────────────────────────────────────────

type generateRequest struct {
	ToolID   string `json:"tool_id"`   // UUID of studio_tools row
	ToolSlug string `json:"tool_slug"` // alternative lookup by slug
	Prompt   string `json:"prompt"`

	// Template-specific extra parameters.
	// These are forwarded to the AI provider via the Prompt field after being
	// serialised into a structured prefix or separate payload fields.
	// Frontend sends only the fields relevant to the active ui_template.
	AspectRatio     string                 `json:"aspect_ratio,omitempty"`      // image / video: "1:1", "16:9", "9:16"
	Duration        int                    `json:"duration,omitempty"`          // music (seconds) or video (seconds)
	VoiceID         string                 `json:"voice_id,omitempty"`          // narrate-pro: ElevenLabs voice id
	Language        string                 `json:"language,omitempty"`          // TTS / transcribe language code
	Vocals          *bool                  `json:"vocals,omitempty"`            // music: true = with vocals
	Lyrics          string                 `json:"lyrics,omitempty"`            // music: user-supplied lyrics
	StyleTags       []string               `json:"style_tags,omitempty"`        // image / video: style hints
	NegativePrompt  string                 `json:"negative_prompt,omitempty"`   // image / video: what to avoid
	ImageURL        string                 `json:"image_url,omitempty"`         // image-editor / video-animator: source image (pre-uploaded URL)
	DocumentURL     string                 `json:"document_url,omitempty"`      // FEAT-01: knowledge tools — pre-uploaded PDF/TXT URL for Gemini multimodal analysis
	ExtraParams     map[string]interface{} `json:"extra_params,omitempty"`      // catch-all for future template fields
}

func (h *StudioHandler) Generate(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.ContextUserID).(string)
	userID, err := uuid.Parse(uid)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid user token"})
		return
	}

	var req generateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.Prompt == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "prompt is required"})
		return
	}

	// Resolve tool ID — accept either tool_id (UUID) or tool_slug
	var toolID uuid.UUID
	if req.ToolID != "" {
		toolID, err = uuid.Parse(req.ToolID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid tool_id"})
			return
		}
	} else if req.ToolSlug != "" {
		tool, err := h.studioSvc.FindToolBySlug(r.Context(), req.ToolSlug)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "tool not found: " + req.ToolSlug})
			return
		}
		toolID = tool.ID
	} else {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tool_id or tool_slug required"})
		return
	}

	// Check daily generation quota (from network_configs — zero-hardcoding rule)
	dailyLimit := h.cfg.GetInt("studio_daily_gen_limit", 10)
	count, _ := h.studioSvc.CountUserGenerationsToday(r.Context(), userID)
	if count >= dailyLimit {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{
			"error": "daily generation limit reached",
			"limit": strconv.Itoa(dailyLimit),
			"used":  strconv.Itoa(count),
		})
		return
	}

	// Enrich prompt with structured template params before dispatching.
	// The AI orchestrator receives a single enriched prompt string; extra params
	// are serialised into a structured prefix so the provider can parse them.
	enrichedPrompt := buildEnrichedPrompt(req)

	// Atomic: deduct PulsePoints + create job
	gen, err := h.studioSvc.RequestGeneration(r.Context(), userID, toolID, enrichedPrompt)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Dispatch to background worker (non-blocking — handler returns immediately)
	h.worker.DispatchGeneration(gen, nil)

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"generation_id":   gen.ID,
		"status":          gen.Status,
		"tool_slug":       gen.ToolSlug,
		"points_deducted": gen.PointsDeducted,
		"message":         "Generation queued — you'll be notified when it's ready",
	})
}

// ─── GET /api/v1/studio/generate/{id} ────────────────────────────────────────

func (h *StudioHandler) GetGenerationStatus(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.ContextUserID).(string)
	userID, _ := uuid.Parse(uid)

	genID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid generation id"})
		return
	}

	gen, err := h.studioSvc.FindGenerationByID(r.Context(), genID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "generation not found"})
		return
	}

	// Ensure the job belongs to the requesting user
	if gen.UserID != userID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "access denied"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":              gen.ID,
		"status":          gen.Status,
		"tool_slug":       gen.ToolSlug,
		"output_url":      gen.OutputURL,
		"output_text":     gen.OutputText,
		"provider":        gen.Provider,
		"points_deducted": gen.PointsDeducted,
		"created_at":      gen.CreatedAt,
		"updated_at":      gen.UpdatedAt,
	})
}

// ─── GET /api/v1/studio/gallery ──────────────────────────────────────────────

func (h *StudioHandler) GetGallery(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.ContextUserID).(string)
	userID, _ := uuid.Parse(uid)

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	items, err := h.studioSvc.GetUserGallery(r.Context(), userID, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load gallery"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":  items,
		"count":  len(items),
		"offset": offset,
		"limit":  limit,
	})
}

// ─── POST /api/v1/studio/chat ─────────────────────────────────────────────────

type chatRequest struct {
	Message   string   `json:"message"`
	SessionID string   `json:"session_id,omitempty"`
	ToolSlug  string   `json:"tool_slug,omitempty"` // "web-search-ai" | "code-helper" | "" (general)
	History   []string `json:"history,omitempty"`
	FileURL   string   `json:"file_url,omitempty"`  // pre-uploaded file URL (PDF, TXT, etc.) from /studio/upload
	LinkURL   string   `json:"link_url,omitempty"`  // web URL or Google Drive link to read
	FileName  string   `json:"file_name,omitempty"` // display name of the attached file
}

func (h *StudioHandler) Chat(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.ContextUserID).(string)

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message is required"})
		return
	}

	// ── Extract text from attached file or link ────────────────────────────
	var attachedContext, attachedName string
	extractor := external.NewTextExtractor()
	if req.FileURL != "" {
		if text, err := extractor.ExtractFromURL(r.Context(), req.FileURL); err == nil && text != "" {
			attachedContext = text
			attachedName = req.FileName
			if attachedName == "" {
				attachedName = "uploaded file"
			}
		}
	} else if req.LinkURL != "" {
		if text, err := extractor.ExtractFromURL(r.Context(), req.LinkURL); err == nil && text != "" {
			attachedContext = text
			attachedName = req.LinkURL
		}
	}

	// ── Session ID: if frontend doesn't send one, mint a new one ──────────────
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = "sess_" + uid[:8] + "_" + fmt.Sprintf("%d", timeNowUnix())
	}

	// ── Route by tool_slug ─────────────────────────────────────────────────
	// web-search-ai and code-helper are dispatched through AIStudioOrchestrator
	// (which uses Pollinations gemini-search and Qwen-Coder respectively).
	// All other slugs (including empty) use the standard LLMOrchestrator chain.
	switch req.ToolSlug {
	case "web-search-ai", "code-helper":
		// Resolve tool → get the tool entity so we can deduct points if needed
		tool, err := h.studioSvc.FindToolBySlug(r.Context(), req.ToolSlug)
		if err != nil {
			// Tool not found — graceful fallback to general chat
			h.handleGeneralChat(w, r, uid, sessionID, req, attachedContext, attachedName)
			return
		}

		// Check if user can afford it (most chat tools are free)
		if !tool.IsFree && tool.PointCost > 0 {
			userID, _ := uuid.Parse(uid)
			gen, genErr := h.studioSvc.RequestGeneration(r.Context(), userID, tool.ID, req.Message)
			if genErr != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": genErr.Error()})
				return
			}
			// Dispatch as full generation job (async)
			h.worker.DispatchGeneration(gen, nil)
			writeJSON(w, http.StatusAccepted, map[string]interface{}{
				"response":   "Processing your request… check Gallery for results.",
				"provider":   req.ToolSlug,
				"session_id": sessionID,
				"generation_id": gen.ID,
			})
			return
		}

		// Free tool — call the AI studio orchestrator's text dispatcher inline
		resp, err := h.llmOrch.ChatWithTool(r.Context(), external.LLMRequest{
			UserID:          uid,
			SessionID:       sessionID,
			Prompt:          req.Message,
			History:         req.History,
			ToolSlug:        req.ToolSlug,
			AttachedContext: attachedContext,
			AttachedName:    attachedName,
		})
		if err != nil {
			// Fallback to general chat on tool failure
			h.handleGeneralChat(w, r, uid, sessionID, req, attachedContext, attachedName)
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"response":      resp.Text,
			"provider":      resp.Provider,
			"session_id":    sessionID,
			"message_count": h.llmOrch.IncrDailyChatCount(r.Context(), uid),
		})
		return

	default:
		h.handleGeneralChat(w, r, uid, sessionID, req, attachedContext, attachedName)
	}
}

// handleGeneralChat routes to the standard Gemini → Groq → DeepSeek cascade.
func (h *StudioHandler) handleGeneralChat(w http.ResponseWriter, r *http.Request, uid, sessionID string, req chatRequest, attachedContext, attachedName string) {
	resp, err := h.llmOrch.Chat(r.Context(), external.LLMRequest{
		UserID:          uid,
		SessionID:       sessionID,
		Prompt:          req.Message,
		History:         req.History,
		ToolSlug:        req.ToolSlug,
		AttachedContext: attachedContext,
		AttachedName:    attachedName,
	})
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "Nexus Chat is temporarily unavailable — please try again shortly",
		})
		return
	}
	// Increment daily chat counter and return count so frontend can display it
	msgCount := h.llmOrch.IncrDailyChatCount(r.Context(), uid)
	// Use the resolved session UUID from the LLM response (so frontend can persist it)
	resolvedSession := resp.SessionID
	if resolvedSession == "" {
		resolvedSession = sessionID
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"response":      resp.Text,
		"provider":      resp.Provider,
		"session_id":    resolvedSession,
		"message_count": msgCount,
	})
}

// timeNowUnix returns the current Unix timestamp (abstracted so tests can stub it).
func timeNowUnix() int64 {
	return time.Now().UnixNano() / 1e6 // milliseconds
}
// ─── GET /api/v1/studio/chat/history ──────────────────────────────────────────────
// Returns the active session ID and all messages for the given mode (?mode=general|search|code).
// Used by the frontend to restore chat history on page load (BUG-05 fix).

func (h *StudioHandler) GetChatHistory(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.ContextUserID).(string)
	toolSlug := r.URL.Query().Get("mode")
	if toolSlug == "" {
		toolSlug = "general"
	}
	sessionID, msgs, err := h.llmOrch.GetChatHistory(r.Context(), uid, toolSlug)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load history"})
		return
	}
	type msgDTO struct {
		Role      string `json:"role"`
		Content   string `json:"content"`
		CreatedAt string `json:"created_at"`
	}
	dtos := make([]msgDTO, 0, len(msgs))
	for _, m := range msgs {
		dtos = append(dtos, msgDTO{
			Role:      m.Role,
			Content:   m.Content,
			CreatedAt: m.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"session_id": sessionID,
		"tool_slug":  toolSlug,
		"messages":   dtos,
	})
}

// ─── GET /api/v1/studio/chat/usage ───────────────────────────────────────────────

func (h *StudioHandler) GetChatUsage(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.ContextUserID).(string)

	// Read from dedicated Redis chat counter (set by Chat handler on each message)
	used := h.llmOrch.GetDailyChatCount(r.Context(), uid)
	limit := h.cfg.GetInt("chat_daily_message_limit", 100)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"used":      used,
		"limit":     limit,
		"remaining": max(0, limit-used),
	})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ─── POST /api/v1/studio/generate/{id}/dispute ────────────────────────────────

func (h *StudioHandler) DisputeGeneration(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.ContextUserID).(string)
	userID, err := uuid.Parse(uid)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid user"})
		return
	}
	genID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid generation id"})
		return
	}
	if err := h.studioSvc.DisputeGeneration(r.Context(), genID, userID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":  "Dispute recorded. Points have been refunded to your wallet.",
		"refunded": true,
	})
}

// ─── GET /api/v1/studio/session ───────────────────────────────────────────────

func (h *StudioHandler) GetSessionUsage(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.ContextUserID).(string)
	userID, _ := uuid.Parse(uid)

	sess, err := h.studioSvc.GetSessionUsage(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load session"})
		return
	}
	if sess == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"active":           false,
			"total_pts_used":   0,
			"generation_count": 0,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"active":           true,
		"session_id":       sess.ID,
		"started_at":       sess.StartedAt,
		"total_pts_used":   sess.TotalPtsUsed,
		"generation_count": sess.GenerationCount,
		"last_active_at":   sess.LastActiveAt,
	})
}

// ─── buildEnrichedPrompt ──────────────────────────────────────────────────────
// Serialises all template-specific parameters from a generateRequest into a
// structured JSON envelope that each dispatcher can parse via parseEnvelope().
//
// The envelope is the single source of truth stored in the Prompt DB column.
// All dispatchers must call parseEnvelope(gen.Prompt) to access fields —
// never parse with string splitting.
//
// Envelope fields:
//   prompt        string  — human-readable text prompt
//   image_url     string  — source image URL (ImageEditor, VideoAnimator, VisionAsk)
//   voice_id      string  — TTS voice name (VoiceStudio / narrate-pro)
//   language      string  — BCP-47 code (Transcribe, Translate, TTS)
//   aspect_ratio  string  — "16:9", "9:16", "1:1" etc.
//   duration      int     — seconds
//   vocals        bool    — music with/without vocals
//   lyrics        string  — user-supplied lyrics
//   style_tags    []str   — style hint array
//   negative_prompt string
//   extra         object  — catch-all for future template fields (speed, bpm, etc.)
func buildEnrichedPrompt(req generateRequest) string {
	payload := map[string]interface{}{
		"prompt": req.Prompt,
	}
	if req.AspectRatio != "" {
		payload["aspect_ratio"] = req.AspectRatio
	}
	if req.Duration > 0 {
		payload["duration"] = req.Duration
	}
	if req.VoiceID != "" {
		payload["voice_id"] = req.VoiceID
	}
	if req.Language != "" {
		payload["language"] = req.Language
	}
	if req.Vocals != nil {
		payload["vocals"] = *req.Vocals
	}
	if req.Lyrics != "" {
		payload["lyrics"] = req.Lyrics
	}
	if len(req.StyleTags) > 0 {
		payload["style_tags"] = req.StyleTags
	}
	if req.NegativePrompt != "" {
		payload["negative_prompt"] = req.NegativePrompt
	}
	if req.ImageURL != "" {
		payload["image_url"] = req.ImageURL
	}
	if req.DocumentURL != "" {
		payload["document_url"] = req.DocumentURL
	}
	if len(req.ExtraParams) > 0 {
		payload["extra"] = req.ExtraParams
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return req.Prompt // safe fallback
	}
	return string(b)
}
