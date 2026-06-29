package http

import (
	"net/http"
	"testing"
)

func TestProviderSettingsEndpoint(t *testing.T) {
	srv, _ := newScanServer(t)
	api := srv + "/api/v1"

	// Nothing configured yet.
	var got providerSettingsDTO
	getJSON(t, api+"/settings/providers", &got)
	if got.ComicVine.Configured {
		t.Fatalf("comicvine should start unconfigured: %+v", got)
	}

	// Save a Comic Vine key → it reconfigures live and reports configured.
	var saved providerSettingsDTO
	decode(t, sendJSON(t, http.MethodPut, api+"/settings/providers",
		`{"comicVineApiKey":"secret-key"}`), &saved)
	if !saved.ComicVine.Configured {
		t.Fatalf("comicvine should be configured after save: %+v", saved)
	}

	// /providers reflects it too (the secret itself is never returned).
	var provs struct {
		Providers []struct {
			Name       string `json:"name"`
			Configured bool   `json:"configured"`
		} `json:"providers"`
	}
	getJSON(t, api+"/providers", &provs)
	var cvConfigured bool
	for _, p := range provs.Providers {
		if p.Name == "comicvine" {
			cvConfigured = p.Configured
		}
	}
	if !cvConfigured {
		t.Fatalf("/providers should show comicvine configured: %+v", provs.Providers)
	}

	// Metron username round-trips (non-secret), password stays write-only.
	decode(t, sendJSON(t, http.MethodPut, api+"/settings/providers",
		`{"metronUsername":"reader42"}`), &saved)
	getJSON(t, api+"/settings/providers", &got)
	if got.Metron.Username != "reader42" {
		t.Fatalf("metron username = %q, want reader42", got.Metron.Username)
	}
}
