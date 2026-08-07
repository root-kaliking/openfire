# OpenFire Central Server

The centralized backend for the OpenFire Godot 4 project: account auth,
a WebSocket lobby, FIFO matchmaking, and lifecycle management for Godot
headless dedicated-server processes. Target scale: 8–16 players per
match.

## Tech stack

- **Language:** Go (module `openfire-server`)
- **HTTP router:** `github.com/go-chi/chi/v5`
- **WebSocket:** `github.com/gorilla/websocket`
- **DB:** PostgreSQL via `github.com/lib/pq`; Redis via `github.com/redis/go-redis/v9`
- **Auth:** JWT (HS256, `github.com/golang-jwt/jwt/v5`), bcrypt passwords
- **Config:** environment variables loaded from `.env` via `github.com/joho/godotenv`
- **Game servers:** Godot headless subprocesses managed by `exec.Command`

## Layout

```
server/
  go.mod
  .env.example
  migrations/001_init.sql
  internal/
    config/   config.go        # env-driven config
    db/       db.go            # PG pool (pq) + Redis client + migrate
    models/   user.go match.go match_player.go
    auth/     jwt.go password.go
    lobby/    hub.go client.go # WS lobby, gorilla/websocket
    matchmaking/ queue.go      # per-mode FIFO, opens a room when full
    gs/       manager.go       # Godot headless process + port pool
  cmd/server/main.go           # single entry point
  handlers/  auth_handler.go lobby_handler.go matchmaking_handler.go
             matches_handler.go internal_handler.go health_handler.go
  middleware/ auth.go internal.go
  logs/                        # GS per-match logs (created at runtime)
```

## Prerequisites

- Go 1.22+
- PostgreSQL 13+ (uses built-in `gen_random_uuid()`)
- Redis 6+
- A Godot 4 binary on PATH (or set `OPENFIRE_GODOT_BIN`) able to run
  `res://scenes/dedicated_server.tscn` headless from `/workspace`

## Setup

### 1. Database & Redis

Create the Postgres role/db used by the default `DB_URL`:

```sh
psql -U postgres -c "CREATE ROLE openfire WITH LOGIN PASSWORD 'openfire';"
psql -U postgres -c "CREATE DATABASE openfire OWNER openfire;"
```

Start Redis (e.g. `redis-server` or `docker run -p 6379:6379 redis:7`).

### 2. Configure environment

```sh
cd server
cp .env.example .env
# edit .env: set JWT_SECRET and INTERNAL_TOKEN to strong random values
```

See `.env.example` for every variable and its default.

### 3. Run migrations

Migrations run **automatically on server startup** by reading
`migrations/001_init.sql` (idempotent `CREATE TABLE IF NOT EXISTS`).
You can also apply it manually:

```sh
psql "$DB_URL" -f migrations/001_init.sql
```

### 4. Build & run

```sh
cd server
go mod tidy        # first time only; resolves indirect deps
go run ./cmd/server
```

The server logs `listening on :8080` and serves `/healthz`.

> The server must be started from the `server/` directory so it can find
> `migrations/001_init.sql` and write `logs/gs-{match_id}.log`. Godot GS
> processes are launched with `--path /workspace`.

## API

All non-2xx responses are `{"error":"..."}`. Authenticated routes require
`Authorization: Bearer <jwt>`.

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/api/auth/register` | – | body `{username,password}` → `{token,user:{id,username}}` |
| POST | `/api/auth/login` | – | body `{username,password}` → `{token,user:{id,username}}` |
| GET | `/api/ws` | Bearer | upgrade to WS lobby |
| POST | `/api/match/queue` | Bearer | body `{mode}` → `{queued:true}` |
| POST | `/api/match/cancel` | Bearer | body `{}` → `{canceled:true}` |
| GET | `/api/matches/{id}` | Bearer | match detail + players |
| GET | `/api/matches` | Bearer | caller's 20 most recent matches |
| POST | `/internal/matches/{id}/result` | `X-Internal-Token` | GS reports `{status,players:[{user_id,kills,deaths,placement,team}]}` |
| GET | `/healthz` | – | `{ok:true}` |

### WebSocket protocol (`/api/ws`)

Client → server:

```json
{"type":"match_queue","mode":"deathmatch"}
{"type":"match_cancel"}
```

Server → client:

```json
{"type":"queued"}
{"type":"match_found","match_id":"...","gs_host":"127.0.0.1","gs_port":27020,"players":[{"user_id":"...","username":"..."}],"mode":"deathmatch"}
{"type":"match_canceled"}
{"type":"error","message":"..."}
```

On `match_found`, the client connects to the game server with its
existing `Net.join_game(gs_host, gs_port)`.

### Matchmaking modes & room sizes

| mode | players/match |
|---|---|
| `deathmatch` | 8 |
| `team_dm` | 8 |
| `domination` | 8 |
| `battle_royale` | 16 |
| `coop` | 4 |
| `adventure` | 4 |

The queue is a per-mode FIFO. When a mode reaches its room size the
server creates a `pending` match, launches a Godot headless GS, flips the
match to `in_progress`, and pushes `match_found`. Players waiting longer
than `MATCHMAKING_TIMEOUT` are evicted and notified.

## Game server integration

For each match the central server runs:

```sh
$OPENFIRE_GODOT_BIN --headless --path /workspace res://scenes/dedicated_server.tscn
```

with these environment variables injected:

| Variable | Meaning |
|---|---|
| `OPENFIRE_GS_PORT` | UDP port the GS should listen on |
| `OPENFIRE_GS_MATCH_ID` | UUID of the match |
| `OPENFIRE_INTERNAL_TOKEN` | token for reporting results back |
| `OPENFIRE_CENTRAL_URL` | base URL of this central server |

GS stdout/stderr are tee'd to `server/logs/gs-{match_id}.log`. When the
process exits, the match is marked `abandoned` unless the GS already
reported a `finished`/`abandoned` result via
`POST /internal/matches/{id}/result`.

The dedicated server should, at end of match, POST to
`{OPENFIRE_CENTRAL_URL}/internal/matches/{OPENFIRE_GS_MATCH_ID}/result`
with header `X-Internal-Token: {OPENFIRE_INTERNAL_TOKEN}` and body:

```json
{"status":"finished","players":[{"user_id":"...","kills":0,"deaths":0,"placement":1,"team":0}]}
```

## Notes

- Matchmaking is in-memory (single central-server instance). The Redis
  client is wired up for session/presence use and can back a distributed
  queue later; for the 8–16/room target a local FIFO keeps `match_found`
  delivery on the same instance that owns the players' sockets.
- `go.mod` lists direct dependencies only; run `go mod tidy` once to
  populate indirect entries before `go build`.
