package domain

import "context"

// Settings keys (string constants so the store and services agree). Provider credentials
// are stored here so they can be set at runtime via the UI, overriding the env defaults.
const (
	SettingComicVineAPIKey = "provider.comicvine.apikey"
	SettingMetronUsername  = "provider.metron.username"
	SettingMetronPassword  = "provider.metron.password"
)

// SettingsRepository persists app key/value settings.
type SettingsRepository interface {
	// Get returns a setting's value, or ErrNotFound if unset.
	Get(ctx context.Context, key string) (string, error)
	// Set upserts a setting.
	Set(ctx context.Context, key, value string) error
	// GetAll returns every setting as a map.
	GetAll(ctx context.Context) (map[string]string, error)
}
