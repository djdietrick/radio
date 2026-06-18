// Package art stores and locates album cover images on disk.
//
// Cover bytes are cached as files under a single directory, named by album id
// (e.g. "<albumID>.jpg"). The albums.has_art column remains the source of truth
// for whether art exists; this cache holds the actual bytes so image blobs stay
// out of Postgres. The cache directory is mounted as a writable volume.
package art

import (
	"errors"
	"os"
	"path/filepath"
)

// ErrNotFound indicates no cached art exists for an album.
var ErrNotFound = errors.New("art: not found")

// Cache reads and writes album art files in a directory.
type Cache struct {
	dir string
}

// New returns a Cache rooted at dir. The directory is expected to already
// exist (main.go creates it on startup).
func New(dir string) *Cache {
	return &Cache{dir: dir}
}

// extForMime maps a cover MIME type to a file extension. Defaults to .jpg,
// which covers the overwhelmingly common case of embedded JPEG covers.
func extForMime(mime string) string {
	switch mime {
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".jpg"
	}
}

// mimeForExt is the inverse, used when serving a cached file.
func mimeForExt(ext string) string {
	switch ext {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return "image/jpeg"
	}
}

// Has reports whether art is cached for albumID.
func (c *Cache) Has(albumID string) bool {
	_, _, err := c.locate(albumID)
	return err == nil
}

// Save writes cover bytes for albumID, returning false (no error) if art for
// the album is already cached — first art wins, so re-scans don't churn files.
func (c *Cache) Save(albumID, mime string, data []byte) (bool, error) {
	if len(data) == 0 {
		return false, nil
	}
	if c.Has(albumID) {
		return false, nil
	}
	path := filepath.Join(c.dir, albumID+extForMime(mime))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// Open returns a readable file handle for an album's cached art, its MIME type,
// and the file's size. The caller must Close the returned file.
func (c *Cache) Open(albumID string) (*os.File, string, int64, error) {
	path, ext, err := c.locate(albumID)
	if err != nil {
		return nil, "", 0, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, "", 0, ErrNotFound
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, "", 0, err
	}
	return f, mimeForExt(ext), info.Size(), nil
}

// locate finds the cached file for albumID across supported extensions.
func (c *Cache) locate(albumID string) (path, ext string, err error) {
	for _, e := range []string{".jpg", ".png", ".webp", ".gif"} {
		p := filepath.Join(c.dir, albumID+e)
		if _, statErr := os.Stat(p); statErr == nil {
			return p, e, nil
		}
	}
	return "", "", ErrNotFound
}
