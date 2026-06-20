import { useState } from "react";
import {
  useAddPlaylistGroup,
  useAddPlaylistItem,
  useCreatePlaylist,
  usePlaylists,
} from "../hooks/queries";

// Target describes what to add: a single track, a whole album (one item), or a
// hand-picked group of tracks that plays in order and shuffles as a unit.
export type AddTarget =
  | { kind: "track"; trackId: string }
  | { kind: "album"; albumId: string }
  | { kind: "group"; trackIds: string[] };

// AddToPlaylist is a small dropdown that adds the target to one of the user's
// playlists (creating one inline if needed). This is where the typed-item design
// surfaces: albums and groups are added as indivisible units, not flattened.
export function AddToPlaylist({
  target,
  label = "+ Playlist",
  disabled = false,
}: {
  target: AddTarget;
  label?: string;
  disabled?: boolean;
}) {
  const [open, setOpen] = useState(false);
  const [newName, setNewName] = useState("");
  const [done, setDone] = useState<string | null>(null);
  const { data: playlists } = usePlaylists();
  const addItem = useAddPlaylistItem();
  const addGroup = useAddPlaylistGroup();
  const createPlaylist = useCreatePlaylist();

  const add = async (playlistID: string) => {
    if (target.kind === "group") {
      await addGroup.mutateAsync({ playlistID, trackIds: target.trackIds });
    } else {
      await addItem.mutateAsync({
        playlistID,
        kind: target.kind,
        trackId: target.kind === "track" ? target.trackId : undefined,
        albumId: target.kind === "album" ? target.albumId : undefined,
      });
    }
    setDone(playlistID);
    setTimeout(() => {
      setOpen(false);
      setDone(null);
    }, 700);
  };

  const createAndAdd = async () => {
    const name = newName.trim();
    if (!name) return;
    const pl = await createPlaylist.mutateAsync(name);
    setNewName("");
    await add(pl.id);
  };

  return (
    <span className="add-to-playlist">
      <button
        className="btn btn--ghost"
        disabled={disabled}
        onClick={() => setOpen((o) => !o)}
      >
        {label}
      </button>
      {open && (
        <div className="menu" role="menu">
          {(playlists ?? []).length === 0 && (
            <div className="menu__empty">No playlists yet</div>
          )}
          {(playlists ?? []).map((pl) => (
            <button
              key={pl.id}
              className="menu__item"
              onClick={() => add(pl.id)}
            >
              {done === pl.id ? "✓ Added" : pl.name}
            </button>
          ))}
          <div className="menu__create">
            <input
              placeholder="New playlist…"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && createAndAdd()}
            />
            <button className="btn" onClick={createAndAdd}>
              Add
            </button>
          </div>
        </div>
      )}
    </span>
  );
}
