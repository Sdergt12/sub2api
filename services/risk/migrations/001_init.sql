CREATE TABLE IF NOT EXISTS risk_user_snapshot (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(128) NOT NULL,
    risk_score INTEGER NOT NULL DEFAULT 0,
    risk_level VARCHAR(32) NOT NULL DEFAULT 'low',
    request_count_1h BIGINT NOT NULL DEFAULT 0,
    request_count_24h BIGINT NOT NULL DEFAULT 0,
    cost_1h NUMERIC(18,6) NOT NULL DEFAULT 0,
    cost_24h NUMERIC(18,6) NOT NULL DEFAULT 0,
    unique_ip_24h INTEGER NOT NULL DEFAULT 0,
    top_models_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    risk_tags_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    last_event_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_risk_user_snapshot_user_id UNIQUE (user_id)
);

CREATE INDEX IF NOT EXISTS idx_risk_user_snapshot_risk_score
    ON risk_user_snapshot (risk_score DESC);

CREATE INDEX IF NOT EXISTS idx_risk_user_snapshot_risk_level
    ON risk_user_snapshot (risk_level);

CREATE INDEX IF NOT EXISTS idx_risk_user_snapshot_last_event_at
    ON risk_user_snapshot (last_event_at DESC);

CREATE TABLE IF NOT EXISTS risk_events (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(128) NOT NULL,
    api_key_id VARCHAR(128),
    event_type VARCHAR(64) NOT NULL,
    severity VARCHAR(32) NOT NULL DEFAULT 'info',
    score_delta INTEGER NOT NULL DEFAULT 0,
    title VARCHAR(255) NOT NULL,
    detail_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_risk_events_user_id
    ON risk_events (user_id);

CREATE INDEX IF NOT EXISTS idx_risk_events_event_type
    ON risk_events (event_type);

CREATE INDEX IF NOT EXISTS idx_risk_events_severity
    ON risk_events (severity);

CREATE INDEX IF NOT EXISTS idx_risk_events_occurred_at
    ON risk_events (occurred_at DESC);

CREATE TABLE IF NOT EXISTS risk_rule_hits (
    id BIGSERIAL PRIMARY KEY,
    event_id BIGINT NOT NULL REFERENCES risk_events(id) ON DELETE CASCADE,
    rule_code VARCHAR(128) NOT NULL,
    metric_value NUMERIC(18,6),
    threshold_value NUMERIC(18,6),
    extra_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_risk_rule_hits_event_id
    ON risk_rule_hits (event_id);

CREATE INDEX IF NOT EXISTS idx_risk_rule_hits_rule_code
    ON risk_rule_hits (rule_code);
