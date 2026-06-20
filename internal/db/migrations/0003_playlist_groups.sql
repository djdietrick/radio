-- Track "groups": a hand-picked subset of tracks that should shuffle as one
-- indivisible unit while playing in order — the same album-aware-shuffle
-- behaviour an "album" item gets, but for a user-chosen selection (typically a
-- handful of songs from one album).
--
-- A group is modelled as several kind='track' rows sharing a group_id, so the
-- existing per-row CHECK constraint still holds and single tracks are unaffected
-- (their group_id is NULL).
ALTER TABLE playlist_items
    ADD COLUMN IF NOT EXISTS group_id UUID;

CREATE INDEX IF NOT EXISTS idx_playlist_items_group ON playlist_items(group_id);
