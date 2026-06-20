import { Link } from "react-router-dom";
import { formatCount } from "../format";
import { useArtists } from "../hooks/queries";

export function ArtistsPage() {
  const { data: artists, isLoading, error } = useArtists();

  return (
    <div>
      <h1>Artists</h1>
      {isLoading && <p className="muted">Loading…</p>}
      {error && <p className="error">{String(error)}</p>}
      {artists && artists.length === 0 && <p className="muted">No artists.</p>}
      <ul className="artist-list">
        {artists?.map((a) => (
          <li key={a.id}>
            <Link to={`/artists/${a.id}`} className="artist-list__item">
              <span className="artist-list__name">{a.name}</span>
              <span className="muted">
                {formatCount(a.albumCount, "album")} ·{" "}
                {formatCount(a.trackCount, "track")}
              </span>
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}
