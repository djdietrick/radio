package metadata

import (
	"context"
	"errors"
	"testing"
)

// fakeExternal is a controllable externalClient for testing EnrichExternal
// without any network access.
type fakeExternal struct {
	mbid     string
	mbidErr  error
	cover    []byte
	coverMime string
	coverErr error

	findCalls  int
	coverCalls int
}

func (f *fakeExternal) FindReleaseMBID(_ context.Context, _, _ string) (string, error) {
	f.findCalls++
	return f.mbid, f.mbidErr
}

func (f *fakeExternal) FetchFrontCover(_ context.Context, _ string) ([]byte, string, error) {
	f.coverCalls++
	return f.cover, f.coverMime, f.coverErr
}

func TestEnrichExternal_DisabledIsNoOp(t *testing.T) {
	e := NewExtractor(false)
	in := &Tags{Album: "A", AlbumArtist: "B"}
	out, err := e.EnrichExternal(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.HasEmbeddedArt {
		t.Fatal("disabled extractor should not fetch art")
	}
}

func TestEnrichExternal_SkipsWhenAlreadyHasArt(t *testing.T) {
	fake := &fakeExternal{mbid: "x", cover: []byte("img"), coverMime: "image/png"}
	e := NewExtractorWithExternal(fake)
	in := &Tags{Album: "A", AlbumArtist: "B", HasEmbeddedArt: true, Art: []byte("orig")}

	out, err := e.EnrichExternal(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if fake.findCalls != 0 {
		t.Fatal("should not query MusicBrainz when art already present")
	}
	if string(out.Art) != "orig" {
		t.Fatal("existing art must be preserved")
	}
}

func TestEnrichExternal_SkipsWhenNoAlbum(t *testing.T) {
	fake := &fakeExternal{}
	e := NewExtractorWithExternal(fake)
	_, err := e.EnrichExternal(context.Background(), &Tags{AlbumArtist: "B"})
	if err != nil {
		t.Fatal(err)
	}
	if fake.findCalls != 0 {
		t.Fatal("should not search without an album title")
	}
}

func TestEnrichExternal_FetchesAndSetsArt(t *testing.T) {
	fake := &fakeExternal{mbid: "rel-123", cover: []byte("coverbytes"), coverMime: "image/png"}
	e := NewExtractorWithExternal(fake)

	out, err := e.EnrichExternal(context.Background(), &Tags{Album: "A", AlbumArtist: "B"})
	if err != nil {
		t.Fatal(err)
	}
	if !out.HasEmbeddedArt {
		t.Fatal("expected HasEmbeddedArt set after fetch")
	}
	if string(out.Art) != "coverbytes" || out.ArtMime != "image/png" {
		t.Fatalf("art not set correctly: %q %q", out.Art, out.ArtMime)
	}
	if fake.coverCalls != 1 {
		t.Fatalf("expected one cover fetch, got %d", fake.coverCalls)
	}
}

func TestEnrichExternal_NoReleaseMatch(t *testing.T) {
	fake := &fakeExternal{mbid: ""} // no match
	e := NewExtractorWithExternal(fake)

	out, err := e.EnrichExternal(context.Background(), &Tags{Album: "A", AlbumArtist: "B"})
	if err != nil {
		t.Fatal(err)
	}
	if out.HasEmbeddedArt {
		t.Fatal("should not set art when no release matched")
	}
	if fake.coverCalls != 0 {
		t.Fatal("should not fetch cover without an MBID")
	}
}

func TestEnrichExternal_ReleaseFoundButNoCover(t *testing.T) {
	fake := &fakeExternal{mbid: "rel-123", cover: nil} // matched, but no archived cover
	e := NewExtractorWithExternal(fake)

	out, err := e.EnrichExternal(context.Background(), &Tags{Album: "A", AlbumArtist: "B"})
	if err != nil {
		t.Fatal(err)
	}
	if out.HasEmbeddedArt {
		t.Fatal("should not set art when no cover returned")
	}
}

func TestEnrichExternal_PropagatesErrors(t *testing.T) {
	fake := &fakeExternal{mbidErr: errors.New("network down")}
	e := NewExtractorWithExternal(fake)

	_, err := e.EnrichExternal(context.Background(), &Tags{Album: "A", AlbumArtist: "B"})
	if err == nil {
		t.Fatal("expected the MusicBrainz error to propagate")
	}
}
