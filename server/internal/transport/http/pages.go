package http

import (
	"bytes"
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/siposbnc/comic-hub/server/internal/image"
	"github.com/siposbnc/comic-hub/server/internal/service/reader"
)

func handleManifest(rdr *reader.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m, err := rdr.Manifest(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, m)
	}
}

func handlePage(rdr *reader.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idx, err := strconv.Atoi(chi.URLParam(r, "idx"))
		if err != nil || idx < 0 {
			writeError(w, http.StatusBadRequest, "invalid_page", "page index must be a non-negative integer")
			return
		}
		img, err := rdr.Page(r.Context(), chi.URLParam(r, "id"), idx, pageOptionsFromQuery(r))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		serveImage(w, r, img)
	}
}

func handlePageThumb(rdr *reader.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idx, err := strconv.Atoi(chi.URLParam(r, "idx"))
		if err != nil || idx < 0 {
			writeError(w, http.StatusBadRequest, "invalid_page", "page index must be a non-negative integer")
			return
		}
		img, err := rdr.Thumb(r.Context(), chi.URLParam(r, "id"), idx)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		serveImage(w, r, img)
	}
}

func handleCover(rdr *reader.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		width := 0
		if v, err := strconv.Atoi(r.URL.Query().Get("w")); err == nil && v > 0 {
			width = v
		}
		img, err := rdr.Cover(r.Context(), chi.URLParam(r, "id"), width)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		serveImage(w, r, img)
	}
}

func handlePrefetch(rdr *reader.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			From  int `json:"from"`
			Count int `json:"count"`
		}
		if r.ContentLength > 0 && !decodeJSON(w, r, &req) {
			return
		}
		if req.Count > 0 {
			// Best-effort: warm the cache without blocking the response. Use a detached
			// context since the work outlives this request.
			id := chi.URLParam(r, "id")
			go rdr.Prefetch(context.Background(), id, req.From, req.Count)
		}
		w.WriteHeader(http.StatusAccepted)
	}
}

// pageOptionsFromQuery parses ?w=&fmt=&q=. WebP/AVIF are not produced by the pure-Go
// pipeline yet, so they fall back to JPEG (honest Content-Type on the response).
func pageOptionsFromQuery(r *http.Request) reader.PageOptions {
	q := r.URL.Query()
	var opts reader.PageOptions
	if v, err := strconv.Atoi(q.Get("w")); err == nil && v > 0 {
		opts.Width = v
	}
	switch strings.ToLower(q.Get("fmt")) {
	case "png":
		opts.Format = image.FormatPNG
	case "jpeg", "jpg", "webp", "avif":
		opts.Format = image.FormatJPEG
	}
	if v, err := strconv.Atoi(q.Get("q")); err == nil && v > 0 {
		opts.Quality = v
	}
	return opts
}

// serveImage writes an image with immutable caching + ETag, delegating range and
// If-None-Match handling to http.ServeContent.
func serveImage(w http.ResponseWriter, r *http.Request, img reader.Image) {
	h := w.Header()
	h.Set("Content-Type", img.ContentType)
	h.Set("Cache-Control", "public, max-age=31536000, immutable")
	h.Set("ETag", img.ETag)
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(img.Data))
}
