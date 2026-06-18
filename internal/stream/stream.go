// Package stream serves audio files to the browser over HTTP, supporting range
// requests (seeking) via the standard library's http.ServeContent.
package stream

import (
	"net/http"
	"os"

	"github.com/djdietrick/radio/internal/catalog"
)

// Handler streams track files by id.
type Handler struct {
	catalog *catalog.Store
}

func New(c *catalog.Store) *Handler {
	return &Handler{catalog: c}
}

// ServeTrack writes the file for trackID, honoring Range headers for seeking.
// http.ServeContent sets Content-Type, Content-Length, Accept-Ranges, and
// handles 206 Partial Content / If-Range / If-Modified-Since automatically.
func (h *Handler) ServeTrack(w http.ResponseWriter, r *http.Request, trackID string) {
	path, mime, err := h.catalog.TrackPathByID(r.Context(), trackID)
	if err != nil {
		http.Error(w, "track not found", http.StatusNotFound)
		return
	}

	f, err := os.Open(path)
	if err != nil {
		// Catalogued but missing on disk (deleted out from under us).
		http.Error(w, "track file unavailable", http.StatusNotFound)
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		http.Error(w, "cannot stat track", http.StatusInternalServerError)
		return
	}

	if mime != "" {
		w.Header().Set("Content-Type", mime)
	}
	// Allow the browser to cache audio bodies; content at a path is immutable
	// for a given track id within a session.
	w.Header().Set("Cache-Control", "private, max-age=3600")

	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
}
