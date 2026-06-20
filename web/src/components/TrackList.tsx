import { Track } from "../api";
import { formatDuration } from "../format";
import { usePlayer } from "../store/player";
import { AddToPlaylist } from "./AddToPlaylist";

// TrackList renders an ordered list of tracks. Clicking a row plays the whole
// list starting from that track (so the rest becomes the queue).
export function TrackList({
  tracks,
  showArtist = true,
}: {
  tracks: Track[];
  showArtist?: boolean;
}) {
  const playQueue = usePlayer((s) => s.playQueue);
  const currentTrack = usePlayer((s) => s.currentTrack());

  return (
    <ol className="tracklist">
      {tracks.map((t, i) => {
        const active = currentTrack?.id === t.id;
        return (
          <li
            key={t.id}
            className={`tracklist__row${active ? " is-active" : ""}`}
            onDoubleClick={() => playQueue(tracks, i)}
          >
            <button
              className="tracklist__play"
              title="Play"
              onClick={() => playQueue(tracks, i)}
            >
              {active ? "♪" : t.trackNumber || i + 1}
            </button>
            <span className="tracklist__title">{t.title}</span>
            {showArtist && (
              <span className="tracklist__artist">{t.artistName}</span>
            )}
            <span className="tracklist__dur">{formatDuration(t.durationMs)}</span>
            <span className="tracklist__actions">
              <AddToPlaylist kind="track" refId={t.id} />
            </span>
          </li>
        );
      })}
    </ol>
  );
}
