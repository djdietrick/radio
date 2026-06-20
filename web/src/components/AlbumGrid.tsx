import { Link } from "react-router-dom";
import { Album } from "../api";
import { formatCount } from "../format";
import { AlbumArt } from "./AlbumArt";

// AlbumGrid is the reusable cover grid used on the library, artist, and search
// screens.
export function AlbumGrid({ albums }: { albums: Album[] }) {
  if (albums.length === 0) return <p className="muted">No albums.</p>;
  return (
    <div className="album-grid">
      {albums.map((a) => (
        <Link key={a.id} to={`/albums/${a.id}`} className="album-card">
          <AlbumArt albumId={a.id} hasArt={a.hasArt} size={160} />
          <div className="album-card__title">{a.title}</div>
          <div className="album-card__sub">{a.albumArtist}</div>
          <div className="album-card__meta muted">
            {a.year ? `${a.year} · ` : ""}
            {formatCount(a.trackCount, "track")}
          </div>
        </Link>
      ))}
    </div>
  );
}
