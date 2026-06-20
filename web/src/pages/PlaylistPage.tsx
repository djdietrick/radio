import { Link, useParams } from "react-router-dom";
import { PlaylistItemRow } from "../components/PlaylistItemRow";
import { formatCount } from "../format";
import { usePlaylist, useRemovePlaylistItem } from "../hooks/queries";
import { resolvePlaylistTracks } from "../lib/resolve";
import { usePlayer } from "../store/player";

export function PlaylistPage() {
  const { playlistId = "" } = useParams();
  const { data: playlist, isLoading } = usePlaylist(playlistId);
  const removeItem = useRemovePlaylistItem();
  const playQueue = usePlayer((s) => s.playQueue);

  const items = playlist?.items ?? [];

  const play = async () => {
    if (!playlist) return;
    const tracks = await resolvePlaylistTracks(playlist);
    if (tracks.length > 0) playQueue(tracks, 0);
  };

  return (
    <div>
      <div className="detail-head detail-head--simple">
        <div>
          <h1>{playlist?.name ?? "Playlist"}</h1>
          <p className="muted">{formatCount(items.length, "item")}</p>
        </div>
        <div className="detail-head__actions">
          <button className="btn btn--primary" disabled={items.length === 0} onClick={play}>
            ▶ Play
          </button>
          <Link className="btn btn--ghost" to={`/stations?playlist=${playlistId}`}>
            Create station
          </Link>
          <Link className="btn btn--ghost" to="/playlists">
            Back
          </Link>
        </div>
      </div>

      <p className="muted">
        Add albums or single tracks from the Albums and Search screens with the
        “+ Playlist” buttons.
      </p>

      {isLoading && <p className="muted">Loading…</p>}
      {items.length === 0 && !isLoading && <p className="muted">This playlist is empty.</p>}
      <ul className="row-list">
        {items.map((item) => (
          <PlaylistItemRow
            key={item.id}
            item={item}
            onRemove={() => removeItem.mutate({ playlistID: playlistId, itemID: item.id })}
          />
        ))}
      </ul>
    </div>
  );
}
