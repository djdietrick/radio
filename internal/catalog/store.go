package catalog

import (
	"context"
	"strings"

	"github.com/djdietrick/radio/internal/catalog/metadata"
	"github.com/djdietrick/radio/internal/db"
	"github.com/djdietrick/radio/internal/models"
	"github.com/jackc/pgx/v5"
)

// Store provides catalog persistence: upserting tracks (with their artist and
// album), and read queries used by the API.
type Store struct {
	db *db.DB
}

func NewStore(database *db.DB) *Store {
	return &Store{db: database}
}

func artistKey(name string) string { return strings.ToLower(strings.TrimSpace(name)) }

func albumKey(albumArtist, title string) string {
	return strings.ToLower(strings.TrimSpace(albumArtist)) + "\x1f" + strings.ToLower(strings.TrimSpace(title))
}

// UpsertResult reports the ids touched by an UpsertTrack call.
type UpsertResult struct {
	TrackID string
	// AlbumID is empty when the track has no album metadata.
	AlbumID string
}

// UpsertTrack inserts or updates a track and its associated artist/album rows.
// Path is the natural key for a track.
func (s *Store) UpsertTrack(ctx context.Context, path string, t *metadata.Tags, sizeBytes int64, durationMs int64) (UpsertResult, error) {
	var res UpsertResult

	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer tx.Rollback(ctx)

	var artistID string
	if t.Artist != "" {
		if err := tx.QueryRow(ctx, `
			INSERT INTO artists (name, name_key) VALUES ($1, $2)
			ON CONFLICT (name_key) DO UPDATE SET name = EXCLUDED.name
			RETURNING id`, t.Artist, artistKey(t.Artist),
		).Scan(&artistID); err != nil {
			return res, err
		}
	}

	var albumID string
	if t.Album != "" {
		if err := tx.QueryRow(ctx, `
			INSERT INTO albums (title, album_artist, year, has_art, album_key)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (album_key) DO UPDATE SET
				has_art = albums.has_art OR EXCLUDED.has_art,
				year = CASE WHEN albums.year = 0 THEN EXCLUDED.year ELSE albums.year END
			RETURNING id`,
			t.Album, t.AlbumArtist, t.Year, t.HasEmbeddedArt, albumKey(t.AlbumArtist, t.Album),
		).Scan(&albumID); err != nil {
			return res, err
		}
	}

	var trackID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO tracks
			(path, title, artist_id, album_id, track_number, disc_number,
			 duration_ms, genre, year, codec, mime_type, size_bytes, modified_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12, now())
		ON CONFLICT (path) DO UPDATE SET
			title=EXCLUDED.title, artist_id=EXCLUDED.artist_id, album_id=EXCLUDED.album_id,
			track_number=EXCLUDED.track_number, disc_number=EXCLUDED.disc_number,
			duration_ms=EXCLUDED.duration_ms, genre=EXCLUDED.genre, year=EXCLUDED.year,
			codec=EXCLUDED.codec, mime_type=EXCLUDED.mime_type, size_bytes=EXCLUDED.size_bytes,
			modified_at=now()
		RETURNING id`,
		path, t.Title, nullIfEmpty(artistID), nullIfEmpty(albumID), t.TrackNumber, t.DiscNumber,
		durationMs, t.Genre, t.Year, t.Codec, t.MimeType, sizeBytes,
	).Scan(&trackID); err != nil {
		return res, err
	}

	if err := tx.Commit(ctx); err != nil {
		return res, err
	}
	res.TrackID = trackID
	res.AlbumID = albumID
	return res, nil
}

// DeleteTrackByPath removes a track that no longer exists on disk.
func (s *Store) DeleteTrackByPath(ctx context.Context, path string) error {
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM tracks WHERE path = $1`, path)
	return err
}

// TrackPathByID returns the on-disk path for a track id (used by streaming).
func (s *Store) TrackPathByID(ctx context.Context, id string) (string, string, error) {
	var path, mime string
	err := s.db.Pool.QueryRow(ctx, `SELECT path, mime_type FROM tracks WHERE id = $1`, id).Scan(&path, &mime)
	return path, mime, err
}

// --- backfill support ---

// TrackRef is a minimal (id, path) pair used by backfill jobs.
type TrackRef struct {
	ID   string
	Path string
}

// AlbumRef pairs an album id with one representative track path, used to
// re-extract embedded art for albums that currently have none.
type AlbumRef struct {
	AlbumID   string
	TrackPath string
}

