// Package metadata extracts tags and embedded cover art from audio files.
//
// Text metadata and embedded art are read locally via the dhowden/tag library.
// When configured with an external client, missing cover art is filled from
// MusicBrainz + the Cover Art Archive via EnrichExternal.
package metadata

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/dhowden/tag"
)

// Tags is the normalized metadata extracted from a file.
type Tags struct {
	Title       string
	Artist      string
	Album       string
	AlbumArtist string
	Genre       string
	Year        int
	TrackNumber int
	DiscNumber  int
	Codec       string // e.g. "MP3", "FLAC", "AAC"
	MimeType    string
	// HasEmbeddedArt is true when cover art is embedded in the file.
	HasEmbeddedArt bool
	// Art holds raw embedded cover bytes (nil if absent).
	Art []byte
	ArtMime string
}

// externalClient is the subset of the external package's Client used here,
// declared as an interface so the extractor can be constructed without a real
// network client (and tested) when external metadata is disabled.
type externalClient interface {
	FindReleaseMBID(ctx context.Context, albumArtist, album string) (string, error)
	FetchFrontCover(ctx context.Context, releaseMBID string) ([]byte, string, error)
}

// Extractor reads metadata from files.
type Extractor struct {
	externalEnabled bool
	external        externalClient
}

// NewExtractor builds an Extractor. When externalEnabled is true an external
// client must be supplied (see WithExternal); otherwise EnrichExternal is a
// no-op. Use NewExtractor for the disabled case and NewExtractorWithExternal
// for the enabled case.
func NewExtractor(externalEnabled bool) *Extractor {
	return &Extractor{externalEnabled: externalEnabled}
}

// NewExtractorWithExternal builds an Extractor wired to an external metadata
// client (MusicBrainz / Cover Art Archive).
func NewExtractorWithExternal(client externalClient) *Extractor {
	return &Extractor{externalEnabled: true, external: client}
}

// SupportedExt reports whether path has an audio extension we catalog.
func SupportedExt(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3", ".flac", ".m4a", ".aac", ".ogg", ".opus", ".wav", ".wma":
		return true
	}
	return false
}

// mimeForExt returns a best-effort MIME type used for HTTP streaming.
func mimeForExt(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3":
		return "audio/mpeg"
	case ".flac":
		return "audio/flac"
	case ".m4a", ".aac":
		return "audio/mp4"
	case ".ogg":
		return "audio/ogg"
	case ".opus":
		return "audio/opus"
	case ".wav":
		return "audio/wav"
	case ".wma":
		return "audio/x-ms-wma"
	default:
		return "application/octet-stream"
	}
}

// Extract reads embedded tags and art from the file at path.
func (e *Extractor) Extract(path string) (*Tags, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	t := &Tags{MimeType: mimeForExt(path)}

	m, err := tag.ReadFrom(f)
	if err != nil {
		// File is unreadable as a known tag format; fall back to filename.
		t.Title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		t.Codec = strings.ToUpper(strings.TrimPrefix(filepath.Ext(path), "."))
		return t, nil
	}

	t.Title = m.Title()
	t.Artist = m.Artist()
	t.Album = m.Album()
	t.AlbumArtist = m.AlbumArtist()
	t.Genre = m.Genre()
	t.Year = m.Year()
	t.Codec = string(m.FileType())
	t.TrackNumber, _ = m.Track()
	t.DiscNumber, _ = m.Disc()

	if pic := m.Picture(); pic != nil {
		t.HasEmbeddedArt = true
		t.Art = pic.Data
		t.ArtMime = pic.MIMEType
	}

	if t.Title == "" {
		t.Title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	if t.AlbumArtist == "" {
		t.AlbumArtist = t.Artist
	}
	return t, nil
}

// EnrichExternal fills missing cover art from MusicBrainz + the Cover Art
// Archive. It is a no-op when external metadata is disabled, when the file
// already has embedded art, or when there isn't enough album info to search.
// On success it sets t.Art / t.ArtMime / t.HasEmbeddedArt so downstream art
// caching treats it identically to embedded art.
//
// Network errors are returned to the caller, which logs and continues; a failed
// enrichment never blocks indexing.
func (e *Extractor) EnrichExternal(ctx context.Context, t *Tags) (*Tags, error) {
	if !e.externalEnabled || e.external == nil {
		return t, nil
	}
	if t.HasEmbeddedArt {
		return t, nil // already have art; nothing to fetch
	}
	if strings.TrimSpace(t.Album) == "" {
		return t, nil // can't match a release without an album title
	}

	mbid, err := e.external.FindReleaseMBID(ctx, t.AlbumArtist, t.Album)
	if err != nil {
		return t, err
	}
	if mbid == "" {
		return t, nil // no matching release
	}

	art, mime, err := e.external.FetchFrontCover(ctx, mbid)
	if err != nil {
		return t, err
	}
	if len(art) == 0 {
		return t, nil // matched a release but no cover archived
	}

	t.Art = art
	t.ArtMime = mime
	t.HasEmbeddedArt = true
	return t, nil
}
