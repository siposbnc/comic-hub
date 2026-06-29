package http

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/service/metadata"
)

// providerSettingsReq is the PUT body. Pointers distinguish "leave unchanged" (omitted)
// from "clear" (empty string).
type providerSettingsReq struct {
	ComicVineAPIKey *string `json:"comicVineApiKey"`
	MetronUsername  *string `json:"metronUsername"`
	MetronPassword  *string `json:"metronPassword"`
	WriteSidecar    *bool   `json:"writeSidecar"`
}

// providerSettingsDTO is the GET response: editable, non-secret state only. Secrets are
// never returned — only whether each is set.
type providerSettingsDTO struct {
	ComicVine    providerCV     `json:"comicvine"`
	Metron       providerMetron `json:"metron"`
	WriteSidecar bool           `json:"writeSidecar"`
}
type providerCV struct {
	Configured bool `json:"configured"`
}
type providerMetron struct {
	Configured bool   `json:"configured"`
	Username   string `json:"username"`
}

// currentProviderSettings builds the settings DTO from live provider status + stored values.
func currentProviderSettings(ctx context.Context, repo domain.Repository, meta *metadata.Service, cfg providerEnv) providerSettingsDTO {
	return providerSettingsDTO{
		ComicVine: providerCV{Configured: meta.Has("comicvine")},
		Metron: providerMetron{
			Configured: meta.Has("metron"),
			// Username isn't secret — echo the effective value so the field is prefilled.
			Username: settingOr(ctx, repo, domain.SettingMetronUsername, cfg.MetronUsername),
		},
		WriteSidecar: settingOr(ctx, repo, domain.SettingWriteSidecar, "") == "true",
	}
}

// handleGetProviderSettings returns provider configuration for the settings screen.
func handleGetProviderSettings(repo domain.Repository, meta *metadata.Service, cfg providerEnv) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, currentProviderSettings(r.Context(), repo, meta, cfg))
	}
}

// handlePutProviderSettings persists provider credentials and live-reconfigures matching.
func handlePutProviderSettings(repo domain.Repository, meta *metadata.Service, cfg providerEnv, reload func(context.Context) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req providerSettingsReq
		if !decodeJSON(w, r, &req) {
			return
		}
		set := func(key string, v *string) bool {
			if v == nil {
				return true
			}
			if err := repo.Settings().Set(r.Context(), key, strings.TrimSpace(*v)); err != nil {
				writeDomainError(w, err)
				return false
			}
			return true
		}
		if !set(domain.SettingComicVineAPIKey, req.ComicVineAPIKey) ||
			!set(domain.SettingMetronUsername, req.MetronUsername) ||
			!set(domain.SettingMetronPassword, req.MetronPassword) {
			return
		}
		if req.WriteSidecar != nil {
			val := "false"
			if *req.WriteSidecar {
				val = "true"
			}
			if err := repo.Settings().Set(r.Context(), domain.SettingWriteSidecar, val); err != nil {
				writeDomainError(w, err)
				return
			}
		}
		if err := reload(r.Context()); err != nil {
			writeError(w, http.StatusInternalServerError, "reload_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, currentProviderSettings(r.Context(), repo, meta, cfg))
	}
}

// providerEnv carries the env-var fallbacks used when a setting hasn't been saved.
type providerEnv struct {
	ComicVineAPIKey string
	MetronUsername  string
	MetronPassword  string
}

// settingOr returns a persisted setting, falling back to the env default when unset.
func settingOr(ctx context.Context, repo domain.Repository, key, fallback string) string {
	v, err := repo.Settings().Get(ctx, key)
	if errors.Is(err, domain.ErrNotFound) {
		return fallback
	}
	if err != nil {
		return fallback
	}
	return v
}
