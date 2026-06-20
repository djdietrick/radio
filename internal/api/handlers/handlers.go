package handlers

import (
	"encoding/json"
	"log/slog"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/djdietrick/radio/internal/auth"
	"github.com/djdietrick/radio/internal/catalog"
	"github.com/djdietrick/radio/internal/catalog/art"
	"github.com/djdietrick/radio/internal/catalog/scanner"
	"github.com/djdietrick/radio/internal/models"
	"github.com/djdietrick/radio/internal/playlist"
	"github.com/djdietrick/radio/internal/radio"
	"github.com/djdietrick/radio/internal/stream"
	"github.com/djdietrick/radio/internal/users"
	"github.com/go-chi/chi/v5"
)

// Deps are the subsystems handlers depend on.
type Deps struct {
	Catalog   *catalog.Store
	Playlists *playlist.Store
	Radio     *radio.Engine
	Stream    *stream.Handler
	Scanner   *scanner.Scanner
	Art       *art.Cache
	Users     *users.Store
	Auth      *auth.Authenticator
	Log       *slog.Logger
}

// Handlers groups all HTTP handlers.
type Handlers struct {
	d Deps
}

func New(d Deps) *Handlers { return &Handlers{d: d} }

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// --- health ---

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- library ---

func (h *Handlers) ListAlbums(w http.ResponseWriter, r *http.Request) {
	limit, offset := paginate(r)
	albums, err := h.d.Catalog.ListAlbums(r.Context(), limit, offset)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to list albums")
		return
	}
	writeJSON(w, http.StatusOK, albums)
}

func (h *Handlers) GetAlbum(w http.ResponseWriter, r *http.Request) {
	album, err := h.d.Catalog.GetAlbum(r.Context(), chi.URLParam(r, "albumID"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "album not found")
		return
	}
	writeJSON(w, http.StatusOK, album)
}

func (h *Handlers) AlbumTracks(w http.ResponseWriter, r *http.Request) {
	albumID := chi.URLParam(r, "albumID")
	tracks, err := h.d.Catalog.ListTracksByAlbum(r.Context(), albumID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to list tracks")
		return
	}
	writeJSON(w, http.StatusOK, tracks)
}

// --- artists ---

func (h *Handlers) ListArtists(w http.ResponseWriter, r *http.Request) {
	limit, offset := paginate(r)
	artists, err := h.d.Catalog.ListArtists(r.Context(), limit, offset)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to list artists")
		return
	}
	writeJSON(w, http.StatusOK, artists)
}

func (h *Handlers) GetArtist(w http.ResponseWriter, r *http.Request) {
	artist, err := h.d.Catalog.GetArtist(r.Context(), chi.URLParam(r, "artistID"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "artist not found")
		return
	}
	writeJSON(w, http.StatusOK, artist)
}

func (h *Handlers) ArtistAlbums(w http.ResponseWriter, r *http.Request) {
	albums, err := h.d.Catalog.ListAlbumsByArtist(r.Context(), chi.URLParam(r, "artistID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to list artist albums")
		return
	}
	writeJSON(w, http.StatusOK, albums)
}

// --- search ---

// Search runs a substring query over tracks/albums/artists. An empty query
// returns empty result sets rather than erroring.
func (h *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeJSON(w, http.StatusOK, &models.SearchResults{
			Tracks:  []models.Track{},
			Albums:  []models.Album{},
			Artists: []models.Artist{},
		})
		return
	}
	limit := 25
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := parseIntClamp(v, 1, 100); err == nil {
			limit = n
		}
	}
	results, err := h.d.Catalog.Search(r.Context(), q, limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "search failed")
		return
	}
	writeJSON(w, http.StatusOK, results)
}

func (h *Handlers) GetTrack(w http.ResponseWriter, r *http.Request) {
	track, err := h.d.Catalog.GetTrack(r.Context(), chi.URLParam(r, "trackID"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "track not found")
		return
	}
	writeJSON(w, http.StatusOK, track)
}

func (h *Handlers) TriggerScan(w http.ResponseWriter, r *http.Request) {
	// Run asynchronously so the request returns immediately.
	go func() {
		if err := h.d.Scanner.ScanAll(contextDetached()); err != nil {
			h.d.Log.Error("manual scan failed", "err", err)
		}
	}()
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "scan started"})
}

// TriggerBackfill re-probes durations and re-extracts art for already-cataloged
// rows that are missing that data — without a full re-scan. Runs asynchronously.
func (h *Handlers) TriggerBackfill(w http.ResponseWriter, r *http.Request) {
	go func() {
		if _, err := h.d.Scanner.Backfill(contextDetached()); err != nil {
			h.d.Log.Error("backfill failed", "err", err)
		}
	}()
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "backfill started"})
}

// --- streaming ---

func (h *Handlers) StreamTrack(w http.ResponseWriter, r *http.Request) {
	h.d.Stream.ServeTrack(w, r, chi.URLParam(r, "trackID"))
}

// AlbumArt serves an album's cached cover image, or 404 if none is cached.
func (h *Handlers) AlbumArt(w http.ResponseWriter, r *http.Request) {
	albumID := chi.URLParam(r, "albumID")
	f, mime, _, err := h.d.Art.Open(albumID)
	if err != nil {
		writeErr(w, http.StatusNotFound, "no art for album")
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	// ServeContent handles conditional/range requests and Content-Length.
	info, statErr := f.Stat()
	var modTime time.Time
	if statErr == nil {
		modTime = info.ModTime()
	}
	http.ServeContent(w, r, albumID, modTime, f)
}

// --- playlists ---

func (h *Handlers) ListPlaylists(w http.ResponseWriter, r *http.Request) {
	pls, err := h.d.Playlists.List(r.Context(), auth.UserID(r.Context()))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to list playlists")
		return
	}
	writeJSON(w, http.StatusOK, pls)
}

