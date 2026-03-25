package handlers

import (
	"context"
	"log"
	"os/exec"
	"strings"
	"time"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/infrastructure/external"

	"github.com/google/uuid"
)

// AsyncStudioWorker dispatches AI generation jobs asynchronously.
// Each tool type is routed to the appropriate backend adapter.
type AsyncStudioWorker struct {
	studioSvc *services.StudioService
	kbGen     external.KnowledgeGenerator // NotebookLM adapter (may be nil during init)
}

func NewAsyncStudioWorker(ss *services.StudioService, kg external.KnowledgeGenerator) *AsyncStudioWorker {
	return &AsyncStudioWorker{studioSvc: ss, kbGen: kg}
}

// DispatchGeneration routes a generation to the correct provider.
func (w *AsyncStudioWorker) DispatchGeneration(gen *entities.AIGeneration, sources []string) {
	tool, err := w.studioSvc.FindToolByID(context.Background(), gen.ToolID)
	if err != nil {
		log.Printf("[WORKER] tool not found for generation %s: %v", gen.ID, err)
		w.studioSvc.FailGeneration(context.Background(), gen.ID, "tool not found")
		return
	}

	switch tool.Provider {
	case "NOTEBOOK_LM":
		w.dispatchNotebookLM(gen, tool, sources)
	case "HUGGING_FACE":
		w.dispatchHuggingFace(gen, tool)
	case "FAL_AI":
		w.dispatchFALAI(gen, tool)
	case "MUBERT":
		w.dispatchMubert(gen, tool)
	case "ASSEMBLY_AI":
		w.dispatchAssemblyAI(gen, tool)
	case "GOOGLE":
		w.dispatchGoogle(gen, tool)
	case "REM_BG":
		w.dispatchRemBG(gen, tool)
	case "PIPELINE":
		w.dispatchCompositePipeline(gen, tool, sources)
	default:
		log.Printf("[WORKER] unknown provider %s for tool %s", tool.Provider, tool.Name)
		w.studioSvc.FailGeneration(context.Background(), gen.ID, "unknown provider")
	}
}

// dispatchNotebookLM calls the notebooklm-py CLI as a subprocess (free tier).
// notebooklm-py creates a notebook, adds sources, triggers generation, polls, downloads.
func (w *AsyncStudioWorker) dispatchNotebookLM(gen *entities.AIGeneration, tool *entities.StudioTool, sources []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Map tool to notebooklm-py generation type
	genType := notebookLMGenType(tool.ProviderTool)

	args := []string{
		"generate",
		"--type", genType,
		"--topic", gen.Prompt,
		"--output-id", gen.ID.String(),
	}
	if len(sources) > 0 {
		args = append(args, "--sources", strings.Join(sources, ","))
	}

	cmd := exec.CommandContext(ctx, "notebooklm", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[NOTEBOOKLM] generation failed for %s: %v\nOutput: %s", gen.ID, err, string(out))
		w.studioSvc.FailGeneration(context.Background(), gen.ID, "NotebookLM generation failed: "+err.Error())
		return
	}

	// Expected output: a file path or S3 URL
	outputURL := strings.TrimSpace(string(out))
	if outputURL == "" {
		w.studioSvc.FailGeneration(context.Background(), gen.ID, "NotebookLM returned no output")
		return
	}

	w.studioSvc.CompleteGeneration(context.Background(), gen.ID, outputURL, "NOTEBOOK_LM", 0)
	log.Printf("[NOTEBOOKLM] ✓ %s completed → %s", gen.ID, outputURL)
}

func (w *AsyncStudioWorker) dispatchHuggingFace(gen *entities.AIGeneration, tool *entities.StudioTool) {
	log.Printf("[HUGGINGFACE] Dispatching %s: %s", tool.Name, gen.Prompt)
	// TODO: Phase 2 — implement HuggingFace Flux inference
	w.studioSvc.FailGeneration(context.Background(), gen.ID, "HuggingFace integration: Phase 2")
}

func (w *AsyncStudioWorker) dispatchFALAI(gen *entities.AIGeneration, tool *entities.StudioTool) {
	log.Printf("[FAL.AI] Dispatching %s: %s", tool.Name, gen.Prompt)
	// TODO: Phase 3 — implement FAL.AI video generation
	w.studioSvc.FailGeneration(context.Background(), gen.ID, "FAL.AI integration: Phase 3")
}

func (w *AsyncStudioWorker) dispatchMubert(gen *entities.AIGeneration, tool *entities.StudioTool) {
	log.Printf("[MUBERT] Dispatching %s", tool.Name)
	// TODO: Phase 3 — Mubert jingle generation
	w.studioSvc.FailGeneration(context.Background(), gen.ID, "Mubert integration: Phase 3")
}

func (w *AsyncStudioWorker) dispatchAssemblyAI(gen *entities.AIGeneration, tool *entities.StudioTool) {
	log.Printf("[ASSEMBLYAI] Dispatching voice-to-plan")
	// TODO: Phase 2 — AssemblyAI transcription + Gemini business plan
	w.studioSvc.FailGeneration(context.Background(), gen.ID, "AssemblyAI integration: Phase 2")
}

func (w *AsyncStudioWorker) dispatchGoogle(gen *entities.AIGeneration, tool *entities.StudioTool) {
	log.Printf("[GOOGLE] Dispatching %s", tool.ProviderTool)
	// TODO: Phase 2 — Google Translate / Cloud TTS
	w.studioSvc.FailGeneration(context.Background(), gen.ID, "Google API integration: Phase 2")
}

func (w *AsyncStudioWorker) dispatchRemBG(gen *entities.AIGeneration, tool *entities.StudioTool) {
	log.Printf("[REMBG] Dispatching background removal")
	// TODO: Phase 2 — rembg self-hosted
	w.studioSvc.FailGeneration(context.Background(), gen.ID, "RemBG integration: Phase 2")
}

func (w *AsyncStudioWorker) dispatchCompositePipeline(gen *entities.AIGeneration, tool *entities.StudioTool, sources []string) {
	log.Printf("[PIPELINE] My Video Story — dispatching composite pipeline")
	// TODO: Phase 3 — HuggingFace + FAL.AI + Mubert composite
	w.studioSvc.FailGeneration(context.Background(), gen.ID, "Composite pipeline: Phase 3")
}

func notebookLMGenType(providerTool string) string {
	m := map[string]string{
		"pdf-gen":          "study_guide",
		"quiz-gen":         "quiz",
		"mindmap-gen":      "mind_map",
		"research-gen":     "deep_research",
		"audio-gen":        "podcast",
		"pptx-gen":         "slide_deck",
		"infographic-gen":  "infographic",
		"business-plan":    "business_plan",
	}
	if v, ok := m[providerTool]; ok {
		return v
	}
	return providerTool
}

// Ensure uuid is used
var _ = uuid.New
