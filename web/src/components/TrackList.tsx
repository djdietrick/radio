import { Track } from "../api";
import { formatDuration } from "../format";
import { usePlayer } from "../store/player";
import { AddToPlaylist } from "./AddToPlaylist";

// TrackList renders an ordered list of tracks. Clicking a row plays the whole
// list starting from that track (so the rest becomes the queue). When
// `selectable` is set, each row gets a checkbox so the parent can build a subset
// (used on the album page to add a hand-picked group).
export function TrackList({
  tracks,
  showArtist = true,
  selectable = false,
  selected,
  onToggle,
}: {
  tracks: Track[];
  showArtist?: boolean;
  selectable?: boolean;
  selected?: Set<string>;
  onToggle?: (trackId: string) => void;
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
            className={`tracklist__row${active ? " is-active" : ""}${
              selectable ? " tracklist__row--selectable" : ""
            }`}
            onDoubleClick={() => playQueue(tracks, i)}
          >
            {selectable && (
              <input
                type="checkbox"
                checked={selected?.has(t.id) ?? false}
                onChange={() => onToggle?.(t.id)}
                aria-label={`Select ${t.title}`}
              />
            )}
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
              <AddToPlaylist target={{ kind: "track", trackId: t.id }} />
            </span>
          </li>
        );
      })}
    </ol>
  );
}
