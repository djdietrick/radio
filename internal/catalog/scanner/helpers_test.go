package scanner

import (
	"context"
	"io"
	"log/slog"
	"os/exec"
	"strconv"
	"time"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// runFFmpegTone generates a sine-wave mp3 of the given length using ffmpeg.
// Returns an error (rather than failing) so callers can skip when ffmpeg is
// absent.
func runFFmpegTone(path string, secs int) error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffmpeg", "-y",
		"-f", "lavfi", "-i", "sine=frequency=440:duration="+strconv.Itoa(secs),
		"-c:a", "libmp3lame", path,
	)
	return cmd.Run()
}
