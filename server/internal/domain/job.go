package domain

import "context"

// JobState is the lifecycle state of a background job.
type JobState string

const (
	JobQueued   JobState = "queued"
	JobRunning  JobState = "running"
	JobDone     JobState = "done"
	JobFailed   JobState = "failed"
	JobCanceled JobState = "canceled"
)

// Job types.
const (
	JobScan          = "scan"
	JobThumbnail     = "thumbnail"
	JobMetadataMatch = "metadata_match"
)

// Job is a unit of background work (scan, thumbnail, …). Progress is reported as
// done/total plus a 0..1 fraction; the WS jobs topic broadcasts updates (docs §10).
type Job struct {
	ID         string
	Type       string
	State      JobState
	Payload    string // JSON
	Progress   float64
	Total      int64
	Done       int64
	Error      string
	CreatedAt  int64
	StartedAt  int64
	FinishedAt int64
}

// JobRepository persists background job records.
type JobRepository interface {
	Create(ctx context.Context, j Job) (Job, error)
	Update(ctx context.Context, j Job) error
	Get(ctx context.Context, id string) (Job, error)
	ListByState(ctx context.Context, state JobState, limit int) ([]Job, error)
}
