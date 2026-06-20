import { Link, useSearchParams } from "react-router-dom";
import { AlbumGrid } from "../components/AlbumGrid";
import { TrackList } from "../components/TrackList";
import { formatCount } from "../format";
import { useSearch } from "../hooks/queries";

export function SearchPage() {
  const [params] = useSearchParams();
  const q = params.get("q") ?? "";
  const { data, isLoading } = useSearch(q);

  return (
    <div>
      <h1>Search</h1>
      {q ? <p className="muted">Results for “{q}”</p> : <p className="muted">Type a query above.</p>}
      {isLoading && <p className="muted">Searching…</p>}

      {data && (
        <>
          {data.artists.length > 0 && (
            <section>
              <h2>Artists</h2>
              <ul className="artist-list">
                {data.artists.map((a) => (
                  <li key={a.id}>
                    <Link to={`/artists/${a.id}`} className="artist-list__item">
                      <span className="artist-list__name">{a.name}</span>
                      <span className="muted">{formatCount(a.albumCount, "album")}</span>
                    </Link>
                  </li>
                ))}
              </ul>
            </section>
          )}

          {data.albums.length > 0 && (
            <section>
              <h2>Albums</h2>
              <AlbumGrid albums={data.albums} />
            </section>
          )}

          {data.tracks.length > 0 && (
            <section>
              <h2>Tracks</h2>
              <TrackList tracks={data.tracks} />
            </section>
          )}

          {q &&
            data.artists.length === 0 &&
            data.albums.length === 0 &&
            data.tracks.length === 0 && <p className="muted">No matches.</p>}
        </>
      )}
    </div>
  );
}
