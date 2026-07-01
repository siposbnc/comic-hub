package jobs

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/store/sqlstore"
)

func newRepo(t *testing.T) domain.Repository {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "jobs.db") +
		"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	db, err := sqlstore.OpenSQLite(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := sqlstore.Migrate(context.Background(), db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return sqlstore.NewStore(db)
}

func newRunner(t *testing.T, repo domain.Repository) *Runner {
	return NewRunner(repo, slog.New(slog.NewTextHandler(io.Discard, nil)), 2)
}

// waitForState polls the job row until it reaches state or the deadline.
func waitForState(t *testing.T, repo domain.Repository, id string, state domain.JobState) domain.Job {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		j, err := repo.Jobs().Get(context.Background(), id)
		if err != nil {
			t.Fatalf("get job: %v", err)
		}
		if j.State == state {
			return j
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("job %s did not reach state %q in time", id, state)
	return domain.Job{}
}

func TestRunnerSuccess(t *testing.T) {
	repo := newRepo(t)
	r := newRunner(t, repo)

	r.Register("test", func(_ context.Context, payload string, progress ProgressFunc) error {
		progress(5, 10)
		progress(10, 10)
		return nil
	})

	id, err := r.Submit(context.Background(), "test", `{"x":1}`)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	j := waitForState(t, repo, id, domain.JobDone)
	if j.Progress != 1 || j.Done != 10 {
		t.Fatalf("done job = %+v", j)
	}
}

func TestRunnerFailure(t *testing.T) {
	repo := newRepo(t)
	r := newRunner(t, repo)
	r.Register("boom", func(_ context.Context, _ string, _ ProgressFunc) error {
		return errors.New("kaboom")
	})
	id, _ := r.Submit(context.Background(), "boom", "")
	j := waitForState(t, repo, id, domain.JobFailed)
	if j.Error != "kaboom" {
		t.Fatalf("expected error message, got %q", j.Error)
	}
}

func TestRunnerCancel(t *testing.T) {
	repo := newRepo(t)
	r := newRunner(t, repo)
	started := make(chan struct{})
	r.Register("long", func(ctx context.Context, _ string, _ ProgressFunc) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})
	id, _ := r.Submit(context.Background(), "long", "")
	<-started
	r.Cancel(id)
	j := waitForState(t, repo, id, domain.JobCanceled)
	if j.State != domain.JobCanceled {
		t.Fatalf("expected canceled, got %q", j.State)
	}
}

func TestRunnerUnknownType(t *testing.T) {
	repo := newRepo(t)
	r := newRunner(t, repo)
	if _, err := r.Submit(context.Background(), "nope", ""); err == nil {
		t.Fatal("expected error for unregistered job type")
	}
}
