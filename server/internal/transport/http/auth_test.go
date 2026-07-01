package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/siposbnc/comic-hub/server/internal/config"
	"github.com/siposbnc/comic-hub/server/internal/service/auth"
	"github.com/siposbnc/comic-hub/server/internal/store/sqlstore"
)

// newAuthServer builds an auth-enabled test server with a bootstrapped admin "alice".
func newAuthServer(t *testing.T) (string, *auth.Service) {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "auth.db") +
		"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	db, err := sqlstore.OpenSQLite(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := sqlstore.Migrate(context.Background(), db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := sqlstore.NewStore(db)
	authSvc := auth.New(store, []byte("integration-secret"))
	if err := authSvc.EnsureAdmin(context.Background(), "alice", "Alice", "hunter2hunter"); err != nil {
		t.Fatalf("ensure admin: %v", err)
	}
	router := NewRouter(Deps{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:     db.Unwrap(),
		Config: config.Config{Mode: config.ModeServer, AuthEnabled: true},
		Repo:   store,
		Auth:   authSvc,
	})
	srv := httptest.NewServer(router)
	t.Cleanup(func() { srv.Close(); _ = db.Close() })
	return srv.URL, authSvc
}

func TestAuthFlowEndToEnd(t *testing.T) {
	srv, _ := newAuthServer(t)
	api := srv + "/api/v1"

	// A protected route without a token is rejected.
	resp, err := http.Get(api + "/server/info")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no-token /server/info = %d, want 401", resp.StatusCode)
	}

	// Wrong password → 401.
	mustStatus(t, sendJSON(t, http.MethodPost, api+"/auth/login", `{"username":"alice","password":"nope"}`), http.StatusUnauthorized)

	// Login → 200 with tokens.
	loginResp := sendJSON(t, http.MethodPost, api+"/auth/login", `{"username":"alice","password":"hunter2hunter"}`)
	defer loginResp.Body.Close()
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", loginResp.StatusCode)
	}
	var toks struct {
		Access  string `json:"access"`
		Refresh string `json:"refresh"`
		User    struct {
			Username string `json:"username"`
			Role     string `json:"role"`
		} `json:"user"`
	}
	if err := json.NewDecoder(loginResp.Body).Decode(&toks); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	if toks.Access == "" || toks.Refresh == "" || toks.User.Username != "alice" || toks.User.Role != "admin" {
		t.Fatalf("login payload = %+v", toks)
	}

	// The access token unlocks protected routes and identifies the user via handshake.
	var hs struct {
		User struct {
			Username string `json:"username"`
			Role     string `json:"role"`
		} `json:"user"`
	}
	getJSONAuth(t, api+"/auth/handshake", toks.Access, &hs)
	if hs.User.Username != "alice" || hs.User.Role != "admin" {
		t.Fatalf("handshake user = %+v", hs.User)
	}

	// /server/info with the token → 200.
	req, _ := http.NewRequest(http.MethodGet, api+"/server/info", nil)
	req.Header.Set("Authorization", "Bearer "+toks.Access)
	r2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("authed get: %v", err)
	}
	mustStatus(t, r2, http.StatusOK)

	// Refresh issues a new pair; logout revokes it.
	refreshResp := sendJSON(t, http.MethodPost, api+"/auth/refresh", `{"refresh":"`+toks.Refresh+`"}`)
	defer refreshResp.Body.Close()
	if refreshResp.StatusCode != http.StatusOK {
		t.Fatalf("refresh status = %d, want 200", refreshResp.StatusCode)
	}
	var rt struct {
		Refresh string `json:"refresh"`
	}
	_ = json.NewDecoder(refreshResp.Body).Decode(&rt)
	mustStatus(t, sendJSON(t, http.MethodPost, api+"/auth/logout", `{"refresh":"`+rt.Refresh+`"}`), http.StatusNoContent)
	// The revoked refresh token no longer works.
	mustStatus(t, sendJSON(t, http.MethodPost, api+"/auth/refresh", `{"refresh":"`+rt.Refresh+`"}`), http.StatusUnauthorized)
}

func getJSONAuth(t *testing.T, url, token string, dst any) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get %s = %d, want 200", url, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
}
