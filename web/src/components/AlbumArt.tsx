import { useEffect, useState } from "react";
import { api } from "../api";

// AlbumArt renders an album's cover, or a neutral placeholder square when the
// album has no cached art (or the image fails to load — the player bar and
// now-playing card reference art optimistically without knowing hasArt).
export function AlbumArt({
  albumId,
  hasArt = true,
  size = 48,
}: {
  albumId: string;
  hasArt?: boolean;
  size?: number;
}) {
  const [failed, setFailed] = useState(false);

  // Reset the error state if the album changes (e.g. in the persistent player).
  useEffect(() => setFailed(false), [albumId]);

  if (!hasArt || failed || !albumId) {
    return (
      <span
        className="album-art album-art--placeholder"
        style={{ width: size, height: size }}
        aria-hidden
      />
    );
  }
  return (
    <img
      className="album-art"
      src={api.albumArtUrl(albumId)}
      alt=""
      width={size}
      height={size}
      loading="lazy"
      onError={() => setFailed(true)}
    />
  );
}
