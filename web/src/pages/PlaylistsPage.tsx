import { useState } from "react";
import { Link } from "react-router-dom";
import { formatCount } from "../format";
import {
  useCreatePlaylist,
  useDeletePlaylist,
  usePlaylists,
} from "../hooks/queries";

export function PlaylistsPage() {
  const { data: playlists, isLoading } = usePlaylists();
  const createPlaylist = useCreatePlaylist();
  const deletePlaylist = useDeletePlaylist();
  const [name, setName] = useState("");

  const create = (e: React.FormEvent) => {
    e.preventDefault();
    const n = name.trim();
    if (!n) return;
    createPlaylist.mutate(n);
    setName("");
  };

  return (
    <div>
      <h1>Playlists</h1>

      <form className="inline-form" onSubmit={create}>
        <input
          placeholder="New playlist name"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <button className="btn btn--primary" type="submit">
          Create
        </button>
      </form>

      {isLoading && <p className="muted">Loading…</p>}
      {playlists && playlists.length === 0 && (
        <p className="muted">No playlists yet.</p>
      )}
      <ul className="row-list">
        {playlists?.map((pl) => (
          <li key={pl.id} className="row-list__item">
            <Link to={`/playlists/${pl.id}`} className="row-list__main">
              {pl.name}
            </Link>
            <button
              className="btn btn--ghost"
              onClick={() => {
                if (confirm(`Delete playlist “${pl.name}”?`)) {
                  deletePlaylist.mutate(pl.id);
                }
              }}
            >
              Delete
            </button>
          </li>
        ))}
      </ul>
      <p className="muted">{formatCount(playlists?.length ?? 0, "playlist")}</p>
    </div>
  );
}
