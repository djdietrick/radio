// Package probe extracts technical stream info (currently duration) from audio
// files using ffprobe. ffprobe is treated as optional: if the binary is not
// available, probing degrades gracefully to an unknown (0) duration, which the
// radio engine already handles with a fallback length.
package probe

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strconv"
	"time"
)

// ErrUnavailable indicates ffprobe is not installed / not on PATH.
var ErrUnavailable = errors.New("probe: ffprobe not available")

// Prober runs ffprobe against files. The zero value is not usable; use New.
type Prober struct {
	// binPath is the resolved ffprobe executable, empty when unavailable.
	binPath string
	// timeout bounds a single probe invocation.
	timeout time.Duration
}

// New locates ffprobe on PATH. Available() reports whether it was found; a
// Prober is always returned so callers can probe unconditionally and let it
// no-op when ffprobe is missing.
func New() *Prober {
	p := &Prober{timeout: 30 * time.Second}
	if path, err := exec.LookPath("ffprobe"); err == nil {
		p.binPath = path
	}
	return p
}

// Available reports whether ffprobe was found.
func (p *Prober) Available() bool { return p.binPath != "" }

// ffprobeOutput is the minimal subset of ffprobe -print_format json we read.
type ffprobeOutput struct {
	Format struct {
		Duration string `json:"duration"` // seconds, as a string like "243.21"
	} `json:"format"`
}

// DurationMs returns the file's duration in milliseconds. It returns
// (0, ErrUnavailable) when ffprobe is not installed, and (0, err) on probe
// failure; callers typically treat both as "unknown duration" and proceed.
func (p *Prober) DurationMs(ctx context.Context, path string) (int64, error) {
	if !p.Available() {
		return 0, ErrUnavailable
	}

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, p.binPath,
		"-v", "error",
		"-show_entries", "format=duration",
		"-print_format", "json",
		path,
	)

	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	return parseDurationMs(out)
}

// parseDurationMs converts ffprobe's JSON output into milliseconds. Factored
// out from DurationMs so the parsing is unit-testable without invoking ffprobe.
func parseDurationMs(out []byte) (int64, error) {
	var parsed ffprobeOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return 0, err
	}
	if parsed.Format.Duration == "" {
		return 0, nil // probed fine but the container reports no duration
	}

	secs, err := strconv.ParseFloat(parsed.Format.Duration, 64)
	if err != nil {
		return 0, err
	}
	if secs < 0 {
		secs = 0
	}
	return int64(secs * 1000), nil
}
