import { Link, useParams } from "react-router-dom";
import { PlaylistItem } from "../api";
import { PlaylistGroupRow, PlaylistItemRow } from "../components/PlaylistItemRow";
import { formatCount } from "../format";
import {
  usePlaylist,
  useRemovePlaylistGroup,
  useRemovePlaylistItem,
} from "../hooks/queries";
import { resolvePlaylistTracks } from "../lib/resolve";
import { usePlayer } from "../store/player";

// A render row is either a standalone item or a folded group of track items.
type Row =
  | { type: "item"; item: PlaylistItem }
  | { type: "group"; groupId: string; items: PlaylistItem[] };

// foldGroups collapses consecutive track items sharing a group_id into one row,
// matching how the backend resolver treats them as a single unit.
function foldGroups(items: PlaylistItem[]): Row[] {
  const rows: Row[] = [];
  for (const item of items) {
    const last = rows[rows.length - 1];
    if (
      item.groupId &&
      last?.type === "group" &&
      last.groupId === item.groupId
    ) {
      last.items.push(item);
    } else if (item.groupId) {
      rows.push({ type: "group", groupId: item.groupId, items: [item] });
    } else {
      rows.push({ type: "item", item });
    }
  }
  return rows;
}

export function PlaylistPage() {
  const { playlistId = "" } = useParams();
  const { data: playlist, isLoading } = usePlaylist(playlistId);
  const removeItem = useRemovePlaylistItem();
  const removeGroup = useRemovePlaylistGroup();
  const playQueue = usePlayer((s) => s.playQueue);

  const items = playlist?.items ?? [];
  const rows = foldGroups(items);

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
        Add albums, single tracks, or a hand-picked group from an album (Select
        tracks on an album page) with the “+ Playlist” buttons. Albums and groups
        shuffle as one unit and play in order.
      </p>

      {isLoading && <p className="muted">Loading…</p>}
      {items.length === 0 && !isLoading && <p className="muted">This playlist is empty.</p>}
      <ul className="row-list">
        {rows.map((row) =>
          row.type === "group" ? (
            <PlaylistGroupRow
              key={row.groupId}
              items={row.items}
              onRemove={() =>
                removeGroup.mutate({ playlistID: playlistId, groupID: row.groupId })
              }
            />
          ) : (
            <PlaylistItemRow
              key={row.item.id}
              item={row.item}
              onRemove={() =>
                removeItem.mutate({ playlistID: playlistId, itemID: row.item.id })
              }
            />
          ),
        )}
      </ul>
    </div>
  );
}
