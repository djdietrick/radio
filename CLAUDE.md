# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A self-hosted music library, player, and **virtual radio** server: Go backend +
PostgreSQL + React (Vite/TS) frontend, shipped as a Docker Compose stack. It
scans configured directories for audio files, catalogs them, streams them to the
browser, and supports playlists and "radio stations."

Two features define the product and most of the non-obvious code:

- **Album-aware shuffle.** Playlist entries are *typed* — a single track or a
  whole album. With album-shuffle enabled, albums shuffle as indivisible units
  while their tracks stay in disc/track order.
- **Virtual radio.** A station has a fixed start time and a deterministic,
  seeded queue. Nothing streams continuously on the server; the current track +
  offset is *computed* from elapsed wall-clock time on each tune-in, so every
  listener is in sync. This is the single most important invariant — see below.

## Commands

```bash
# Build / vet / test everything (run from repo root)
go build ./...
go vet ./...
go test ./...

# A single package or test
go test ./internal/radio/
go test ./internal/radio/ -run TestComputeNowPlaying_Loops -v

# Run the whole stack (point MUSIC_DIR at a real library; set an admin password)
MUSIC_DIR=/path/to/music RADIO_ADMIN_PASSWORD=secret docker compose up --build

# Backend alone (needs a reachable Postgres via RADIO_DATABASE_URL)
RADIO_MUSIC_DIRS=/path/to/music RADIO_ADMIN_PASSWORD=secret go run ./cmd/server

# Frontend dev server (proxies /api + /healthz to localhost:8080)
cd web && npm install && npm run dev
cd web && npx tsc -b   # typecheck
```

## Test strategy (important)

The suite is deliberately **DB-free and network-free** so `go test ./...` runs
anywhere. Logic that needs Postgres or external services is tested by:

- Extracting pure functions and testing those (e.g. `computeNowPlaying`,
  `parseDurationMs`, `luceneQuote`).
- Injecting **interfaces** for external dependencies (e.g. `externalClient` in
  the metadata package is faked in tests; no MusicBrainz calls).
- DB-backed tests are **gated behind an env var** and skip cleanly when unset —
  see `internal/catalog/scanner/backfill_test.go` (`RADIO_TEST_DATABASE_URL`).

There is currently **no Postgres integration harness**, so the store/handler/
router wiring is verified only by `go build`/`go vet` plus these unit tests. When
touching SQL or request flows, that's the untested seam — call it out, and
prefer a gated integration test over leaving it unverified.

## Architecture

Request → `internal/api` (chi router) → `internal/api/handlers` → domain stores/
engines → `internal/db` (pgx pool). `cmd/server/main.go` constructs every
subsystem explicitly and injects it; there is no DI framework or global state.

Domain packages under `internal/`:

- **catalog** — `Store` is the catalog repository (tracks/albums/artists upsert
  + read queries). Subpackages: `metadata` (tag + embedded-art extraction;
  `external` is the MusicBrainz/Cover Art Archive client behind an interface),
  `probe` (ffprobe duration), `art` (file-backed cover cache keyed by album id),
  `scanner` (directory walk + fsnotify watcher + the combined backfill).
- **playlist** — `Store` (typed items) and `Resolver`, which flattens typed
  items into an ordered track queue. The resolver is the heart of album-aware
  shuffle: it builds shufflable *units* (a lone track, an album as one ordered
  unit, or a hand-picked *group* — consecutive `track` items sharing a
  `group_id` — kept together and in order) and shuffles units vs. flat tracks
  based on options. Groups behave like albums under shuffle: indivisible and
  ordered with `AlbumShuffle` on, broken apart in flat shuffle.
- **radio** — `Engine` computes station state. `computeNowPlaying` is the pure
  positional math; `LiveSession` caches a resolved queue so a WebSocket
  connection doesn't re-resolve on every track boundary.
- **stream** — Range-aware audio serving via `http.ServeContent`.
- **auth** / **users** — JWT bearer tokens (HMAC), bcrypt, middleware, user
  store + admin bootstrap.
- **api/ws** — the station live-sync WebSocket (`github.com/coder/websocket`).

### Invariants and conventions worth knowing before editing

- **Radio determinism.** A station's queue ordering comes from
  `playlist.Resolver` seeded with `Station.ShuffleSeed`, and position is a pure
  function of `(StartedAt, queue, now)`. Never introduce per-listener state or
  non-deterministic ordering into station resolution — it would desync
  listeners. `computeNowPlaying` defends against bad metadata: a 0-duration
  track gets a 3-minute fallback so the station never stalls.

- **WebSocket scheduling.** `NowPlaying.MsUntilNextTrack` is how the WS handler
  schedules its next push (one timer per connection, fired at the track
  boundary — not a poll). If you add station-mutation features, those fields and
  the boundary math must stay consistent.

- **Scan idempotency & backfill.** `UpsertTrack` keys on file path with
  `ON CONFLICT`; album/artist dedup uses normalized keys (`albumKey`/
  `artistKey`). New data (duration, art) only lands on the **next scan**.
  `POST /api/backfill` fills *only* rows missing that data (`duration_ms = 0`,
  `has_art = false`) and pages from offset 0 each iteration with a no-progress
  termination guard — preserve that guard or you risk an infinite loop.

