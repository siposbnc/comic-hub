package sqlite

import (
	"context"
	"sync"
	"testing"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
)

// TestSeriesUpsertConvergesOnFolder reproduces the concurrent-scan race that previously
// created duplicate series: two scans both miss the by-folder lookup, mint different ULIDs,
// and upsert the same folder. The UNIQUE(library_id, folder_path) index plus the upsert's
// recovery must converge them onto one row, returning the surviving id.
func TestSeriesUpsertConvergesOnFolder(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	lib := seedLibrary(t, store)
	const folder = `C:\DC\Batman`

	mk := func(id string) domain.Series {
		return domain.Series{
			ID: id, LibraryID: lib, FolderPath: folder,
			Name: "Batman", SortName: "Batman", CreatedAt: 1, UpdatedAt: 1,
		}
	}

	first, err := store.Series().Upsert(ctx, mk(ulid.New()))
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	// A second scan with a *different* id for the same folder must not create a new row.
	second, err := store.Series().Upsert(ctx, mk(ulid.New()))
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("second upsert returned id %q, want converged %q", second.ID, first.ID)
	}

	all, err := store.Series().ListByLibrary(ctx, lib)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("got %d series rows for one folder, want 1", len(all))
	}
}

// TestSeriesUpsertConcurrent hammers Upsert from many goroutines with distinct ids but the
// same folder — the real shape of two scan worker pools racing — and asserts exactly one row
// survives.
func TestSeriesUpsertConcurrent(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	lib := seedLibrary(t, store)
	const folder = `C:\DC\Detective Comics`

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = store.Series().Upsert(ctx, domain.Series{
				ID: ulid.New(), LibraryID: lib, FolderPath: folder,
				Name: "Detective Comics", SortName: "Detective Comics", CreatedAt: 1, UpdatedAt: 1,
			})
		}()
	}
	wg.Wait()

	all, err := store.Series().ListByLibrary(ctx, lib)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("got %d series rows under concurrency, want 1", len(all))
	}
}