// TracksMissingDuration returns up to limit tracks whose duration is unknown
// (duration_ms = 0), oldest first for stable paging.
func (s *Store) TracksMissingDuration(ctx context.Context, limit, offset int) ([]TrackRef, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, path FROM tracks
		WHERE duration_ms = 0
		ORDER BY id
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TrackRef
	for rows.Next() {
		var r TrackRef
		if err := rows.Scan(&r.ID, &r.Path); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// UpdateTrackDuration sets only the duration for a track.
func (s *Store) UpdateTrackDuration(ctx context.Context, id string, durationMs int64) error {
	_, err := s.db.Pool.Exec(ctx,
		`UPDATE tracks SET duration_ms = $2 WHERE id = $1`, id, durationMs)
	return err
}

// AlbumsMissingArt returns up to limit albums that have no art, each paired with
// the path of one of its tracks (the lowest disc/track number) so the caller
// can re-extract embedded cover bytes.
func (s *Store) AlbumsMissingArt(ctx context.Context, limit, offset int) ([]AlbumRef, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT a.id, t.path
		FROM albums a
		JOIN LATERAL (
			SELECT path FROM tracks
			WHERE album_id = a.id
			ORDER BY disc_number, track_number
			LIMIT 1
		) t ON true
		WHERE a.has_art = FALSE
		ORDER BY a.id
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AlbumRef
	for rows.Next() {
		var r AlbumRef
		if err := rows.Scan(&r.AlbumID, &r.TrackPath); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// MarkAlbumHasArt flips an album's has_art flag to true.
func (s *Store) MarkAlbumHasArt(ctx context.Context, albumID string) error {
	_, err := s.db.Pool.Exec(ctx,
		`UPDATE albums SET has_art = TRUE WHERE id = $1`, albumID)
	return err
}

// GetTrack returns a fully-populated track including artist/album names.
func (s *Store) GetTrack(ctx context.Context, id string) (*models.Track, error) {
	row := s.db.Pool.QueryRow(ctx, trackSelect+` WHERE t.id = $1`, id)
	return scanTrack(row)
}

// ListTracksByAlbum returns an album's tracks in disc/track order.
func (s *Store) ListTracksByAlbum(ctx context.Context, albumID string) ([]models.Track, error) {
	rows, err := s.db.Pool.Query(ctx, trackSelect+`
		WHERE t.album_id = $1 ORDER BY t.disc_number, t.track_number`, albumID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTracks(rows)
}

// albumSelect is the shared projection for album rows with computed track
// counts and durations. Callers append a WHERE/GROUP BY/ORDER BY as needed.
const albumSelect = `
	SELECT a.id, a.title, a.album_artist, a.year, a.has_art, a.added_at,
	       COUNT(t.id), COALESCE(SUM(t.duration_ms), 0)
	FROM albums a LEFT JOIN tracks t ON t.album_id = a.id`

// ListAlbums returns albums with computed track counts and durations.
func (s *Store) ListAlbums(ctx context.Context, limit, offset int) ([]models.Album, error) {
	rows, err := s.db.Pool.Query(ctx, albumSelect+`
		GROUP BY a.id
		ORDER BY a.album_artist, a.year, a.title
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAlbums(rows)
}

// GetAlbum returns a single album with computed track count and duration.
func (s *Store) GetAlbum(ctx context.Context, id string) (*models.Album, error) {
	rows, err := s.db.Pool.Query(ctx, albumSelect+`
		WHERE a.id = $1
		GROUP BY a.id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	albums, err := scanAlbums(rows)
	if err != nil {
		return nil, err
	}
	if len(albums) == 0 {
		return nil, pgx.ErrNoRows
	}
	return &albums[0], nil
}

func scanAlbums(rows rowsIface) ([]models.Album, error) {
	var out []models.Album
	for rows.Next() {
		var a models.Album
		if err := rows.Scan(&a.ID, &a.Title, &a.AlbumArtist, &a.Year, &a.HasArt,
			&a.AddedAt, &a.TrackCount, &a.DurationMs); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// trackSelect is the shared projection for track rows joined to artist/album.
const trackSelect = `
	SELECT t.id, t.path, t.title,
	       COALESCE(t.artist_id::text,''), COALESCE(ar.name,''),
	       COALESCE(t.album_id::text,''),  COALESCE(al.title,''),
	       t.track_number, t.disc_number, t.duration_ms, t.genre, t.year,
	       t.codec, t.mime_type, t.size_bytes, t.modified_at, t.added_at
	FROM tracks t
	LEFT JOIN artists ar ON ar.id = t.artist_id
	LEFT JOIN albums  al ON al.id = t.album_id`

type scannable interface {
	Scan(dest ...any) error
}

func scanTrack(row scannable) (*models.Track, error) {
	var t models.Track
	if err := row.Scan(&t.ID, &t.Path, &t.Title, &t.ArtistID, &t.ArtistName,
		&t.AlbumID, &t.AlbumTitle, &t.TrackNumber, &t.DiscNumber, &t.DurationMs,
		&t.Genre, &t.Year, &t.Codec, &t.MimeType, &t.SizeBytes, &t.ModifiedAt, &t.AddedAt); err != nil {
		return nil, err
	}
	return &t, nil
}

type rowsIface interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func scanTracks(rows rowsIface) ([]models.Track, error) {
	var out []models.Track
	for rows.Next() {
		t, err := scanTrack(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
