package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

// loginToken logs in and returns the access + refresh tokens.
func loginToken(t *testing.T, api, username, pass string) (string, string) {
	t.Helper()
	resp := sendJSON(t, http.MethodPost, api+"/auth/login",
		`{"username":"`+username+`","password":"`+pass+`"}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login %s = %d, want 200", username, resp.StatusCode)
	}
	var b struct{ Access, Refresh string }
	if err := json.NewDecoder(resp.Body).Decode(&b); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	return b.Access, b.Refresh
}

// authReq sends a request with a bearer token and returns the response.
func authReq(t *testing.T, method, url, token, body string) *http.Response {
	t.Helper()
	var r *http.Request
	var err error
	if body == "" {
		r, err = http.NewRequest(method, url, nil)
	} else {
		r, err = http.NewRequest(method, url, bytes.NewBufferString(body))
		r.Header.Set("Content-Type", "application/json")
	}
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	r.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

func TestUserManagementAndRoleGating(t *testing.T) {
	srv, _ := newAuthServer(t)
	api := srv + "/api/v1"

	adminAccess, _ := loginToken(t, api, "alice", "hunter2hunter")

	// Admin creates a member.
	created := authReq(t, http.MethodPost, api+"/users", adminAccess,
		`{"username":"bob","displayName":"Bob","role":"member","password":"bobsecret1"}`)
	defer created.Body.Close()
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create user = %d, want 201", created.StatusCode)
	}
	var bob struct {
		ID   string `json:"id"`
		Role string `json:"role"`
	}
	_ = json.NewDecoder(created.Body).Decode(&bob)
	if bob.Role != "member" {
		t.Fatalf("created role = %q", bob.Role)
	}

	bobAccess, bobRefresh := loginToken(t, api, "bob", "bobsecret1")

	// A member can't list users (admin-only) or shut the server down.
	if r := authReq(t, http.MethodGet, api+"/users", bobAccess, ""); r.StatusCode != http.StatusForbidden {
		r.Body.Close()
		t.Fatalf("member GET /users = %d, want 403", r.StatusCode)
	}
	if r := authReq(t, http.MethodPost, api+"/admin/shutdown", bobAccess, ""); r.StatusCode != http.StatusForbidden {
		r.Body.Close()
		t.Fatalf("member POST /admin/shutdown = %d, want 403", r.StatusCode)
	}

	// Admin lists users (owner seed + alice + bob).
	listed := authReq(t, http.MethodGet, api+"/users", adminAccess, "")
	defer listed.Body.Close()
	if listed.StatusCode != http.StatusOK {
		t.Fatalf("admin GET /users = %d, want 200", listed.StatusCode)
	}

	// Changing bob's role revokes his sessions — his refresh token stops working.
	patch := authReq(t, http.MethodPatch, api+"/users/"+bob.ID, adminAccess, `{"role":"restricted"}`)
	patch.Body.Close()
	if patch.StatusCode != http.StatusOK {
		t.Fatalf("patch role = %d, want 200", patch.StatusCode)
	}
	if r := sendJSON(t, http.MethodPost, api+"/auth/refresh", `{"refresh":"`+bobRefresh+`"}`); r.StatusCode != http.StatusUnauthorized {
		r.Body.Close()
		t.Fatalf("refresh after role change = %d, want 401 (sessions revoked)", r.StatusCode)
	}

	// The owner account can't be deleted; a normal user can.
	if r := authReq(t, http.MethodDelete, api+"/users/owner", adminAccess, ""); r.StatusCode != http.StatusBadRequest {
		r.Body.Close()
		t.Fatalf("delete owner = %d, want 400", r.StatusCode)
	}
	if r := authReq(t, http.MethodDelete, api+"/users/"+bob.ID, adminAccess, ""); r.StatusCode != http.StatusNoContent {
		r.Body.Close()
		t.Fatalf("delete bob = %d, want 204", r.StatusCode)
	}
}
