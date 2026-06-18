package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"sub2api-risk/internal/model"
)

func (r *RiskRepository) UpsertGameMonitorRound(ctx context.Context, input model.GameMonitorRoundUpsertInput) error {
	const query = `
INSERT INTO game_monitor_rounds (
	round_id, day_key, user_id, username, game_id, risk_mode, status, claim_status, is_paid_round,
	choice_index, entry_fee_amount, gross_reward_amount, net_reward_amount, created_at, revealed_at, updated_at
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8, $9,
	$10, $11, $12, $13, $14, $15, $16
)
ON CONFLICT (round_id) DO UPDATE SET
	day_key = EXCLUDED.day_key,
	user_id = EXCLUDED.user_id,
	username = EXCLUDED.username,
	game_id = EXCLUDED.game_id,
	risk_mode = EXCLUDED.risk_mode,
	status = EXCLUDED.status,
	claim_status = EXCLUDED.claim_status,
	is_paid_round = EXCLUDED.is_paid_round,
	choice_index = EXCLUDED.choice_index,
	entry_fee_amount = EXCLUDED.entry_fee_amount,
	gross_reward_amount = EXCLUDED.gross_reward_amount,
	net_reward_amount = EXCLUDED.net_reward_amount,
	created_at = EXCLUDED.created_at,
	revealed_at = EXCLUDED.revealed_at,
	updated_at = EXCLUDED.updated_at
`
	_, err := r.db.Exec(
		ctx,
		query,
		input.RoundID,
		input.DayKey,
		input.UserID,
		input.Username,
		input.GameID,
		input.RiskMode,
		input.Status,
		input.ClaimStatus,
		input.IsPaidRound,
		input.ChoiceIndex,
		input.EntryFeeAmount,
		input.GrossRewardAmount,
		input.NetRewardAmount,
		input.CreatedAt,
		input.RevealedAt,
		input.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert game monitor round: %w", err)
	}
	return nil
}

func (r *RiskRepository) UpsertGameMonitorClaim(ctx context.Context, input model.GameMonitorClaimUpsertInput) error {
	const query = `
INSERT INTO game_monitor_claims (
	claim_id, round_id, day_key, user_id, username, game_id, risk_mode, claim_kind, claim_status,
	amount, entry_fee_amount, gross_reward_amount, net_reward_amount, attempt_count, last_error,
	created_at, redeemed_at, updated_at
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8, $9,
	$10, $11, $12, $13, $14, $15,
	$16, $17, $18
)
ON CONFLICT (claim_id) DO UPDATE SET
	round_id = EXCLUDED.round_id,
	day_key = EXCLUDED.day_key,
	user_id = EXCLUDED.user_id,
	username = EXCLUDED.username,
	game_id = EXCLUDED.game_id,
	risk_mode = EXCLUDED.risk_mode,
	claim_kind = EXCLUDED.claim_kind,
	claim_status = EXCLUDED.claim_status,
	amount = EXCLUDED.amount,
	entry_fee_amount = EXCLUDED.entry_fee_amount,
	gross_reward_amount = EXCLUDED.gross_reward_amount,
	net_reward_amount = EXCLUDED.net_reward_amount,
	attempt_count = EXCLUDED.attempt_count,
	last_error = EXCLUDED.last_error,
	created_at = EXCLUDED.created_at,
	redeemed_at = EXCLUDED.redeemed_at,
	updated_at = EXCLUDED.updated_at
`
	_, err := r.db.Exec(
		ctx,
		query,
		input.ClaimID,
		input.RoundID,
		input.DayKey,
		input.UserID,
		input.Username,
		input.GameID,
		input.RiskMode,
		input.ClaimKind,
		input.ClaimStatus,
		input.Amount,
		input.EntryFeeAmount,
		input.GrossRewardAmount,
		input.NetRewardAmount,
		input.AttemptCount,
		input.LastError,
		input.CreatedAt,
		input.RedeemedAt,
		input.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert game monitor claim: %w", err)
	}
	return nil
}

