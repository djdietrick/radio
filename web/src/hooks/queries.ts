import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { api, PlaylistItemKind } from "../api";

// Centralized query/mutation hooks. Query keys are kept here so invalidation is
// consistent across screens.

export const keys = {
  albums: ["albums"] as const,
  album: (id: string) => ["album", id] as const,
  albumTracks: (id: string) => ["album", id, "tracks"] as const,
  artists: ["artists"] as const,
  artist: (id: string) => ["artist", id] as const,
  artistAlbums: (id: string) => ["artist", id, "albums"] as const,
  search: (q: string) => ["search", q] as const,
  playlists: ["playlists"] as const,
  playlist: (id: string) => ["playlist", id] as const,
  stations: ["stations"] as const,
  station: (id: string) => ["station", id] as const,
  users: ["users"] as const,
  me: ["me"] as const,
};

// --- library ---

export const useAlbums = () =>
  useQuery({ queryKey: keys.albums, queryFn: api.listAlbums });

export const useAlbum = (id: string) =>
  useQuery({ queryKey: keys.album(id), queryFn: () => api.getAlbum(id) });

export const useAlbumTracks = (id: string) =>
  useQuery({ queryKey: keys.albumTracks(id), queryFn: () => api.albumTracks(id) });

export const useArtists = () =>
  useQuery({ queryKey: keys.artists, queryFn: api.listArtists });

export const useArtist = (id: string) =>
  useQuery({ queryKey: keys.artist(id), queryFn: () => api.getArtist(id) });

export const useArtistAlbums = (id: string) =>
  useQuery({ queryKey: keys.artistAlbums(id), queryFn: () => api.artistAlbums(id) });

export const useTrack = (id: string) =>
  useQuery({ queryKey: ["track", id], queryFn: () => api.getTrack(id) });

export const useSearch = (q: string) =>
  useQuery({
    queryKey: keys.search(q),
    queryFn: () => api.search(q),
    enabled: q.trim().length > 0,
  });

// --- playlists ---

export const usePlaylists = () =>
  useQuery({ queryKey: keys.playlists, queryFn: api.listPlaylists });

export const usePlaylist = (id: string) =>
  useQuery({ queryKey: keys.playlist(id), queryFn: () => api.getPlaylist(id) });

export const useCreatePlaylist = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => api.createPlaylist(name),
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.playlists }),
  });
};

export const useDeletePlaylist = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deletePlaylist(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.playlists }),
  });
};

export const useAddPlaylistItem = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (vars: {
      playlistID: string;
      kind: PlaylistItemKind;
      trackId?: string;
      albumId?: string;
    }) =>
      api.addPlaylistItem(vars.playlistID, {
        kind: vars.kind,
        trackId: vars.trackId,
        albumId: vars.albumId,
      }),
    onSuccess: (_data, vars) =>
      qc.invalidateQueries({ queryKey: keys.playlist(vars.playlistID) }),
  });
};

export const useRemovePlaylistItem = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (vars: { playlistID: string; itemID: string }) =>
      api.removePlaylistItem(vars.playlistID, vars.itemID),
    onSuccess: (_data, vars) =>
      qc.invalidateQueries({ queryKey: keys.playlist(vars.playlistID) }),
  });
};

// --- stations ---

export const useStations = () =>
  useQuery({ queryKey: keys.stations, queryFn: api.listStations });

export const useStation = (id: string) =>
  useQuery({ queryKey: keys.station(id), queryFn: () => api.getStation(id) });

export const useCreateStation = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: api.createStation,
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.stations }),
  });
};

// --- users (admin) ---

export const useUsers = () =>
  useQuery({ queryKey: keys.users, queryFn: api.listUsers });

export const useCreateUser = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: api.createUser,
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.users }),
  });
};
