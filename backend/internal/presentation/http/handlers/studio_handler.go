package handlers

import (
	"encoding/json"
	"net/http"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/infrastructure/config"
	"loyalty-nexus/internal/infrastructure/external"
	"loyalty-nexus/internal/presentation/http/middleware"

	"github.com/google/uuid"
)

type StudioHandler struct {
	studioSvc *services.StudioService
	llmOrch   *external.LLMOrchestrator
	kbWorker  *AsyncStudioWorker
	cfg       *config.ConfigManager
}

func NewStudioHandler(ss *services.StudioService, lo *external.LLMOrchestrator, kb *AsyncStudioWorker, cfg *config.ConfigManager) *StudioHandler {
	return &StudioHandler{studioSvc: ss, llmOrch: lo, kbWorker: kb, cfg: cfg}
}

func (h *StudioHandler) ListTools(w http.ResponseWriter, r *http.Request) {
	tools, err := h.studioSvc.ListActiveTools(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load tools"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"tools": tools})
}

type ChatRequest struct {
	Message   string   `json:"message"`
	SessionID string   `json:"session_id,omitempty"`
	History   []string `json:"history,omitempty"`
}

func (h *StudioHandler) Chat(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.ContextUserID).(string)

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message is required"})
		return
	}

	// Check daily chat limit
	limit := h.cfg.GetInt("chat_daily_message_limit", 20)
	_ = limit // enforced in llm orchestrator

	llmReq := external.LLMRequest{
		UserID:  uid,
		Prompt:  req.Message,
		History: req.History,
	}

	resp, err := h.llmOrch.Chat(r.Context(), llmReq)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "AI service temporarily unavailable"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"response": resp.Text,
		"provider": resp.Provider,
	})
}

type GenerateRequest struct {
	ToolID string `json:"tool_id"`
	Prompt string `json:"prompt"`
	Sources []string `json:"sources,omitempty"` // For NotebookLM tools
}

func (h *StudioHandler) Generate(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.ContextUserID).(string)
	userID, _ := uuid.Parse(uid)

	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	toolID, err := uuid.Parse(req.ToolID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid tool_id"})
		return
	}

	gen, err := h.studioSvc.RequestGeneration(r.Context(), userID, toolID, req.Prompt)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Dispatch async generation
	go h.kbWorker.DispatchGeneration(gen, req.Sources)

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"generation_id": gen.ID,
		"status":        "pending",
		"message":       "Your generation is being processed. You'll be notified when it's ready.",
	})
}

func (h *StudioHandler) GetGenerationStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	genID, err := uuid.Parse(id)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	gen, err := h.studioSvc.FindGenerationByID(r.Context(), genID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, gen)
}

func (h *StudioHandler) GetGallery(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.ContextUserID).(string)
	userID, _ := uuid.Parse(uid)
	gallery, err := h.studioSvc.GetUserGallery(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load gallery"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"items": gallery})
}
