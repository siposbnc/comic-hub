// Package sidecar writes a book's catalog metadata back into its archive as a ComicInfo.xml
// (only .cbz today). It's an opt-in step run after an online match so matched metadata
// travels with the file and survives re-import. See docs/04-server.md §6.3.
package sidecar

import (
	"context"
	"os"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/archive"
	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/contenthash"
	"github.com/siposbnc/comic-hub/server/internal/scanner"
)

// Writer serializes catalog metadata into book archives.
type Writer struct {
	repo          domain.Repository
	hashThreshold int64
}

// New constructs a sidecar Writer. hashThreshold must match the scanner's so the rewritten
// file's content hash stays comparable across scans.
func New(repo domain.Repository, hashThreshold int64) *Writer {
	return &Writer{repo: repo, hashThreshold: hashThreshold}
}

// Write builds a ComicInfo.xml from the book's stored metadata and writes it into the
// archive, then refreshes the book's size/mtime/hash so a later scan doesn't see a phantom
// change. Books in a format we can't write (e.g. .cbr) are skipped (no error).
func (w *Writer) Write(ctx context.Context, bookID string) error {
	book, err := w.repo.Books().Get(ctx, bookID)
	if err != nil {
		return err
	}
	if book.IsCorrupt || !archive.CanWriteSidecar(book.FilePath) {
		return nil
	}

	series, _ := w.repo.Series().Get(ctx, book.SeriesID)
	credits, _ := w.repo.Metadata().BookCredits(ctx, bookID)
	genres, _ := w.repo.Metadata().BookGenres(ctx, bookID)
	characters, _ := w.repo.Metadata().BookCharacters(ctx, bookID)

	xmlBytes, err := scanner.BuildComicInfoXML(scanner.SidecarData{
		Series:      series.Name,
		Number:      book.Number,
		Title:       book.Title,
		Summary:     book.Summary,
		Publisher:   series.Publisher,
		AgeRating:   book.AgeRating,
		Language:    book.Language,
		Volume:      book.Volume,
		ReleaseDate: book.ReleaseDate,
		ReadingDir:  series.ReadingDir,
		PageCount:   book.PageCount,
		Credits:     credits,
		Genres:      genres,
		Characters:  characters,
	})
	if err != nil {
		return err
	}
	if err := archive.WriteCBZComicInfo(book.FilePath, xmlBytes); err != nil {
		return err
	}

	// Re-stat + re-hash so the change we just made doesn't look like an external edit on the
	// next scan (which would otherwise re-process it or flag a duplicate).
	fi, err := os.Stat(book.FilePath)
	if err != nil {
		return nil // written ok; just couldn't refresh stats
	}
	hash, herr := contenthash.OfFile(book.FilePath, w.hashThreshold)
	if herr != nil {
		return nil
	}
	book.FileSize = fi.Size()
	book.FileMTime = fi.ModTime().UnixMilli()
	book.ContentHash = hash
	book.UpdatedAt = time.Now().UnixMilli()
	_, _ = w.repo.Books().Upsert(ctx, book)
	return nil
}
