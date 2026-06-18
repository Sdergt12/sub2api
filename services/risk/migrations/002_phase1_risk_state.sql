ALTER TABLE risk_user_snapshot
    ADD COLUMN IF NOT EXISTS window_start TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS window_end TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS refreshed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS is_stale BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE risk_user_snapshot
SET
    window_end = COALESCE(window_end, updated_at, created_at),
    refreshed_at = COALESCE(refreshed_at, updated_at, created_at),
    window_start = COALESCE(
        window_start,
        COALESCE(updated_at, created_at) - INTERVAL '24 hours'
    )
WHERE window_start IS NULL
   OR window_end IS NULL
   OR refreshed_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_risk_user_snapshot_is_stale
    ON risk_user_snapshot (is_stale);

CREATE INDEX IF NOT EXISTS idx_risk_user_snapshot_refreshed_at
    ON risk_user_snapshot (refreshed_at DESC);

ALTER TABLE risk_events
    ADD COLUMN IF NOT EXISTS event_fingerprint VARCHAR(255),
    ADD COLUMN IF NOT EXISTS status VARCHAR(32) NOT NULL DEFAULT 'open',
    ADD COLUMN IF NOT EXISTS first_seen_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS hit_count INTEGER NOT NULL DEFAULT 1;

UPDATE risk_events
SET
    event_fingerprint = COALESCE(event_fingerprint, 'legacy:' || id::text),
    status = COALESCE(NULLIF(status, ''), 'resolved'),
    first_seen_at = COALESCE(first_seen_at, occurred_at),
    last_seen_at = COALESCE(last_seen_at, occurred_at),
    hit_count = COALESCE(NULLIF(hit_count, 0), 1)
WHERE event_fingerprint IS NULL
   OR status IS NULL
   OR status = ''
   OR first_seen_at IS NULL
   OR last_seen_at IS NULL
   OR hit_count IS NULL
   OR hit_count = 0;

CREATE UNIQUE INDEX IF NOT EXISTS uq_risk_events_event_fingerprint
    ON risk_events (event_fingerprint);

DELETE FROM risk_rule_hits a
USING risk_rule_hits b
WHERE a.ctid < b.ctid
  AND a.event_id = b.event_id
  AND a.rule_code = b.rule_code;

CREATE UNIQUE INDEX IF NOT EXISTS uq_risk_rule_hits_event_rule
    ON risk_rule_hits (event_id, rule_code);
