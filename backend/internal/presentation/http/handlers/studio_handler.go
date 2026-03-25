package handlers

import (
	"encoding/json"
	"net/http"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/infrastructure/external"
	"github.com/google/uuid"
)

type StudioHandler struct {
	studioService   *services.StudioService
	llmOrchestrator *external.LLMOrchestrator
}

func NewStudioHandler(ss *services.StudioService, lo *external.LLMOrchestrator) *StudioHandler {
	return &StudioHandler{
		studioService:   ss,
		llmOrchestrator: lo,
	}
}

func (h *StudioHandler) Chat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		Prompt string `json:"prompt"`
		MSISDN string `json:"msisdn"` // In production, get from JWT
	}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// 1. Prepare LLM Request
	llmReq := external.LLMRequest{
		UserID: reqBody.MSISDN,
		Prompt: reqBody.Prompt,
	}

	// 2. Execute Orchestration (Groq -> Gemini -> DeepSeek)
	resp, err := h.llmOrchestrator.Chat(r.Context(), llmReq)
	if err != nil {
		http.Error(w, "AI Studio unavailable", http.StatusInternalServerError)
		return
	}

	// 3. Return Response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *StudioHandler) ListTools(w http.ResponseWriter, r *http.Request) {
	tools, err := h.studioService.ListActiveTools(r.Context())
	if err != nil {
		http.Error(w, "Failed to load tools", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(tools)
}
