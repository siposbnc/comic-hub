package scanner

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/archive"
	"github.com/siposbnc/comic-hub/server/internal/domain"
	imageproc "github.com/siposbnc/comic-hub/server/internal/image"
	"github.com/siposbnc/comic-hub/server/internal/pkg/contenthash"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
)

// ProgressFunc reports scan progress (files done / total). It matches jobs.ProgressFunc
// so the scanner can be driven directly by a job handler.
type ProgressFunc func(done, total int64)

// JobPayload is the JSON payload for a scan job.
type JobPayload struct {
	LibraryID string `json:"libraryId"`
	Full      bool   `json:"full"`
}

// Scanner turns a library's folder tree into catalog rows. It is incremental
// (change-detected by size+mtime), idempotent (safe to re-run), and resilient
// (a corrupt file is flagged, never aborting the scan). See docs/04-server.md §3.
type Scanner struct {
	repo          domain.Repository
	registry      *archive.Registry
	logger        *slog.Logger
	hashThreshold int64
	proc          imageproc.Processor
}

// New constructs a scanner. hashThreshold is the file size above which content hashing
// switches to sampled mode.
func New(repo domain.Repository, registry *archive.Registry, logger *slog.Logger, hashThreshold int64) *Scanner {
	return &Scanner{
		repo:          repo,
		registry:      registry,
		logger:        logger,
		hashThreshold: hashThreshold,
		proc:          imageproc.New(),
	}
}

// Scan walks a library's roots and upserts its catalog. When full is false, files whose
// size and mtime are unchanged since the last scan are skipped.
func (s *Scanner) Scan(ctx context.Context, libraryID string, full bool, progress ProgressFunc) error {
	lib, err := s.repo.Libraries().Get(ctx, libraryID)
	if err != nil {
		return err
	}

	files, err := s.collect(ctx, lib.Roots)
	if err != nil {
		return err
	}
	total := int64(len(files))
	if progress != nil {
		progress(0, total)
	}

	for i, path := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := s.processFile(ctx, lib, full, path); err != nil {
			// Per-file failures are logged and recorded, never fatal to the whole scan.
			s.logger.Warn("scan: file failed", "path", path, "err", err)
		}
		if progress != nil {
			progress(int64(i+1), total)
		}
	}

	// Drop series left empty (e.g. old per-subfolder series after the grouping rule changed,
	// or a series whose every file was moved/removed). Best-effort; never fail the scan.
	if n, err := s.repo.Series().DeleteEmpty(ctx, libraryID); err != nil {
		s.logger.Warn("scan: prune empty series failed", "library", libraryID, "err", err)
	} else if n > 0 {
		s.logger.Info("scan: pruned empty series", "library", libraryID, "count", n)
	}
	return nil
}

