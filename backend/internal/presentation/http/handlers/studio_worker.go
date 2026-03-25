package handlers

// studio_worker.go — Async studio job dispatcher
//
// Design principles:
//   - Worker holds a reference to AIStudioOrchestrator (service layer)
//   - All DB writes go through StudioService / AIStudioOrchestrator — never raw SQL here
//   - The worker is constructed once at startup and injected into StudioHandler
//   - DispatchGeneration runs in a goroutine so the HTTP handler returns immediately
//   - Stale-job watchdog method re-queues pending jobs older than 10 minutes

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/domain/entities"
)

// AsyncStudioWorker owns the goroutine pool for AI generation jobs.
type AsyncStudioWorker struct {
	studioSvc *services.StudioService
	orch      *services.AIStudioOrchestrator // 4-tier provider orchestrator
	jobQueue  chan uuid.UUID                  // buffered channel — back-pressure protection
}

// NewAsyncStudioWorker creates the worker.
// studioSvc is used to look up stale jobs; orch does the actual AI dispatch.
// The second argument signature accepts interface{} so it can be called with
// *services.AIStudioOrchestrator or nil (used in unit tests).
func NewAsyncStudioWorker(studioSvc *services.StudioService, orch interface{}) *AsyncStudioWorker {
	var o *services.AIStudioOrchestrator
	if orch != nil {
		if typed, ok := orch.(*services.AIStudioOrchestrator); ok {
			o = typed
		}
	}
	w := &AsyncStudioWorker{
		studioSvc: studioSvc,
		orch:      o,
		jobQueue:  make(chan uuid.UUID, 256), // 256 concurrent-ish jobs max
	}
	// Start background consumer
	go w.run()
	return w
}

// DispatchGeneration enqueues a generation job for async processing.
// gen must be a *entities.AIGeneration (the record returned by StudioService.RequestGeneration).
func (w *AsyncStudioWorker) DispatchGeneration(gen interface{}, _ []string) {
	if w.orch == nil {
		log.Println("[StudioWorker] orchestrator not configured — skipping dispatch")
		return
	}

	// Accept *entities.AIGeneration only (type-safe)
	typed, ok := gen.(*entities.AIGeneration)
	if !ok {
		log.Printf("[StudioWorker] unexpected type %T — expected *entities.AIGeneration", gen)
		return
	}

	select {
	case w.jobQueue <- typed.ID:
		// enqueued
	default:
		// Queue full — fail the job immediately so points are refunded
		log.Printf("[StudioWorker] queue full, failing gen %s immediately", typed.ID)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = w.studioSvc.FailGeneration(ctx, typed.ID, "server busy — please retry in a moment")
	}
}

// run is the background consumer goroutine.
func (w *AsyncStudioWorker) run() {
	for genID := range w.jobQueue {
		// Give each job its own context with a generous timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		w.orch.Dispatch(ctx, genID)
		cancel()
	}
}

// RecoverStaleJobs is called by the lifecycle worker (cron) to retry jobs that
// got stuck in "pending" or "processing" state (e.g. pod restart mid-job).
func (w *AsyncStudioWorker) RecoverStaleJobs(ctx context.Context) {
	if w.orch == nil {
		return
	}
	const staleAfterSeconds = 10 * 60 // 10 minutes
	stale, err := w.studioSvc.ListStalePendingJobs(ctx, staleAfterSeconds, 20)
	if err != nil {
		log.Printf("[StudioWorker] RecoverStaleJobs query: %v", err)
		return
	}
	if len(stale) == 0 {
		return
	}
	log.Printf("[StudioWorker] recovering %d stale jobs", len(stale))
	for _, gen := range stale {
		id := gen.ID // capture for goroutine
		select {
		case w.jobQueue <- id:
		default:
			log.Printf("[StudioWorker] queue full, skipping stale gen %s", id)
		}
	}
}

// LinkHandler is kept for compatibility — the new worker no longer needs it.
// Previously used to break circular initialization; now orch is injected directly.
func (w *AsyncStudioWorker) LinkHandler(_ *StudioHandler) {}
