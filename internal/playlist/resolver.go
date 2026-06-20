// Package playlist persists playlists of typed items (single track or whole
// album) and resolves them into a flat, ordered track queue. The resolver is
// what makes album-aware shuffle work: an album item is an indivisible unit
// whose tracks always stay in disc/track order, while shuffle reorders the
// *units* (single tracks and whole albums), not individual album tracks.
package playlist

import (
	"context"
	"math/rand"

	"github.com/djdietrick/radio/internal/catalog"
	"github.com/djdietrick/radio/internal/models"
)

// unit is one shufflable element: a lone track, an ordered album, or a
// hand-picked group of tracks (all kept together and in order).
type unit struct {
	tracks []models.Track
}

// trackSource is the slice of the catalog the resolver needs. Depending on an
// interface (rather than *catalog.Store) keeps the unit-building logic DB-free
// and testable; *catalog.Store satisfies it.
type trackSource interface {
	GetTrack(ctx context.Context, id string) (*models.Track, error)
	ListTracksByAlbum(ctx context.Context, albumID string) ([]models.Track, error)
}

// Resolver expands playlist items into a track queue.
type Resolver struct {
	catalog trackSource
}

func NewResolver(c *catalog.Store) *Resolver {
	return &Resolver{catalog: c}
}

// ResolveOptions controls how a playlist is flattened into a queue.
type ResolveOptions struct {
	// AlbumShuffle, when true, shuffles units (keeping album tracks together
	// and in order). When false with Shuffle true, all tracks shuffle flat.
	AlbumShuffle bool
	// Shuffle enables shuffling. With AlbumShuffle false this is per-track.
	Shuffle bool
	// Seed makes shuffling deterministic and shared (used by radio stations so
	// every listener resolves the same ordering).
	Seed int64
}

// Resolve loads the playlist's items and returns the ordered track queue.
func (r *Resolver) Resolve(ctx context.Context, items []models.PlaylistItem, opts ResolveOptions) ([]models.Track, error) {
	// items are assumed pre-sorted by position; build units preserving order.
	units, err := r.buildUnits(ctx, items)
	if err != nil {
		return nil, err
	}

	if opts.Shuffle {
		rng := rand.New(rand.NewSource(opts.Seed))
		if opts.AlbumShuffle {
			// Shuffle whole units; album tracks remain grouped and ordered.
			rng.Shuffle(len(units), func(i, j int) { units[i], units[j] = units[j], units[i] })
		} else {
			// Flatten first, then shuffle every track independently. Album
			// grouping is intentionally discarded in this mode.
			flat := flatten(units)
			rng.Shuffle(len(flat), func(i, j int) { flat[i], flat[j] = flat[j], flat[i] })
			return flat, nil
		}
	}

	return flatten(units), nil
}

func (r *Resolver) buildUnits(ctx context.Context, items []models.PlaylistItem) ([]unit, error) {
	units := make([]unit, 0, len(items))
	// Track the group currently being accumulated so consecutive track items
	// sharing a group_id collapse into one ordered, indivisible unit. Items are
	// position-ordered and a group's rows are always consecutive, so this stays a
	// single pass. lastGroup is "" between groups (and for ungrouped items), and
	// the GroupID != "" guard keeps adjacent single tracks from merging.
	lastGroup := ""
	groupUnitIdx := -1
	for _, it := range items {
		switch it.Kind {
		case models.ItemKindTrack:
			t, err := r.catalog.GetTrack(ctx, it.TrackID)
			if err != nil {
				// Skip dangling references rather than failing the whole resolve.
				continue
			}
			if it.GroupID != "" && it.GroupID == lastGroup && groupUnitIdx >= 0 {
				units[groupUnitIdx].tracks = append(units[groupUnitIdx].tracks, *t)
				continue
			}
			units = append(units, unit{tracks: []models.Track{*t}})
			if it.GroupID != "" {
				lastGroup = it.GroupID
				groupUnitIdx = len(units) - 1
			} else {
				lastGroup = ""
				groupUnitIdx = -1
			}
		case models.ItemKindAlbum:
			lastGroup = ""
			groupUnitIdx = -1
			tracks, err := r.catalog.ListTracksByAlbum(ctx, it.AlbumID)
			if err != nil || len(tracks) == 0 {
				continue
			}
			units = append(units, unit{tracks: tracks})
		}
	}
	return units, nil
}

func flatten(units []unit) []models.Track {
	var out []models.Track
	for _, u := range units {
		out = append(out, u.tracks...)
	}
	return out
}
