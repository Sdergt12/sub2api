-- 140_token_risk_events.sql
-- Token 风险事件表：从 ops_system_logs 的 audit.token 原始日志中提炼可处置的风控事件。

CREATE TABLE IF NOT EXISTS token_risk_events (
  id BIGSERIAL PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  source_log_id BIGINT UNIQUE REFERENCES ops_system_logs(id) ON DELETE SET NULL,
  user_id BIGINT,
  api_key_id BIGINT,
  token_type VARCHAR(64) NOT NULL DEFAULT '',
  token_hash VARCHAR(128) NOT NULL DEFAULT '',
  token_prefix VARCHAR(24) NOT NULL DEFAULT '',
  token_suffix VARCHAR(24) NOT NULL DEFAULT '',
  api_key_summary VARCHAR(64) NOT NULL DEFAULT '',
  client_ip VARCHAR(128) NOT NULL DEFAULT '',
  user_agent TEXT NOT NULL DEFAULT '',
  method VARCHAR(16) NOT NULL DEFAULT '',
  path TEXT NOT NULL DEFAULT '',
  status_code INTEGER NOT NULL DEFAULT 0,
  result VARCHAR(64) NOT NULL DEFAULT '',
  failure_reason VARCHAR(128) NOT NULL DEFAULT '',
  risk_score INTEGER NOT NULL DEFAULT 0,
  risk_level VARCHAR(16) NOT NULL DEFAULT 'low',
  risk_categories JSONB NOT NULL DEFAULT '[]'::jsonb,
  matched_rules JSONB NOT NULL DEFAULT '[]'::jsonb,
  recommended_actions JSONB NOT NULL DEFAULT '[]'::jsonb,
  explanation TEXT NOT NULL DEFAULT '',
  status VARCHAR(32) NOT NULL DEFAULT 'open',
  false_positive BOOLEAN NOT NULL DEFAULT FALSE,
  handled_by_user_id BIGINT,
  handled_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_token_risk_events_created_at
  ON token_risk_events (created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_token_risk_events_risk_level_created_at
  ON token_risk_events (risk_level, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_token_risk_events_status_created_at
  ON token_risk_events (status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_token_risk_events_user_created_at
  ON token_risk_events (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_token_risk_events_token_hash_created_at
  ON token_risk_events (token_hash, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_token_risk_events_api_key_created_at
  ON token_risk_events (api_key_id, created_at DESC);

CREATE TABLE IF NOT EXISTS token_risk_actions (
  id BIGSERIAL PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  event_id BIGINT REFERENCES token_risk_events(id) ON DELETE CASCADE,
  actor_user_id BIGINT NOT NULL,
  action VARCHAR(64) NOT NULL,
  note TEXT NOT NULL DEFAULT '',
  result VARCHAR(32) NOT NULL DEFAULT 'recorded',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_token_risk_actions_event_created_at
  ON token_risk_actions (event_id, created_at DESC);

CREATE TABLE IF NOT EXISTS token_risk_watchlist (
  id BIGSERIAL PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  subject_type VARCHAR(32) NOT NULL,
  subject_value VARCHAR(256) NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  actor_user_id BIGINT NOT NULL,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  UNIQUE(subject_type, subject_value)
);

CREATE INDEX IF NOT EXISTS idx_token_risk_watchlist_active
  ON token_risk_watchlist (active, subject_type, subject_value);

CREATE OR REPLACE FUNCTION token_risk_same_subject(left_event token_risk_events, right_event token_risk_events)
RETURNS BOOLEAN
LANGUAGE SQL
IMMUTABLE
AS $$
  SELECT
    (left_event.token_hash <> '' AND left_event.token_hash = right_event.token_hash)
    OR (left_event.api_key_id IS NOT NULL AND left_event.api_key_id = right_event.api_key_id)
    OR (left_event.user_id IS NOT NULL AND left_event.user_id = right_event.user_id)
$$;
