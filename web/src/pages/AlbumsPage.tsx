import { AlbumGrid } from "../components/AlbumGrid";
import { useAlbums } from "../hooks/queries";

export function AlbumsPage() {
  const { data: albums, isLoading, error } = useAlbums();

  return (
    <div>
      <h1>Albums</h1>
      {isLoading && <p className="muted">Loading…</p>}
      {error && <p className="error">{String(error)}</p>}
      {albums && <AlbumGrid albums={albums} />}
    </div>
  );
}