func (r *RiskRepository) UpsertGameMonitorEvent(ctx context.Context, input model.GameMonitorEventUpsertInput) error {
	detailJSON, err := json.Marshal(input.Detail)
	if err != nil {
		return fmt.Errorf("marshal game monitor event detail: %w", err)
	}

	const query = `
INSERT INTO game_monitor_events (
	event_key, day_key, user_id, username, game_id, round_id, claim_id, risk_mode,
	event_type, severity, message, detail_json, occurred_at, updated_at
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8,
	$9, $10, $11, $12::jsonb, $13, NOW()
)
ON CONFLICT (event_key) DO UPDATE SET
	day_key = EXCLUDED.day_key,
	user_id = EXCLUDED.user_id,
	username = EXCLUDED.username,
	game_id = EXCLUDED.game_id,
	round_id = EXCLUDED.round_id,
	claim_id = EXCLUDED.claim_id,
	risk_mode = EXCLUDED.risk_mode,
	event_type = EXCLUDED.event_type,
	severity = EXCLUDED.severity,
	message = EXCLUDED.message,
	detail_json = EXCLUDED.detail_json,
	occurred_at = EXCLUDED.occurred_at,
	updated_at = NOW()
`
	_, err = r.db.Exec(
		ctx,
		query,
		input.EventKey,
		input.DayKey,
		input.UserID,
		input.Username,
		input.GameID,
		input.RoundID,
		input.ClaimID,
		input.RiskMode,
		input.EventType,
		input.Severity,
		input.Message,
		string(detailJSON),
		input.OccurredAt,
	)
	if err != nil {
		return fmt.Errorf("upsert game monitor event: %w", err)
	}
	return nil
}

func (r *RiskRepository) UpsertGameMonitorUserSnapshot(ctx context.Context, input model.GameMonitorUserSnapshotUpsertInput) error {
	const query = `
INSERT INTO game_monitor_user_snapshot (
	day_key, user_id, username, play_count, free_play_count, paid_play_count, gross_reward_amount,
	entry_fee_amount, net_reward_amount, pending_claim_count, failed_claim_count, updated_at
) VALUES (
	$1, $2, $3, $4, $5, $6, $7,
	$8, $9, $10, $11, NOW()
)
ON CONFLICT (user_id, day_key) DO UPDATE SET
	username = EXCLUDED.username,
	play_count = EXCLUDED.play_count,
	free_play_count = EXCLUDED.free_play_count,
	paid_play_count = EXCLUDED.paid_play_count,
	gross_reward_amount = EXCLUDED.gross_reward_amount,
	entry_fee_amount = EXCLUDED.entry_fee_amount,
	net_reward_amount = EXCLUDED.net_reward_amount,
	pending_claim_count = EXCLUDED.pending_claim_count,
	failed_claim_count = EXCLUDED.failed_claim_count,
	updated_at = NOW()
`
	_, err := r.db.Exec(
		ctx,
		query,
		input.DayKey,
		input.UserID,
		input.Username,
		input.PlayCount,
		input.FreePlayCount,
		input.PaidPlayCount,
		input.GrossRewardAmount,
		input.EntryFeeAmount,
		input.NetRewardAmount,
		input.PendingClaimCount,
		input.FailedClaimCount,
	)
	if err != nil {
		return fmt.Errorf("upsert game monitor user snapshot: %w", err)
	}
	return nil
}

