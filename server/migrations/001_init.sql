-- OpenFire central server schema (idempotent; safe to run on every boot).

CREATE TABLE IF NOT EXISTS users (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    username      TEXT         NOT NULL UNIQUE,
    password_hash TEXT         NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS matches (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    mode         TEXT         NOT NULL,
    status       TEXT         NOT NULL CHECK (status IN ('pending','in_progress','finished','abandoned')),
    gs_host      TEXT         NOT NULL DEFAULT '',
    gs_port      INTEGER      NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    finished_at  TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_matches_status ON matches(status);
CREATE INDEX IF NOT EXISTS idx_matches_created_at ON matches(created_at DESC);

CREATE TABLE IF NOT EXISTS match_players (
    match_id    UUID        NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    username    TEXT        NOT NULL,
    kills       INTEGER     NOT NULL DEFAULT 0,
    deaths      INTEGER     NOT NULL DEFAULT 0,
    placement   INTEGER,
    team        INTEGER,
    PRIMARY KEY (match_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_match_players_user_id ON match_players(user_id);
