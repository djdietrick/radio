import { PlaylistItem } from "../api";
import { formatCount, formatDuration } from "../format";
import { useAlbum, useTrack } from "../hooks/queries";
import { AlbumArt } from "./AlbumArt";

// PlaylistItemRow renders one typed item. The kind badge ("Album" vs "Track") is
// deliberately prominent — the whole-album-vs-single-track distinction is the
// product's defining playlist feature.
export function PlaylistItemRow({
  item,
  onRemove,
}: {
  item: PlaylistItem;
  onRemove: () => void;
}) {
  return (
    <li className="row-list__item playlist-item">
      {item.kind === "album" ? (
        <AlbumItem albumId={item.albumId!} />
      ) : (
        <TrackItem trackId={item.trackId!} />
      )}
      <button className="btn btn--ghost" title="Remove" onClick={onRemove}>
        ✕
      </button>
    </li>
  );
}

function AlbumItem({ albumId }: { albumId: string }) {
  const { data: album } = useAlbum(albumId);
  return (
    <span className="playlist-item__main">
      <span className="badge">Album</span>
      <AlbumArt albumId={albumId} hasArt={album?.hasArt ?? false} size={36} />
      <span className="playlist-item__text">
        <span className="playlist-item__title">{album?.title ?? "…"}</span>
        <span className="muted">
          {album ? `${album.albumArtist} · ${formatCount(album.trackCount, "track")}` : ""}
        </span>
      </span>
    </span>
  );
}

function TrackItem({ trackId }: { trackId: string }) {
  const { data: track } = useTrack(trackId);
  return (
    <span className="playlist-item__main">
      <span className="badge badge--track">Track</span>
      <span className="playlist-item__text">
        <span className="playlist-item__title">{track?.title ?? "…"}</span>
        <span className="muted">
          {track ? `${track.artistName} · ${formatDuration(track.durationMs)}` : ""}
        </span>
      </span>
    </span>
  );
}