func (r *RiskRepository) RefreshGameMonitorDailySnapshot(ctx context.Context, dayKey string) error {
	const query = `
WITH round_stats AS (
	SELECT
		COUNT(*) AS total_rounds,
		COUNT(*) FILTER (WHERE is_paid_round = FALSE) AS free_rounds,
		COUNT(*) FILTER (WHERE is_paid_round = TRUE) AS paid_rounds,
		COALESCE(SUM(gross_reward_amount), 0)::float8 AS gross_reward_amount,
		COALESCE(SUM(entry_fee_amount), 0)::float8 AS entry_fee_amount,
		COALESCE(SUM(net_reward_amount), 0)::float8 AS net_reward_amount
	FROM game_monitor_rounds
	WHERE day_key = $1
),
claim_stats AS (
	SELECT
		COUNT(*) FILTER (WHERE claim_status = 'reconcile_pending') AS pending_claims,
		COUNT(*) FILTER (WHERE claim_status = 'failed') AS failed_claims
	FROM game_monitor_claims
	WHERE day_key = $1
),
event_stats AS (
	SELECT
		COUNT(*) FILTER (WHERE event_type = 'cap_hit') AS cap_hits
	FROM game_monitor_events
	WHERE day_key = $1
)
INSERT INTO game_monitor_daily_snapshot (
	day_key, total_rounds, free_rounds, paid_rounds, gross_reward_amount, entry_fee_amount,
	net_reward_amount, pending_claims, failed_claims, cap_hits, refreshed_at, updated_at
)
SELECT
	$1,
	round_stats.total_rounds,
	round_stats.free_rounds,
	round_stats.paid_rounds,
	round_stats.gross_reward_amount,
	round_stats.entry_fee_amount,
	round_stats.net_reward_amount,
	claim_stats.pending_claims,
	claim_stats.failed_claims,
	event_stats.cap_hits,
	NOW(),
	NOW()
FROM round_stats, claim_stats, event_stats
ON CONFLICT (day_key) DO UPDATE SET
	total_rounds = EXCLUDED.total_rounds,
	free_rounds = EXCLUDED.free_rounds,
	paid_rounds = EXCLUDED.paid_rounds,
	gross_reward_amount = EXCLUDED.gross_reward_amount,
	entry_fee_amount = EXCLUDED.entry_fee_amount,
	net_reward_amount = EXCLUDED.net_reward_amount,
	pending_claims = EXCLUDED.pending_claims,
	failed_claims = EXCLUDED.failed_claims,
	cap_hits = EXCLUDED.cap_hits,
	refreshed_at = EXCLUDED.refreshed_at,
	updated_at = NOW()
`
	_, err := r.db.Exec(ctx, query, dayKey)
	if err != nil {
		return fmt.Errorf("refresh game monitor daily snapshot: %w", err)
	}
	return nil
}

func (r *RiskRepository) GetGameMonitorOverview(ctx context.Context, dayKey string) (model.GameMonitorOverview, error) {
	if strings.TrimSpace(dayKey) == "" {
		const latestQuery = `
SELECT day_key, total_rounds, free_rounds, paid_rounds,
       gross_reward_amount::float8, entry_fee_amount::float8, net_reward_amount::float8,
       pending_claims, failed_claims, cap_hits, refreshed_at
FROM game_monitor_daily_snapshot
ORDER BY (total_rounds > 0) DESC, day_key DESC
LIMIT 1
`
		var item model.GameMonitorOverview
		var refreshedAt sql.NullTime
		err := r.db.QueryRow(ctx, latestQuery).Scan(
			&item.DayKey,
			&item.TotalRounds,
			&item.FreeRounds,
			&item.PaidRounds,
			&item.GrossRewardAmount,
			&item.EntryFeeAmount,
			&item.NetRewardAmount,
			&item.PendingClaims,
			&item.FailedClaims,
			&item.CapHits,
			&refreshedAt,
		)
		if err != nil {
			if err == sql.ErrNoRows || err == pgx.ErrNoRows {
				return model.GameMonitorOverview{}, nil
			}
			return model.GameMonitorOverview{}, fmt.Errorf("query latest game monitor overview: %w", err)
		}
		if refreshedAt.Valid {
			item.RefreshedAt = &refreshedAt.Time
		}
		return item, nil
	}

	const query = `
SELECT day_key, total_rounds, free_rounds, paid_rounds,
       gross_reward_amount::float8, entry_fee_amount::float8, net_reward_amount::float8,
       pending_claims, failed_claims, cap_hits, refreshed_at
FROM game_monitor_daily_snapshot
WHERE day_key = $1
`
	var item model.GameMonitorOverview
	var refreshedAt sql.NullTime
	err := r.db.QueryRow(ctx, query, dayKey).Scan(
		&item.DayKey,
		&item.TotalRounds,
		&item.FreeRounds,
		&item.PaidRounds,
		&item.GrossRewardAmount,
		&item.EntryFeeAmount,
		&item.NetRewardAmount,
		&item.PendingClaims,
		&item.FailedClaims,
		&item.CapHits,
		&refreshedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows || err == pgx.ErrNoRows {
			return model.GameMonitorOverview{DayKey: dayKey}, nil
		}
		return model.GameMonitorOverview{}, fmt.Errorf("query game monitor overview: %w", err)
	}
	if refreshedAt.Valid {
		item.RefreshedAt = &refreshedAt.Time
	}
	return item, nil
}

