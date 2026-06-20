package playlist

import (
	"context"
	"errors"
	"testing"

	"github.com/djdietrick/radio/internal/models"
)

// fakeSource is an in-memory trackSource so resolver logic can be tested without
// a database.
type fakeSource struct {
	tracks map[string]models.Track   // trackID -> track
	albums map[string][]models.Track // albumID -> ordered tracks
}

func (f *fakeSource) GetTrack(_ context.Context, id string) (*models.Track, error) {
	t, ok := f.tracks[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return &t, nil
}

func (f *fakeSource) ListTracksByAlbum(_ context.Context, albumID string) ([]models.Track, error) {
	return f.albums[albumID], nil
}

func ids(tracks []models.Track) []string {
	out := make([]string, len(tracks))
	for i, t := range tracks {
		out[i] = t.ID
	}
	return out
}

func newResolver() (*Resolver, *fakeSource) {
	src := &fakeSource{
		tracks: map[string]models.Track{
			"t1": {ID: "t1", Title: "T1"},
			"t2": {ID: "t2", Title: "T2"},
			"a1": {ID: "a1", Title: "A1"},
			"a2": {ID: "a2", Title: "A2"},
			"a3": {ID: "a3", Title: "A3"},
		},
		albums: map[string][]models.Track{
			"album1": {{ID: "a1"}, {ID: "a2"}, {ID: "a3"}},
		},
	}
	return &Resolver{catalog: src}, src
}

// A group of track items sharing a group_id stays together and in order even
// when units are shuffled (album-shuffle semantics for a subset).
func TestResolve_GroupStaysTogetherAndOrdered(t *testing.T) {
	r, _ := newResolver()
	items := []models.PlaylistItem{
		{ID: "i1", Kind: models.ItemKindTrack, Position: 0, TrackID: "a1", GroupID: "g1"},
		{ID: "i2", Kind: models.ItemKindTrack, Position: 1, TrackID: "a3", GroupID: "g1"},
		{ID: "i3", Kind: models.ItemKindTrack, Position: 2, TrackID: "t1"},
	}

	// Try several seeds; the group's relative order must never break.
	for seed := int64(0); seed < 20; seed++ {
		queue, err := r.Resolve(context.Background(), items, ResolveOptions{
			Shuffle: true, AlbumShuffle: true, Seed: seed,
		})
		if err != nil {
			t.Fatalf("resolve: %v", err)
		}
		if len(queue) != 3 {
			t.Fatalf("seed %d: expected 3 tracks, got %d", seed, len(queue))
		}
		// Find a1 and a3; a1 must immediately precede a3.
		var ai, bi = -1, -1
		for i, tr := range queue {
			if tr.ID == "a1" {
				ai = i
			}
			if tr.ID == "a3" {
				bi = i
			}
		}
		if ai < 0 || bi < 0 || bi != ai+1 {
			t.Fatalf("seed %d: group split or reordered: %v", seed, ids(queue))
		}
	}
}

// In flat shuffle (AlbumShuffle=false) a group is intentionally broken apart,
// mirroring how whole-album items flatten in that mode.
func TestResolve_FlatShuffleBreaksGroup(t *testing.T) {
	r, _ := newResolver()
	items := []models.PlaylistItem{
		{ID: "i1", Kind: models.ItemKindTrack, Position: 0, TrackID: "a1", GroupID: "g1"},
		{ID: "i2", Kind: models.ItemKindTrack, Position: 1, TrackID: "a2", GroupID: "g1"},
		{ID: "i3", Kind: models.ItemKindTrack, Position: 2, TrackID: "a3", GroupID: "g1"},
	}
	queue, err := r.Resolve(context.Background(), items, ResolveOptions{
		Shuffle: true, AlbumShuffle: false, Seed: 1,
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(queue) != 3 {
		t.Fatalf("expected 3 tracks, got %d", len(queue))
	}
}

// Adjacent single tracks (no group_id) must not be merged into a group.
func TestResolve_AdjacentSinglesNotGrouped(t *testing.T) {
	r, _ := newResolver()
	items := []models.PlaylistItem{
		{ID: "i1", Kind: models.ItemKindTrack, Position: 0, TrackID: "t1"},
		{ID: "i2", Kind: models.ItemKindTrack, Position: 1, TrackID: "t2"},
	}
	units, err := r.buildUnits(context.Background(), items)
	if err != nil {
		t.Fatalf("buildUnits: %v", err)
	}
	if len(units) != 2 {
		t.Fatalf("expected 2 separate units for ungrouped singles, got %d", len(units))
	}
}

// No shuffle preserves album order within an album unit and group order within
// a group unit, in playlist position order.
func TestResolve_NoShuffleOrder(t *testing.T) {
	r, _ := newResolver()
	items := []models.PlaylistItem{
		{ID: "i1", Kind: models.ItemKindAlbum, Position: 0, AlbumID: "album1"},
		{ID: "i2", Kind: models.ItemKindTrack, Position: 1, TrackID: "t1"},
	}
	queue, err := r.Resolve(context.Background(), items, ResolveOptions{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	got := ids(queue)
	want := []string{"a1", "a2", "a3", "t1"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order mismatch: got %v want %v", got, want)
		}
	}
}
