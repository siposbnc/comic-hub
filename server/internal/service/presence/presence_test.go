package presence

import (
	"testing"
	"time"
)

type change struct {
	e      Entry
	active bool
}

func newTest(ttl time.Duration) (*Tracker, *[]change, *time.Time) {
	t := New(ttl)
	now := time.Unix(1_700_000_000, 0)
	t.now = func() time.Time { return now }
	var changes []change
	t.OnChange(func(e Entry, active bool) { changes = append(changes, change{e, active}) })
	return t, &changes, &now
}

func TestTouchSnapshotClear(t *testing.T) {
	tr, changes, _ := newTest(time.Minute)

	tr.Touch(Entry{UserID: "u1", DisplayName: "Alex", BookID: "b1", Page: 3})
	tr.Touch(Entry{UserID: "u2", BookID: "b2"})
	if got := len(tr.Snapshot()); got != 2 {
		t.Fatalf("Snapshot() len = %d, want 2", got)
	}

	// Re-touch replaces, not duplicates; latest book wins.
	tr.Touch(Entry{UserID: "u1", BookID: "b9", Page: 7})
	snap := tr.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("Snapshot() after re-touch len = %d, want 2", len(snap))
	}
	for _, e := range snap {
		if e.UserID == "u1" && e.BookID != "b9" {
			t.Errorf("u1 book = %s, want b9", e.BookID)
		}
	}

	tr.Clear("u1")
	if got := len(tr.Snapshot()); got != 1 {
		t.Fatalf("Snapshot() after clear len = %d, want 1", got)
	}
	// Clearing an absent user notifies nothing.
	before := len(*changes)
	tr.Clear("u1")
	if len(*changes) != before {
		t.Error("Clear of absent user should not notify")
	}

	// 3 touches (active) + 1 clear (inactive).
	var actives, clears int
	for _, c := range *changes {
		if c.active {
			actives++
		} else {
			clears++
		}
	}
	if actives != 3 || clears != 1 {
		t.Errorf("changes = %d active / %d cleared, want 3/1", actives, clears)
	}
}

func TestIgnoresIncompleteEntries(t *testing.T) {
	tr, changes, _ := newTest(time.Minute)
	tr.Touch(Entry{UserID: "", BookID: "b1"})
	tr.Touch(Entry{UserID: "u1", BookID: ""})
	if len(tr.Snapshot()) != 0 || len(*changes) != 0 {
		t.Error("incomplete entries must be ignored")
	}
}

func TestExpiry(t *testing.T) {
	tr, changes, now := newTest(time.Minute)
	tr.Touch(Entry{UserID: "u1", BookID: "b1"})
	tr.Touch(Entry{UserID: "u2", BookID: "b2"})

	*now = now.Add(30 * time.Second)
	tr.Touch(Entry{UserID: "u2", BookID: "b2", Page: 5}) // keeps u2 fresh

	*now = now.Add(45 * time.Second) // u1 is now 75s old, u2 45s
	if got := len(tr.Snapshot()); got != 1 {
		t.Fatalf("Snapshot() len = %d, want 1 (u1 expired)", got)
	}

	tr.sweep()
	last := (*changes)[len(*changes)-1]
	if last.active || last.e.UserID != "u1" {
		t.Errorf("sweep should notify u1 cleared, got %+v", last)
	}
	if got := len(tr.Snapshot()); got != 1 {
		t.Fatalf("after sweep len = %d, want 1", got)
	}
}
