package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/djdietrick/radio/internal/catalog"
	"github.com/djdietrick/radio/internal/catalog/art"
	"github.com/djdietrick/radio/internal/catalog/metadata"
	"github.com/djdietrick/radio/internal/catalog/probe"
	"github.com/djdietrick/radio/internal/db"
)

// TestBackfill exercises the full backfill against a real Postgres. It is
// skipped unless RADIO_TEST_DATABASE_URL points at a disposable database, since
// it writes catalog rows. Run with, e.g.:
//
//	RADIO_TEST_DATABASE_URL=postgres://radio:radio@localhost:5432/radio_test?sslmode=disable \
//	  go test ./internal/catalog/scanner/ -run Backfill
func TestBackfill(t *testing.T) {
	url := os.Getenv("RADIO_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set RADIO_TEST_DATABASE_URL to run backfill integration test")
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
	prober := probe.New()
	artCache := art.New(t.TempDir())
	ext := metadata.NewExtractor(false)
	s := New([]string{t.TempDir()}, store, ext, prober, artCache, discardLogger())

	// Seed a track with unknown duration pointing at a real audio file if one
	// can be generated; otherwise just assert the run completes cleanly.
	if !prober.Available() {
		t.Skip("ffprobe unavailable; cannot verify duration backfill")
	}
	fixture := genTone(t, 3)
	if _, err := store.UpsertTrack(ctx, fixture, &metadata.Tags{Title: "x", Album: "A", AlbumArtist: "B"}, 0, 0); err != nil {
		t.Fatalf("seed track: %v", err)
	}

	res, err := s.Backfill(ctx)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if res.DurationsFilled < 1 {
		t.Fatalf("expected at least one duration filled, got %d", res.DurationsFilled)
	}

	// A second run should be a no-op (nothing left missing).
	res2, err := s.Backfill(ctx)
	if err != nil {
		t.Fatalf("second backfill: %v", err)
	}
	if res2.DurationsFilled != 0 {
		t.Fatalf("expected idempotent second run, got %d durations filled", res2.DurationsFilled)
	}
}

// genTone writes a sine .mp3 of the given seconds via ffmpeg, skipping the test
// if ffmpeg isn't available.
func genTone(t *testing.T, secs int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tone.mp3")
	if err := runFFmpegTone(path, secs); err != nil {
		t.Skipf("ffmpeg unavailable to generate fixture: %v", err)
	}
	return path
}
