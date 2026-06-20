import { create } from "zustand";
import { NowPlaying, Station, Track } from "../api";

// The player store is the single source of truth for what's playing. The actual
// <audio> element lives in PlayerBar and reacts to this state; the WebSocket for
// radio mode is driven by RadioController. Two modes coexist:
//
//   - "library": a user-built queue with full transport (next/prev/shuffle/
//     repeat/seek).
//   - "radio":   synced to a station's deterministic broadcast position. Transport
//     is disabled — position is computed server-side and pushed over the socket.

export type RepeatMode = "off" | "all" | "one";
export type PlayerMode = "library" | "radio";

interface PlayerState {
  mode: PlayerMode;

  // library mode
  queue: Track[]; // current playback order
  originalQueue: Track[]; // pre-shuffle order, to restore on unshuffle
  index: number;
  shuffle: boolean;
  repeat: RepeatMode;

  // radio mode
  station: Station | null;
  nowPlaying: NowPlaying | null;

  // transport state (PlayerBar mirrors the <audio> element here)
  isPlaying: boolean;
  positionMs: number;
  durationMs: number;

  // selectors
  currentTrack: () => Track | null;

  // library actions
  playQueue: (tracks: Track[], startIndex?: number) => void;
  addToQueue: (tracks: Track[]) => void;
  playNext: () => boolean; // returns false when the queue is exhausted
  playPrev: () => void;
  jumpTo: (index: number) => void;
  removeFromQueue: (index: number) => void;
  toggleShuffle: () => void;
  cycleRepeat: () => void;

  // radio actions
  tuneStation: (station: Station, np: NowPlaying) => void;
  updateNowPlaying: (np: NowPlaying) => void;
  leaveStation: () => void;

  // shared transport
  setPlaying: (playing: boolean) => void;
  setProgress: (positionMs: number, durationMs: number) => void;
}

function shuffled<T>(items: T[]): T[] {
  const out = items.slice();
  for (let i = out.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [out[i], out[j]] = [out[j], out[i]];
  }
  return out;
}

export const usePlayer = create<PlayerState>((set, get) => ({
  mode: "library",
  queue: [],
  originalQueue: [],
  index: -1,
  shuffle: false,
  repeat: "off",
  station: null,
  nowPlaying: null,
  isPlaying: false,
  positionMs: 0,
  durationMs: 0,

  currentTrack: () => {
    const s = get();
    if (s.mode === "radio") return s.nowPlaying?.track ?? null;
    return s.index >= 0 && s.index < s.queue.length ? s.queue[s.index] : null;
  },

  playQueue: (tracks, startIndex = 0) => {
    if (tracks.length === 0) return;
    const { shuffle } = get();
    if (shuffle) {
      // Keep the chosen track first, shuffle the rest.
      const head = tracks[startIndex];
      const rest = tracks.filter((_, i) => i !== startIndex);
      set({
        mode: "library",
        originalQueue: tracks,
        queue: [head, ...shuffled(rest)],
        index: 0,
        station: null,
        nowPlaying: null,
        isPlaying: true,
      });
    } else {
      set({
        mode: "library",
        originalQueue: tracks,
        queue: tracks,
        index: startIndex,
        station: null,
        nowPlaying: null,
        isPlaying: true,
      });
    }
  },

  addToQueue: (tracks) => {
    const s = get();
    if (s.mode === "radio" || s.queue.length === 0) {
      // Nothing playing locally — start a fresh library queue.
      get().playQueue(tracks, 0);
      return;
    }
    set({
      queue: [...s.queue, ...tracks],
      originalQueue: [...s.originalQueue, ...tracks],
    });
  },

  playNext: () => {
    const s = get();
    if (s.queue.length === 0) return false;
    if (s.index < s.queue.length - 1) {
      set({ index: s.index + 1, isPlaying: true });
      return true;
    }
    if (s.repeat === "all") {
      set({ index: 0, isPlaying: true });
      return true;
    }
    // End of queue, no repeat.
    set({ isPlaying: false });
    return false;
  },

  playPrev: () => {
    const s = get();
    if (s.queue.length === 0 || s.index <= 0) return;
    set({ index: s.index - 1, isPlaying: true });
  },

  jumpTo: (index) => {
    const s = get();
    if (index < 0 || index >= s.queue.length) return;
    set({ index, isPlaying: true });
  },

  removeFromQueue: (index) => {
    const s = get();
    if (index < 0 || index >= s.queue.length) return;
    const queue = s.queue.filter((_, i) => i !== index);
    let newIndex = s.index;
    if (index < s.index) newIndex = s.index - 1;
    else if (index === s.index) newIndex = Math.min(s.index, queue.length - 1);
    set({ queue, index: queue.length === 0 ? -1 : newIndex });
  },

  toggleShuffle: () => {
    const s = get();
    const next = !s.shuffle;
    if (s.queue.length === 0) {
      set({ shuffle: next });
      return;
    }
    const current = s.queue[s.index];
    if (next) {
      const rest = s.queue.filter((_, i) => i !== s.index);
      set({ shuffle: true, queue: [current, ...shuffled(rest)], index: 0 });
    } else {
      // Restore original order, keeping the current track active.
      const idx = s.originalQueue.findIndex((t) => t === current);
      set({
        shuffle: false,
        queue: s.originalQueue,
        index: idx >= 0 ? idx : 0,
      });
    }
  },

  cycleRepeat: () => {
    const order: RepeatMode[] = ["off", "all", "one"];
    const s = get();
    set({ repeat: order[(order.indexOf(s.repeat) + 1) % order.length] });
  },

  tuneStation: (station, np) => {
    set({
      mode: "radio",
      station,
      nowPlaying: np,
      queue: [],
      originalQueue: [],
      index: -1,
      isPlaying: true,
    });
  },

  updateNowPlaying: (np) => {
    // Ignore stale pushes for a station we've left.
    if (get().station?.id !== np.stationId) return;
    set({ nowPlaying: np });
  },

  leaveStation: () => {
    set({
      mode: "library",
      station: null,
      nowPlaying: null,
      isPlaying: false,
    });
  },

  setPlaying: (playing) => set({ isPlaying: playing }),
  setProgress: (positionMs, durationMs) => set({ positionMs, durationMs }),
}));
