package http

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWebSocketJobsAndProgress(t *testing.T) {
	srv, store := newScanServer(t)
	api := srv + "/api/v1"

	// A library with one book so we can exercise the progress topic.
	root := t.TempDir()
	writeImageCBZ(t, filepath.Join(root, "Saga", "Saga 001.cbz"), map[string][]byte{
		"p1.png": makePNGBytes(40, 60), "p2.png": makePNGBytes(40, 60),
	})

	// Connect and subscribe.
	wsURL := "ws" + strings.TrimPrefix(srv, "http") + "/api/v1/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	mustWrite(t, conn, map[string]any{"type": "subscribe", "topics": []string{"jobs", "progress"}})
	// Ping/pong round-trip guarantees the subscribe above has been applied before we
	// trigger events (same connection, processed in order).
	mustWrite(t, conn, map[string]any{"type": "ping"})
	waitFrame(t, conn, func(f frame) bool { return f.Type == "pong" })

	// Trigger a scan -> expect job.* frames on the jobs topic.
	lib := createLibrary(t, api, root)
	res, err := http.Post(api+"/libraries/"+lib.ID+"/scan", "application/json", strings.NewReader(`{"mode":"full"}`))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	res.Body.Close()
	waitFrame(t, conn, func(f frame) bool { return f.Topic == "jobs" && strings.HasPrefix(f.Type, "job.") })

	// Wait for the scan to finish, then resolve the book id.
	deadline := time.Now().Add(5 * time.Second)
	var bookID string
	for time.Now().Before(deadline) && bookID == "" {
		series, _ := store.Series().ListByLibrary(context.Background(), lib.ID)
		if len(series) == 1 {
			books, _ := store.Books().ListBySeries(context.Background(), series[0].ID)
			if len(books) == 1 {
				bookID = books[0].ID
			}
		}
		if bookID == "" {
			time.Sleep(20 * time.Millisecond)
		}
	}
	if bookID == "" {
		t.Fatal("scan did not produce a book")
	}

	// Write progress -> expect a progress.updated frame.
	putJSON(t, api+"/me/progress/"+bookID, `{"page":1,"status":"in_progress"}`)
	waitFrame(t, conn, func(f frame) bool { return f.Topic == "progress" && f.Type == "progress.updated" })
}

type frame struct {
	Type  string          `json:"type"`
	Topic string          `json:"topic"`
	Data  json.RawMessage `json:"data"`
}

func mustWrite(t *testing.T, conn *websocket.Conn, v any) {
	t.Helper()
	if err := conn.WriteJSON(v); err != nil {
		t.Fatalf("ws write: %v", err)
	}
}

// waitFrame reads frames until pred matches or the read deadline fires.
func waitFrame(t *testing.T, conn *websocket.Conn, pred func(frame) bool) frame {
	t.Helper()
	for {
		var f frame
		if err := conn.ReadJSON(&f); err != nil {
			t.Fatalf("ws read (waiting for frame): %v", err)
		}
		if pred(f) {
			return f
		}
	}
}