- **Auth ownership.** Owned rows (playlists, stations) take their `user_id` from
  `auth.UserID(r.Context())`, populated by the `Require` middleware — *not* from
  config. The seeded default user (`RADIO_DEFAULT_USER_ID`) exists only to own
  pre-auth data and is bootstrapped into the admin account. **Radio tune-in
  (station read/now/ws) is public**; everything else requires a token, user
  management requires admin. Browser element URLs (`<audio>`, `<img>`, WS) carry
  the token as a `?token=` query param since they can't set headers.

- **Migrations** are embedded SQL in `internal/db/migrations/NNNN_*.sql`, applied
  in lexical order and tracked in `schema_migrations`. They run once — **add a
  new file; never edit an applied one.**

- **External services are optional and graceful.** ffprobe absence, MusicBrainz/
  CAA being disabled or failing, and missing files during a scan are all
  warn-and-continue, never fatal. Keep new external integrations behind a config
  flag and non-fatal.

## Config

All config is env-based (`internal/config`). Notable: `RADIO_MUSIC_DIRS`
(comma-separated, required), `RADIO_DATABASE_URL`, `RADIO_JWT_SECRET` (random
ephemeral if unset — **set in production or tokens reset on restart**),
`RADIO_ADMIN_PASSWORD` (must be set for first login), `RADIO_EXTERNAL_METADATA`.
See the README's config table for the full list.

## What's left to build (to complete the original vision)

The original goal: scan/catalog directories; a UI to browse and play with
features "standard in other music apps"; add songs **or whole albums** to
playlists and radio stations; album-aware shuffle; shared-timeline radio.

The **backend domain logic is the mature part** — scan/catalog, streaming,
album-aware shuffle, virtual-radio math, auth, art, duration, external metadata,
and the station WebSocket all exist and are unit-tested. What remains is mostly
**frontend** (the React app is an intentional stub) and a handful of **API/infra
holes**. A future agent picking this up should weight effort accordingly.

### API gaps (backend mostly done, these endpoints don't exist yet)

- **Search** — no full-text/substring search over tracks/albums/artists. Needed
  for any usable library UI. Add a `GET /api/search?q=` (Postgres `ILIKE` or a
  `tsvector` column) backed by a new `catalog.Store` method.
- **Artist browsing** — the `artists` table and `Artist` model exist, but there
  is **no list/get-artist or artist→albums endpoint**. Browse-by-artist is a
  baseline music-app feature.
- **Station listing** — you can create/read a station by id but there's **no
  `GET /api/stations`** to list them. Stations aren't discoverable in the UI
  without this.
- **Station item editing** — stations reference a playlist; there's no API to
  reorder/swap a station's playlist or edit items in place once created.
- **Pagination/sorting** — `ListAlbums` paginates; most other reads don't.
  Libraries get large; add consistent limit/offset/sort.

### Frontend (the largest gap — `web/src/App.tsx` is a single-file demo)

The current UI only proves the plumbing: login, list albums, play a track, tune
into a station by pasting an id. To reach "standard music app" it needs, roughly
in priority order:

- **Real navigation/layout** — album grid, artist view, track lists, a
  persistent player bar (the demo is one scrolling page with a bare `<audio>`).
- **Playback queue + transport** — next/prev, shuffle/repeat toggles, a visible
  upcoming queue, seek. None of this exists client-side today.
- **Playlist management UI** — create/edit playlists and **add whole albums vs.
  single tracks** (the typed-item API supports this; nothing surfaces it). This
  is the feature that motivates the whole typed-item design — it must be visible.
- **Station management UI** — create a station from a playlist with the
  album-shuffle/loop toggles, list/browse stations, share links. The
  album-shuffle mode is the product's differentiator and currently has **no UI**.
- **Now-playing / radio sync polish** — the WS pushes work; the UI should show
  station now-playing, queue position, and handle reconnect/clock-skew (the
  handler sends `ServerTime` for skew correction; the client ignores it).
- **State management** — `App.tsx` uses raw `useState`/`fetch`. A real app wants
  a query/cache layer and a player store.

### Infra / ops gaps

- **Postgres integration test harness** — there is none; the store/handler/router
  wiring is unverified by tests (see Test strategy). Highest-leverage infra task.
- **Production CORS/origins** — both REST CORS and the WS origin check are
  wildcard-open for dev (`AllowedOrigins: ["*"]`, `InsecureSkipVerify`). Lock
  these down for real deployment.
- **TLS / reverse proxy** — compose serves plain HTTP; WS uses `ws://`. A
  production deploy needs TLS termination (and the frontend already picks
  `wss://` when served over https).
- **Token revocation & password management** — stateless JWTs are valid until
  expiry; no logout-everywhere, no password change/reset endpoint.
- **Health/readiness depth & migrations on deploy** — `/healthz` is shallow (no
  DB check); migrations run in-process at startup (fine, but no separate
  migrate-only path for zero-downtime deploys).
- **Backups / volume strategy** — `db_data` and `art_cache` are named volumes
  with no documented backup path.
- **MusicBrainz `User-Agent`** contact URL in
  `internal/catalog/metadata/external` is a placeholder — set before public use.
