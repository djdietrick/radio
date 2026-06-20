package catalog

import (
	"context"

	"github.com/djdietrick/radio/internal/models"
)

// Search performs a case-insensitive substring search over track titles, album
// titles/artists, and artist names. Each entity type is capped at limit hits.
//
// This is the simple ILIKE form noted in the project plan; if the library grows
// large enough to need it, a tsvector column is the natural upgrade — callers
// shouldn't need to change.
func (s *Store) Search(ctx context.Context, q string, limit int) (*models.SearchResults, error) {
	pattern := "%" + q + "%"
	out := &models.SearchResults{
		Tracks:  []models.Track{},
		Albums:  []models.Album{},
		Artists: []models.Artist{},
	}

	trackRows, err := s.db.Pool.Query(ctx, trackSelect+`
		WHERE t.title ILIKE $1
		ORDER BY t.title
		LIMIT $2`, pattern, limit)
	if err != nil {
		return nil, err
	}
	tracks, err := scanTracks(trackRows)
	trackRows.Close()
	if err != nil {
		return nil, err
	}
	out.Tracks = tracks

	albumRows, err := s.db.Pool.Query(ctx, albumSelect+`
		WHERE a.title ILIKE $1 OR a.album_artist ILIKE $1
		GROUP BY a.id
		ORDER BY a.album_artist, a.title
		LIMIT $2`, pattern, limit)
	if err != nil {
		return nil, err
	}
	albums, err := scanAlbums(albumRows)
	albumRows.Close()
	if err != nil {
		return nil, err
	}
	out.Albums = albums

	artistRows, err := s.db.Pool.Query(ctx, artistSelect+`
		WHERE ar.name ILIKE $1
		GROUP BY ar.id
		ORDER BY ar.name
		LIMIT $2`, pattern, limit)
	if err != nil {
		return nil, err
	}
	artists, err := scanArtists(artistRows)
	artistRows.Close()
	if err != nil {
		return nil, err
	}
	out.Artists = artists

	return out, nil
}
