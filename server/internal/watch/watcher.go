// Package watch keeps the catalog in step with the filesystem: it watches each library's
// root folders with fsnotify and, after a short debounce, fires a callback (the server
// enqueues an incremental scan). New subfolders are picked up automatically; a moved file
// is reconciled by the scanner via content hash. See docs/04-server.md (file-watching).
package watch

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher debounces filesystem events per library and invokes onChange when a library's
// tree changes. It is safe for concurrent use.
type Watcher struct {
	fsw      *fsnotify.Watcher
	log      *slog.Logger
	onChange func(libraryID string)
	debounce time.Duration

	mu     sync.Mutex
	dirLib map[string]string      // watched directory -> library id
	timers map[string]*time.Timer // library id -> pending debounce timer
}

// New creates a Watcher. onChange is called (off the event goroutine) with a library id
// after its tree has been quiet for debounce.
func New(log *slog.Logger, debounce time.Duration, onChange func(libraryID string)) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if debounce <= 0 {
		debounce = 2 * time.Second
	}
	return &Watcher{
		fsw:      fsw,
		log:      log,
		onChange: onChange,
		debounce: debounce,
		dirLib:   map[string]string{},
		timers:   map[string]*time.Timer{},
	}, nil
}

// Add starts watching a library's roots (recursively). Errors on individual directories
// are logged, not returned, so one unreadable folder doesn't disable watching.
func (w *Watcher) Add(libraryID string, roots []string) {
	for _, root := range roots {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil //nolint:nilerr — skip unreadable, keep walking
			}
			if d.IsDir() {
				w.addDir(path, libraryID)
			}
			return nil
		})
	}
}

// Remove stops watching every directory registered to a library.
func (w *Watcher) Remove(libraryID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for dir, lib := range w.dirLib {
		if lib == libraryID {
			_ = w.fsw.Remove(dir)
			delete(w.dirLib, dir)
		}
	}
	if t := w.timers[libraryID]; t != nil {
		t.Stop()
		delete(w.timers, libraryID)
	}
}

func (w *Watcher) addDir(dir, libraryID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, ok := w.dirLib[dir]; ok {
		return
	}
	if err := w.fsw.Add(dir); err != nil {
		w.log.Debug("watch: add dir failed", "dir", dir, "err", err)
		return
	}
	w.dirLib[dir] = libraryID
}

// Run processes events until ctx is cancelled. Call it once in a goroutine.
func (w *Watcher) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			w.handle(ev)
		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			w.log.Debug("watch: error", "err", err)
		}
	}
}

func (w *Watcher) handle(ev fsnotify.Event) {
	w.mu.Lock()
	// The event path's directory is the watched node (file events) or itself (dir events).
	libID, ok := w.dirLib[ev.Name]
	if !ok {
		libID, ok = w.dirLib[filepath.Dir(ev.Name)]
	}
	w.mu.Unlock()
	if !ok {
		return
	}

	// A newly created directory must be watched too, so files dropped into it are seen.
	if ev.Op&fsnotify.Create != 0 && isDir(ev.Name) {
		w.Add(libID, []string{ev.Name})
	}

	w.schedule(libID)
}

// schedule (re)arms the per-library debounce timer; onChange fires once the tree is quiet.
func (w *Watcher) schedule(libraryID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if t := w.timers[libraryID]; t != nil {
		t.Stop()
	}
	w.timers[libraryID] = time.AfterFunc(w.debounce, func() {
		w.mu.Lock()
		delete(w.timers, libraryID)
		w.mu.Unlock()
		w.onChange(libraryID)
	})
}

// Close releases the underlying fsnotify watcher and stops pending timers.
func (w *Watcher) Close() error {
	w.mu.Lock()
	for _, t := range w.timers {
		t.Stop()
	}
	w.timers = map[string]*time.Timer{}
	w.mu.Unlock()
	return w.fsw.Close()
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
