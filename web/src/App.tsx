import { useEffect, useRef, useState } from "react";
import { api, Album, NowPlaying, Track } from "./api";

// Stub UI demonstrating the core flows. It's intentionally minimal — the
// backend is the focus of this scaffold. Three things are shown:
//   1. Browse albums (catalog read).
//   2. Play a track via the Range-aware /api/stream endpoint.
//   3. Tune into a radio station over WebSocket: the server pushes the current
//      track + offset on connect and again at every track boundary, and we
//      seek the <audio> element so this listener stays in sync with everyone.

export function App() {
  const [authed, setAuthed] = useState(api.isAuthenticated());
  const [albums, setAlbums] = useState<Album[]>([]);
  const [tracks, setTracks] = useState<Track[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [live, setLive] = useState<NowPlaying | null>(null);
  const audioRef = useRef<HTMLAudioElement>(null);
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    if (!authed) return;
    api.listAlbums().then(setAlbums).catch((e) => {
      // An expired/invalid token surfaces as unauthorized; drop back to login.
      if (String(e).includes("unauthorized")) {
        api.logout();
        setAuthed(false);
      } else {
        setError(String(e));
      }
    });
    // Close any open station socket on unmount.
    return () => wsRef.current?.close();
  }, [authed]);

  if (!authed) {
    return <LoginForm onLoggedIn={() => setAuthed(true)} />;
  }

  const logout = () => {
    wsRef.current?.close();
    api.logout();
    setAuthed(false);
    setAlbums([]);
    setTracks([]);
    setLive(null);
  };

  const openAlbum = (a: Album) =>
    api.albumTracks(a.id).then(setTracks).catch((e) => setError(String(e)));

  const play = (t: Track) => {
    const el = audioRef.current;
    if (!el) return;
    el.src = api.streamUrl(t.id);
    el.play();
  };

  // Apply a NowPlaying push: load the current track and seek to the offset so
  // playback matches the "broadcast" position. Only re-load the <audio> source
  // when the track actually changes to avoid restarting mid-track.
  const applyNowPlaying = (np: NowPlaying) => {
    setLive(np);
    const el = audioRef.current;
    if (!el || np.ended) return;
    const nextSrc = api.streamUrl(np.track.id);
    if (!el.src.endsWith(nextSrc)) {
      el.src = nextSrc;
    }
    el.currentTime = np.offsetMs / 1000;
    el.play().catch(() => {/* autoplay may require a user gesture */});
  };

  // Tune in over WebSocket: receive the initial state plus every track change.
  const tuneIn = (stationID: string) => {
    wsRef.current?.close();
    setError(null);
    try {
      const ws = new WebSocket(api.stationWsUrl(stationID));
      wsRef.current = ws;
      ws.onmessage = (ev) => {
        try {
          applyNowPlaying(JSON.parse(ev.data) as NowPlaying);
        } catch (e) {
          setError(String(e));
        }
      };
      ws.onerror = () => setError("station socket error");
    } catch (e) {
      setError(String(e));
    }
  };

  return (
    <div style={{ fontFamily: "system-ui", padding: 24, maxWidth: 800, margin: "0 auto" }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <h1>Radio</h1>
        <button onClick={logout}>Log out</button>
      </div>
      {error && <p style={{ color: "crimson" }}>Error: {error}</p>}

      <div style={{ display: "flex", gap: 8 }}>
        <button onClick={() => api.triggerScan()}>Rescan library</button>
        <button onClick={() => api.triggerBackfill()}>Backfill durations &amp; art</button>
      </div>

      <h2>Albums</h2>
      <ul style={{ listStyle: "none", padding: 0 }}>
        {albums.map((a) => (
          <li key={a.id} style={{ marginBottom: 8 }}>
            <button
              onClick={() => openAlbum(a)}
              style={{ display: "flex", alignItems: "center", gap: 12, width: "100%", textAlign: "left" }}
            >
              {a.hasArt ? (
                <img
                  src={api.albumArtUrl(a.id)}
                  alt=""
                  width={48}
                  height={48}
                  style={{ objectFit: "cover", borderRadius: 4 }}
                  loading="lazy"
                />
              ) : (
                <span
                  style={{
                    width: 48,
                    height: 48,
                    background: "#ddd",
                    borderRadius: 4,
                    display: "inline-block",
                  }}
                />
              )}
              <span>
                {a.albumArtist} — {a.title} ({a.trackCount} tracks)
              </span>
            </button>
          </li>
        ))}
      </ul>

      {tracks.length > 0 && (
        <>
          <h2>Tracks</h2>
          <ol>
            {tracks.map((t) => (
              <li key={t.id}>
                <button onClick={() => play(t)}>{t.title}</button>
              </li>
            ))}
          </ol>
        </>
      )}

      {/* Example: paste a station id to tune in. */}
      <h2>Tune into a station (live)</h2>
      <StationTuner onTune={tuneIn} />

      {live && (
        <p style={{ marginTop: 12 }}>
          {live.ended ? (
            <em>Station program ended.</em>
          ) : (
            <>
              Now playing: <strong>{live.track.title}</strong> — {live.track.artistName} (track{" "}
              {live.indexInQueue + 1}/{live.queueLength})
            </>
          )}
        </p>
      )}

      <audio ref={audioRef} controls style={{ width: "100%", marginTop: 24 }} />
    </div>
  );
}

function StationTuner({ onTune }: { onTune: (id: string) => void }) {
  const [id, setId] = useState("");
  return (
    <div style={{ display: "flex", gap: 8 }}>
      <input
        placeholder="station id"
        value={id}
        onChange={(e) => setId(e.target.value)}
        style={{ flex: 1 }}
      />
      <button onClick={() => id && onTune(id)}>Tune in</button>
    </div>
  );
}

function LoginForm({ onLoggedIn }: { onLoggedIn: () => void }) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    try {
      await api.login(username, password);
      onLoggedIn();
    } catch (err) {
      setError(String(err instanceof Error ? err.message : err));
    }
  };

  return (
    <div style={{ fontFamily: "system-ui", maxWidth: 320, margin: "80px auto", padding: 24 }}>
      <h1>Radio</h1>
      <form onSubmit={submit} style={{ display: "flex", flexDirection: "column", gap: 8 }}>
        <input
          placeholder="username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          autoFocus
        />
        <input
          type="password"
          placeholder="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
        <button type="submit">Log in</button>
        {error && <p style={{ color: "crimson", margin: 0 }}>{error}</p>}
      </form>
    </div>
  );
}
