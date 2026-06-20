import { PlaylistItem } from "../api";
import { formatCount, formatDuration } from "../format";
import { useAlbum, useTrack } from "../hooks/queries";
import { AlbumArt } from "./AlbumArt";

// PlaylistGroupRow renders a hand-picked group (several track items sharing a
// group_id) as one block: it shuffles together and plays in order, so it reads
// as a unit with a single remove control.
export function PlaylistGroupRow({
  items,
  onRemove,
}: {
  items: PlaylistItem[];
  onRemove: () => void;
}) {
  return (
    <li className="row-list__item playlist-group">
      <div className="playlist-group__main">
        <span className="badge">Group · in order</span>
        <ol className="playlist-group__tracks">
          {items.map((it) => (
            <GroupTrack key={it.id} trackId={it.trackId!} />
          ))}
        </ol>
      </div>
      <button className="btn btn--ghost" title="Remove group" onClick={onRemove}>
        ✕
      </button>
    </li>
  );
}

function GroupTrack({ trackId }: { trackId: string }) {
  const { data: track } = useTrack(trackId);
  return (
    <li className="playlist-group__track">
      <span className="playlist-item__title">{track?.title ?? "…"}</span>
      <span className="muted">{track ? track.artistName : ""}</span>
    </li>
  );
}

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
