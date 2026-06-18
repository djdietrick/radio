// Minimal typed API client for the Go backend. URLs are same-origin (proxied in
// dev by Vite, served behind nginx in prod).

export interface Album {
  id: string;
  title: string;
  albumArtist: string;
  year: number;
  trackCount: number;
  durationMs: number;
  hasArt: boolean;
}

export interface Track {
  id: string;
  title: string;
  artistName: string;
  albumTitle: string;
  trackNumber: number;
  durationMs: number;
  mimeType: string;
}

export interface NowPlaying {
  stationId: string;
  track: Track;
  offsetMs: number;
  indexInQueue: number;
  queueLength: number;
  serverTime: string;
  ended: boolean;
}

export interface Session {
  token: string;
  userId: string;
  isAdmin: boolean;
}

const TOKEN_KEY = "radio.token";

// Token is persisted in localStorage so a refresh keeps the session. (For a
// hardened setup you'd prefer an HttpOnly cookie; bearer-in-JS is the tradeoff
// that comes with the JWT choice.)
function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}
function setToken(token: string | null) {
  if (token) localStorage.setItem(TOKEN_KEY, token);
  else localStorage.removeItem(TOKEN_KEY);
}

function authHeaders(extra: Record<string, string> = {}): Record<string, string> {
  const token = getToken();
  return token ? { ...extra, Authorization: `Bearer ${token}` } : extra;
}

async function get<T>(path: string): Promise<T> {
  const res = await fetch(path, { headers: authHeaders() });
  if (res.status === 401) throw new Error("unauthorized");
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return res.json() as Promise<T>;
}

async function post<T>(path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method: "POST",
    headers: authHeaders({ "Content-Type": "application/json" }),
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (res.status === 401) throw new Error("unauthorized");
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return (res.status === 204 ? (undefined as T) : ((await res.json()) as T));
}

// appendToken adds the bearer token as a query param, for URLs used in element
// attributes (<audio src>, <img src>, WebSocket) that cannot send headers.
function appendToken(url: string): string {
  const token = getToken();
  if (!token) return url;
  const sep = url.includes("?") ? "&" : "?";
  return `${url}${sep}token=${encodeURIComponent(token)}`;
}

export const api = {
  isAuthenticated: () => getToken() !== null,
  logout: () => setToken(null),

  async login(username: string, password: string): Promise<Session> {
    const res = await fetch("/api/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password }),
    });
    if (!res.ok) throw new Error("invalid username or password");
    const s = (await res.json()) as Session;
    setToken(s.token);
    return s;
  },

  listAlbums: () => get<Album[]>("/api/albums"),
  albumTracks: (albumID: string) => get<Track[]>(`/api/albums/${albumID}/tracks`),
  nowPlaying: (stationID: string) => get<NowPlaying>(`/api/stations/${stationID}/now`),

  // These return URLs for element attributes, so the token rides as a query
  // param. Streaming/art require auth; the radio WS is public but the token is
  // harmless when present.
  streamUrl: (trackID: string) => appendToken(`/api/stream/${trackID}`),
  albumArtUrl: (albumID: string) => appendToken(`/api/albums/${albumID}/art`),
  stationWsUrl: (stationID: string) => {
    const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
    return appendToken(`${proto}//${window.location.host}/api/stations/${stationID}/ws`);
  },

  triggerScan: () => post("/api/scan"),
  triggerBackfill: () => post("/api/backfill"),
};
