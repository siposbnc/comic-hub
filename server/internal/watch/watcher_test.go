package watch

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestWatcher(t *testing.T, fired chan<- string) *Watcher {
	t.Helper()
	w, err := New(slog.New(slog.NewTextHandler(io.Discard, nil)), 60*time.Millisecond, func(id string) {
		fired <- id
	})
	if err != nil {
		t.Fatalf("new watcher: %v", err)
	}
	t.Cleanup(func() { _ = w.Close() })
	return w
}

func waitFor(t *testing.T, ch <-chan string, want string) {
	t.Helper()
	select {
	case got := <-ch:
		if got != want {
			t.Fatalf("fired for %q, want %q", got, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for watcher to fire")
	}
}

func TestWatcherFiresOnNewFile(t *testing.T) {
	root := t.TempDir()
	fired := make(chan string, 4)
	w := newTestWatcher(t, fired)
	w.Add("lib1", []string{root})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Run(ctx)

	if err := os.WriteFile(filepath.Join(root, "Saga 001.cbz"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	waitFor(t, fired, "lib1")
}

func TestWatcherPicksUpNewSubdir(t *testing.T) {
	root := t.TempDir()
	fired := make(chan string, 8)
	w := newTestWatcher(t, fired)
	w.Add("lib1", []string{root})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Run(ctx)

	// Creating a subdir fires once; a file created inside the (now-watched) subdir fires
	// again — proving the new directory was added to the watch set.
	sub := filepath.Join(root, "Batman")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	waitFor(t, fired, "lib1")

	// Drain any coalesced events, then write into the subdir.
	time.Sleep(80 * time.Millisecond)
	drain(fired)
	if err := os.WriteFile(filepath.Join(sub, "Batman 001.cbz"), []byte("y"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	waitFor(t, fired, "lib1")
}

func TestWatcherRemoveStopsEvents(t *testing.T) {
	root := t.TempDir()
	fired := make(chan string, 4)
	w := newTestWatcher(t, fired)
	w.Add("lib1", []string{root})
	w.Remove("lib1")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Run(ctx)

	if err := os.WriteFile(filepath.Join(root, "x.cbz"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	select {
	case id := <-fired:
		t.Fatalf("unexpected fire for %q after Remove", id)
	case <-time.After(300 * time.Millisecond):
		// good: no events
	}
}

func drain(ch <-chan string) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}
