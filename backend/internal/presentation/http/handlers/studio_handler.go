package handlers

import (
	"encoding/json"
	"net/http"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/infrastructure/external"
	"github.com/google/uuid"
)

type StudioHandler struct {
	studioService      *services.StudioService
	llmOrchestrator    *external.LLMOrchestrator
	asyncWorker        *AsyncStudioWorker
	knowledgeGenerator external.KnowledgeGenerator
}

func NewStudioHandler(ss *services.StudioService, lo *external.LLMOrchestrator, aw *AsyncStudioWorker, kg external.KnowledgeGenerator) *StudioHandler {
	return &StudioHandler{
		studioService:      ss,
		llmOrchestrator:    lo,
		asyncWorker:        aw,
		knowledgeGenerator: kg,
	}
}

func (h *StudioHandler) GenerateKnowledge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		UserID string    `json:"user_id"`
		ToolID uuid.UUID `json:"tool_id"`
		Topic  string    `json:"topic"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	uid, _ := uuid.Parse(reqBody.UserID)

	// 1. Initial Request (Atomic Point Deduction)
	gen, err := h.studioService.RequestGeneration(r.Context(), uid, reqBody.ToolID, reqBody.Topic)
	if err != nil {
		http.Error(w, err.Error(), http.StatusPaymentRequired)
		return
	}

	// 2. Trigger Async Generation at Provider (NotebookLM)
	providerGenID, err := h.knowledgeGenerator.TriggerGeneration(r.Context(), reqBody.Topic, "pdf")
	if err != nil {
		h.studioService.FailGeneration(r.Context(), gen.ID, "Provider trigger failed")
		http.Error(w, "Trigger failed", http.StatusInternalServerError)
		return
	}

	// 3. Start Background Polling
	h.asyncWorker.StartJob(gen.ID, providerGenID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"generation_id": gen.ID.String(), "status": "processing"})
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

func (h *StudioHandler) GenerateImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		UserID string    `json:"user_id"`
		ToolID uuid.UUID `json:"tool_id"`
		Prompt string    `json:"prompt"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	uid, _ := uuid.Parse(reqBody.UserID)

	// 1. Initial Request (Atomic Point Deduction)
	gen, err := h.studioService.RequestGeneration(r.Context(), uid, reqBody.ToolID, reqBody.Prompt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusPaymentRequired)
		return
	}

	// 2. Determine Provider & Tool ID
	tool, _ := h.studioService.FindToolByID(r.Context(), reqBody.ToolID)
	
	// 3. Dispatch to External Generator (Mocking the sync call)
	// In production, this would use h.imageGenerator.Generate(...)
	outputURL := fmt.Sprintf("https://cdn.loyalty-nexus.ai/generated/%s.webp", gen.ID.String())
	
	// Simulation of success
	// costMicros is internal tracking of API costs (Innovation 6.4)
	if err := h.studioService.CompleteGeneration(r.Context(), gen.ID, outputURL, "FAL_AI", 50000); err != nil {
		h.studioService.FailGeneration(r.Context(), gen.ID, "Storage failure")
		http.Error(w, "Processing failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"output_url": outputURL})
}

func (h *StudioHandler) GetGallery(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	uid, _ := uuid.Parse(userID)
	
	gallery, err := h.studioService.GetUserGallery(r.Context(), uid)
	if err != nil {
		http.Error(w, "Failed to load gallery", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(gallery)
}

func (h *StudioHandler) GenerateBuild(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		UserID      string    `json:"user_id"`
		ToolID      uuid.UUID `json:"tool_id"`
		Description string    `json:"description"`
		AudioURL    string    `json:"audio_url,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	uid, _ := uuid.Parse(reqBody.UserID)

	// 1. Point Deduction
	gen, err := h.studioService.RequestGeneration(r.Context(), uid, reqBody.ToolID, reqBody.Description)
	if err != nil {
		http.Error(w, err.Error(), http.StatusPaymentRequired)
		return
	}

	// 2. Handle Build Pipeline (Sync for now, can be moved to async worker)
	var outputURL string
	// Mocking tool logic based on catalogue
	// In production, we check tool.Name or tool.ProviderToolID
	outputURL = fmt.Sprintf("https://cdn.loyalty-nexus.ai/build/%s.pdf", gen.ID.String())

	if err := h.studioService.CompleteGeneration(r.Context(), gen.ID, outputURL); err != nil {
		h.studioService.FailGeneration(r.Context(), gen.ID, "Document gen failed")
		http.Error(w, "Build failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"output_url": outputURL})
}
