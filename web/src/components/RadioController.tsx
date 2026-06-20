import { useEffect } from "react";
import { api, NowPlaying } from "../api";
import { usePlayer } from "../store/player";

// RadioController owns the station WebSocket so radio playback survives page
// navigation (it's mounted once in the Layout, not per-page). The server pushes
// the current track + offset on connect and again at each track boundary; we
// fan those into the player store, which PlayerBar reacts to.
//
// On unexpected socket close it retries with backoff, then falls back to the
// REST now-playing endpoint so a dropped connection doesn't strand the listener.
export function RadioController() {
  const stationId = usePlayer((s) => s.station?.id ?? null);
  const updateNowPlaying = usePlayer((s) => s.updateNowPlaying);

  useEffect(() => {
    if (!stationId) return;

    let ws: WebSocket | null = null;
    let retry: ReturnType<typeof setTimeout> | null = null;
    let attempts = 0;
    let closed = false;

    const apply = (np: NowPlaying) => updateNowPlaying(np);

    const connect = () => {
      if (closed) return;
      ws = new WebSocket(api.stationWsUrl(stationId));
      ws.onmessage = (ev) => {
        try {
          apply(JSON.parse(ev.data) as NowPlaying);
          attempts = 0;
        } catch {
          /* ignore malformed frame */
        }
      };
      ws.onclose = () => {
        if (closed) return;
        // Backoff reconnect, and resync via REST in the meantime.
        api.nowPlaying(stationId).then(apply).catch(() => undefined);
        const delay = Math.min(1000 * 2 ** attempts, 15000);
        attempts++;
        retry = setTimeout(connect, delay);
      };
      ws.onerror = () => ws?.close();
    };

    connect();

    return () => {
      closed = true;
      if (retry) clearTimeout(retry);
      ws?.close();
    };
  }, [stationId, updateNowPlaying]);

  return null;
}
