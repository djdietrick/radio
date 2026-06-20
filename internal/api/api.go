// Package api wires HTTP routes to the catalog, playlist, radio, and streaming
// subsystems.
package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/djdietrick/radio/internal/api/handlers"
	"github.com/djdietrick/radio/internal/api/ws"
	"github.com/djdietrick/radio/internal/auth"
	"github.com/djdietrick/radio/internal/catalog"
	"github.com/djdietrick/radio/internal/catalog/art"
	"github.com/djdietrick/radio/internal/catalog/scanner"
	"github.com/djdietrick/radio/internal/playlist"
	"github.com/djdietrick/radio/internal/radio"
	"github.com/djdietrick/radio/internal/stream"
	"github.com/djdietrick/radio/internal/users"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// Deps holds the constructed subsystems the handlers need.
type Deps struct {
	Catalog   *catalog.Store
	Playlists *playlist.Store
	Resolver  *playlist.Resolver
	Radio     *radio.Engine
	Stream    *stream.Handler
	Scanner   *scanner.Scanner
	Art       *art.Cache
	Users     *users.Store
	Auth      *auth.Authenticator
	Log       *slog.Logger
}

// NewRouter builds the HTTP handler with all routes mounted under /api.
func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	// CORS is permissive in dev so the Vite frontend (different port) can call
	// the API; tighten AllowedOrigins for production.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Range"},
		ExposedHeaders:   []string{"Content-Length", "Content-Range", "Accept-Ranges"},
		AllowCredentials: false,
	}))

	h := handlers.New(handlers.Deps{
		Catalog:   d.Catalog,
		Playlists: d.Playlists,
		Radio:     d.Radio,
		Stream:    d.Stream,
		Scanner:   d.Scanner,
		Art:       d.Art,
		Users:     d.Users,
		Auth:      d.Auth,
		Log:       d.Log,
	})

	// allowAllOrigins mirrors the permissive CORS policy above for the WS
	// upgrade check (Vite dev server is a different origin).
	stationWS := ws.NewStationHandler(d.Radio, true, d.Log)

	r.Get("/healthz", h.Health)

	r.Route("/api", func(r chi.Router) {
		// --- public routes (no auth) ---

		// Auth: login is necessarily public.
		r.Group(func(r chi.Router) {
			r.Use(middleware.Timeout(60 * time.Second))
			r.Post("/auth/login", h.Login)

			// Radio tune-in is public so stations are shareable/listenable
			// without an account.
			r.Get("/stations/{stationID}", h.GetStation)
			r.Get("/stations/{stationID}/now", h.StationNowPlaying)
		})

		// Radio live-sync WebSocket: public and long-lived (no request timeout).
		r.Get("/stations/{stationID}/ws", func(w http.ResponseWriter, req *http.Request) {
			stationWS.Serve(w, req, chi.URLParam(req, "stationID"))
		})

		// --- authenticated routes ---
		r.Group(func(r chi.Router) {
			r.Use(middleware.Timeout(60 * time.Second))
			r.Use(d.Auth.Require)

			r.Get("/auth/me", h.Me)

			// Library
			r.Get("/albums", h.ListAlbums)
			r.Get("/albums/{albumID}", h.GetAlbum)
			r.Get("/albums/{albumID}/tracks", h.AlbumTracks)
			r.Get("/albums/{albumID}/art", h.AlbumArt)
			r.Get("/artists", h.ListArtists)
			r.Get("/artists/{artistID}", h.GetArtist)
			r.Get("/artists/{artistID}/albums", h.ArtistAlbums)
			r.Get("/tracks/{trackID}", h.GetTrack)
			r.Get("/search", h.Search)
			r.Post("/scan", h.TriggerScan)
			r.Post("/backfill", h.TriggerBackfill)

			// Streaming (Range-aware)
			r.Get("/stream/{trackID}", h.StreamTrack)

			// Playlists (owned by the authenticated user)
			r.Get("/playlists", h.ListPlaylists)
			r.Post("/playlists", h.CreatePlaylist)
			r.Get("/playlists/{playlistID}", h.GetPlaylist)
			r.Delete("/playlists/{playlistID}", h.DeletePlaylist)
			r.Post("/playlists/{playlistID}/items", h.AddPlaylistItem)
			r.Delete("/playlists/{playlistID}/items/{itemID}", h.RemovePlaylistItem)

			// Radio management (create requires auth; single-station reads
			// above are public, but listing is gated to logged-in users)
			r.Get("/stations", h.ListStations)
			r.Post("/stations", h.CreateStation)

			// User management (admin only)
			r.Group(func(r chi.Router) {
				r.Use(d.Auth.RequireAdmin)
				r.Get("/users", h.ListUsers)
				r.Post("/users", h.CreateUser)
			})
		})
	})

	return r
}
