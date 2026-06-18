# Radio

A self-hosted music library, player, and virtual-radio server. Scans your music
directories, catalogs tracks/albums/artists, streams to the browser, and lets
you build playlists and "radio stations" — playlists that play on a fixed
timeline so every listener who tunes in hears the same thing at the same moment.

## Highlights

- **Album-aware shuffle.** Playlist entries are *typed*: a single track or a
  whole album. In album-shuffle mode, albums shuffle as indivisible units while
  their tracks stay in order — so a playlist of full albums shuffles by album.
- **Virtual radio.** A station has a fixed start time and a deterministic,
  seeded queue. Nothing streams continuously; the server *computes* the current
  track and offset from elapsed time on each tune-in, keeping listeners in sync.
- **Direct HTTP streaming** with Range requests (seeking) via `http.ServeContent`.
- **Live library updates** via startup scan, manual rescan, and an fsnotify watcher.

## Architecture

```
cmd/server            entrypoint, wiring, graceful shutdown
internal/config       env-based configuration
internal/db           pgx pool + embedded SQL migrations
internal/models       domain types
internal/catalog      catalog store (tracks/albums/artists)
  ├── metadata        tag + embedded-art extraction (external fallback stubbed)
  └── scanner         directory walker + fsnotify watcher
internal/playlist     playlist store + album-aware resolver
internal/radio        virtual-station position engine (+ unit tests)
internal/stream       Range-aware audio streaming
internal/api          chi router + handlers
web/                  React + Vite frontend stub (nginx in prod)
```

Stack: **Go + PostgreSQL + React**, all in Docker Compose.

## Run

```bash
# Point at your music library and start the stack.
MUSIC_DIR=/path/to/your/music docker compose up --build
```

- Frontend: http://localhost:5173
- API:      http://localhost:8080/api
- Health:   http://localhost:8080/healthz

### Local dev (without Docker)

```bash
# Backend (needs a Postgres reachable via RADIO_DATABASE_URL)
RADIO_MUSIC_DIRS=/path/to/music go run ./cmd/server

# Frontend
cd web && npm install && npm run dev
```

## Configuration (env vars)

| Var | Default | Meaning |
|-----|---------|---------|
| `RADIO_LISTEN_ADDR` | `:8080` | HTTP listen address |
| `RADIO_DATABASE_URL` | `postgres://radio:radio@localhost:5432/radio?sslmode=disable` | Postgres DSN |
| `RADIO_MUSIC_DIRS` | `/music` | Comma-separated library roots |
| `RADIO_ART_CACHE_DIR` | `/data/art` | Album-art cache directory |
| `RADIO_SCAN_ON_STARTUP` | `true` | Scan on boot |
| `RADIO_WATCH_ENABLED` | `true` | fsnotify live watcher |
| `RADIO_EXTERNAL_METADATA` | `false` | MusicBrainz/Cover Art Archive fallback |
| `RADIO_DEFAULT_USER_ID` | `0000…0001` | Seeded user that owns pre-auth data; bootstrapped to admin |
| `RADIO_JWT_SECRET` | _(random)_ | HMAC secret for JWTs. **Set in production** — otherwise tokens reset on restart |
| `RADIO_TOKEN_TTL` | `24h` | Issued token lifetime |
| `RADIO_ADMIN_USERNAME` | `admin` | Bootstrap admin username |
| `RADIO_ADMIN_PASSWORD` | _(empty)_ | Bootstrap admin password. Set on first run to enable login |

### Auth

Auth uses **JWT bearer tokens**. There is no open signup: the seeded default
user is bootstrapped into an **admin** account on first startup from
`RADIO_ADMIN_USERNAME` / `RADIO_ADMIN_PASSWORD` (the password is set only if the
account has none yet, so a later change isn't clobbered). Admins create
additional accounts via `POST /api/users`.

`POST /api/auth/login` returns a token; send it as `Authorization: Bearer <t>`.
For element URLs that can't set headers (`<audio>`, `<img>`, the station
WebSocket) the token may instead be passed as a `?token=` query param.
**Radio tune-in is public** (station read / now / ws); everything else requires
a token, and user management requires an admin.

## API sketch

```
POST   /api/auth/login                  # { username, password } -> { token, userId, isAdmin }  (public)
GET    /api/auth/me                      # current user
GET    /api/users                        # admin only
POST   /api/users                        # admin only: { username, password, isAdmin }

GET    /api/albums
GET    /api/albums/{id}/tracks
GET    /api/albums/{id}/art             # cached cover image, 404 if none
GET    /api/tracks/{id}
POST   /api/scan                       # full re-scan (async)
POST   /api/backfill                   # fill missing durations + art, no full scan (async)
GET    /api/stream/{trackID}            # Range-aware

GET    /api/playlists
POST   /api/playlists                   # { name }
GET    /api/playlists/{id}
DELETE /api/playlists/{id}
POST   /api/playlists/{id}/items        # { kind: "track"|"album", trackId|albumId }
DELETE /api/playlists/{id}/items/{itemID}

POST   /api/stations                    # { name, playlistId, albumShuffle, loop }  (auth)
GET    /api/stations/{id}               # public
GET    /api/stations/{id}/now           # computed NowPlaying (one-shot, public)
GET    /api/stations/{id}/ws            # WebSocket live-sync (public)
```

## Known gaps / next steps

- **Token revocation**: JWTs are stateless, so a token is valid until it
  expires (`RADIO_TOKEN_TTL`). There's no logout-everywhere / blocklist yet.
- **Password management**: no self-service password change or reset endpoint.

Done so far: **JWT auth** (admin-bootstrapped accounts, per-user playlists/
stations, public radio tune-in), catalog scan + watch, Range streaming,
album-aware shuffle, virtual radio, **WebSocket live-sync** for stations
(pushes NowPlaying on connect and at every track boundary), **ffprobe duration
probing**, **embedded album-art
extraction + serving** (cached to `RADIO_ART_CACHE_DIR`, keyed by album id),
a **combined backfill** (`POST /api/backfill`) that fills missing durations and
art for already-cataloged rows without a full re-scan, and an **external
metadata fallback** (MusicBrainz + Cover Art Archive) that fetches album covers
for art-less albums when `RADIO_EXTERNAL_METADATA=true`.

### External metadata

When `RADIO_EXTERNAL_METADATA=true`, albums with no embedded cover are matched
against MusicBrainz by (album-artist, album) and their front cover is pulled
from the Cover Art Archive, then cached like embedded art. Requests are
rate-limited to ~1/sec and send a descriptive `User-Agent` per MusicBrainz
policy — update the contact URL in `internal/catalog/metadata/external` before
public deployment. This applies on both live scans and `POST /api/backfill`.
```
