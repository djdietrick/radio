// Package scanner walks configured music directories, extracts metadata, and
// upserts tracks into the catalog. It supports a full scan on startup, manual
// rescans, and a live fsnotify watcher.
package scanner

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/djdietrick/radio/internal/catalog"
	"github.com/djdietrick/radio/internal/catalog/art"
	"github.com/djdietrick/radio/internal/catalog/metadata"
	"github.com/djdietrick/radio/internal/catalog/probe"
	"github.com/fsnotify/fsnotify"
)

// Scanner indexes audio files into the catalog store.
type Scanner struct {
	dirs      []string
	store     *catalog.Store
	extractor *metadata.Extractor
	prober    *probe.Prober
	art       *art.Cache
	log       *slog.Logger

	mu       sync.Mutex
	scanning bool
}

func New(dirs []string, store *catalog.Store, extractor *metadata.Extractor, prober *probe.Prober, artCache *art.Cache, log *slog.Logger) *Scanner {
	if !prober.Available() {
		log.Warn("ffprobe not found on PATH; track durations will be unknown (radio uses a fallback length)")
	}
	return &Scanner{dirs: dirs, store: store, extractor: extractor, prober: prober, art: artCache, log: log}
}

// ScanAll walks every configured directory and indexes supported files.
// Concurrent scans are coalesced: a second caller returns immediately.
func (s *Scanner) ScanAll(ctx context.Context) error {
	s.mu.Lock()
	if s.scanning {
		s.mu.Unlock()
		s.log.Info("scan already in progress, skipping")
		return nil
	}
	s.scanning = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.scanning = false
		s.mu.Unlock()
	}()

	start := time.Now()
	var indexed int
	for _, dir := range s.dirs {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				s.log.Warn("walk error", "path", path, "err", err)
				return nil // skip unreadable entries, keep going
			}
			if d.IsDir() || !metadata.SupportedExt(path) {
				return nil
			}
			if err := s.indexFile(ctx, path); err != nil {
				s.log.Warn("index failed", "path", path, "err", err)
				return nil
			}
			indexed++
			return ctx.Err()
		})
		if err != nil {
			s.log.Error("scan dir failed", "dir", dir, "err", err)
		}
	}
	s.log.Info("scan complete", "indexed", indexed, "elapsed", time.Since(start))
	return nil
}

// indexFile extracts metadata for a single file and upserts it.
func (s *Scanner) indexFile(ctx context.Context, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	tags, err := s.extractor.Extract(path)
	if err != nil {
		return err
	}
	if tags, err = s.extractor.EnrichExternal(ctx, tags); err != nil {
		s.log.Warn("external enrich failed", "path", path, "err", err)
	}

	// Probe duration via ffprobe. Failures (including ffprobe being absent) are
	// non-fatal: we index with duration 0, which the radio engine handles with
	// a fallback length.
	durationMs, err := s.prober.DurationMs(ctx, path)
	if err != nil && err != probe.ErrUnavailable {
		s.log.Warn("duration probe failed", "path", path, "err", err)
	}

	res, err := s.store.UpsertTrack(ctx, path, tags, info.Size(), durationMs)
	if err != nil {
		return err
	}

	// Cache embedded cover art against its album (first art per album wins).
	if tags.HasEmbeddedArt && res.AlbumID != "" {
		if _, saveErr := s.art.Save(res.AlbumID, tags.ArtMime, tags.Art); saveErr != nil {
			s.log.Warn("art cache write failed", "albumId", res.AlbumID, "err", saveErr)
		}
	}
	return nil
}

// BackfillResult summarizes a backfill run.
type BackfillResult struct {
	DurationsFilled int `json:"durationsFilled"`
	ArtFilled       int `json:"artFilled"`
}

// backfillPage bounds how many rows are pulled per query iteration.
const backfillPage = 200

// Backfill fixes already-cataloged rows that are missing data, without a full
// re-scan: it re-probes duration for tracks where it's unknown, and re-extracts
// embedded cover art for albums that have none. It only touches files that are
// actually missing the corresponding data, so it's cheap to re-run.
//
// Backfill shares the scan lock, so it won't run concurrently with a full scan.
func (s *Scanner) Backfill(ctx context.Context) (BackfillResult, error) {
	var res BackfillResult

	s.mu.Lock()
	if s.scanning {
		s.mu.Unlock()
		s.log.Info("scan/backfill already in progress, skipping")
		return res, nil
	}
	s.scanning = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.scanning = false
		s.mu.Unlock()
	}()

	start := time.Now()

	durFilled, err := s.backfillDurations(ctx)
	if err != nil {
		return res, err
	}
	res.DurationsFilled = durFilled

	artFilled, err := s.backfillArt(ctx)
	if err != nil {
		return res, err
	}
	res.ArtFilled = artFilled

	s.log.Info("backfill complete",
		"durationsFilled", res.DurationsFilled,
		"artFilled", res.ArtFilled,
		"elapsed", time.Since(start))
	return res, nil
}

