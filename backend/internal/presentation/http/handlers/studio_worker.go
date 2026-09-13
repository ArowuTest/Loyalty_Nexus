package handlers

// studio_worker.go — bounded, class-separated async Studio dispatcher.
//
// Jobs are already durable in ai_generations before they reach this worker.
// In-memory queues provide fast admission/back-pressure; the lifecycle recovery
// path remains the safety net for process restarts.

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/domain/entities"
)

type queuedStudioJob struct {
	ID              uuid.UUID
	ToolSlug        string
	QueueClass      string
	EnqueuedAt      time.Time
	MaxQueueSeconds int
}

// AsyncStudioWorker owns class-separated bounded worker pools so long-running
// video work cannot starve interactive/image/audio generations.
type AsyncStudioWorker struct {
	studioSvc *services.StudioService
	orch      *services.AIStudioOrchestrator
	queues    map[string]chan queuedStudioJob
}

var studioQueueClasses = []string{
	entities.QueueRealtime,
	entities.QueueInteractive,
	entities.QueueAsync,
	entities.QueueHeavyAsync,
	entities.QueueBackground,
}

func NewAsyncStudioWorker(studioSvc *services.StudioService, orch interface{}) *AsyncStudioWorker {
	var o *services.AIStudioOrchestrator
	if typed, ok := orch.(*services.AIStudioOrchestrator); ok {
		o = typed
	}
	w := &AsyncStudioWorker{
		studioSvc: studioSvc,
		orch:      o,
		queues:    make(map[string]chan queuedStudioJob, len(studioQueueClasses)),
	}
	for _, class := range studioQueueClasses {
		cfg := o.StudioWorkerPoolConfig(class)
		ch := make(chan queuedStudioJob, cfg.Buffer)
		w.queues[class] = ch
		for i := 0; i < cfg.Workers; i++ {
			go w.runQueue(class, i, ch)
		}
		log.Printf("[StudioWorker] class=%s workers=%d buffer=%d", class, cfg.Workers, cfg.Buffer)
	}
	return w
}

func (w *AsyncStudioWorker) queueFor(class string) chan queuedStudioJob {
	if q := w.queues[class]; q != nil {
		return q
	}
	return w.queues[entities.QueueAsync]
}

func studioJobTimeout(class string) time.Duration {
	switch class {
	case entities.QueueHeavyAsync:
		return 12 * time.Minute
	case entities.QueueAsync:
		return 7 * time.Minute
	case entities.QueueBackground:
		return 10 * time.Minute
	case entities.QueueRealtime, entities.QueueInteractive:
		return 3 * time.Minute
	default:
		return 7 * time.Minute
	}
}

func (w *AsyncStudioWorker) classifyJob(toolSlug string) (string, int) {
	if w.orch == nil {
		return entities.QueueAsync, 60
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return w.orch.QueuePolicyForTool(ctx, toolSlug)
}

// DispatchGeneration enqueues a generation job after the DB transaction has
// committed. Queue admission is non-blocking so API latency never depends on an
// AI provider or a saturated worker pool.
func (w *AsyncStudioWorker) DispatchGeneration(gen interface{}, _ []string) {
	if w.orch == nil {
		log.Println("[StudioWorker] orchestrator not configured — skipping dispatch")
		return
	}
	typed, ok := gen.(*entities.AIGeneration)
	if !ok {
		log.Printf("[StudioWorker] unexpected type %T — expected *entities.AIGeneration", gen)
		return
	}

	class, maxQueueSeconds := w.classifyJob(typed.ToolSlug)
	job := queuedStudioJob{
		ID: typed.ID, ToolSlug: typed.ToolSlug, QueueClass: class,
		EnqueuedAt: time.Now(), MaxQueueSeconds: maxQueueSeconds,
	}
	select {
	case w.queueFor(class) <- job:
		return
	default:
		// Extreme overload beyond the bounded queue. Fail/refund immediately
		// rather than allocating unbounded goroutines or memory.
		log.Printf("[StudioWorker] queue full class=%s gen=%s", class, typed.ID)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = w.studioSvc.FailGeneration(ctx, typed.ID, "AI queue is at capacity — points refunded; please retry shortly")
	}
}

func (w *AsyncStudioWorker) runQueue(class string, workerID int, jobs <-chan queuedStudioJob) {
	for job := range jobs {
		if job.MaxQueueSeconds > 0 && time.Since(job.EnqueuedAt) > time.Duration(job.MaxQueueSeconds)*time.Second {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_ = w.studioSvc.FailGeneration(ctx, job.ID, "AI queue wait limit exceeded — points refunded; please retry")
			cancel()
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), studioJobTimeout(class))
		if dispatchErr := w.orch.Dispatch(ctx, job.ID); dispatchErr != nil {
			log.Printf("[StudioWorker] class=%s worker=%d gen=%s dispatch error: %v", class, workerID, job.ID, dispatchErr)
		}
		cancel()
	}
}

// RecoverStaleJobs re-admits durable jobs after an interrupted worker/process.
// The DB remains the source of truth; duplicate safety is handled by the
// generation status/refund guards in StudioService.
func (w *AsyncStudioWorker) RecoverStaleJobs(ctx context.Context) {
	if w.orch == nil {
		return
	}
	const staleAfterSeconds = 10 * 60
	stale, err := w.studioSvc.ListStalePendingJobs(ctx, staleAfterSeconds, 100)
	if err != nil {
		log.Printf("[StudioWorker] RecoverStaleJobs query: %v", err)
		return
	}
	for _, gen := range stale {
		class, maxQueueSeconds := w.orch.QueuePolicyForTool(ctx, gen.ToolSlug)
		job := queuedStudioJob{
			ID: gen.ID, ToolSlug: gen.ToolSlug, QueueClass: class,
			EnqueuedAt: time.Now(), MaxQueueSeconds: maxQueueSeconds,
		}
		select {
		case w.queueFor(class) <- job:
		default:
			log.Printf("[StudioWorker] recovery queue full class=%s gen=%s; leaving durable job for next recovery", class, gen.ID)
		}
	}
}

func (w *AsyncStudioWorker) LinkHandler(_ *StudioHandler) {}
