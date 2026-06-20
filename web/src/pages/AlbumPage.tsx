import { useState } from "react";
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

  // Selection mode lets the user pick a subset to add as an ordered group.
  const [selecting, setSelecting] = useState(false);
  const [selected, setSelected] = useState<Set<string>>(new Set());

  const toggle = (id: string) =>
    setSelected((prev) => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });

  const exitSelect = () => {
    setSelecting(false);
    setSelected(new Set());
  };

  // Selected track ids in album (disc/track) order, so the group plays in order.
  const selectedInOrder =
    tracks?.filter((t) => selected.has(t.id)).map((t) => t.id) ?? [];

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
            <AddToPlaylist
              target={{ kind: "album", albumId }}
              label="+ Add album to playlist"
            />
            {selecting ? (
              <button className="btn btn--ghost" onClick={exitSelect}>
                Cancel selection
              </button>
            ) : (
              <button className="btn btn--ghost" onClick={() => setSelecting(true)}>
                Select tracks
              </button>
            )}
            <Link className="btn btn--ghost" to="/albums">
              Back
            </Link>
          </div>
        </div>
      </div>

      {selecting && (
        <div className="selection-bar">
          <span>{formatCount(selected.size, "track")} selected</span>
          <AddToPlaylist
            target={{ kind: "group", trackIds: selectedInOrder }}
            label="+ Add selection as group (plays in order)"
            disabled={selected.size === 0}
          />
          <span className="muted">
            Added as one unit — shuffles together, plays in order.
          </span>
        </div>
      )}

      {isLoading && <p className="muted">Loading…</p>}
      {tracks && (
        <TrackList
          tracks={tracks}
          selectable={selecting}
          selected={selected}
          onToggle={toggle}
        />
      )}
    </div>
  );
}
