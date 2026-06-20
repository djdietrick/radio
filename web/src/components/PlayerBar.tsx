import { useEffect, useRef } from "react";
import { Link } from "react-router-dom";
import { api } from "../api";
import { formatDuration } from "../format";
import { usePlayer } from "../store/player";
import { AlbumArt } from "./AlbumArt";

// PlayerBar is the persistent transport at the bottom of the app. It owns the
// single <audio> element and reconciles it with the player store. In library
// mode it offers full transport; in radio mode it follows the station's pushed
// now-playing position and disables seek/skip.
export function PlayerBar() {
  const audioRef = useRef<HTMLAudioElement>(null);
  const mode = usePlayer((s) => s.mode);
  const track = usePlayer((s) => s.currentTrack());
  const isPlaying = usePlayer((s) => s.isPlaying);
  const positionMs = usePlayer((s) => s.positionMs);
  const durationMs = usePlayer((s) => s.durationMs);
  const shuffle = usePlayer((s) => s.shuffle);
  const repeat = usePlayer((s) => s.repeat);
  const nowPlaying = usePlayer((s) => s.nowPlaying);
  const station = usePlayer((s) => s.station);

  const setPlaying = usePlayer((s) => s.setPlaying);
  const setProgress = usePlayer((s) => s.setProgress);
  const playNext = usePlayer((s) => s.playNext);
  const playPrev = usePlayer((s) => s.playPrev);
  const toggleShuffle = usePlayer((s) => s.toggleShuffle);
  const cycleRepeat = usePlayer((s) => s.cycleRepeat);
  const leaveStation = usePlayer((s) => s.leaveStation);

  const trackId = track?.id ?? null;

  // Load a new source when the active track changes. In radio mode also seek to
  // the broadcast offset so this listener is in sync with everyone else.
  useEffect(() => {
    const el = audioRef.current;
    if (!el || !trackId) return;
    const src = api.streamUrl(trackId);
    if (!el.src.endsWith(src)) el.src = src;
    if (mode === "radio" && nowPlaying) {
      el.currentTime = nowPlaying.offsetMs / 1000;
    }
    el.play().catch(() => {
      /* autoplay may require a user gesture; the play button recovers */
    });
    // nowPlaying identity changes on every push; we intentionally re-sync on
    // track change only (offset is read fresh above).
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [trackId, mode]);

  // Reflect play/pause intent onto the element.
  useEffect(() => {
    const el = audioRef.current;
    if (!el || !trackId) return;
    if (isPlaying) el.play().catch(() => undefined);
    else el.pause();
  }, [isPlaying, trackId]);

  const onEnded = () => {
    if (mode === "radio") return; // station pushes drive radio advancement
    if (repeat === "one") {
      const el = audioRef.current;
      if (el) {
        el.currentTime = 0;
        el.play().catch(() => undefined);
      }
      return;
    }
    playNext();
  };

  const onSeek = (e: React.ChangeEvent<HTMLInputElement>) => {
    const el = audioRef.current;
    if (!el || mode === "radio") return;
    el.currentTime = Number(e.target.value) / 1000;
  };

  const prev = () => {
    const el = audioRef.current;
    if (el && el.currentTime > 3) {
      el.currentTime = 0;
      return;
    }
    playPrev();
  };

  const empty = !track;

  return (
    <div className="playerbar">
      <audio
        ref={audioRef}
        onPlay={() => setPlaying(true)}
        onPause={() => setPlaying(false)}
        onEnded={onEnded}
        onTimeUpdate={(e) => {
          const el = e.currentTarget;
          setProgress(el.currentTime * 1000, (el.duration || 0) * 1000);
        }}
      />

      <div className="playerbar__now">
        {track && (
          <>
            <AlbumArt albumId={track.albumId} hasArt size={40} />
            <div className="playerbar__meta">
              <div className="playerbar__title">{track.title}</div>
              <div className="playerbar__artist">
                {mode === "radio" && <span className="badge-live">LIVE</span>}
                {track.artistName}
                {mode === "radio" && station && (
                  <>
                    {" · "}
                    <Link to={`/stations/${station.id}`}>{station.name}</Link>
                  </>
                )}
              </div>
            </div>
          </>
        )}
        {empty && <span className="playerbar__idle">Nothing playing</span>}
      </div>

      <div className="playerbar__controls">
        {mode === "library" ? (
          <>
            <button
              className={`btn btn--icon${shuffle ? " is-on" : ""}`}
              title="Shuffle"
              onClick={toggleShuffle}
            >
              🔀
            </button>
            <button className="btn btn--icon" title="Previous" onClick={prev}>
              ⏮
            </button>
            <button
              className="btn btn--icon btn--play"
              title={isPlaying ? "Pause" : "Play"}
              onClick={() => setPlaying(!isPlaying)}
              disabled={empty}
            >
              {isPlaying ? "⏸" : "▶"}
            </button>
            <button className="btn btn--icon" title="Next" onClick={() => playNext()}>
              ⏭
            </button>
            <button
              className={`btn btn--icon${repeat !== "off" ? " is-on" : ""}`}
              title={`Repeat: ${repeat}`}
              onClick={cycleRepeat}
            >
              {repeat === "one" ? "🔂" : "🔁"}
            </button>
          </>
        ) : (
          <>
            <button
              className="btn btn--icon btn--play"
              title={isPlaying ? "Pause" : "Play"}
              onClick={() => setPlaying(!isPlaying)}
            >
              {isPlaying ? "⏸" : "▶"}
            </button>
            <button className="btn btn--ghost" onClick={leaveStation}>
              Leave station
            </button>
          </>
        )}
      </div>

      <div className="playerbar__seek">
        <span className="playerbar__time">{formatDuration(positionMs)}</span>
        <input
          type="range"
          min={0}
          max={durationMs || 0}
          value={Math.min(positionMs, durationMs || 0)}
          onChange={onSeek}
          disabled={mode === "radio" || empty}
        />
        <span className="playerbar__time">{formatDuration(durationMs)}</span>
      </div>
    </div>
  );
}
