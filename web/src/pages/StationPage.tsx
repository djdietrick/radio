import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { api } from "../api";
import { AlbumArt } from "../components/AlbumArt";
import { formatDuration } from "../format";
import { useStation } from "../hooks/queries";
import { useQuery } from "@tanstack/react-query";
import { usePlayer } from "../store/player";

export function StationPage() {
  const { stationId = "" } = useParams();
  const { data: station } = useStation(stationId);
  const [copied, setCopied] = useState(false);

  const activeStationId = usePlayer((s) => s.station?.id ?? null);
  const liveNp = usePlayer((s) => s.nowPlaying);
  const tuneStation = usePlayer((s) => s.tuneStation);

  const isTuned = activeStationId === stationId;

  // A one-off snapshot for preview when not tuned in. When tuned, the live store
  // value (driven by the WebSocket) is the source of truth.
  const { data: snapshot } = useQuery({
    queryKey: ["station", stationId, "now"],
    queryFn: () => api.nowPlaying(stationId),
    enabled: !isTuned,
    refetchInterval: 15000,
  });

  const np = isTuned ? liveNp : snapshot;

  const tuneIn = async () => {
    if (!station) return;
    const current = await api.nowPlaying(stationId);
    tuneStation(station, current);
  };

  const share = async () => {
    try {
      await navigator.clipboard.writeText(window.location.href);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      /* clipboard may be blocked; the URL is in the address bar regardless */
    }
  };

  return (
    <div>
      <div className="detail-head detail-head--simple">
        <div>
          <h1>{station?.name ?? "Station"}</h1>
          <p className="muted">
            {station?.albumShuffle ? "Album shuffle · " : "Shuffle · "}
            {station?.loop ? "Looping" : "Plays once"}
          </p>
        </div>
        <Link className="btn btn--ghost" to="/stations">
          Back
        </Link>
      </div>

      <div className="detail-head__actions">
        <button className="btn btn--primary" onClick={tuneIn} disabled={isTuned}>
          {isTuned ? "Tuned in" : "▶ Tune in"}
        </button>
        <button className="btn btn--ghost" onClick={share}>
          {copied ? "Link copied" : "Share link"}
        </button>
      </div>

      <div className="card now-card">
        <h2>Now playing</h2>
        {!np && <p className="muted">Loading…</p>}
        {np?.ended && <p className="muted">This station's program has ended.</p>}
        {np && !np.ended && (
          <div className="now-card__body">
            <AlbumArt albumId={np.track.albumId} hasArt size={72} />
            <div>
              <div className="now-card__title">{np.track.title}</div>
              <div className="muted">{np.track.artistName}</div>
              <div className="muted">
                Track {np.indexInQueue + 1} of {np.queueLength} ·{" "}
                {formatDuration(np.offsetMs)} / {formatDuration(np.track.durationMs)}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
