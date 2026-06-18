// Package ws implements the WebSocket live-sync endpoint for radio stations.
//
// A connected client receives the station's NowPlaying immediately, then a
// fresh NowPlaying each time the station advances to the next track. The server
// schedules each push at the exact track boundary (using MsUntilNextTrack)
// rather than polling, so an idle connection costs one timer, not a busy loop.
package ws

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/djdietrick/radio/internal/radio"
)

// StationHandler upgrades requests to WebSocket and streams station state.
type StationHandler struct {
	engine *radio.Engine
	log    *slog.Logger
	// allowedOrigins is passed to the WS accept options. Empty means same-origin
	// only; use InsecureSkipVerify-style behavior via "*" for dev.
	allowAllOrigins bool
}

func NewStationHandler(engine *radio.Engine, allowAllOrigins bool, log *slog.Logger) *StationHandler {
	return &StationHandler{engine: engine, log: log, allowAllOrigins: allowAllOrigins}
}

// minBoundaryDelay clamps how soon the next push can be scheduled, guarding
// against a tight loop if a track reports a near-zero remaining time.
const minBoundaryDelay = 250 * time.Millisecond

// Serve handles one station live-sync connection. stationID is supplied by the
// router.
func (h *StationHandler) Serve(w http.ResponseWriter, r *http.Request, stationID string) {
	opts := &websocket.AcceptOptions{}
	if h.allowAllOrigins {
		opts.InsecureSkipVerify = true // dev: allow cross-origin (Vite on :5173)
	}

	conn, err := websocket.Accept(w, r, opts)
	if err != nil {
		// Accept already wrote an error response.
		return
	}
	defer conn.Close(websocket.StatusInternalError, "closing")

	ctx := r.Context()

	// Resolve the queue once for the lifetime of this connection.
	st, err := h.engine.GetStation(ctx, stationID)
	if err != nil {
		conn.Close(websocket.StatusPolicyViolation, "station not found")
		return
	}
	session, err := h.engine.NewLiveSession(ctx, st)
	if err != nil {
		conn.Close(websocket.StatusPolicyViolation, "station has no playable tracks")
		return
	}

	// Detect client disconnect: read in the background; any result cancels.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		// We don't expect client messages; a read returning means the peer
		// closed or errored. Either way, tear down.
		conn.Read(ctx)
		cancel()
	}()

	h.loop(ctx, conn, session)
	conn.Close(websocket.StatusNormalClosure, "")
}

// loop pushes NowPlaying now and at each subsequent track boundary until the
// context is cancelled or the (non-looping) station ends.
func (h *StationHandler) loop(ctx context.Context, conn *websocket.Conn, session *radio.LiveSession) {
	for {
		np := session.At(time.Now())

		if err := h.write(ctx, conn, np); err != nil {
			return
		}

		if np.Ended {
			// Non-looping station finished; nothing more will change.
			return
		}

		delay := time.Duration(np.MsUntilNextTrack) * time.Millisecond
		if delay < minBoundaryDelay {
			delay = minBoundaryDelay
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			// Loop around: recompute at the new boundary and push again.
		}
	}
}

func (h *StationHandler) write(ctx context.Context, conn *websocket.Conn, v any) error {
	// Bound a single write so a stuck client can't block forever.
	writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := wsjson.Write(writeCtx, conn, v); err != nil {
		if ctx.Err() == nil {
			h.log.Debug("ws write failed", "err", err)
		}
		return err
	}
	return nil
}
