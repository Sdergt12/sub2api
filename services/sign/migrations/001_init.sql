CREATE TABLE IF NOT EXISTS sign_checkins (
    id BIGSERIAL PRIMARY KEY,
    sub2api_user_id BIGINT NOT NULL,
    sign_date DATE NOT NULL,
    base_reward NUMERIC(12,2) NOT NULL DEFAULT 0,
    bonus_reward NUMERIC(12,2) NOT NULL DEFAULT 0,
    total_reward NUMERIC(12,2) NOT NULL DEFAULT 0,
    reward_code VARCHAR(128) NOT NULL,
    grant_method VARCHAR(32) NOT NULL DEFAULT 'create_and_redeem',
    grant_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    idempotency_key VARCHAR(128) NOT NULL,
    ip VARCHAR(128) NOT NULL DEFAULT '',
    ua TEXT NOT NULL DEFAULT '',
    failure_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_sign_checkins_user_date UNIQUE (sub2api_user_id, sign_date),
    CONSTRAINT uq_sign_checkins_reward_code UNIQUE (reward_code)
);

CREATE INDEX IF NOT EXISTS idx_sign_checkins_status_date
    ON sign_checkins (grant_status, sign_date DESC);

CREATE INDEX IF NOT EXISTS idx_sign_checkins_user_created
    ON sign_checkins (sub2api_user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS sign_reward_attempts (
    id BIGSERIAL PRIMARY KEY,
    checkin_id BIGINT NOT NULL REFERENCES sign_checkins(id) ON DELETE CASCADE,
    reward_code VARCHAR(128) NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    request_payload TEXT NOT NULL DEFAULT '',
    response_payload TEXT NOT NULL DEFAULT '',
    http_status INTEGER NOT NULL DEFAULT 0,
    success BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sign_reward_attempts_checkin_id
    ON sign_reward_attempts (checkin_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_sign_reward_attempts_reward_code
    ON sign_reward_attempts (reward_code, created_at DESC);

CREATE TABLE IF NOT EXISTS sign_risk_events (
    id BIGSERIAL PRIMARY KEY,
    sub2api_user_id BIGINT NOT NULL,
    risk_type VARCHAR(64) NOT NULL,
    risk_score INTEGER NOT NULL DEFAULT 0,
    detail TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sign_risk_events_user_created
    ON sign_risk_events (sub2api_user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS sign_user_stats (
    sub2api_user_id BIGINT PRIMARY KEY,
    last_sign_date DATE NOT NULL,
    current_streak INTEGER NOT NULL DEFAULT 1,
    total_reward NUMERIC(14,2) NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
