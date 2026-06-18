package radio

import (
	"testing"
	"time"

	"github.com/djdietrick/radio/internal/models"
)

func mkQueue(durationsMs ...int64) []models.Track {
	q := make([]models.Track, len(durationsMs))
	for i, d := range durationsMs {
		q[i] = models.Track{ID: string(rune('a' + i)), DurationMs: d}
	}
	return q
}

func TestComputeNowPlaying_WithinFirstTrack(t *testing.T) {
	start := time.Unix(0, 0)
	st := &models.Station{ID: "s", StartedAt: start, Loop: true}
	q := mkQueue(60000, 120000)

	np := computeNowPlaying(st, q, start.Add(30*time.Second))
	if np.IndexInQueue != 0 || np.OffsetMs != 30000 {
		t.Fatalf("got index=%d offset=%d, want 0/30000", np.IndexInQueue, np.OffsetMs)
	}
}

func TestComputeNowPlaying_SecondTrack(t *testing.T) {
	start := time.Unix(0, 0)
	st := &models.Station{ID: "s", StartedAt: start, Loop: true}
	q := mkQueue(60000, 120000)

	np := computeNowPlaying(st, q, start.Add(90*time.Second))
	if np.IndexInQueue != 1 || np.OffsetMs != 30000 {
		t.Fatalf("got index=%d offset=%d, want 1/30000", np.IndexInQueue, np.OffsetMs)
	}
}

func TestComputeNowPlaying_Loops(t *testing.T) {
	start := time.Unix(0, 0)
	st := &models.Station{ID: "s", StartedAt: start, Loop: true}
	q := mkQueue(60000, 120000) // total 180s

	// 190s in => 10s into the loop => 10s into first track
	np := computeNowPlaying(st, q, start.Add(190*time.Second))
	if np.IndexInQueue != 0 || np.OffsetMs != 10000 {
		t.Fatalf("got index=%d offset=%d, want 0/10000", np.IndexInQueue, np.OffsetMs)
	}
}

func TestComputeNowPlaying_NonLoopEnds(t *testing.T) {
	start := time.Unix(0, 0)
	st := &models.Station{ID: "s", StartedAt: start, Loop: false}
	q := mkQueue(60000, 120000) // total 180s

	np := computeNowPlaying(st, q, start.Add(300*time.Second))
	if !np.Ended {
		t.Fatalf("expected Ended=true past program end")
	}
	if np.IndexInQueue != 1 {
		t.Fatalf("expected to pin to last track, got index=%d", np.IndexInQueue)
	}
}

func TestComputeNowPlaying_UnknownDurationUsesFallback(t *testing.T) {
	start := time.Unix(0, 0)
	st := &models.Station{ID: "s", StartedAt: start, Loop: true}
	q := mkQueue(0) // unknown duration -> 180s fallback

	np := computeNowPlaying(st, q, start.Add(60*time.Second))
	if np.IndexInQueue != 0 || np.OffsetMs != 60000 {
		t.Fatalf("got index=%d offset=%d, want 0/60000", np.IndexInQueue, np.OffsetMs)
	}
}

func TestComputeNowPlaying_FutureStartPinsToStart(t *testing.T) {
	start := time.Unix(1000, 0)
	st := &models.Station{ID: "s", StartedAt: start, Loop: true}
	q := mkQueue(60000)

	np := computeNowPlaying(st, q, start.Add(-30*time.Second))
	if np.IndexInQueue != 0 || np.OffsetMs != 0 {
		t.Fatalf("got index=%d offset=%d, want 0/0", np.IndexInQueue, np.OffsetMs)
	}
}

func TestComputeNowPlaying_MsUntilNextTrack(t *testing.T) {
	start := time.Unix(0, 0)
	st := &models.Station{ID: "s", StartedAt: start, Loop: true}
	q := mkQueue(60000, 120000)

	// 30s into a 60s first track => 30s remain.
	np := computeNowPlaying(st, q, start.Add(30*time.Second))
	if np.MsUntilNextTrack != 30000 {
		t.Fatalf("msUntilNextTrack=%d, want 30000", np.MsUntilNextTrack)
	}

	// 90s in => 30s into the 120s second track => 90s remain.
	np = computeNowPlaying(st, q, start.Add(90*time.Second))
	if np.MsUntilNextTrack != 90000 {
		t.Fatalf("msUntilNextTrack=%d, want 90000", np.MsUntilNextTrack)
	}
}

func TestComputeNowPlaying_EndedHasZeroUntilNext(t *testing.T) {
	start := time.Unix(0, 0)
	st := &models.Station{ID: "s", StartedAt: start, Loop: false}
	q := mkQueue(60000, 120000)

	np := computeNowPlaying(st, q, start.Add(300*time.Second))
	if !np.Ended || np.MsUntilNextTrack != 0 {
		t.Fatalf("ended=%v msUntilNextTrack=%d, want true/0", np.Ended, np.MsUntilNextTrack)
	}
}
