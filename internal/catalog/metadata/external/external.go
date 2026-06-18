// Package external fetches album metadata and cover art from MusicBrainz and
// the Cover Art Archive (CAA) when local tags are incomplete.
//
// Flow: a release is matched on MusicBrainz by (album-artist, album) to obtain
// a release MBID, then the front cover is fetched from CAA using that MBID.
// MusicBrainz asks clients to send a descriptive User-Agent and to cap requests
// at roughly one per second; this client enforces both.
package external

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

const (
	musicBrainzBase = "https://musicbrainz.org/ws/2"
	coverArtBase    = "https://coverartarchive.org"
	// userAgent identifies this app to MusicBrainz, per their requirements.
	// (They ask for app/version (contact) — adjust the contact when deploying.)
	userAgent = "radio-server/0.1 (https://github.com/djdietrick/radio)"
	// maxArtBytes guards against pathologically large covers.
	maxArtBytes = 12 << 20 // 12 MiB
)

// Client talks to MusicBrainz + Cover Art Archive over HTTP.
type Client struct {
	http    *http.Client
	limiter *rate.Limiter
}

// New returns a Client. MusicBrainz's rate guidance is ~1 req/sec sustained;
// the limiter applies to MusicBrainz lookups (CAA is more lenient but we let it
// share the budget for simplicity).
func New() *Client {
	return &Client{
		http:    &http.Client{Timeout: 15 * time.Second},
		limiter: rate.NewLimiter(rate.Every(time.Second), 1),
	}
}

// release is the subset of the MusicBrainz release-search response we read.
type mbReleaseSearch struct {
	Releases []struct {
		ID    string `json:"id"`
		Score int    `json:"score"`
	} `json:"releases"`
}

// FindReleaseMBID searches MusicBrainz for a release matching the given album
// artist and title, returning the best-scoring release's MBID. Returns an empty
// id (no error) when nothing matches.
func (c *Client) FindReleaseMBID(ctx context.Context, albumArtist, album string) (string, error) {
	if strings.TrimSpace(album) == "" {
		return "", nil
	}

	// Lucene-style query against the release index.
	var q strings.Builder
	fmt.Fprintf(&q, `release:%s`, luceneQuote(album))
	if strings.TrimSpace(albumArtist) != "" {
		fmt.Fprintf(&q, ` AND artist:%s`, luceneQuote(albumArtist))
	}

	u := fmt.Sprintf("%s/release/?query=%s&fmt=json&limit=3",
		musicBrainzBase, url.QueryEscape(q.String()))

	body, err := c.get(ctx, u)
	if err != nil {
		return "", err
	}
	var parsed mbReleaseSearch
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Releases) == 0 {
		return "", nil
	}
	// MusicBrainz returns results sorted by score; take the top one.
	return parsed.Releases[0].ID, nil
}

// FetchFrontCover downloads the front cover image for a release MBID from the
// Cover Art Archive. Returns (nil, "", nil) when no front cover exists (CAA
// responds 404), which callers treat as "no art available".
func (c *Client) FetchFrontCover(ctx context.Context, releaseMBID string) ([]byte, string, error) {
	if releaseMBID == "" {
		return nil, "", nil
	}
	// The /front endpoint 307-redirects to the actual image; the default
	// http.Client follows redirects.
	u := fmt.Sprintf("%s/release/%s/front", coverArtBase, url.PathEscape(releaseMBID))

	req, err := c.newRequest(ctx, u)
	if err != nil {
		return nil, "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, "", nil // no cover archived for this release
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("external: cover art archive returned %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxArtBytes))
	if err != nil {
		return nil, "", err
	}
	mime := resp.Header.Get("Content-Type")
	if mime == "" {
		mime = "image/jpeg"
	}
	return data, mime, nil
}

// get issues a rate-limited GET against MusicBrainz and returns the body for a
// 200 response.
func (c *Client) get(ctx context.Context, u string) ([]byte, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	req, err := c.newRequest(ctx, u)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("external: %s returned %d", u, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxArtBytes))
}

func (c *Client) newRequest(ctx context.Context, u string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	return req, nil
}

// luceneQuote wraps a value in quotes and escapes embedded quotes/backslashes
// so it can be used safely inside a MusicBrainz Lucene query term.
func luceneQuote(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return `"` + r.Replace(s) + `"`
}