func (r *RiskRepository) ListGameMonitorUsers(ctx context.Context, filter model.GameMonitorListFilter) ([]model.GameMonitorUserSnapshot, int64, error) {
	whereSQL, args := buildGameUserFilter(filter)
	countQuery := `SELECT COUNT(*) FROM game_monitor_user_snapshot` + whereSQL

	var total int64
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count game monitor users: %w", err)
	}

	orderBy := "ORDER BY net_reward_amount DESC, play_count DESC, id DESC"
	switch filter.SortBy {
	case "play_count":
		orderBy = "ORDER BY play_count DESC, net_reward_amount DESC, id DESC"
	case "pending_claim_count":
		orderBy = "ORDER BY pending_claim_count DESC, net_reward_amount DESC, id DESC"
	case "failed_claim_count":
		orderBy = "ORDER BY failed_claim_count DESC, net_reward_amount DESC, id DESC"
	}

	dataQuery := `
SELECT
	id, day_key, user_id, username, play_count, free_play_count, paid_play_count,
	gross_reward_amount::float8, entry_fee_amount::float8, net_reward_amount::float8,
	pending_claim_count, failed_claim_count, created_at, updated_at
FROM game_monitor_user_snapshot` + whereSQL + fmt.Sprintf(`
%s
LIMIT $%d OFFSET $%d`, orderBy, len(args)+1, len(args)+2)

	rows, err := r.db.Query(ctx, dataQuery, append(args, filter.PageSize, offset(filter.Page, filter.PageSize))...)
	if err != nil {
		return nil, 0, fmt.Errorf("query game monitor users: %w", err)
	}
	defer rows.Close()

	items := make([]model.GameMonitorUserSnapshot, 0, filter.PageSize)
	for rows.Next() {
		var item model.GameMonitorUserSnapshot
		if err := rows.Scan(
			&item.ID,
			&item.DayKey,
			&item.UserID,
			&item.Username,
			&item.PlayCount,
			&item.FreePlayCount,
			&item.PaidPlayCount,
			&item.GrossRewardAmount,
			&item.EntryFeeAmount,
			&item.NetRewardAmount,
			&item.PendingClaimCount,
			&item.FailedClaimCount,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan game monitor user row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate game monitor user rows: %w", err)
	}
	return items, total, nil
}

