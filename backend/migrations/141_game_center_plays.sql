-- 141_game_center_plays.sql
-- 游戏中心战绩表：排行榜以 Sub2API 后端为唯一权威数据源，避免 iframe/Worker 本地伪造排名。
CREATE TABLE IF NOT EXISTS game_center_plays (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  game_key VARCHAR(64) NOT NULL,
  round_id VARCHAR(128) NOT NULL,
  stake_type VARCHAR(16) NOT NULL,
  cost_amount NUMERIC(20, 8) NOT NULL DEFAULT 0,
  reward_amount NUMERIC(20, 8) NOT NULL DEFAULT 0,
  net_amount NUMERIC(20, 8) NOT NULL DEFAULT 0,
  played_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(round_id)
);

CREATE INDEX IF NOT EXISTS idx_game_center_plays_user_round
  ON game_center_plays (user_id, round_id);

CREATE INDEX IF NOT EXISTS idx_game_center_plays_played_at
  ON game_center_plays (played_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_game_center_plays_user_played_at
  ON game_center_plays (user_id, played_at DESC);

CREATE INDEX IF NOT EXISTS idx_game_center_plays_game_played_at
  ON game_center_plays (game_key, played_at DESC);

CREATE INDEX IF NOT EXISTS idx_game_center_plays_leaderboard
  ON game_center_plays (game_key, played_at DESC, net_amount DESC);
