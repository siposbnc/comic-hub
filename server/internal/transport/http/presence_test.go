package http

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/siposbnc/comic-hub/server/internal/config"
	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
	"github.com/siposbnc/comic-hub/server/internal/service/auth"
	"github.com/siposbnc/comic-hub/server/internal/service/presence"
	"github.com/siposbnc/comic-hub/server/internal/service/reading"
	"github.com/siposbnc/comic-hub/server/internal/store/sqlstore"
)

// newPresenceServer wires an auth-enabled server with the presence stack (tracker +
// hub + reading notifier, exactly as main wires it) and two seeded books: one adult,
// one all-ages. Returns the server URL, the store, and the two book ids.
func newPresenceServer(t *testing.T) (string, *sqlstore.Store, string, string) {
	t.Helper()
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "p.db") +
		"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	db, err := sqlstore.OpenSQLite(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := sqlstore.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := sqlstore.NewStore(db)

	lib, _ := store.Libraries().Create(ctx, domain.Library{
		ID: ulid.New(), Name: "DC", Kind: "comic", Roots: []string{`C:\DC`}, CreatedAt: 1, UpdatedAt: 1,
	})
	ser, _ := store.Series().Upsert(ctx, domain.Series{
		ID: ulid.New(), LibraryID: lib.ID, Name: "Batman", SortName: "Batman", CreatedAt: 1, UpdatedAt: 1,
	})
	adult, _ := store.Books().Upsert(ctx, domain.Book{
		ID: ulid.New(), SeriesID: ser.ID, LibraryID: lib.ID, FilePath: `C:\DC\black-label.cbz`,
		FileFormat: "cbz", PageCount: 10, Title: "Batman: Damned", Number: "1",
		AgeRating: "Adults Only 18+", AddedAt: 1, UpdatedAt: 1,
	})
	kidsBook, _ := store.Books().Upsert(ctx, domain.Book{
		ID: ulid.New(), SeriesID: ser.ID, LibraryID: lib.ID, FilePath: `C:\DC\adventures.cbz`,
		FileFormat: "cbz", PageCount: 20, Title: "Batman Adventures", Number: "2",
		AgeRating: "Everyone", AddedAt: 1, UpdatedAt: 1,
	})

	authSvc := auth.New(store, []byte("presence-secret"))
	if err := authSvc.EnsureAdmin(ctx, "alice", "Alice", "hunter2hunter"); err != nil {
		t.Fatalf("ensure admin: %v", err)
	}
	if _, err := authSvc.CreateUser(ctx, auth.CreateUserInput{
		Username: "kid", DisplayName: "Kid", Role: domain.RoleRestricted,
		Password: "kidsecret1", AgeRatingMax: "Everyone 10+",
	}); err != nil {
		t.Fatalf("create restricted: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	hub := NewHub(logger)
	tracker := presence.New(time.Minute)
	tracker.OnChange(hub.BroadcastPresence)
	observe := tracker.ObserveProgress(store)
	readingSvc := reading.New(store, func(userID string, p domain.Progress) {
		hub.BroadcastProgress(p)
		observe(userID, p)
	})

	router := NewRouter(Deps{
		Logger:   logger,
		DB:       db.Unwrap(),
		Config:   config.Config{Mode: config.ModeServer, AuthEnabled: true},
		Repo:     store,
		Reading:  readingSvc,
		Auth:     authSvc,
		Hub:      hub,
		Presence: tracker,
	})
	srv := httptest.NewServer(router)
	t.Cleanup(func() { srv.Close(); _ = db.Close() })
	return srv.URL, store, adult.ID, kidsBook.ID
}

// TestPresenceSnapshotRespectsCeiling: an adult book being read appears in the
// unrestricted snapshot but is withheld from a restricted viewer — invisible, not
// teased, matching browse filtering.
func TestPresenceSnapshotRespectsCeiling(t *testing.T) {
	srv, _, adultBook, kidsBook := newPresenceServer(t)
	api := srv + "/api/v1"

	alice, _ := loginToken(t, api, "alice", "hunter2hunter")
	kid, _ := loginToken(t, api, "kid", "kidsecret1")

	// Alice starts reading the adult book.
	res := authReq(t, http.MethodPut, api+"/me/progress/"+adultBook, alice, `{"page":3,"status":"in_progress"}`)
	mustStatus(t, res, http.StatusOK)

	var mine struct {
		Items []presence.Entry `json:"items"`
	}
	getJSONAuth(t, api+"/presence", alice, &mine)
	if len(mine.Items) != 1 || mine.Items[0].DisplayName != "Alice" || mine.Items[0].BookTitle != "Batman: Damned" {
		t.Fatalf("unrestricted snapshot = %+v, want Alice reading Batman: Damned", mine.Items)
	}
	if mine.Items[0].SeriesTitle != "Batman" || mine.Items[0].Page != 3 {
		t.Errorf("entry enrichment = %+v, want series Batman page 3", mine.Items[0])
	}

	var kids struct {
		Items []presence.Entry `json:"items"`
	}
	getJSONAuth(t, api+"/presence", kid, &kids)
	if len(kids.Items) != 0 {
		t.Fatalf("restricted snapshot = %+v, want empty (adult entry withheld)", kids.Items)
	}

	// Alice switches to the all-ages book: now the kid sees her too.
	res = authReq(t, http.MethodPut, api+"/me/progress/"+kidsBook, alice, `{"page":1,"status":"in_progress"}`)
	mustStatus(t, res, http.StatusOK)
	getJSONAuth(t, api+"/presence", kid, &kids)
	if len(kids.Items) != 1 || kids.Items[0].BookTitle != "Batman Adventures" {
		t.Fatalf("restricted snapshot after switch = %+v, want Batman Adventures", kids.Items)
	}

	// Finishing clears presence for everyone.
	res = authReq(t, http.MethodPut, api+"/me/progress/"+kidsBook, alice, `{"page":19,"status":"read"}`)
	mustStatus(t, res, http.StatusOK)
	getJSONAuth(t, api+"/presence", alice, &mine)
	if len(mine.Items) != 0 {
		t.Fatalf("snapshot after finishing = %+v, want empty", mine.Items)
	}
}

// TestPresenceAndProgressOverWS: presence events apply the viewer's content ceiling,
// and progress events are delivered only to the owning user's sockets.
func TestPresenceAndProgressOverWS(t *testing.T) {
	srv, _, adultBook, kidsBook := newPresenceServer(t)
	api := srv + "/api/v1"

	alice, _ := loginToken(t, api, "alice", "hunter2hunter")
	kid, _ := loginToken(t, api, "kid", "kidsecret1")

	dial := func(token string) *websocket.Conn {
		t.Helper()
		wsURL := "ws" + strings.TrimPrefix(srv, "http") + "/api/v1/ws"
		hdr := http.Header{"Authorization": []string{"Bearer " + token}}
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, hdr)
		if err != nil {
			t.Fatalf("dial ws: %v", err)
		}
		t.Cleanup(func() { _ = conn.Close() })
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		mustWrite(t, conn, map[string]any{"type": "subscribe", "topics": []string{"progress", "presence"}})
		mustWrite(t, conn, map[string]any{"type": "ping"})
		waitFrame(t, conn, func(f frame) bool { return f.Type == "pong" })
		return conn
	}
	aliceConn := dial(alice)
	kidConn := dial(kid)

	// Alice reads the adult book: her socket gets progress + presence; the kid's must not.
	res := authReq(t, http.MethodPut, api+"/me/progress/"+adultBook, alice, `{"page":4,"status":"in_progress"}`)
	mustStatus(t, res, http.StatusOK)
	waitFrame(t, aliceConn, func(f frame) bool { return f.Topic == "progress" && f.Type == "progress.updated" })
	waitFrame(t, aliceConn, func(f frame) bool { return f.Topic == "presence" && f.Type == "presence.updated" })

	// Alice switches to the all-ages book. The kid's FIRST frame must be that presence
	// update — proving neither Alice's progress event nor the adult-book presence event
	// reached the restricted socket.
	res = authReq(t, http.MethodPut, api+"/me/progress/"+kidsBook, alice, `{"page":2,"status":"in_progress"}`)
	mustStatus(t, res, http.StatusOK)
	var first frame
	if err := kidConn.ReadJSON(&first); err != nil {
		t.Fatalf("kid ws read: %v", err)
	}
	if first.Topic != "presence" || first.Type != "presence.updated" ||
		!strings.Contains(string(first.Data), "Batman Adventures") {
		t.Fatalf("kid's first frame = %+v, want presence.updated for Batman Adventures", first)
	}

	// Finishing broadcasts presence.cleared to both.
	res = authReq(t, http.MethodPut, api+"/me/progress/"+kidsBook, alice, `{"page":19,"status":"read"}`)
	mustStatus(t, res, http.StatusOK)
	waitFrame(t, kidConn, func(f frame) bool { return f.Topic == "presence" && f.Type == "presence.cleared" })
	waitFrame(t, aliceConn, func(f frame) bool { return f.Topic == "presence" && f.Type == "presence.cleared" })
}
