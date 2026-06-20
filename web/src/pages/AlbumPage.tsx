import { Link, useParams } from "react-router-dom";
import { AddToPlaylist } from "../components/AddToPlaylist";
import { AlbumArt } from "../components/AlbumArt";
import { TrackList } from "../components/TrackList";
import { formatCount, formatDuration } from "../format";
import { useAlbum, useAlbumTracks } from "../hooks/queries";
import { usePlayer } from "../store/player";

export function AlbumPage() {
  const { albumId = "" } = useParams();
  const { data: album } = useAlbum(albumId);
  const { data: tracks, isLoading } = useAlbumTracks(albumId);
  const playQueue = usePlayer((s) => s.playQueue);

  return (
    <div>
      <div className="detail-head">
        {album && <AlbumArt albumId={album.id} hasArt={album.hasArt} size={180} />}
        <div className="detail-head__meta">
          <h1>{album?.title ?? "Album"}</h1>
          {album && (
            <p className="muted">
              {album.albumArtist}
              {album.year ? ` · ${album.year}` : ""} ·{" "}
              {formatCount(album.trackCount, "track")} ·{" "}
              {formatDuration(album.durationMs)}
            </p>
          )}
          <div className="detail-head__actions">
            <button
              className="btn btn--primary"
              disabled={!tracks || tracks.length === 0}
              onClick={() => tracks && playQueue(tracks, 0)}
            >
              ▶ Play
            </button>
            <AddToPlaylist kind="album" refId={albumId} label="+ Add album to playlist" />
            <Link className="btn btn--ghost" to="/albums">
              Back
            </Link>
          </div>
        </div>
      </div>

      {isLoading && <p className="muted">Loading…</p>}
      {tracks && <TrackList tracks={tracks} />}
    </div>
  );
}
