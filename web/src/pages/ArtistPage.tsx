import { Link, useParams } from "react-router-dom";
import { AlbumGrid } from "../components/AlbumGrid";
import { formatCount } from "../format";
import { useArtist, useArtistAlbums } from "../hooks/queries";

export function ArtistPage() {
  const { artistId = "" } = useParams();
  const { data: artist } = useArtist(artistId);
  const { data: albums, isLoading } = useArtistAlbums(artistId);

  return (
    <div>
      <div className="detail-head detail-head--simple">
        <div>
          <h1>{artist?.name ?? "Artist"}</h1>
          {artist && (
            <p className="muted">
              {formatCount(artist.albumCount, "album")} ·{" "}
              {formatCount(artist.trackCount, "track")}
            </p>
          )}
        </div>
        <Link className="btn btn--ghost" to="/artists">
          Back
        </Link>
      </div>

      {isLoading && <p className="muted">Loading…</p>}
      {albums && <AlbumGrid albums={albums} />}
    </div>
  );
}
