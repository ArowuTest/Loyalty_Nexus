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
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/infrastructure/config"
	"loyalty-nexus/internal/infrastructure/external"
	"loyalty-nexus/internal/presentation/http/middleware"
)

// StudioHandler handles all Nexus Studio HTTP endpoints.
type StudioHandler struct {
	studioSvc *services.StudioService
	llmOrch   *external.LLMOrchestrator
	worker    *AsyncStudioWorker
	cfg       *config.ConfigManager
}

func NewStudioHandler(
	ss *services.StudioService,
	lo *external.LLMOrchestrator,
	kb *AsyncStudioWorker,
	cfg *config.ConfigManager,
) *StudioHandler {
	return &StudioHandler{studioSvc: ss, llmOrch: lo, worker: kb, cfg: cfg}
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
	ToolID  string `json:"tool_id"`   // UUID of studio_tools row
	ToolSlug string `json:"tool_slug"` // alternative lookup by slug
	Prompt  string `json:"prompt"`
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

	// Atomic: deduct PulsePoints + create job
	gen, err := h.studioSvc.RequestGeneration(r.Context(), userID, toolID, req.Prompt)
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
	History   []string `json:"history,omitempty"`
}

func (h *StudioHandler) Chat(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.ContextUserID).(string)

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message is required"})
		return
	}

	// Daily chat message limit (from network_configs — zero-hardcoding rule)
	// Enforcement is delegated to LLMOrchestrator (it tracks per-user daily usage)
	resp, err := h.llmOrch.Chat(r.Context(), external.LLMRequest{
		UserID:    uid,
		SessionID: req.SessionID,
		Prompt:    req.Message,
		History:   req.History,
	})
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "Nexus Chat is temporarily unavailable — please try again shortly",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"response":   resp.Text,
		"provider":   resp.Provider,
		"session_id": req.SessionID,
	})
}

// ─── GET /api/v1/studio/chat/usage ───────────────────────────────────────────

func (h *StudioHandler) GetChatUsage(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.ContextUserID).(string)
	userID, _ := uuid.Parse(uid)

	// Daily generation count doubles as chat quota indicator
	used, _ := h.studioSvc.CountUserGenerationsToday(r.Context(), userID)
	limit := h.cfg.GetInt("chat_daily_message_limit", 20)

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
