// Package radio implements virtual ("never actually playing") radio stations.
//
// A station has a fixed start time and a deterministically-resolved track
// queue. Nothing streams continuously on the server; instead, when a listener
// tunes in, the engine computes — purely from elapsed wall-clock time — which
// track is "currently" playing and the offset into it. Because the queue
// ordering is seeded and the start time is fixed, every listener computes the
// same NowPlaying for the same instant, keeping them in sync.
package radio

import (
	"context"
	"errors"
	"time"

	"github.com/djdietrick/radio/internal/db"
	"github.com/djdietrick/radio/internal/models"
	"github.com/djdietrick/radio/internal/playlist"
)

// ErrEmptyStation is returned when a station resolves to no playable tracks.
var ErrEmptyStation = errors.New("radio: station has no tracks")

// Engine computes station state. It resolves the underlying playlist on demand
// using the station's shuffle settings and seed.
type Engine struct {
	db        *db.DB
	playlists *playlist.Store
	resolver  *playlist.Resolver
}

func NewEngine(database *db.DB, playlists *playlist.Store, resolver *playlist.Resolver) *Engine {
	return &Engine{db: database, playlists: playlists, resolver: resolver}
}

// CreateStation creates a station from a playlist, fixing its start time now.
func (e *Engine) CreateStation(ctx context.Context, userID, name, playlistID string, albumShuffle, loop bool, seed int64) (*models.Station, error) {
	var st models.Station
	err := e.db.Pool.QueryRow(ctx, `
		INSERT INTO stations (user_id, name, playlist_id, album_shuffle, shuffle_seed, loop)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, started_at, created_at`,
		userID, name, playlistID, albumShuffle, seed, loop,
	).Scan(&st.ID, &st.StartedAt, &st.CreatedAt)
	if err != nil {
		return nil, err
	}
	st.UserID = userID
	st.Name = name
	st.PlaylistID = playlistID
	st.AlbumShuffle = albumShuffle
	st.ShuffleSeed = seed
	st.Loop = loop
	return &st, nil
}

// GetStation loads a station by id.
func (e *Engine) GetStation(ctx context.Context, id string) (*models.Station, error) {
	var st models.Station
	err := e.db.Pool.QueryRow(ctx, `
		SELECT id, user_id, name, playlist_id, started_at, album_shuffle, shuffle_seed, loop, created_at
		FROM stations WHERE id = $1`, id,
	).Scan(&st.ID, &st.UserID, &st.Name, &st.PlaylistID, &st.StartedAt,
		&st.AlbumShuffle, &st.ShuffleSeed, &st.Loop, &st.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &st, nil
}

// resolveQueue expands the station's playlist into its deterministic queue.
// Radio stations always shuffle (that's the point of a "station"); AlbumShuffle
// decides whether albums stay together.
func (e *Engine) resolveQueue(ctx context.Context, st *models.Station) ([]models.Track, error) {
	pl, err := e.playlists.Get(ctx, st.PlaylistID)
	if err != nil {
		return nil, err
	}
	return e.resolver.Resolve(ctx, pl.Items, playlist.ResolveOptions{
		Shuffle:      true,
		AlbumShuffle: st.AlbumShuffle,
		Seed:         st.ShuffleSeed,
	})
}

// NowPlaying computes the station's live state at time `at`.
func (e *Engine) NowPlaying(ctx context.Context, st *models.Station, at time.Time) (*models.NowPlaying, error) {
	queue, err := e.resolveQueue(ctx, st)
	if err != nil {
		return nil, err
	}
	if len(queue) == 0 {
		return nil, ErrEmptyStation
	}
	return computeNowPlaying(st, queue, at), nil
}

// NowPlayingByID is a convenience that loads and computes in one call.
func (e *Engine) NowPlayingByID(ctx context.Context, id string, at time.Time) (*models.NowPlaying, error) {
	st, err := e.GetStation(ctx, id)
	if err != nil {
		return nil, err
	}
	return e.NowPlaying(ctx, st, at)
}

// LiveSession holds a station's resolved queue so successive NowPlaying
// computations (e.g. for a WebSocket connection) avoid re-resolving the
// playlist on every track boundary. The queue ordering is fixed by the
// station's seed, so it's stable for the station's lifetime.
type LiveSession struct {
	station *models.Station
	queue   []models.Track
}

// NewLiveSession resolves the station's queue once for repeated computation.
func (e *Engine) NewLiveSession(ctx context.Context, st *models.Station) (*LiveSession, error) {
	queue, err := e.resolveQueue(ctx, st)
	if err != nil {
		return nil, err
	}
	if len(queue) == 0 {
		return nil, ErrEmptyStation
	}
	return &LiveSession{station: st, queue: queue}, nil
}

// At computes the station's NowPlaying at the given instant using the cached
// queue.
func (s *LiveSession) At(at time.Time) *models.NowPlaying {
	return computeNowPlaying(s.station, s.queue, at)
}

// computeNowPlaying is the pure positional math, factored out for testability.
//
// It walks the queue accumulating durations until it finds the track whose
// span contains the elapsed time. For looping stations it first reduces elapsed
// modulo the total program duration. Tracks with unknown (0) duration are given
// a nominal length so the station never stalls on bad metadata.
func computeNowPlaying(st *models.Station, queue []models.Track, at time.Time) *models.NowPlaying {
	const fallbackMs int64 = 3 * 60 * 1000 // assume 3 min when duration unknown

	durations := make([]int64, len(queue))
	var total int64
	for i, t := range queue {
		d := t.DurationMs
		if d <= 0 {
			d = fallbackMs
		}
		durations[i] = d
		total += d
	}

	elapsed := at.Sub(st.StartedAt).Milliseconds()
	if elapsed < 0 {
		elapsed = 0 // station scheduled in the future: pin to start
	}

	np := &models.NowPlaying{
		StationID:   st.ID,
		QueueLength: len(queue),
		ServerTime:  at,
	}

	if st.Loop {
		if total > 0 {
			elapsed %= total
		}
	} else if elapsed >= total {
		// Program finished and not looping: report ended on the last track.
		last := len(queue) - 1
		np.Track = queue[last]
		np.IndexInQueue = last
		np.OffsetMs = durations[last]
		np.MsUntilNextTrack = 0
		np.Ended = true
		return np
	}

	// Find the active track by walking accumulated spans.
	var acc int64
	for i, d := range durations {
		if elapsed < acc+d {
			np.Track = queue[i]
			np.IndexInQueue = i
			np.OffsetMs = elapsed - acc
			np.MsUntilNextTrack = d - np.OffsetMs
			return np
		}
		acc += d
	}

	// Numerical edge (e.g. elapsed == total after modulo): pin to first track.
	np.Track = queue[0]
	np.IndexInQueue = 0
	np.OffsetMs = 0
	np.MsUntilNextTrack = durations[0]
	return np
}
