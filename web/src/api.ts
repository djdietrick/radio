// Typed API client for the Go backend. URLs are same-origin (proxied in dev by
// Vite, served behind nginx in prod).

export interface Album {
  id: string;
  title: string;
  albumArtist: string;
  year: number;
  trackCount: number;
  durationMs: number;
  hasArt: boolean;
  addedAt: string;
}

export interface Track {
  id: string;
  title: string;
  artistId: string;
  artistName: string;
  albumId: string;
  albumTitle: string;
  trackNumber: number;
  discNumber: number;
  durationMs: number;
  mimeType: string;
}

export interface Artist {
  id: string;
  name: string;
  albumCount: number;
  trackCount: number;
}

export type PlaylistItemKind = "track" | "album";

export interface PlaylistItem {
  id: string;
  kind: PlaylistItemKind;
  position: number;
  trackId?: string;
  albumId?: string;
  // Set on the track items of a hand-picked group; same value ties them into one
  // ordered shuffle unit.
  groupId?: string;
}

export interface Playlist {
  id: string;
  userId: string;
  name: string;
  items?: PlaylistItem[];
  createdAt: string;
  updatedAt: string;
}

export interface Station {
  id: string;
  userId: string;
  name: string;
  playlistId: string;
  startedAt: string;
  albumShuffle: boolean;
  shuffleSeed: number;
  loop: boolean;
  createdAt: string;
}

export interface NowPlaying {
  stationId: string;
  track: Track;
  offsetMs: number;
  indexInQueue: number;
  queueLength: number;
  serverTime: string;
  msUntilNextTrack: number;
  ended: boolean;
}

export interface SearchResults {
  tracks: Track[];
  albums: Album[];
  artists: Artist[];
}

export interface User {
  id: string;
  username: string;
  isAdmin: boolean;
  createdAt: string;
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

// ApiError carries the HTTP status so callers (and the query layer) can react to
// 401s by dropping back to the login screen.
export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: authHeaders(init?.headers as Record<string, string>),
  });
  if (!res.ok) {
    let msg = `${res.status} ${res.statusText}`;
    try {
      const body = (await res.json()) as { error?: string };
      if (body.error) msg = body.error;
    } catch {
      /* non-JSON error body */
    }
    throw new ApiError(res.status, msg);
  }
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

function get<T>(path: string): Promise<T> {
  return request<T>(path);
}

function post<T>(path: string, body?: unknown): Promise<T> {
  return request<T>(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
}

function del<T>(path: string): Promise<T> {
  return request<T>(path, { method: "DELETE" });
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
    if (!res.ok) throw new ApiError(res.status, "invalid username or password");
    const s = (await res.json()) as Session;
    setToken(s.token);
    return s;
  },

  me: () => get<User>("/api/auth/me"),

  // Library
  listAlbums: () => get<Album[]>("/api/albums"),
  getAlbum: (id: string) => get<Album>(`/api/albums/${id}`),
  albumTracks: (id: string) => get<Track[]>(`/api/albums/${id}/tracks`),
  listArtists: () => get<Artist[]>("/api/artists"),
  getArtist: (id: string) => get<Artist>(`/api/artists/${id}`),
  artistAlbums: (id: string) => get<Album[]>(`/api/artists/${id}/albums`),
  getTrack: (id: string) => get<Track>(`/api/tracks/${id}`),
  search: (q: string) =>
    get<SearchResults>(`/api/search?q=${encodeURIComponent(q)}`),

  // Playlists
  listPlaylists: () => get<Playlist[]>("/api/playlists"),
  getPlaylist: (id: string) => get<Playlist>(`/api/playlists/${id}`),
  createPlaylist: (name: string) =>
    post<Playlist>("/api/playlists", { name }),
  deletePlaylist: (id: string) => del<void>(`/api/playlists/${id}`),
  addPlaylistItem: (
    playlistID: string,
    item: { kind: PlaylistItemKind; trackId?: string; albumId?: string },
  ) => post<PlaylistItem>(`/api/playlists/${playlistID}/items`, item),
  // Add a hand-picked set of tracks as one ordered group (trackIds in play order).
  addPlaylistGroup: (playlistID: string, trackIds: string[]) =>
    post<PlaylistItem[]>(`/api/playlists/${playlistID}/items`, {
      kind: "group",
      trackIds,
    }),
  removePlaylistItem: (playlistID: string, itemID: string) =>
    del<void>(`/api/playlists/${playlistID}/items/${itemID}`),
  removePlaylistGroup: (playlistID: string, groupID: string) =>
    del<void>(`/api/playlists/${playlistID}/groups/${groupID}`),

  // Stations
  listStations: () => get<Station[]>("/api/stations"),
  getStation: (id: string) => get<Station>(`/api/stations/${id}`),
  createStation: (req: {
    name: string;
    playlistId: string;
    albumShuffle: boolean;
    loop: boolean;
  }) => post<Station>("/api/stations", req),
  nowPlaying: (id: string) => get<NowPlaying>(`/api/stations/${id}/now`),

  // Users (admin)
  listUsers: () => get<User[]>("/api/users"),
  createUser: (req: { username: string; password: string; isAdmin: boolean }) =>
    post<User>("/api/users", req),

  // Maintenance
  triggerScan: () => post("/api/scan"),
  triggerBackfill: () => post("/api/backfill"),

  // Element-attribute URLs carry the token as a query param.
  streamUrl: (trackID: string) => appendToken(`/api/stream/${trackID}`),
  albumArtUrl: (albumID: string) => appendToken(`/api/albums/${albumID}/art`),
  stationWsUrl: (stationID: string) => {
    const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
    return appendToken(
      `${proto}//${window.location.host}/api/stations/${stationID}/ws`,
    );
  },
};
