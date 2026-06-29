package archive

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
)

// WriteCBZComicInfo writes (or replaces) the ComicInfo.xml entry inside a .cbz, preserving
// every other entry. It rebuilds the archive into a temp file in the same directory and
// atomically renames it over the original, so a crash mid-write can't corrupt the comic.
func WriteCBZComicInfo(filePath string, comicInfoXML []byte) error {
	zr, err := zip.OpenReader(filePath)
	if err != nil {
		return fmt.Errorf("open cbz: %w", err)
	}
	defer zr.Close()

	tmp, err := os.CreateTemp(filepath.Dir(filePath), ".comichub-cbz-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// Clean up the temp file on any failure path (no-op once the rename succeeds).
	committed := false
	defer func() {
		_ = tmp.Close()
		if !committed {
			_ = os.Remove(tmpName)
		}
	}()

	zw := zip.NewWriter(tmp)
	for _, f := range zr.File {
		if isSidecarEntry(f.Name) {
			continue // drop any existing ComicInfo.xml; we write a fresh one below
		}
		if err := copyZipEntry(zw, f); err != nil {
			_ = zw.Close()
			return err
		}
	}
	// Add the new ComicInfo.xml at the archive root (deflate-compressed).
	w, err := zw.CreateHeader(&zip.FileHeader{Name: "ComicInfo.xml", Method: zip.Deflate})
	if err != nil {
		_ = zw.Close()
		return err
	}
	if _, err := w.Write(comicInfoXML); err != nil {
		_ = zw.Close()
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	// Release the read handle before replacing the file (required on Windows).
	_ = zr.Close()
	if err := os.Rename(tmpName, filePath); err != nil {
		return err
	}
	committed = true
	return nil
}

// copyZipEntry copies one entry verbatim (preserving its name, mode, and modtime).
func copyZipEntry(zw *zip.Writer, f *zip.File) error {
	hdr := f.FileHeader // copy
	w, err := zw.CreateHeader(&hdr)
	if err != nil {
		return err
	}
	if f.FileInfo().IsDir() {
		return nil
	}
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("read entry %q: %w", f.Name, err)
	}
	defer rc.Close()
	if _, err := io.Copy(w, io.LimitReader(rc, maxEntryBytes)); err != nil {
		return fmt.Errorf("copy entry %q: %w", f.Name, err)
	}
	return nil
}

// CanWriteSidecar reports whether a file is a format we can write a ComicInfo.xml into
// (only ZIP-based .cbz today; .cbr/.cb7/.cbt are read-only here).
func CanWriteSidecar(filePath string) bool {
	switch path.Ext(filePath) {
	case ".cbz", ".zip", ".CBZ", ".ZIP":
		return true
	default:
		return false
	}
}
