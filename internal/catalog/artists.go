package catalog

import (
	"context"

	"github.com/djdietrick/radio/internal/models"
	"github.com/jackc/pgx/v5"
)

// artistSelect is the shared projection for artist rows with computed album and
// track counts. Callers append a WHERE/GROUP BY/ORDER BY as needed.
const artistSelect = `
	SELECT ar.id, ar.name,
	       COUNT(DISTINCT t.album_id) FILTER (WHERE t.album_id IS NOT NULL),
	       COUNT(t.id)
	FROM artists ar LEFT JOIN tracks t ON t.artist_id = ar.id`

// ListArtists returns artists with album/track counts, alphabetical.
func (s *Store) ListArtists(ctx context.Context, limit, offset int) ([]models.Artist, error) {
	rows, err := s.db.Pool.Query(ctx, artistSelect+`
		GROUP BY ar.id
		ORDER BY ar.name
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanArtists(rows)
}

// GetArtist returns a single artist with computed counts.
func (s *Store) GetArtist(ctx context.Context, id string) (*models.Artist, error) {
	rows, err := s.db.Pool.Query(ctx, artistSelect+`
		WHERE ar.id = $1
		GROUP BY ar.id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	artists, err := scanArtists(rows)
	if err != nil {
		return nil, err
	}
	if len(artists) == 0 {
		return nil, pgx.ErrNoRows
	}
	return &artists[0], nil
}

// ListAlbumsByArtist returns the albums that contain at least one track by the
// given artist, ordered by year then title.
func (s *Store) ListAlbumsByArtist(ctx context.Context, artistID string) ([]models.Album, error) {
	rows, err := s.db.Pool.Query(ctx, albumSelect+`
		WHERE a.id IN (
			SELECT DISTINCT album_id FROM tracks
			WHERE artist_id = $1 AND album_id IS NOT NULL
		)
		GROUP BY a.id
		ORDER BY a.year, a.title`, artistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAlbums(rows)
}

func scanArtists(rows rowsIface) ([]models.Artist, error) {
	var out []models.Artist
	for rows.Next() {
		var a models.Artist
		if err := rows.Scan(&a.ID, &a.Name, &a.AlbumCount, &a.TrackCount); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
