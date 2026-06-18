package playlist

import (
	"context"

	"github.com/djdietrick/radio/internal/db"
	"github.com/djdietrick/radio/internal/models"
)

// Store persists playlists and their typed items.
type Store struct {
	db *db.DB
}

func NewStore(database *db.DB) *Store {
	return &Store{db: database}
}

// Create makes an empty playlist owned by userID.
func (s *Store) Create(ctx context.Context, userID, name string) (*models.Playlist, error) {
	var p models.Playlist
	p.UserID = userID
	p.Name = name
	err := s.db.Pool.QueryRow(ctx, `
		INSERT INTO playlists (user_id, name) VALUES ($1, $2)
		RETURNING id, created_at, updated_at`, userID, name,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	return &p, err
}

// List returns a user's playlists (without items).
func (s *Store) List(ctx context.Context, userID string) ([]models.Playlist, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, user_id, name, created_at, updated_at
		FROM playlists WHERE user_id = $1 ORDER BY name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Playlist
	for rows.Next() {
		var p models.Playlist
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Get returns a playlist with its items ordered by position.
func (s *Store) Get(ctx context.Context, id string) (*models.Playlist, error) {
	var p models.Playlist
	if err := s.db.Pool.QueryRow(ctx, `
		SELECT id, user_id, name, created_at, updated_at
		FROM playlists WHERE id = $1`, id,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}

	items, err := s.Items(ctx, id)
	if err != nil {
		return nil, err
	}
	p.Items = items
	return &p, nil
}

// Items returns a playlist's items ordered by position.
func (s *Store) Items(ctx context.Context, playlistID string) ([]models.PlaylistItem, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, kind, position, COALESCE(track_id::text,''), COALESCE(album_id::text,'')
		FROM playlist_items WHERE playlist_id = $1 ORDER BY position`, playlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.PlaylistItem
	for rows.Next() {
		var it models.PlaylistItem
		if err := rows.Scan(&it.ID, &it.Kind, &it.Position, &it.TrackID, &it.AlbumID); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// AddItem appends a typed item to the end of the playlist.
func (s *Store) AddItem(ctx context.Context, playlistID string, kind models.PlaylistItemKind, refID string) (*models.PlaylistItem, error) {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var nextPos int
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(MAX(position)+1, 0) FROM playlist_items WHERE playlist_id = $1`, playlistID,
	).Scan(&nextPos); err != nil {
		return nil, err
	}

	var trackID, albumID any
	if kind == models.ItemKindTrack {
		trackID = refID
	} else {
		albumID = refID
	}

	var it models.PlaylistItem
	it.Kind = kind
	it.Position = nextPos
	if err := tx.QueryRow(ctx, `
		INSERT INTO playlist_items (playlist_id, kind, position, track_id, album_id)
		VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		playlistID, kind, nextPos, trackID, albumID,
	).Scan(&it.ID); err != nil {
		return nil, err
	}
	if kind == models.ItemKindTrack {
		it.TrackID = refID
	} else {
		it.AlbumID = refID
	}

	if _, err := tx.Exec(ctx, `UPDATE playlists SET updated_at = now() WHERE id = $1`, playlistID); err != nil {
		return nil, err
	}
	return &it, tx.Commit(ctx)
}

// RemoveItem deletes an item and leaves remaining positions as-is (gaps are
// harmless since resolution only relies on ordering, not contiguity).
func (s *Store) RemoveItem(ctx context.Context, itemID string) error {
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM playlist_items WHERE id = $1`, itemID)
	return err
}

// Delete removes a playlist and (via cascade) its items.
func (s *Store) Delete(ctx context.Context, id string) error {
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM playlists WHERE id = $1`, id)
	return err
}