func (r *RiskRepository) ListGameMonitorRounds(ctx context.Context, filter model.GameMonitorListFilter) ([]model.GameMonitorRound, int64, error) {
	whereSQL, args := buildGameRoundFilter(filter)
	countQuery := `SELECT COUNT(*) FROM game_monitor_rounds` + whereSQL

	var total int64
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count game monitor rounds: %w", err)
	}

	dataQuery := `
SELECT
	round_id, day_key, user_id, username, game_id, risk_mode, status, claim_status, is_paid_round,
	choice_index, entry_fee_amount::float8, gross_reward_amount::float8, net_reward_amount::float8,
	created_at, revealed_at, updated_at
FROM game_monitor_rounds` + whereSQL + fmt.Sprintf(`
ORDER BY created_at DESC, round_id DESC
LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)

	rows, err := r.db.Query(ctx, dataQuery, append(args, filter.PageSize, offset(filter.Page, filter.PageSize))...)
	if err != nil {
		return nil, 0, fmt.Errorf("query game monitor rounds: %w", err)
	}
	defer rows.Close()

	items := make([]model.GameMonitorRound, 0, filter.PageSize)
	for rows.Next() {
		var item model.GameMonitorRound
		var choiceIndex sql.NullInt32
		var revealedAt sql.NullTime
		if err := rows.Scan(
			&item.RoundID,
			&item.DayKey,
			&item.UserID,
			&item.Username,
			&item.GameID,
			&item.RiskMode,
			&item.Status,
			&item.ClaimStatus,
			&item.IsPaidRound,
			&choiceIndex,
			&item.EntryFeeAmount,
			&item.GrossRewardAmount,
			&item.NetRewardAmount,
			&item.CreatedAt,
			&revealedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan game monitor round row: %w", err)
		}
		if choiceIndex.Valid {
			value := int(choiceIndex.Int32)
			item.ChoiceIndex = &value
		}
		if revealedAt.Valid {
			item.RevealedAt = &revealedAt.Time
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate game monitor round rows: %w", err)
	}
	return items, total, nil
}

func (r *RiskRepository) ListGameMonitorClaims(ctx context.Context, filter model.GameMonitorListFilter) ([]model.GameMonitorClaim, int64, error) {
	whereSQL, args := buildGameClaimFilter(filter)
	countQuery := `SELECT COUNT(*) FROM game_monitor_claims` + whereSQL

	var total int64
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count game monitor claims: %w", err)
	}

	dataQuery := `
SELECT
	claim_id, round_id, day_key, user_id, username, game_id, risk_mode, claim_kind, claim_status,
	amount::float8, entry_fee_amount::float8, gross_reward_amount::float8, net_reward_amount::float8,
	attempt_count, last_error, created_at, redeemed_at, updated_at
FROM game_monitor_claims` + whereSQL + fmt.Sprintf(`
ORDER BY created_at DESC, claim_id DESC
LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)

	rows, err := r.db.Query(ctx, dataQuery, append(args, filter.PageSize, offset(filter.Page, filter.PageSize))...)
	if err != nil {
		return nil, 0, fmt.Errorf("query game monitor claims: %w", err)
	}
	defer rows.Close()

	items := make([]model.GameMonitorClaim, 0, filter.PageSize)
	for rows.Next() {
		var item model.GameMonitorClaim
		var redeemedAt sql.NullTime
		if err := rows.Scan(
			&item.ClaimID,
			&item.RoundID,
			&item.DayKey,
			&item.UserID,
			&item.Username,
			&item.GameID,
			&item.RiskMode,
			&item.ClaimKind,
			&item.ClaimStatus,
			&item.Amount,
			&item.EntryFeeAmount,
			&item.GrossRewardAmount,
			&item.NetRewardAmount,
			&item.AttemptCount,
			&item.LastError,
			&item.CreatedAt,
			&redeemedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan game monitor claim row: %w", err)
		}
		if redeemedAt.Valid {
			item.RedeemedAt = &redeemedAt.Time
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate game monitor claim rows: %w", err)
	}
	return items, total, nil
}