// collect walks the roots and returns every supported comic file path.
func (s *Scanner) collect(ctx context.Context, roots []string) ([]string, error) {
	var files []string
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				// Unreadable dir/file: skip it, keep scanning the rest.
				return nil //nolint:nilerr
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if d.IsDir() {
				if isHiddenOrSystem(d.Name()) && path != root {
					return fs.SkipDir
				}
				return nil
			}
			if s.registry.Supports(path) {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

func (s *Scanner) processFile(ctx context.Context, lib domain.Library, full bool, path string) error {
	fi, err := statFile(path)
	if err != nil {
		return err
	}
	size := fi.Size()
	mtime := fi.ModTime().UnixMilli()

	existing, getErr := s.repo.Books().GetByPath(ctx, path)
	haveExisting := getErr == nil
	if haveExisting && !full && !existing.IsCorrupt &&
		existing.FileSize == size && existing.FileMTime == mtime {
		return nil // unchanged
	}

	hash, err := contenthash.OfFile(path, s.hashThreshold)
	if err != nil {
		return s.persistCorrupt(ctx, lib, existing, haveExisting, path, size, mtime, "")
	}

	// A file with no row at this path but matching the content of a row whose file has
	// vanished is a move/rename: reuse that row (updating its path) instead of creating a
	// new book and orphaning the old one.
	if !haveExisting {
		if moved, ok := s.reconcileMovedBook(ctx, lib.ID, path, hash); ok {
			existing, haveExisting = moved, true
		}
	}

	src, err := s.registry.Open(path)
	if err != nil {
		return s.persistCorrupt(ctx, lib, existing, haveExisting, path, size, mtime, hash)
	}
	defer src.Close()

	var ci ComicInfo
	haveCI := false
	if r, ok := src.Sidecar(); ok {
		if parsed, perr := ParseComicInfo(r); perr == nil {
			ci, haveCI = parsed, true
		}
	}

	pages := buildPages(src.Pages(), ci)
	s.fillPageDimensions(src, pages)
	series, err := s.resolveSeries(ctx, lib, path, ci, haveCI)
	if err != nil {
		return err
	}

	book := s.buildBook(lib, series.ID, existing, haveExisting, path, size, mtime, hash, len(pages), ci, haveCI)
	saved, err := s.repo.Books().Upsert(ctx, book)
	if err != nil {
		return err
	}
	if !haveExisting {
		// A newly cataloged book may be the return of a deleted one (series rescan,
		// re-imported file): re-attach any stale reading-list entries with its hash.
		if _, err := s.repo.ReadingLists().RelinkStaleByHash(ctx, hash, saved.ID); err != nil {
			s.logger.Warn("relink stale reading-list entries", "book", saved.ID, "err", err)
		}
	}
	return s.repo.Books().ReplacePages(ctx, saved.ID, pages)
}

// buildPages converts archive page metadata into domain pages, enriching with
// ComicInfo per-page types/double-spread flags when present.
func buildPages(infos []archive.PageInfo, ci ComicInfo) []domain.Page {
	pages := make([]domain.Page, len(infos))
	for i, in := range infos {
		// BookID is supplied by ReplacePages, which keys inserts on its bookID argument.
		p := domain.Page{
			Index:    in.Index,
			FileName: in.FileName,
			Size:     in.Size,
		}
		if meta, ok := ci.Pages[in.Index]; ok {
			p.PageType = meta.Type
			p.IsDouble = meta.Double
		}
		pages[i] = p
	}
	return pages
}

// fillPageDimensions decodes each page's header to record its pixel size, so the reader's
// manifest carries real dimensions (for layout + double-spread detection) instead of zeros.
// Best-effort: a page whose format has no pure-Go decoder (e.g. AVIF) keeps zero dims, and
// a single unreadable page never fails the scan.
func (s *Scanner) fillPageDimensions(src archive.PageSource, pages []domain.Page) {
	for i := range pages {
		rc, _, err := src.Page(i)
		if err != nil {
			continue
		}
		w, h, err := s.proc.Dimensions(rc)
		rc.Close()
		if err == nil {
			pages[i].Width = w
			pages[i].Height = h
		}
	}
}

func (s *Scanner) resolveSeries(ctx context.Context, lib domain.Library, path string, ci ComicInfo, haveCI bool) (domain.Series, error) {
	// The series is the first folder beneath a library root, NOT the file's immediate
	// parent — so subfolders inside a series (variant covers, volumes, annuals) all group
	// under the one series instead of fragmenting into separate ones. ComicInfo.Series
	// still overrides the name. The filename only supplies per-issue fields.
	folder := seriesFolder(lib.Roots, path)
	name := filepath.Base(folder)
	if haveCI && ci.Series != "" {
		name = ci.Series
	}

	readingDir := domain.LTR
	if lib.Kind == "manga" {
		readingDir = domain.RTL
	}
	if haveCI && ci.ReadingDir != "" {
		readingDir = ci.ReadingDir
	}

	now := time.Now().UnixMilli()
	existing, err := s.repo.Series().GetByFolder(ctx, lib.ID, folder)
	series := domain.Series{
		LibraryID:  lib.ID,
		FolderPath: folder,
		Name:       name,
		SortName:   sortName(name),
		ReadingDir: readingDir,
		UpdatedAt:  now,
	}
	if err == nil {
		series.ID = existing.ID
		series.CreatedAt = existing.CreatedAt
		// The scanner has no file-level source for these — they only ever come from a
		// provider match (WriteMatch) or the user — so a rescan must carry them over,
		// not wipe them via the upsert.
		series.Year = existing.Year
		series.Publisher = existing.Publisher
		series.Description = existing.Description
		series.CoverBookID = existing.CoverBookID
	} else {
		series.ID = ulid.New()
		series.CreatedAt = now
	}
	return s.repo.Series().Upsert(ctx, series)
}

func (s *Scanner) buildBook(
	lib domain.Library, seriesID string, existing domain.Book, haveExisting bool,
	path string, size, mtime int64, hash string, pageCount int, ci ComicInfo, haveCI bool,
) domain.Book {
	parsed := ParseFilename(path)

	number := parsed.Number
	volume := parsed.Volume
	var releaseDate int64
	if parsed.Year > 0 {
		releaseDate = time.Date(parsed.Year, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	}
	title, ageRating, language, summary := "", "", "", ""
	state := domain.MetaNone

	if haveCI {
		state = domain.MetaSidecar
		if ci.Number != "" {
			number = ci.Number
		}
		if ci.Volume > 0 {
			volume = ci.Volume
		}
		if ci.ReleaseDate > 0 {
			releaseDate = ci.ReleaseDate
		}
		title, ageRating, language, summary = ci.Title, ci.AgeRating, ci.Language, ci.Summary
	}

	now := time.Now().UnixMilli()
	book := domain.Book{
		SeriesID:      seriesID,
		LibraryID:     lib.ID,
		FilePath:      path,
		FileFormat:    formatOf(path),
		FileSize:      size,
		FileMTime:     mtime,
		ContentHash:   hash,
		PageCount:     pageCount,
		Title:         title,
		Number:        number,
		SortNumber:    SortNumber(number),
		Volume:        volume,
		ReleaseDate:   releaseDate,
		AgeRating:     ageRating,
		Language:      language,
		Summary:       summary,
		CoverPage:     0,
		MetadataState: state,
		UpdatedAt:     now,
	}
	if haveExisting {
		book.ID = existing.ID
		book.AddedAt = existing.AddedAt
		// Provider-matched and user-locked metadata outrank anything re-derivable from
		// the file: a rescan refreshes file facts (size, hash, pages) but must not
		// overwrite these fields or downgrade their state back to sidecar/none.
		if existing.MetadataState == domain.MetaMatched || existing.MetadataState == domain.MetaLocked {
			book.Title = existing.Title
			book.Number = existing.Number
			book.SortNumber = existing.SortNumber
			book.Volume = existing.Volume
			book.ReleaseDate = existing.ReleaseDate
			book.AgeRating = existing.AgeRating
			book.Language = existing.Language
			book.Summary = existing.Summary
			book.CoverPage = existing.CoverPage
			book.MetadataState = existing.MetadataState
		}
	} else {
		book.ID = ulid.New()
		book.AddedAt = now
	}
	// Classify from the resolved number + filename + sidecar Format, so a rescan always
	// reflects the current facts (independent of metadata state).
	ciFormat := ""
	if haveCI {
		ciFormat = ci.Format
	}
	book.Kind = classifyKind(book.Number, path, ciFormat)
	return book
}

// reconcileMovedBook looks for an existing book in the library with the same content hash
// whose file no longer exists on disk — i.e. the same file under a new path. It returns
// that row (so the caller reuses its id) and true. Genuine duplicates (both files still
// present) are left alone, so they surface separately in Library Health.
func (s *Scanner) reconcileMovedBook(ctx context.Context, libraryID, newPath, hash string) (domain.Book, bool) {
	cands, err := s.repo.Books().ByContentHash(ctx, libraryID, hash)
	if err != nil {
		return domain.Book{}, false
	}
	for _, c := range cands {
		if c.FilePath == newPath {
			continue
		}
		if _, err := statFile(c.FilePath); err != nil {
			return c, true // old file gone → this is a move
		}
	}
	return domain.Book{}, false
}

// persistCorrupt records a file that could not be opened/hashed so it surfaces in
// Library Health, without aborting the scan.
func (s *Scanner) persistCorrupt(
	ctx context.Context, lib domain.Library, existing domain.Book, haveExisting bool,
	path string, size, mtime int64, hash string,
) error {
	series, err := s.resolveSeries(ctx, lib, path, ComicInfo{}, false)
	if err != nil {
		return err
	}
	parsed := ParseFilename(path)
	now := time.Now().UnixMilli()
	book := domain.Book{
		SeriesID:      series.ID,
		LibraryID:     lib.ID,
		FilePath:      path,
		FileFormat:    formatOf(path),
		FileSize:      size,
		FileMTime:     mtime,
		ContentHash:   hash,
		PageCount:     0,
		Number:        parsed.Number,
		SortNumber:    SortNumber(parsed.Number),
		Volume:        parsed.Volume,
		MetadataState: domain.MetaNone,
		IsCorrupt:     true,
		UpdatedAt:     now,
	}
	if haveExisting {
		book.ID = existing.ID
		book.AddedAt = existing.AddedAt
	} else {
		book.ID = ulid.New()
		book.AddedAt = now
	}
	_, err = s.repo.Books().Upsert(ctx, book)
	return err
}

// seriesFolder returns the folder that defines a file's series: the first directory beneath
// the matching library root. Files sitting directly in a root use the root itself. This is
// what makes nested folders (variant covers, volumes, annuals) part of their parent series.
func seriesFolder(roots []string, filePath string) string {
	for _, root := range roots {
		root = filepath.Clean(root)
		rel, err := filepath.Rel(root, filePath)
		if err != nil {
			continue
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue // filePath is not under this root
		}
		parts := strings.Split(rel, string(filepath.Separator))
		if len(parts) <= 1 {
			return root // file is directly in the root
		}
		return filepath.Join(root, parts[0])
	}
	// No root matched (shouldn't happen for scanned files): fall back to the parent folder.
	return filepath.Dir(filePath)
}

func formatOf(path string) string {
	return strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
}

func isHiddenOrSystem(name string) bool {
	return strings.HasPrefix(name, ".") || strings.EqualFold(name, "$RECYCLE.BIN") ||
		strings.EqualFold(name, "System Volume Information") || name == "@eaDir"
}

// sortName produces a sort key that moves a leading article to the end
// ("The Sandman" -> "Sandman, The").
func sortName(name string) string {
	for _, art := range []string{"The ", "A ", "An "} {
		if strings.HasPrefix(name, art) {
			return strings.TrimSpace(name[len(art):]) + ", " + strings.TrimSpace(art)
		}
	}
	return name
}

func statFile(path string) (os.FileInfo, error) { return os.Stat(path) }
