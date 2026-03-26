package services

// ussd_knowledge_service.go — USSD Knowledge Tools SMS delivery service.
//
// REQ-6.4: Feature phone users can access Nexus Studio Knowledge Tools
// (Study Guide, Quiz, Mind Map) by submitting a topic via USSD. The system
// processes the request asynchronously and delivers the result via SMS as a
// text summary (first 280 chars) plus a short URL to the full asset.
//
// Design:
//   - USSDKnowledgeService wraps StudioService + AIStudioOrchestrator + NotificationService.
//   - SubmitKnowledgeTool: finds the tool by slug, calls StudioService.RequestGeneration,
//     dispatches the job to AIStudioOrchestrator, then returns immediately.
//     The USSD response is "END ✅ Your [tool] is being prepared. You'll receive an SMS shortly."
//   - DeliverKnowledgeSMS: called by the lifecycle worker after a generation completes.
//     Sends a 280-char summary + short URL via NotificationService.SendSMS.
//   - The short URL is built from cfg key "app_base_url" + "/studio/result/{genID}".
//   - All tool slugs are read from DB — zero hardcoding.

import (
	"context"
	"fmt"
	"log"
	"time"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/infrastructure/config"

	"github.com/google/uuid"
)

// GenerationDispatcher is a minimal interface satisfied by *handlers.AsyncStudioWorker.
// Defined here to avoid a circular import between services and handlers packages.
type GenerationDispatcher interface {
	DispatchGeneration(gen interface{}, refs []string)
}

// knowledgeToolSlugs are the three tools accessible via USSD.
// These slugs must match studio_tools.slug in the DB.
var knowledgeToolSlugs = []string{"study-guide", "quiz", "mind-map"}

// USSDKnowledgeService orchestrates the USSD → AI Studio → SMS pipeline.
type USSDKnowledgeService struct {
	studioSvc *StudioService
	worker    GenerationDispatcher
	notifySvc *NotificationService
	cfg       *config.ConfigManager
}

// NewUSSDKnowledgeService constructs the service.
func NewUSSDKnowledgeService(
	studioSvc *StudioService,
	worker GenerationDispatcher,
	notifySvc *NotificationService,
	cfg *config.ConfigManager,
) *USSDKnowledgeService {
	return &USSDKnowledgeService{
		studioSvc: studioSvc,
		worker:    worker,
		notifySvc: notifySvc,
		cfg:       cfg,
	}
}

// KnowledgeToolOption describes a knowledge tool for the USSD menu.
type KnowledgeToolOption struct {
	Slug  string
	Label string
}

// ListKnowledgeTools returns the three knowledge tools available via USSD,
// in menu order. Only returns tools that are active in the DB.
func (s *USSDKnowledgeService) ListKnowledgeTools(ctx context.Context) ([]KnowledgeToolOption, error) {
	options := make([]KnowledgeToolOption, 0, 3)
	labels := map[string]string{
		"study-guide": "Study Guide",
		"quiz":        "Quiz",
		"mind-map":    "Mind Map",
	}
	for _, slug := range knowledgeToolSlugs {
		tool, err := s.studioSvc.FindToolBySlug(ctx, slug)
		if err != nil || tool == nil || !tool.IsActive {
			continue
		}
		options = append(options, KnowledgeToolOption{
			Slug:  slug,
			Label: labels[slug],
		})
	}
	return options, nil
}

// SubmitKnowledgeTool submits an async AI generation job for the given tool slug
// and topic. Returns the generation ID so the caller can track it.
// The USSD handler calls this and immediately returns an END response to the user.
func (s *USSDKnowledgeService) SubmitKnowledgeTool(
	ctx context.Context,
	userID uuid.UUID,
	toolSlug string,
	topic string,
) (*entities.AIGeneration, error) {
	// Resolve tool
	tool, err := s.studioSvc.FindToolBySlug(ctx, toolSlug)
	if err != nil || tool == nil {
		return nil, fmt.Errorf("tool %q not found", toolSlug)
	}
	if !tool.IsActive {
		return nil, fmt.Errorf("tool %q is currently unavailable", toolSlug)
	}

	// Build enriched prompt — USSD requests are plain text topics
	prompt := fmt.Sprintf("[USSD] %s", topic)

	// Atomic: deduct PulsePoints + create generation job
	gen, err := s.studioSvc.RequestGeneration(ctx, userID, tool.ID, prompt)
	if err != nil {
		return nil, err
	}

	// Dispatch to background AI worker — non-blocking
	s.worker.DispatchGeneration(gen, nil)

	log.Printf("[USSDKnowledge] submitted gen %s for user %s tool %s topic=%q",
		gen.ID, userID, toolSlug, topic)

	return gen, nil
}