func (r *RiskRepository) ListGameMonitorEvents(ctx context.Context, filter model.GameMonitorListFilter) ([]model.GameMonitorEvent, int64, error) {
	whereSQL, args := buildGameEventFilter(filter)
	countQuery := `SELECT COUNT(*) FROM game_monitor_events` + whereSQL

	var total int64
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count game monitor events: %w", err)
	}

	dataQuery := `
SELECT
	event_key, day_key, user_id, username, game_id, round_id, claim_id, risk_mode,
	event_type, severity, message, detail_json, occurred_at, created_at, updated_at
FROM game_monitor_events` + whereSQL + fmt.Sprintf(`
ORDER BY occurred_at DESC, event_key DESC
LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)

	rows, err := r.db.Query(ctx, dataQuery, append(args, filter.PageSize, offset(filter.Page, filter.PageSize))...)
	if err != nil {
		return nil, 0, fmt.Errorf("query game monitor events: %w", err)
	}
	defer rows.Close()

	items := make([]model.GameMonitorEvent, 0, filter.PageSize)
	for rows.Next() {
		var item model.GameMonitorEvent
		var detailJSON []byte
		if err := rows.Scan(
			&item.EventKey,
			&item.DayKey,
			&item.UserID,
			&item.Username,
			&item.GameID,
			&item.RoundID,
			&item.ClaimID,
			&item.RiskMode,
			&item.EventType,
			&item.Severity,
			&item.Message,
			&detailJSON,
			&item.OccurredAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan game monitor event row: %w", err)
		}
		item.Detail = decodeDetailMap(detailJSON)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate game monitor event rows: %w", err)
	}
	return items, total, nil
}

func buildGameUserFilter(filter model.GameMonitorListFilter) (string, []any) {
	conditions := make([]string, 0, 2)
	args := make([]any, 0, 2)

	if dayKey := strings.TrimSpace(filter.DayKey); dayKey != "" {
		args = append(args, dayKey)
		conditions = append(conditions, fmt.Sprintf("day_key = $%d", len(args)))
	}
	if userID := strings.TrimSpace(filter.UserID); userID != "" {
		args = append(args, "%"+userID+"%")
		conditions = append(conditions, fmt.Sprintf("user_id ILIKE $%d", len(args)))
	}
	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func buildGameRoundFilter(filter model.GameMonitorListFilter) (string, []any) {
	conditions := make([]string, 0, 5)
	args := make([]any, 0, 5)

	if dayKey := strings.TrimSpace(filter.DayKey); dayKey != "" {
		args = append(args, dayKey)
		conditions = append(conditions, fmt.Sprintf("day_key = $%d", len(args)))
	}
	if userID := strings.TrimSpace(filter.UserID); userID != "" {
		args = append(args, "%"+userID+"%")
		conditions = append(conditions, fmt.Sprintf("user_id ILIKE $%d", len(args)))
	}
	if gameID := strings.TrimSpace(filter.GameID); gameID != "" {
		args = append(args, gameID)
		conditions = append(conditions, fmt.Sprintf("game_id = $%d", len(args)))
	}
	if riskMode := strings.TrimSpace(filter.RiskMode); riskMode != "" {
		args = append(args, riskMode)
		conditions = append(conditions, fmt.Sprintf("risk_mode = $%d", len(args)))
	}
	if claimStatus := strings.TrimSpace(filter.ClaimStatus); claimStatus != "" {
		args = append(args, claimStatus)
		conditions = append(conditions, fmt.Sprintf("claim_status = $%d", len(args)))
	}
	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func buildGameClaimFilter(filter model.GameMonitorListFilter) (string, []any) {
	conditions := make([]string, 0, 5)
	args := make([]any, 0, 5)

	if dayKey := strings.TrimSpace(filter.DayKey); dayKey != "" {
		args = append(args, dayKey)
		conditions = append(conditions, fmt.Sprintf("day_key = $%d", len(args)))
	}
	if userID := strings.TrimSpace(filter.UserID); userID != "" {
		args = append(args, "%"+userID+"%")
		conditions = append(conditions, fmt.Sprintf("user_id ILIKE $%d", len(args)))
	}
	if gameID := strings.TrimSpace(filter.GameID); gameID != "" {
		args = append(args, gameID)
		conditions = append(conditions, fmt.Sprintf("game_id = $%d", len(args)))
	}
	if riskMode := strings.TrimSpace(filter.RiskMode); riskMode != "" {
		args = append(args, riskMode)
		conditions = append(conditions, fmt.Sprintf("risk_mode = $%d", len(args)))
	}
	if claimStatus := strings.TrimSpace(filter.ClaimStatus); claimStatus != "" {
		args = append(args, claimStatus)
		conditions = append(conditions, fmt.Sprintf("claim_status = $%d", len(args)))
	}
	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func buildGameEventFilter(filter model.GameMonitorListFilter) (string, []any) {
	conditions := make([]string, 0, 5)
	args := make([]any, 0, 5)

	if dayKey := strings.TrimSpace(filter.DayKey); dayKey != "" {
		args = append(args, dayKey)
		conditions = append(conditions, fmt.Sprintf("day_key = $%d", len(args)))
	}
	if userID := strings.TrimSpace(filter.UserID); userID != "" {
		args = append(args, "%"+userID+"%")
		conditions = append(conditions, fmt.Sprintf("user_id ILIKE $%d", len(args)))
	}
	if gameID := strings.TrimSpace(filter.GameID); gameID != "" {
		args = append(args, gameID)
		conditions = append(conditions, fmt.Sprintf("game_id = $%d", len(args)))
	}
	if riskMode := strings.TrimSpace(filter.RiskMode); riskMode != "" {
		args = append(args, riskMode)
		conditions = append(conditions, fmt.Sprintf("risk_mode = $%d", len(args)))
	}
	if eventType := strings.TrimSpace(filter.EventType); eventType != "" {
		args = append(args, eventType)
		conditions = append(conditions, fmt.Sprintf("event_type = $%d", len(args)))
	}
	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}
