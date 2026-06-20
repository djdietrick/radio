import { useState } from "react";
import { useAddPlaylistItem, useCreatePlaylist, usePlaylists } from "../hooks/queries";

// AddToPlaylist is a small dropdown that adds a track or a whole album to one of
// the user's playlists (creating one inline if needed). This is where the
// typed-item design surfaces: an album is added as a single "album" item, not
// flattened into tracks.
export function AddToPlaylist({
  kind,
  refId,
  label = "+ Playlist",
}: {
  kind: "track" | "album";
  refId: string;
  label?: string;
}) {
  const [open, setOpen] = useState(false);
  const [newName, setNewName] = useState("");
  const [done, setDone] = useState<string | null>(null);
  const { data: playlists } = usePlaylists();
  const addItem = useAddPlaylistItem();
  const createPlaylist = useCreatePlaylist();

  const add = async (playlistID: string) => {
    await addItem.mutateAsync({
      playlistID,
      kind,
      trackId: kind === "track" ? refId : undefined,
      albumId: kind === "album" ? refId : undefined,
    });
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
      <button className="btn btn--ghost" onClick={() => setOpen((o) => !o)}>
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