// DeliverKnowledgeSMS is called by the lifecycle worker after a generation
// with a USSD origin completes. It sends a 280-char summary + short URL via SMS.
// The caller is responsible for determining whether the generation originated
// from USSD (e.g. by checking if prompt starts with "[USSD]").
func (s *USSDKnowledgeService) DeliverKnowledgeSMS(
	ctx context.Context,
	gen *entities.AIGeneration,
	phoneNumber string,
) error {
	baseURL := s.cfg.GetString("app_base_url", "https://loyalty-nexus.app")
	shortURL := fmt.Sprintf("%s/studio/result/%s", baseURL, gen.ID)

	// Build summary: first 280 chars of output text, then short URL
	summary := gen.OutputText
	if len(summary) > 280 {
		// Trim at last word boundary before 280
		summary = summary[:280]
		for i := len(summary) - 1; i > 0; i-- {
			if summary[i] == ' ' {
				summary = summary[:i]
				break
			}
		}
		summary += "..."
	}

	var msg string
	if summary != "" {
		msg = fmt.Sprintf("✅ Your %s is ready!\n\n%s\n\nFull result: %s",
			gen.ToolSlug, summary, shortURL)
	} else {
		// Image/audio tools — just send the URL
		msg = fmt.Sprintf("✅ Your %s is ready! View it here: %s", gen.ToolSlug, shortURL)
	}

	// Truncate to 160 chars for single SMS (AT billing)
	maxLen := s.cfg.GetInt("ussd_sms_max_chars", 320)
	if len(msg) > maxLen {
		msg = msg[:maxLen]
	}

	return s.notifySvc.SendSMS(ctx, phoneNumber, msg)
}

// IsUSSDGeneration returns true if the generation was submitted via USSD.
// Used by the lifecycle worker to decide whether to call DeliverKnowledgeSMS.
func IsUSSDGeneration(gen *entities.AIGeneration) bool {
	return len(gen.Prompt) > 6 && gen.Prompt[:6] == "[USSD]"
}

// ─── Session timeout rollback helper ─────────────────────────────────────────

// SpinRollbackFunc is the function signature for rolling back a spin.
// Injected from SpinService to avoid a circular import.
type SpinRollbackFunc func(ctx context.Context, spinID uuid.UUID) error

// RollbackExpiredUSSDSpins is called by the lifecycle worker every 5 minutes.
// It finds expired USSD sessions with a pending spin and rolls back the spin,
// then cleans up the expired sessions.
func RollbackExpiredUSSDSpins(
	ctx context.Context,
	sessionRepo interface {
		GetExpiredWithPendingSpin(ctx context.Context) ([]entities.USSDSession, error)
		DeleteExpired(ctx context.Context) error
	},
	rollback SpinRollbackFunc,
) {
	sessions, err := sessionRepo.GetExpiredWithPendingSpin(ctx)
	if err != nil {
		log.Printf("[USSDRollback] query error: %v", err)
		return
	}
	for _, sess := range sessions {
		if sess.PendingSpinID == nil {
			continue
		}
		if rbErr := rollback(ctx, *sess.PendingSpinID); rbErr != nil {
			log.Printf("[USSDRollback] rollback spin %s for session %s: %v",
				*sess.PendingSpinID, sess.SessionID, rbErr)
		} else {
			log.Printf("[USSDRollback] rolled back spin %s (session %s expired at %s)",
				*sess.PendingSpinID, sess.SessionID, sess.ExpiresAt.Format(time.RFC3339))
		}
	}
	if cleanErr := sessionRepo.DeleteExpired(ctx); cleanErr != nil {
		log.Printf("[USSDRollback] delete expired sessions: %v", cleanErr)
	}
}
