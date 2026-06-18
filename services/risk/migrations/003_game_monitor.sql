CREATE TABLE IF NOT EXISTS game_monitor_daily_snapshot (
    day_key VARCHAR(16) PRIMARY KEY,
    total_rounds BIGINT NOT NULL DEFAULT 0,
    free_rounds BIGINT NOT NULL DEFAULT 0,
    paid_rounds BIGINT NOT NULL DEFAULT 0,
    gross_reward_amount NUMERIC(18,2) NOT NULL DEFAULT 0,
    entry_fee_amount NUMERIC(18,2) NOT NULL DEFAULT 0,
    net_reward_amount NUMERIC(18,2) NOT NULL DEFAULT 0,
    pending_claims BIGINT NOT NULL DEFAULT 0,
    failed_claims BIGINT NOT NULL DEFAULT 0,
    cap_hits BIGINT NOT NULL DEFAULT 0,
    refreshed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_game_monitor_daily_snapshot_day_key
    ON game_monitor_daily_snapshot (day_key);

CREATE TABLE IF NOT EXISTS game_monitor_user_snapshot (
    id BIGSERIAL PRIMARY KEY,
    day_key VARCHAR(16) NOT NULL,
    user_id VARCHAR(128) NOT NULL,
    username VARCHAR(255) NOT NULL DEFAULT '',
    play_count BIGINT NOT NULL DEFAULT 0,
    free_play_count BIGINT NOT NULL DEFAULT 0,
    paid_play_count BIGINT NOT NULL DEFAULT 0,
    gross_reward_amount NUMERIC(18,2) NOT NULL DEFAULT 0,
    entry_fee_amount NUMERIC(18,2) NOT NULL DEFAULT 0,
    net_reward_amount NUMERIC(18,2) NOT NULL DEFAULT 0,
    pending_claim_count BIGINT NOT NULL DEFAULT 0,
    failed_claim_count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_game_monitor_user_snapshot_day_user UNIQUE (user_id, day_key)
);

CREATE INDEX IF NOT EXISTS idx_game_monitor_user_snapshot_day_key
    ON game_monitor_user_snapshot (day_key);

CREATE INDEX IF NOT EXISTS idx_game_monitor_user_snapshot_user_day
    ON game_monitor_user_snapshot (user_id, day_key);

CREATE INDEX IF NOT EXISTS idx_game_monitor_user_snapshot_net_reward_desc
    ON game_monitor_user_snapshot (net_reward_amount DESC);

CREATE TABLE IF NOT EXISTS game_monitor_rounds (
    round_id VARCHAR(128) PRIMARY KEY,
    day_key VARCHAR(16) NOT NULL,
    user_id VARCHAR(128) NOT NULL,
    username VARCHAR(255) NOT NULL DEFAULT '',
    game_id VARCHAR(64) NOT NULL,
    risk_mode VARCHAR(64) NOT NULL DEFAULT 'steady',
    status VARCHAR(32) NOT NULL,
    claim_status VARCHAR(32) NOT NULL DEFAULT 'none',
    is_paid_round BOOLEAN NOT NULL DEFAULT FALSE,
    choice_index INTEGER,
    entry_fee_amount NUMERIC(18,2) NOT NULL DEFAULT 0,
    gross_reward_amount NUMERIC(18,2) NOT NULL DEFAULT 0,
    net_reward_amount NUMERIC(18,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL,
    revealed_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_game_monitor_rounds_day_key
    ON game_monitor_rounds (day_key);

CREATE INDEX IF NOT EXISTS idx_game_monitor_rounds_user_day
    ON game_monitor_rounds (user_id, day_key);

CREATE INDEX IF NOT EXISTS idx_game_monitor_rounds_game_day
    ON game_monitor_rounds (game_id, day_key);

CREATE INDEX IF NOT EXISTS idx_game_monitor_rounds_claim_status
    ON game_monitor_rounds (claim_status);

CREATE TABLE IF NOT EXISTS game_monitor_claims (
    claim_id VARCHAR(128) PRIMARY KEY,
    round_id VARCHAR(128) NOT NULL,
    day_key VARCHAR(16) NOT NULL,
    user_id VARCHAR(128) NOT NULL,
    username VARCHAR(255) NOT NULL DEFAULT '',
    game_id VARCHAR(64) NOT NULL,
    risk_mode VARCHAR(64) NOT NULL DEFAULT 'steady',
    claim_kind VARCHAR(32) NOT NULL,
    claim_status VARCHAR(32) NOT NULL,
    amount NUMERIC(18,2) NOT NULL DEFAULT 0,
    entry_fee_amount NUMERIC(18,2) NOT NULL DEFAULT 0,
    gross_reward_amount NUMERIC(18,2) NOT NULL DEFAULT 0,
    net_reward_amount NUMERIC(18,2) NOT NULL DEFAULT 0,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    redeemed_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_game_monitor_claims_day_key
    ON game_monitor_claims (day_key);

CREATE INDEX IF NOT EXISTS idx_game_monitor_claims_user_day
    ON game_monitor_claims (user_id, day_key);

CREATE INDEX IF NOT EXISTS idx_game_monitor_claims_game_day
    ON game_monitor_claims (game_id, day_key);

CREATE INDEX IF NOT EXISTS idx_game_monitor_claims_claim_status
    ON game_monitor_claims (claim_status);

CREATE TABLE IF NOT EXISTS game_monitor_events (
    event_key VARCHAR(255) PRIMARY KEY,
    day_key VARCHAR(16) NOT NULL,
    user_id VARCHAR(128) NOT NULL,
    username VARCHAR(255) NOT NULL DEFAULT '',
    game_id VARCHAR(64) NOT NULL DEFAULT '',
    round_id VARCHAR(128) NOT NULL DEFAULT '',
    claim_id VARCHAR(128) NOT NULL DEFAULT '',
    risk_mode VARCHAR(64) NOT NULL DEFAULT '',
    event_type VARCHAR(64) NOT NULL,
    severity VARCHAR(32) NOT NULL,
    message TEXT NOT NULL,
    detail_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_game_monitor_events_day_key
    ON game_monitor_events (day_key);

CREATE INDEX IF NOT EXISTS idx_game_monitor_events_user_day
    ON game_monitor_events (user_id, day_key);

CREATE INDEX IF NOT EXISTS idx_game_monitor_events_game_day
    ON game_monitor_events (game_id, day_key);

CREATE INDEX IF NOT EXISTS idx_game_monitor_events_type_occurred
    ON game_monitor_events (event_type, occurred_at DESC);
