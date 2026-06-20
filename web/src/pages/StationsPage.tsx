import { useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { usePlaylists, useStations, useCreateStation } from "../hooks/queries";

export function StationsPage() {
  const { data: stations, isLoading } = useStations();
  const { data: playlists } = usePlaylists();
  const createStation = useCreateStation();
  const [params] = useSearchParams();

  const [name, setName] = useState("");
  const [playlistId, setPlaylistId] = useState(params.get("playlist") ?? "");
  const [albumShuffle, setAlbumShuffle] = useState(true);
  const [loop, setLoop] = useState(true);

  const create = (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !playlistId) return;
    createStation.mutate(
      { name: name.trim(), playlistId, albumShuffle, loop },
      { onSuccess: () => setName("") },
    );
  };

  return (
    <div>
      <h1>Stations</h1>

      <form className="card station-form" onSubmit={create}>
        <h2>New station</h2>
        <div className="field">
          <label>Name</label>
          <input
            placeholder="Station name"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </div>
        <div className="field">
          <label>Playlist</label>
          <select value={playlistId} onChange={(e) => setPlaylistId(e.target.value)}>
            <option value="">Select a playlist…</option>
            {playlists?.map((pl) => (
              <option key={pl.id} value={pl.id}>
                {pl.name}
              </option>
            ))}
          </select>
        </div>
        <label className="check">
          <input
            type="checkbox"
            checked={albumShuffle}
            onChange={(e) => setAlbumShuffle(e.target.checked)}
          />
          Album shuffle (keep albums together, shuffle their order)
        </label>
        <label className="check">
          <input type="checkbox" checked={loop} onChange={(e) => setLoop(e.target.checked)} />
          Loop when the program ends
        </label>
        <button
          className="btn btn--primary"
          type="submit"
          disabled={!name.trim() || !playlistId || createStation.isPending}
        >
          Create station
        </button>
        {createStation.isError && (
          <p className="error">{String(createStation.error)}</p>
        )}
        {(playlists?.length ?? 0) === 0 && (
          <p className="muted">Create a playlist first to start a station.</p>
        )}
      </form>

      <h2>All stations</h2>
      {isLoading && <p className="muted">Loading…</p>}
      {stations && stations.length === 0 && <p className="muted">No stations yet.</p>}
      <ul className="row-list">
        {stations?.map((st) => (
          <li key={st.id} className="row-list__item">
            <Link to={`/stations/${st.id}`} className="row-list__main">
              {st.name}
              {st.albumShuffle && <span className="badge">album shuffle</span>}
              {st.loop && <span className="badge">loop</span>}
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}
