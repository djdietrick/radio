import { api, Playlist, Track } from "../api";

// resolvePlaylistTracks flattens a playlist's typed items into a flat track list
// for local (non-radio) playback: track items resolve to themselves, album items
// expand to their tracks in disc/track order. Dangling references are skipped,
// mirroring the backend resolver's tolerance.
export async function resolvePlaylistTracks(pl: Playlist): Promise<Track[]> {
  const out: Track[] = [];
  for (const item of pl.items ?? []) {
    try {
      if (item.kind === "track" && item.trackId) {
        out.push(await api.getTrack(item.trackId));
      } else if (item.kind === "album" && item.albumId) {
        out.push(...(await api.albumTracks(item.albumId)));
      }
    } catch {
      /* skip dangling reference */
    }
  }
  return out;
}
