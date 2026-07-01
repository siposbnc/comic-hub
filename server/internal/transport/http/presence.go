package http

import (
	"net/http"
	"sort"

	"github.com/siposbnc/comic-hub/server/internal/access"
	"github.com/siposbnc/comic-hub/server/internal/service/presence"
)

// handlePresence returns the current "now reading" set (Milestone E), most recent
// first. Entries above the viewer's content ceiling are withheld — same rule as the WS
// presence events and browse filtering. The viewer's own entry is included; the UI
// decides whether to render "you".
func handlePresence(t *presence.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ceiling := access.CeilingFrom(r.Context())
		entries := t.Snapshot()
		items := make([]presence.Entry, 0, len(entries))
		for _, e := range entries {
			if access.Allowed(ceiling, e.AgeRating) {
				items = append(items, e)
			}
		}
		sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt > items[j].UpdatedAt })
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}
