package sqlstore

import (
	"context"
	"errors"
	"testing"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
)

func TestTrackRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	repo := store.Tracks()

	tr := domain.Track{ID: ulid.New(), UserID: "owner", Name: "Flashpoint", CreatedAt: 1, UpdatedAt: 1}
	if _, err := repo.CreateTrack(ctx, tr); err != nil {
		t.Fatalf("create track: %v", err)
	}

	// Owner-scoped: visible to the owner, invisible to anyone else.
	if got, err := repo.GetTrack(ctx, "owner", tr.ID); err != nil || got.Name != "Flashpoint" {
		t.Fatalf("owner get = %+v, err %v", got, err)
	}
	if _, err := repo.GetTrack(ctx, "intruder", tr.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-user get = %v, want ErrNotFound", err)
	}

	tr.Name = "Flashpoint (2011)"
	tr.UpdatedAt = 2
	if err := repo.RenameTrack(ctx, tr); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if got, _ := repo.GetTrack(ctx, "owner", tr.ID); got.Name != "Flashpoint (2011)" {
		t.Fatalf("renamed = %q", got.Name)
	}
	if tracks, _ := repo.ListTracks(ctx, "owner"); len(tracks) != 1 {
		t.Fatalf("list = %d, want 1", len(tracks))
	}
}

func TestTrackIssuesOverlay(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	lib := seedLibrary(t, store)
	series := seedSeries(t, store, lib, "Aquaman")
	repo := store.Tracks()

	tr := domain.Track{ID: ulid.New(), UserID: "owner", Name: "Grifter", CreatedAt: 1, UpdatedAt: 1}
	if _, err := repo.CreateTrack(ctx, tr); err != nil {
		t.Fatalf("create track: %v", err)
	}

	// A track-attached issue and a series-attached (gap) issue, both marked read on add is off.
	trackIssueID := ulid.New()
	seriesIssueID := ulid.New()
	err := repo.AddIssues(ctx, []domain.TrackIssue{
		{ID: trackIssueID, UserID: "owner", TrackID: tr.ID, Number: "1", Sort: 1, CreatedAt: 1},
		{ID: seriesIssueID, UserID: "owner", SeriesID: series, Number: "24", Sort: 24, CreatedAt: 1},
		// Duplicate number in the same track — the unique index drops it.
		{ID: ulid.New(), UserID: "owner", TrackID: tr.ID, Number: "1", Sort: 1, CreatedAt: 1},
	})
	if err != nil {
		t.Fatalf("add issues: %v", err)
	}

	overlay, err := repo.OverlayIssues(ctx, "owner")
	if err != nil {
		t.Fatalf("overlay: %v", err)
	}
	if len(overlay) != 2 {
		t.Fatalf("overlay count = %d, want 2 (dup dropped)", len(overlay))
	}

	// Mark the gap issue read, then confirm the flag round-trips.
	if err := repo.SetIssueRead(ctx, "owner", seriesIssueID, true, 99); err != nil {
		t.Fatalf("mark: %v", err)
	}
	got, err := repo.GetIssue(ctx, "owner", seriesIssueID)
	if err != nil || !got.Read || got.ReadAt != 99 || got.SeriesID != series {
		t.Fatalf("issue after mark = %+v, err %v", got, err)
	}

	// Cross-user cannot mark or remove.
	if err := repo.SetIssueRead(ctx, "intruder", seriesIssueID, false, 0); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-user mark = %v, want ErrNotFound", err)
	}
	if err := repo.RemoveIssue(ctx, "owner", trackIssueID); err != nil {
		t.Fatalf("remove: %v", err)
	}

	// Deleting the track cascades its remaining issues away.
	if err := repo.DeleteTrack(ctx, "owner", tr.ID); err != nil {
		t.Fatalf("delete track: %v", err)
	}
	remaining, _ := repo.OverlayIssues(ctx, "owner")
	if len(remaining) != 1 || remaining[0].ID != seriesIssueID {
		t.Fatalf("after delete = %+v, want only the series issue", remaining)
	}
}
