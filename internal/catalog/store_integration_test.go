package catalog_test

import (
	"context"
	"os"
	"testing"

	"github.com/djdietrick/radio/internal/catalog"
	"github.com/djdietrick/radio/internal/catalog/metadata"
	"github.com/djdietrick/radio/internal/db"
)

// TestCatalogReads exercises the album/artist/search read queries against a real
// Postgres. Like the other DB-backed tests it's skipped unless
// RADIO_TEST_DATABASE_URL points at a disposable database, since it writes rows.
// Run with, e.g.:
//
//	RADIO_TEST_DATABASE_URL=postgres://radio:radio@localhost:5432/radio_test?sslmode=disable \
//	  go test ./internal/catalog/ -run CatalogReads
func TestCatalogReads(t *testing.T) {
	url := os.Getenv("RADIO_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set RADIO_TEST_DATABASE_URL to run catalog read integration test")
	}

	ctx := context.Background()
	database, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer database.Close()
	if err := database.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	store := catalog.NewStore(database)

	// Seed two tracks on one album by one artist. Unique paths keep the test
	// idempotent across runs (ON CONFLICT updates rather than duplicates).
	seed := func(path, title string, track int) {
		_, err := store.UpsertTrack(ctx, path, &metadata.Tags{
			Title:       title,
			Artist:      "Integration Artist",
			Album:       "Integration Album",
			AlbumArtist: "Integration Artist",
			TrackNumber: track,
			Year:        2020,
		}, 0, 120000)
		if err != nil {
			t.Fatalf("seed %s: %v", path, err)
		}
	}
	seed("/it/catalog/reads/one.mp3", "First Song", 1)
	seed("/it/catalog/reads/two.mp3", "Second Song", 2)

	// Artists list/get with counts.
	artists, err := store.ListArtists(ctx, 100, 0)
	if err != nil {
		t.Fatalf("ListArtists: %v", err)
	}
	var artistID string
	for _, a := range artists {
		if a.Name == "Integration Artist" {
			artistID = a.ID
			if a.TrackCount < 2 {
				t.Fatalf("expected >=2 tracks for artist, got %d", a.TrackCount)
			}
			if a.AlbumCount < 1 {
				t.Fatalf("expected >=1 album for artist, got %d", a.AlbumCount)
			}
		}
	}
	if artistID == "" {
		t.Fatal("seeded artist not found in ListArtists")
	}

	if _, err := store.GetArtist(ctx, artistID); err != nil {
		t.Fatalf("GetArtist: %v", err)
	}

	// Artist -> albums.
	albums, err := store.ListAlbumsByArtist(ctx, artistID)
	if err != nil {
		t.Fatalf("ListAlbumsByArtist: %v", err)
	}
	if len(albums) == 0 {
		t.Fatal("expected at least one album for artist")
	}
	albumID := albums[0].ID

	// GetAlbum returns computed counts.
	album, err := store.GetAlbum(ctx, albumID)
	if err != nil {
		t.Fatalf("GetAlbum: %v", err)
	}
	if album.TrackCount < 2 {
		t.Fatalf("expected >=2 tracks on album, got %d", album.TrackCount)
	}

	// Search hits across all three entity types.
	results, err := store.Search(ctx, "Integration", 25)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results.Albums) == 0 {
		t.Error("expected album search hit for 'Integration'")
	}
	if len(results.Artists) == 0 {
		t.Error("expected artist search hit for 'Integration'")
	}
	songs, err := store.Search(ctx, "Song", 25)
	if err != nil {
		t.Fatalf("Search tracks: %v", err)
	}
	if len(songs.Tracks) < 2 {
		t.Errorf("expected >=2 track hits for 'Song', got %d", len(songs.Tracks))
	}
}
