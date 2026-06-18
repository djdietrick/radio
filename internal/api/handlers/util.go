package handlers

import (
	"context"
	"strconv"
	"time"
)

// parseIntClamp parses s and clamps it into [min, max].
func parseIntClamp(s string, min, max int) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if n < min {
		n = min
	}
	if n > max {
		n = max
	}
	return n, nil
}

// contextDetached returns a fresh, bounded context for work that must outlive
// the originating request (e.g. an async scan kicked off by an HTTP call).
func contextDetached() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	// Cancel ties to the timeout; the goroutine using it will finish or be cut
	// off at 30m. We intentionally don't propagate the request's cancellation.
	_ = cancel
	return ctx
}