// backfillDurations re-probes every track with an unknown duration. If ffprobe
// is unavailable it's a no-op (nothing can be filled).
func (s *Scanner) backfillDurations(ctx context.Context) (int, error) {
	if !s.prober.Available() {
		s.log.Info("ffprobe unavailable; skipping duration backfill")
		return 0, nil
	}

	var filled int
	for {
		// Always page from offset 0: rows that get filled drop out of the
		// WHERE duration_ms = 0 result set, so the window naturally advances.
		refs, err := s.store.TracksMissingDuration(ctx, backfillPage, 0)
		if err != nil {
			return filled, err
		}
		if len(refs) == 0 {
			break
		}

		progressed := false
		for _, ref := range refs {
			if err := ctx.Err(); err != nil {
				return filled, err
			}
			ms, err := s.prober.DurationMs(ctx, ref.Path)
			if err != nil {
				s.log.Warn("backfill probe failed", "path", ref.Path, "err", err)
				continue
			}
			if ms <= 0 {
				// Genuinely unknown (e.g. file gone or container has no
				// duration). Skip; leaving it at 0 keeps it out of future
				// passes only if we don't loop forever — guarded below.
				continue
			}
			if err := s.store.UpdateTrackDuration(ctx, ref.ID, ms); err != nil {
				s.log.Warn("backfill duration update failed", "id", ref.ID, "err", err)
				continue
			}
			filled++
			progressed = true
		}

		// If a full page yielded no updates, every remaining row is unfillable
		// (missing files / no probe result). Stop to avoid an infinite loop.
		if !progressed {
			break
		}
	}
	return filled, nil
}

// backfillArt re-extracts embedded cover art for albums lacking it, caches the
// bytes, and flips has_art so the rows leave the missing-art set.
func (s *Scanner) backfillArt(ctx context.Context) (int, error) {
	var filled int
	for {
		refs, err := s.store.AlbumsMissingArt(ctx, backfillPage, 0)
		if err != nil {
			return filled, err
		}
		if len(refs) == 0 {
			break
		}

		progressed := false
		for _, ref := range refs {
			if err := ctx.Err(); err != nil {
				return filled, err
			}

			// Art may already be cached on disk from an earlier run even though
			// the flag is stale; reconcile the flag without re-reading the file.
			if s.art.Has(ref.AlbumID) {
				if err := s.store.MarkAlbumHasArt(ctx, ref.AlbumID); err != nil {
					s.log.Warn("backfill mark has_art failed", "albumId", ref.AlbumID, "err", err)
					continue
				}
				filled++
				progressed = true
				continue
			}

			tags, err := s.extractor.Extract(ref.TrackPath)
			if err != nil {
				s.log.Warn("backfill extract failed", "path", ref.TrackPath, "err", err)
				continue
			}
			// When external metadata is enabled, fall back to MusicBrainz /
			// Cover Art Archive for albums with no embedded art. This is a
			// no-op when external is disabled or the file already has art.
			if tags, err = s.extractor.EnrichExternal(ctx, tags); err != nil {
				s.log.Warn("backfill external enrich failed", "path", ref.TrackPath, "err", err)
			}
			if !tags.HasEmbeddedArt {
				// No embedded art and no external match; leave has_art false.
				continue
			}
			if _, err := s.art.Save(ref.AlbumID, tags.ArtMime, tags.Art); err != nil {
				s.log.Warn("backfill art save failed", "albumId", ref.AlbumID, "err", err)
				continue
			}
			if err := s.store.MarkAlbumHasArt(ctx, ref.AlbumID); err != nil {
				s.log.Warn("backfill mark has_art failed", "albumId", ref.AlbumID, "err", err)
				continue
			}
			filled++
			progressed = true
		}

		// No progress on a full page => remaining albums have no art available
		// (no embedded cover and no external match). Stop to avoid looping.
		if !progressed {
			break
		}
	}
	return filled, nil
}

// Watch starts a recursive fsnotify watcher that indexes created/modified files
// and removes deleted ones. It blocks until ctx is cancelled.
func (s *Scanner) Watch(ctx context.Context) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()

	// fsnotify is non-recursive; add every existing subdirectory.
	for _, root := range s.dirs {
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err == nil && d.IsDir() {
				if addErr := w.Add(path); addErr != nil {
					s.log.Warn("watch add failed", "path", path, "err", addErr)
				}
			}
			return nil
		})
	}

	// debounce rapid bursts (editors write multiple events per save)
	debounce := make(map[string]*time.Timer)
	var dmu sync.Mutex

	s.log.Info("watcher started", "dirs", s.dirs)
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-w.Events:
			if !ok {
				return nil
			}
			s.handleEvent(ctx, w, ev, debounce, &dmu)
		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			s.log.Warn("watcher error", "err", err)
		}
	}
}

func (s *Scanner) handleEvent(ctx context.Context, w *fsnotify.Watcher, ev fsnotify.Event,
	debounce map[string]*time.Timer, dmu *sync.Mutex) {

	// A newly created directory must be watched too.
	if ev.Op&fsnotify.Create != 0 {
		if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
			_ = w.Add(ev.Name)
			return
		}
	}

	if ev.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
		if metadata.SupportedExt(ev.Name) {
			if err := s.store.DeleteTrackByPath(ctx, ev.Name); err != nil {
				s.log.Warn("delete on remove failed", "path", ev.Name, "err", err)
			}
		}
		return
	}

	if ev.Op&(fsnotify.Create|fsnotify.Write) == 0 || !metadata.SupportedExt(ev.Name) {
		return
	}

	dmu.Lock()
	if t, ok := debounce[ev.Name]; ok {
		t.Stop()
	}
	debounce[ev.Name] = time.AfterFunc(500*time.Millisecond, func() {
		if err := s.indexFile(ctx, ev.Name); err != nil {
			s.log.Warn("index on watch failed", "path", ev.Name, "err", err)
		}
		dmu.Lock()
		delete(debounce, ev.Name)
		dmu.Unlock()
	})
	dmu.Unlock()
}