type createPlaylistReq struct {
	Name string `json:"name"`
}

func (h *Handlers) CreatePlaylist(w http.ResponseWriter, r *http.Request) {
	var req createPlaylistReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeErr(w, http.StatusBadRequest, "name is required")
		return
	}
	pl, err := h.d.Playlists.Create(r.Context(), auth.UserID(r.Context()), req.Name)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to create playlist")
		return
	}
	writeJSON(w, http.StatusCreated, pl)
}

func (h *Handlers) GetPlaylist(w http.ResponseWriter, r *http.Request) {
	pl, err := h.d.Playlists.Get(r.Context(), chi.URLParam(r, "playlistID"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "playlist not found")
		return
	}
	writeJSON(w, http.StatusOK, pl)
}

func (h *Handlers) DeletePlaylist(w http.ResponseWriter, r *http.Request) {
	if err := h.d.Playlists.Delete(r.Context(), chi.URLParam(r, "playlistID")); err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to delete playlist")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// kindGroup is an API-only item kind: it creates several track rows sharing a
// group_id (stored as kind='track'). It isn't a models.PlaylistItemKind.
const kindGroup = "group"

type addItemReq struct {
	Kind     models.PlaylistItemKind `json:"kind"`     // "track" | "album" | "group"
	TrackID  string                  `json:"trackId"`  // set when kind=track
	AlbumID  string                  `json:"albumId"`  // set when kind=album
	TrackIDs []string                `json:"trackIds"` // set when kind=group, in play order
}

func (h *Handlers) AddPlaylistItem(w http.ResponseWriter, r *http.Request) {
	var req addItemReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	playlistID := chi.URLParam(r, "playlistID")

	// A group is a hand-picked set of tracks that plays in order and shuffles as
	// one unit (like an album item, but a subset).
	if string(req.Kind) == kindGroup {
		if len(req.TrackIDs) == 0 {
			writeErr(w, http.StatusBadRequest, "trackIds required for a group")
			return
		}
		items, err := h.d.Playlists.AddGroup(r.Context(), playlistID, req.TrackIDs)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "failed to add group")
			return
		}
		writeJSON(w, http.StatusCreated, items)
		return
	}

	var refID string
	switch req.Kind {
	case models.ItemKindTrack:
		refID = req.TrackID
	case models.ItemKindAlbum:
		refID = req.AlbumID
	default:
		writeErr(w, http.StatusBadRequest, "kind must be 'track', 'album' or 'group'")
		return
	}
	if refID == "" {
		writeErr(w, http.StatusBadRequest, "trackId or albumId required for the given kind")
		return
	}

	item, err := h.d.Playlists.AddItem(r.Context(), playlistID, req.Kind, refID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to add item")
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *Handlers) RemovePlaylistItem(w http.ResponseWriter, r *http.Request) {
	if err := h.d.Playlists.RemoveItem(r.Context(), chi.URLParam(r, "itemID")); err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to remove item")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) RemovePlaylistGroup(w http.ResponseWriter, r *http.Request) {
	err := h.d.Playlists.RemoveGroup(r.Context(),
		chi.URLParam(r, "playlistID"), chi.URLParam(r, "groupID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to remove group")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- radio ---

type createStationReq struct {
	Name         string `json:"name"`
	PlaylistID   string `json:"playlistId"`
	AlbumShuffle bool   `json:"albumShuffle"`
	Loop         bool   `json:"loop"`
}

func (h *Handlers) CreateStation(w http.ResponseWriter, r *http.Request) {
	var req createStationReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.PlaylistID == "" {
		writeErr(w, http.StatusBadRequest, "name and playlistId are required")
		return
	}
	seed := rand.Int63()
	st, err := h.d.Radio.CreateStation(r.Context(), auth.UserID(r.Context()), req.Name, req.PlaylistID, req.AlbumShuffle, req.Loop, seed)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to create station")
		return
	}
	writeJSON(w, http.StatusCreated, st)
}

// ListStations returns all stations so they're discoverable in the UI.
func (h *Handlers) ListStations(w http.ResponseWriter, r *http.Request) {
	stations, err := h.d.Radio.ListStations(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to list stations")
		return
	}
	writeJSON(w, http.StatusOK, stations)
}

func (h *Handlers) GetStation(w http.ResponseWriter, r *http.Request) {
	st, err := h.d.Radio.GetStation(r.Context(), chi.URLParam(r, "stationID"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "station not found")
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// StationNowPlaying returns the computed current track/offset. This is the
// virtual-radio endpoint: every listener calling it at the same instant gets
// the same answer.
func (h *Handlers) StationNowPlaying(w http.ResponseWriter, r *http.Request) {
	np, err := h.d.Radio.NowPlayingByID(r.Context(), chi.URLParam(r, "stationID"), time.Now())
	if err == radio.ErrEmptyStation {
		writeErr(w, http.StatusConflict, "station has no playable tracks")
		return
	}
	if err != nil {
		writeErr(w, http.StatusNotFound, "station not found")
		return
	}
	writeJSON(w, http.StatusOK, np)
}

// --- misc ---

func paginate(r *http.Request) (limit, offset int) {
	limit, offset = 100, 0
	q := r.URL.Query()
	if v := q.Get("limit"); v != "" {
		if n, err := parseIntClamp(v, 1, 500); err == nil {
			limit = n
		}
	}
	if v := q.Get("offset"); v != "" {
		if n, err := parseIntClamp(v, 0, 1<<31); err == nil {
			offset = n
		}
	}
	return
}
