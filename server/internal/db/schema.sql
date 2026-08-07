-- OpenFire central server schema
-- Run by internal/db migrations on startup (idempotent).

CREATE TABLE IF NOT EXISTS users (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    username      TEXT         NOT NULL UNIQUE,
    password_hash TEXT         NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS matches (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    mode         TEXT         NOT NULL,
    map          TEXT         NOT NULL,
    status       TEXT         NOT NULL,
    gs_host      TEXT         NOT NULL DEFAULT '',
    gs_port      INTEGER      NOT NULL DEFAULT 0,
    winner_team  INTEGER,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    started_at   TIMESTAMPTZ,
    finished_at  TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_matches_status ON matches(status);
CREATE INDEX IF NOT EXISTS idx_matches_created_at ON matches(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_matches_status_finished_at ON matches(status, finished_at);

CREATE TABLE IF NOT EXISTS match_players (
    match_id    UUID        NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    player_id   UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    username    TEXT        NOT NULL,
    kills       INTEGER     NOT NULL DEFAULT 0,
    deaths      INTEGER     NOT NULL DEFAULT 0,
    score       INTEGER     NOT NULL DEFAULT 0,
    team        INTEGER     NOT NULL DEFAULT 0,
    join_token  TEXT        NOT NULL,
    PRIMARY KEY (match_id, player_id)
);
CREATE INDEX IF NOT EXISTS idx_match_players_player_id ON match_players(player_id);

CREATE TABLE IF NOT EXISTS schema_migrations (
    version    TEXT        PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
