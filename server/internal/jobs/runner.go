// Package jobs runs background work (scans, thumbnails) on an in-process worker pool
// backed by the SQLite job table. Each job's progress is persisted so clients can
// observe it (and, in a later milestone, receive it over the WS jobs topic). Jobs are
// cancelable and survive process restarts as idempotent re-runs. See docs/04-server.md §7.
package jobs

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
)

// ProgressFunc is called by a handler to report incremental progress.
type ProgressFunc func(done, total int64)

// Handler performs one job. It must honor ctx cancellation promptly.
type Handler func(ctx context.Context, payload string, progress ProgressFunc) error

// progressInterval throttles how often progress is flushed to the DB.
const progressInterval = 400 * time.Millisecond

// Runner dispatches jobs to a bounded worker pool.
type Runner struct {
	repo     domain.Repository
	logger   *slog.Logger
	handlers map[string]Handler

	sem chan struct{}
	wg  sync.WaitGroup

	mu      sync.Mutex
	cancels map[string]context.CancelFunc
}

// NewRunner builds a runner with the given max concurrent workers (min 1).
func NewRunner(repo domain.Repository, logger *slog.Logger, workers int) *Runner {
	if workers < 1 {
		workers = 1
	}
	return &Runner{
		repo:     repo,
		logger:   logger,
		handlers: make(map[string]Handler),
		sem:      make(chan struct{}, workers),
		cancels:  make(map[string]context.CancelFunc),
	}
}

// Register associates a handler with a job type. Not safe to call after Submit.
func (r *Runner) Register(jobType string, h Handler) {
	r.handlers[jobType] = h
}

// Submit creates a queued job and starts it asynchronously, returning its id. The job
// runs on a background context (it outlives the request that submitted it).
func (r *Runner) Submit(ctx context.Context, jobType, payload string) (string, error) {
	h, ok := r.handlers[jobType]
	if !ok {
		return "", errors.New("jobs: no handler for type " + jobType)
	}

	job := domain.Job{
		ID:        ulid.New(),
		Type:      jobType,
		State:     domain.JobQueued,
		Payload:   payload,
		CreatedAt: time.Now().UnixMilli(),
	}
	if _, err := r.repo.Jobs().Create(ctx, job); err != nil {
		return "", err
	}

	jobCtx, cancel := context.WithCancel(context.Background())
	r.mu.Lock()
	r.cancels[job.ID] = cancel
	r.mu.Unlock()

	r.wg.Add(1)
	go r.run(jobCtx, h, job)
	return job.ID, nil
}

func (r *Runner) run(ctx context.Context, h Handler, job domain.Job) {
	defer r.wg.Done()
	defer func() {
		r.mu.Lock()
		delete(r.cancels, job.ID)
		r.mu.Unlock()
	}()

	r.sem <- struct{}{}
	defer func() { <-r.sem }()

	// Re-check cancellation after possibly waiting for a worker slot.
	if ctx.Err() != nil {
		r.finish(job, domain.JobCanceled, "")
		return
	}

	job.State = domain.JobRunning
	job.StartedAt = time.Now().UnixMilli()
	r.save(job)

	var lastFlush time.Time
	progress := func(done, total int64) {
		job.Done = done
		job.Total = total
		if total > 0 {
			job.Progress = float64(done) / float64(total)
		}
		if time.Since(lastFlush) >= progressInterval {
			lastFlush = time.Now()
			r.save(job)
		}
	}

	err := h(ctx, job.Payload, progress)
	switch {
	case err == nil:
		job.Progress = 1
		if job.Total > 0 {
			job.Done = job.Total
		}
		r.finish(job, domain.JobDone, "")
	case errors.Is(err, context.Canceled) || ctx.Err() != nil:
		r.finish(job, domain.JobCanceled, "")
	default:
		r.logger.Error("job failed", "id", job.ID, "type", job.Type, "err", err)
		r.finish(job, domain.JobFailed, err.Error())
	}
}

func (r *Runner) finish(job domain.Job, state domain.JobState, errMsg string) {
	job.State = state
	job.Error = errMsg
	job.FinishedAt = time.Now().UnixMilli()
	r.save(job)
}

func (r *Runner) save(job domain.Job) {
	if err := r.repo.Jobs().Update(context.Background(), job); err != nil {
		r.logger.Error("persist job", "id", job.ID, "err", err)
	}
}

// Cancel requests cancellation of a running job. It is a no-op if the job is unknown
// or already finished.
func (r *Runner) Cancel(id string) {
	r.mu.Lock()
	cancel := r.cancels[id]
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Shutdown cancels all in-flight jobs and waits for them to finish.
func (r *Runner) Shutdown() {
	r.mu.Lock()
	for _, cancel := range r.cancels {
		cancel()
	}
	r.mu.Unlock()
	r.wg.Wait()
}
