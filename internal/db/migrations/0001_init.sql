-- Schema is multi-user ready: owned rows carry user_id, defaulted to the
-- single-user id while auth is not yet implemented.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username    TEXT UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed the default single-user account referenced by RADIO_DEFAULT_USER_ID.
INSERT INTO users (id, username)
VALUES ('00000000-0000-0000-0000-000000000001', 'default')
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS artists (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        TEXT NOT NULL,
    name_key    TEXT NOT NULL UNIQUE  -- lower(trim(name)) for dedup
);

CREATE TABLE IF NOT EXISTS albums (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title         TEXT NOT NULL,
    album_artist  TEXT NOT NULL DEFAULT '',
    year          INT  NOT NULL DEFAULT 0,
    has_art       BOOLEAN NOT NULL DEFAULT FALSE,
    -- dedup key: lower(trim(album_artist)) || '\x1f' || lower(trim(title))
    album_key     TEXT NOT NULL UNIQUE,
    added_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS tracks (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    path          TEXT NOT NULL UNIQUE,
    title         TEXT NOT NULL,
    artist_id     UUID REFERENCES artists(id) ON DELETE SET NULL,
    album_id      UUID REFERENCES albums(id)  ON DELETE SET NULL,
    track_number  INT  NOT NULL DEFAULT 0,
    disc_number   INT  NOT NULL DEFAULT 0,
    duration_ms   BIGINT NOT NULL DEFAULT 0,
    genre         TEXT NOT NULL DEFAULT '',
    year          INT  NOT NULL DEFAULT 0,
    codec         TEXT NOT NULL DEFAULT '',
    mime_type     TEXT NOT NULL DEFAULT '',
    size_bytes    BIGINT NOT NULL DEFAULT 0,
    modified_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    added_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_tracks_album   ON tracks(album_id, disc_number, track_number);
CREATE INDEX IF NOT EXISTS idx_tracks_artist  ON tracks(artist_id);

CREATE TABLE IF NOT EXISTS playlists (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS playlist_items (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    playlist_id  UUID NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
    kind         TEXT NOT NULL CHECK (kind IN ('track', 'album')),
    position     INT  NOT NULL,
    track_id     UUID REFERENCES tracks(id) ON DELETE CASCADE,
    album_id     UUID REFERENCES albums(id) ON DELETE CASCADE,
    -- exactly one of track_id / album_id is set, matching kind
    CHECK ((kind = 'track' AND track_id IS NOT NULL AND album_id IS NULL)
        OR (kind = 'album' AND album_id IS NOT NULL AND track_id IS NULL))
);

CREATE INDEX IF NOT EXISTS idx_playlist_items ON playlist_items(playlist_id, position);

CREATE TABLE IF NOT EXISTS stations (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    playlist_id   UUID NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
    started_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    album_shuffle BOOLEAN NOT NULL DEFAULT FALSE,
    shuffle_seed  BIGINT NOT NULL DEFAULT 0,
    loop          BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
