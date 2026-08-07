-- OpenFire central server initial schema
CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,
  username VARCHAR(32) UNIQUE NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE matches (
  id VARCHAR(32) PRIMARY KEY,
  mode VARCHAR(32) NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'queued',  -- queued/running/ended
  gs_port INT,
  gs_ip VARCHAR(64),
  started_at TIMESTAMPTZ,
  ended_at TIMESTAMPTZ
);

CREATE TABLE match_players (
  match_id VARCHAR(32) REFERENCES matches(id) ON DELETE CASCADE,
  user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
  session_token VARCHAR(64),
  placement INT,
  kills INT DEFAULT 0,
  deaths INT DEFAULT 0,
  connected BOOL DEFAULT FALSE,
  PRIMARY KEY (match_id, user_id)
);

CREATE TABLE user_stats (
  user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  wins INT DEFAULT 0,
  losses INT DEFAULT 0,
  kills INT DEFAULT 0,
  deaths INT DEFAULT 0,
  matches INT DEFAULT 0
);

CREATE INDEX idx_matches_status ON matches(status);
CREATE INDEX idx_match_players_user ON match_players(user_id);
