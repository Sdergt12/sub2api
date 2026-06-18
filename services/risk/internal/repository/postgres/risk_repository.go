package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sub2api-risk/internal/model"
)

var validSourceIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_\.]*$`)

type RiskRepository struct {
	db     *pgxpool.Pool
	signDB *pgxpool.Pool
}

func NewRiskRepository(db *pgxpool.Pool, signDB *pgxpool.Pool) *RiskRepository {
	return &RiskRepository{
		db:     db,
		signDB: signDB,
	}
}

func (r *RiskRepository) GetDashboardOverview(ctx context.Context) (model.DashboardOverview, error) {
	const query = `
SELECT
	(SELECT COUNT(*) FROM risk_user_snapshot WHERE risk_level = 'high' AND is_stale = FALSE) AS high_risk_users,
	(SELECT COUNT(*) FROM risk_user_snapshot WHERE risk_level = 'warning' AND is_stale = FALSE) AS warning_users,
	(SELECT COUNT(*) FROM risk_events WHERE occurred_at >= date_trunc('day', NOW())) AS events_today,
	(SELECT COUNT(*) FROM risk_events WHERE status = 'open') AS events_unprocessed,
	(SELECT COUNT(*) FROM risk_user_snapshot WHERE is_stale = TRUE) AS stale_users,
	(SELECT MAX(refreshed_at) FROM risk_user_snapshot) AS last_refresh_at,
	(SELECT MAX(window_start) FROM risk_user_snapshot) AS window_start,
	(SELECT MAX(window_end) FROM risk_user_snapshot) AS window_end
`

	var overview model.DashboardOverview
	var lastRefreshAt sql.NullTime
	var windowStart sql.NullTime
	var windowEnd sql.NullTime

	if err := r.db.QueryRow(ctx, query).Scan(
		&overview.HighRiskUsers,
		&overview.WarningUsers,
		&overview.EventsToday,
		&overview.EventsUnprocessed,
		&overview.StaleUsers,
		&lastRefreshAt,
		&windowStart,
		&windowEnd,
	); err != nil {
		return model.DashboardOverview{}, fmt.Errorf("query dashboard overview: %w", err)
	}

	if lastRefreshAt.Valid {
		overview.LastRefreshAt = &lastRefreshAt.Time
	}
	if windowStart.Valid {
		overview.WindowStart = &windowStart.Time
	}
	if windowEnd.Valid {
		overview.WindowEnd = &windowEnd.Time
	}

	return overview, nil
}

func (r *RiskRepository) GetOperationsOverview(ctx context.Context) (model.OperationsOverview, error) {
	signSummary, err := r.getOperationsSignSummary(ctx)
	if err != nil {
		return model.OperationsOverview{}, err
	}

	paymentSummary, err := r.getOperationsPaymentSummary(ctx)
	if err != nil {
		return model.OperationsOverview{}, err
	}

	conversionSummary, err := r.getOperationsConversionSummary(ctx)
	if err != nil {
		return model.OperationsOverview{}, err
	}

	usageSummary, err := r.getOperationsUsageSummary(ctx)
	if err != nil {
		return model.OperationsOverview{}, err
	}

	trends, err := r.getOperationsTrends(ctx, 7)
	if err != nil {
		return model.OperationsOverview{}, err
	}

	topPayers, err := r.getTopPayers(ctx, 7, 8)
	if err != nil {
		return model.OperationsOverview{}, err
	}

	topCost, err := r.getTopCostUsers(ctx, 24, 8)
	if err != nil {
		return model.OperationsOverview{}, err
	}

	return model.OperationsOverview{
		Sign:       signSummary,
		Payment:    paymentSummary,
		Conversion: conversionSummary,
		Usage:      usageSummary,
		Trends:     trends,
		TopPayers:  topPayers,
		TopCost:    topCost,
	}, nil
}

func (r *RiskRepository) getOperationsSignSummary(ctx context.Context) (model.OperationsSignSummary, error) {
	const query = `
SELECT
	COUNT(DISTINCT sub2api_user_id) FILTER (WHERE grant_status = 'success') AS today_success_users,
	COUNT(*) FILTER (WHERE grant_status = 'success') AS today_success_count,
	COALESCE(SUM(total_reward) FILTER (WHERE grant_status = 'success'), 0)::float8 AS today_total_reward,
	COALESCE(AVG(total_reward) FILTER (WHERE grant_status = 'success'), 0)::float8 AS today_avg_reward,
	COUNT(*) FILTER (WHERE grant_status = 'pending') AS today_pending_count,
	COUNT(*) FILTER (WHERE grant_status = 'failed') AS today_failed_count,
	MAX(created_at) FILTER (WHERE grant_status = 'success') AS last_checkin_at
FROM sign_checkins
WHERE sign_date = CURRENT_DATE
`

	var item model.OperationsSignSummary
	var lastCheckinAt sql.NullTime
	if err := r.signDB.QueryRow(ctx, query).Scan(
		&item.TodaySuccessUsers,
		&item.TodaySuccessCount,
		&item.TodayTotalReward,
		&item.TodayAvgReward,
		&item.TodayPendingCount,
		&item.TodayFailedCount,
		&lastCheckinAt,
	); err != nil {
		return model.OperationsSignSummary{}, fmt.Errorf("query operations sign summary: %w", err)
	}

	if lastCheckinAt.Valid {
		item.LastCheckinAt = &lastCheckinAt.Time
	}

	return item, nil
}

func (r *RiskRepository) getOperationsPaymentSummary(ctx context.Context) (model.OperationsPaymentSummary, error) {
	const query = `
WITH day_start AS (
	SELECT date_trunc('day', NOW()) AS started_at
)
SELECT
	COUNT(*) FILTER (WHERE o.created_at >= d.started_at) AS today_created_orders,
	COALESCE(SUM(o.amount) FILTER (WHERE o.created_at >= d.started_at), 0)::float8 AS today_created_amount,
	COUNT(*) FILTER (WHERE o.paid_at >= d.started_at) AS today_paid_orders,
	COALESCE(SUM(o.pay_amount) FILTER (WHERE o.paid_at >= d.started_at), 0)::float8 AS today_paid_amount,
	COUNT(DISTINCT o.user_id) FILTER (WHERE o.paid_at >= d.started_at) AS today_paid_users,
	COUNT(*) FILTER (
		WHERE o.created_at >= d.started_at
		  AND (
			o.failed_at IS NOT NULL OR UPPER(o.status) IN ('FAILED', 'CANCELED', 'CANCELLED', 'EXPIRED', 'CLOSED')
		  )
	) AS today_canceled_orders,
	COALESCE(AVG(o.pay_amount) FILTER (WHERE o.paid_at >= d.started_at), 0)::float8 AS avg_paid_order_amount
FROM payment_orders o
CROSS JOIN day_start d
`

	var item model.OperationsPaymentSummary
	if err := r.db.QueryRow(ctx, query).Scan(
		&item.TodayCreatedOrders,
		&item.TodayCreatedAmount,
		&item.TodayPaidOrders,
		&item.TodayPaidAmount,
		&item.TodayPaidUsers,
		&item.TodayCanceledOrders,
		&item.AvgPaidOrderAmount,
	); err != nil {
		return model.OperationsPaymentSummary{}, fmt.Errorf("query operations payment summary: %w", err)
	}

	return item, nil
}

func (r *RiskRepository) getOperationsConversionSummary(ctx context.Context) (model.OperationsConversionSummary, error) {
	const signedUsersQuery = `
SELECT COUNT(DISTINCT sub2api_user_id)
FROM sign_checkins
WHERE grant_status = 'success'
  AND created_at >= NOW() - INTERVAL '24 hours'
`
	const paidUsersQuery = `
SELECT COUNT(DISTINCT user_id)
FROM payment_orders
WHERE paid_at >= NOW() - INTERVAL '24 hours'
`

	var item model.OperationsConversionSummary
	if err := r.signDB.QueryRow(ctx, signedUsersQuery).Scan(&item.SignedUsers24h); err != nil {
		return model.OperationsConversionSummary{}, fmt.Errorf("query signed users 24h: %w", err)
	}
	if err := r.db.QueryRow(ctx, paidUsersQuery).Scan(&item.PaidUsers24h); err != nil {
		return model.OperationsConversionSummary{}, fmt.Errorf("query paid users 24h: %w", err)
	}

	// 签到库和主站库是两个独立数据库，不能直接做 SQL JOIN。
	// 这里在应用层按用户和时间窗口合并，避免为了统计面板破坏外挂服务的低耦合边界。
	signRows, err := r.queryRecentSignRows(ctx)
	if err != nil {
		return model.OperationsConversionSummary{}, err
	}
	payRows, err := r.queryRecentPaidRows(ctx)
	if err != nil {
		return model.OperationsConversionSummary{}, err
	}

	signByUser := make(map[int64][]time.Time, len(signRows))
	for _, row := range signRows {
		signByUser[row.UserID] = append(signByUser[row.UserID], row.OccurredAt)
	}

	convertedUsers := make(map[int64]struct{})
	for _, row := range payRows {
		signTimes, ok := signByUser[row.UserID]
		if !ok {
			continue
		}
		for _, signTime := range signTimes {
			if !row.OccurredAt.Before(signTime) && !row.OccurredAt.After(signTime.Add(24*time.Hour)) {
				convertedUsers[row.UserID] = struct{}{}
				break
			}
		}
	}
	item.SignedThenPaidUsers24h = int64(len(convertedUsers))

	if item.SignedUsers24h > 0 {
		item.SignedToPaidRate24h = float64(item.SignedThenPaidUsers24h) / float64(item.SignedUsers24h)
	}

	return item, nil
}

func (r *RiskRepository) getOperationsUsageSummary(ctx context.Context) (model.OperationsUsageSummary, error) {
	const query = `
SELECT
	COUNT(*) AS requests_24h,
	COUNT(DISTINCT user_id) AS active_users_24h,
	COALESCE(SUM(total_cost), 0)::float8 AS total_billed_24h,
	COALESCE(SUM(actual_cost), 0)::float8 AS total_actual_24h,
	COALESCE(AVG(duration_ms) FILTER (WHERE duration_ms IS NOT NULL), 0)::float8 AS avg_latency_ms,
	COALESCE(percentile_cont(0.9) WITHIN GROUP (ORDER BY duration_ms) FILTER (WHERE duration_ms IS NOT NULL), 0)::float8 AS p90_latency_ms,
	COALESCE(AVG(first_token_ms) FILTER (WHERE first_token_ms IS NOT NULL), 0)::float8 AS avg_first_token_ms,
	COALESCE(percentile_cont(0.9) WITHIN GROUP (ORDER BY first_token_ms) FILTER (WHERE first_token_ms IS NOT NULL), 0)::float8 AS p90_first_token_ms
FROM usage_logs
WHERE created_at >= NOW() - INTERVAL '24 hours'
`

	var item model.OperationsUsageSummary
	if err := r.db.QueryRow(ctx, query).Scan(
		&item.Requests24h,
		&item.ActiveUsers24h,
		&item.TotalBilled24h,
		&item.TotalActual24h,
		&item.AvgLatencyMs,
		&item.P90LatencyMs,
		&item.AvgFirstTokenMs,
		&item.P90FirstTokenMs,
	); err != nil {
		return model.OperationsUsageSummary{}, fmt.Errorf("query operations usage summary: %w", err)
	}

	return item, nil
}

func (r *RiskRepository) getOperationsTrends(ctx context.Context, days int) ([]model.OperationsTrendPoint, error) {
	signMap, err := r.querySignTrendMap(ctx, days)
	if err != nil {
		return nil, err
	}

	paymentMap, err := r.queryPaymentTrendMap(ctx, days)
	if err != nil {
		return nil, err
	}

	usageMap, err := r.queryUsageTrendMap(ctx, days)
	if err != nil {
		return nil, err
	}

	// 趋势图需要补齐没有数据的日期，否则前端会出现断点，影响运营判断。
	items := make([]model.OperationsTrendPoint, 0, days)
	for _, day := range buildRecentDateKeys(days) {
		item := model.OperationsTrendPoint{Date: day}
		if value, ok := signMap[day]; ok {
			item.SignUsers = value.SignUsers
			item.SignRewardTotal = value.SignRewardTotal
		}
		if value, ok := paymentMap[day]; ok {
			item.PaidOrders = value.PaidOrders
			item.PaidAmount = value.PaidAmount
		}
		if value, ok := usageMap[day]; ok {
			item.UsageRequests = value.UsageRequests
			item.UsageActiveUsers = value.UsageActiveUsers
		}
		items = append(items, item)
	}

	return items, nil
}

func (r *RiskRepository) getTopPayers(ctx context.Context, days int, limit int) ([]model.OperationsUserRankItem, error) {
	const query = `
SELECT
	user_id,
	COALESCE(NULLIF(MAX(user_name), ''), NULLIF(MAX(user_email), ''), user_id::text) AS user_label,
	COUNT(*) AS paid_orders,
	COALESCE(SUM(pay_amount), 0)::float8 AS paid_amount,
	MAX(paid_at) AS last_active_at
FROM payment_orders
WHERE paid_at >= NOW() - $1::int * INTERVAL '1 day'
GROUP BY user_id
ORDER BY paid_amount DESC, paid_orders DESC, user_id DESC
LIMIT $2
`

	rows, err := r.db.Query(ctx, query, days, limit)
	if err != nil {
		return nil, fmt.Errorf("query top payers: %w", err)
	}
	defer rows.Close()

	items := make([]model.OperationsUserRankItem, 0, limit)
	for rows.Next() {
		var item model.OperationsUserRankItem
		var lastActiveAt sql.NullTime
		if err := rows.Scan(&item.UserID, &item.UserLabel, &item.PaidOrders, &item.MetricValue, &lastActiveAt); err != nil {
			return nil, fmt.Errorf("scan top payer row: %w", err)
		}
		if lastActiveAt.Valid {
			item.LastActiveAt = &lastActiveAt.Time
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate top payer rows: %w", err)
	}

	return items, nil
}

func (r *RiskRepository) getTopCostUsers(ctx context.Context, hours int, limit int) ([]model.OperationsUserRankItem, error) {
	const query = `
SELECT
	u.user_id,
	COALESCE(NULLIF(MAX(users.email), ''), u.user_id::text) AS user_label,
	COALESCE(SUM(u.actual_cost), 0)::float8 AS actual_cost_24h,
	COUNT(*) AS request_count,
	MAX(u.created_at) AS last_active_at
FROM usage_logs u
LEFT JOIN users ON users.id = u.user_id
WHERE u.created_at >= NOW() - $1::int * INTERVAL '1 hour'
GROUP BY u.user_id
ORDER BY actual_cost_24h DESC, request_count DESC, u.user_id DESC
LIMIT $2
`

	rows, err := r.db.Query(ctx, query, hours, limit)
	if err != nil {
		return nil, fmt.Errorf("query top cost users: %w", err)
	}
	defer rows.Close()

	items := make([]model.OperationsUserRankItem, 0, limit)
	for rows.Next() {
		var item model.OperationsUserRankItem
		var lastActiveAt sql.NullTime
		if err := rows.Scan(&item.UserID, &item.UserLabel, &item.MetricValue, &item.RequestCount, &lastActiveAt); err != nil {
			return nil, fmt.Errorf("scan top cost user row: %w", err)
		}
		if lastActiveAt.Valid {
			item.LastActiveAt = &lastActiveAt.Time
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate top cost user rows: %w", err)
	}

	return items, nil
}

func (r *RiskRepository) ListUserSnapshots(ctx context.Context, filter model.UserListFilter) ([]model.RiskUserSnapshot, int64, error) {
	whereSQL, args := buildUserFilter(filter)

	countQuery := `SELECT COUNT(*) FROM risk_user_snapshot` + whereSQL

	var total int64
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count risk users: %w", err)
	}

	dataQuery := `
SELECT
	id,
	user_id,
	risk_score,
	risk_level,
	request_count_1h,
	request_count_24h,
	cost_1h::float8,
	cost_24h::float8,
	unique_ip_24h,
	top_models_json,
	risk_tags_json,
	last_event_at,
	window_start,
	window_end,
	refreshed_at,
	is_stale,
	created_at,
	updated_at
FROM risk_user_snapshot` + whereSQL + fmt.Sprintf(`
ORDER BY risk_score DESC, last_event_at DESC NULLS LAST, id DESC
LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)

	rows, err := r.db.Query(ctx, dataQuery, append(args, filter.PageSize, offset(filter.Page, filter.PageSize))...)
	if err != nil {
		return nil, 0, fmt.Errorf("query risk users: %w", err)
	}
	defer rows.Close()

	items := make([]model.RiskUserSnapshot, 0, filter.PageSize)

	for rows.Next() {
		var item model.RiskUserSnapshot
		var topModelsJSON []byte
		var riskTagsJSON []byte
		var lastEventAt sql.NullTime
		var windowStart sql.NullTime
		var windowEnd sql.NullTime
		var refreshedAt sql.NullTime

		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.RiskScore,
			&item.RiskLevel,
			&item.RequestCount1h,
			&item.RequestCount24h,
			&item.Cost1h,
			&item.Cost24h,
			&item.UniqueIP24h,
			&topModelsJSON,
			&riskTagsJSON,
			&lastEventAt,
			&windowStart,
			&windowEnd,
			&refreshedAt,
			&item.IsStale,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan risk user row: %w", err)
		}

		item.TopModels = decodeStringSlice(topModelsJSON)
		item.RiskTags = decodeStringSlice(riskTagsJSON)

		if lastEventAt.Valid {
			item.LastEventAt = &lastEventAt.Time
		}
		if windowStart.Valid {
			item.WindowStart = &windowStart.Time
		}
		if windowEnd.Valid {
			item.WindowEnd = &windowEnd.Time
		}
		if refreshedAt.Valid {
			item.RefreshedAt = &refreshedAt.Time
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate risk user rows: %w", err)
	}

	return items, total, nil
}

func (r *RiskRepository) ListRiskEvents(ctx context.Context, filter model.EventListFilter) ([]model.RiskEvent, int64, error) {
	whereSQL, args := buildEventFilter(filter)

	countQuery := `SELECT COUNT(*) FROM risk_events` + whereSQL

	var total int64
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count risk events: %w", err)
	}

	dataQuery := `
SELECT
	id,
	user_id,
	api_key_id,
	event_fingerprint,
	event_type,
	status,
	severity,
	score_delta,
	title,
	detail_json,
	first_seen_at,
	last_seen_at,
	hit_count,
	occurred_at,
	created_at
FROM risk_events` + whereSQL + fmt.Sprintf(`
ORDER BY occurred_at DESC, id DESC
LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)

	rows, err := r.db.Query(ctx, dataQuery, append(args, filter.PageSize, offset(filter.Page, filter.PageSize))...)
	if err != nil {
		return nil, 0, fmt.Errorf("query risk events: %w", err)
	}
	defer rows.Close()

	items := make([]model.RiskEvent, 0, filter.PageSize)

	for rows.Next() {
		var item model.RiskEvent
		var apiKeyID sql.NullString
		var detailJSON []byte

		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&apiKeyID,
			&item.EventFingerprint,
			&item.EventType,
			&item.Status,
			&item.Severity,
			&item.ScoreDelta,
			&item.Title,
			&detailJSON,
			&item.FirstSeenAt,
			&item.LastSeenAt,
			&item.HitCount,
			&item.OccurredAt,
			&item.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan risk event row: %w", err)
		}

		if apiKeyID.Valid {
			item.APIKeyID = apiKeyID.String
		}

		item.Detail = decodeDetailMap(detailJSON)
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate risk event rows: %w", err)
	}

	return items, total, nil
}

func (r *RiskRepository) ListRiskAggregates(ctx context.Context, sourceTable string) ([]model.UserRiskAggregate, error) {
	tableName, err := normalizeSourceTable(sourceTable)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
WITH base AS (
	SELECT
		user_id,
		COALESCE(NULLIF(api_key_id, ''), '') AS api_key_id,
		COALESCE(model_name, '') AS model_name,
		COALESCE(client_ip, '') AS client_ip,
		COALESCE(cost, 0)::float8 AS cost,
		occurred_at
	FROM %s
	WHERE occurred_at >= NOW() - INTERVAL '24 hours'
),
model_ranked AS (
	SELECT
		user_id,
		model_name,
		COUNT(*) AS model_count,
		ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY COUNT(*) DESC, model_name ASC) AS rn
	FROM base
	WHERE model_name <> ''
	GROUP BY user_id, model_name
),
model_top AS (
	SELECT
		user_id,
		COALESCE(jsonb_agg(model_name ORDER BY model_count DESC, model_name ASC), '[]'::jsonb) AS top_models_json
	FROM model_ranked
	WHERE rn <= 3
	GROUP BY user_id
)
SELECT
	base.user_id,
	MAX(base.api_key_id) FILTER (WHERE base.api_key_id <> '') AS api_key_id,
	COUNT(*) FILTER (WHERE base.occurred_at >= NOW() - INTERVAL '1 hour') AS request_count_1h,
	COUNT(*) AS request_count_24h,
	COALESCE(SUM(base.cost) FILTER (WHERE base.occurred_at >= NOW() - INTERVAL '1 hour'), 0)::float8 AS cost_1h,
	COALESCE(SUM(base.cost), 0)::float8 AS cost_24h,
	COUNT(DISTINCT NULLIF(base.client_ip, ''))::int AS unique_ip_24h,
	COALESCE(model_top.top_models_json, '[]'::jsonb) AS top_models_json,
	MAX(base.occurred_at) AS window_end
FROM base
LEFT JOIN model_top ON model_top.user_id = base.user_id
GROUP BY base.user_id, model_top.top_models_json
ORDER BY base.user_id ASC
`, tableName)

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query risk aggregates: %w", err)
	}
	defer rows.Close()

	items := make([]model.UserRiskAggregate, 0, 64)

	for rows.Next() {
		var item model.UserRiskAggregate
		var apiKeyID sql.NullString
		var topModelsJSON []byte

		if err := rows.Scan(
			&item.UserID,
			&apiKeyID,
			&item.RequestCount1h,
			&item.RequestCount24h,
			&item.Cost1h,
			&item.Cost24h,
			&item.UniqueIP24h,
			&topModelsJSON,
			&item.WindowEnd,
		); err != nil {
			return nil, fmt.Errorf("scan risk aggregate row: %w", err)
		}

		if apiKeyID.Valid {
			item.APIKeyID = apiKeyID.String
		}

		item.TopModels = decodeStringSlice(topModelsJSON)
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate risk aggregate rows: %w", err)
	}

	return items, nil
}

func (r *RiskRepository) ReplaceRiskSnapshot(ctx context.Context, snapshot model.RiskSnapshotUpsertInput, events []model.RiskEventCreateInput, ruleHits [][]model.RiskRuleHitCreateInput) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin risk refresh transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := upsertSnapshot(ctx, tx, snapshot); err != nil {
		return err
	}

	currentFingerprints := make([]string, 0, len(events))
	for idx, event := range events {
		currentFingerprints = append(currentFingerprints, event.EventFingerprint)
		eventID, err := upsertRiskEvent(ctx, tx, event)
		if err != nil {
			return err
		}

		if idx < len(ruleHits) {
			if err := upsertRuleHits(ctx, tx, eventID, ruleHits[idx]); err != nil {
				return err
			}
		}
	}

	if err := resolveInactiveEvents(ctx, tx, snapshot.UserID, currentFingerprints, snapshot.WindowEnd); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit risk refresh transaction: %w", err)
	}

	return nil
}

func (r *RiskRepository) MarkStaleSnapshots(ctx context.Context, activeUserIDs []string, snapshotTime time.Time, windowStart time.Time, windowEnd time.Time) (int64, error) {
	const query = `
UPDATE risk_user_snapshot
SET
	risk_score = 0,
	risk_level = 'normal',
	request_count_1h = 0,
	request_count_24h = 0,
	cost_1h = 0,
	cost_24h = 0,
	unique_ip_24h = 0,
	top_models_json = '[]'::jsonb,
	risk_tags_json = '[]'::jsonb,
	is_stale = TRUE,
	window_start = $2,
	window_end = $3,
	refreshed_at = $4,
	updated_at = NOW()
WHERE
	(COALESCE(array_length($1::varchar[], 1), 0) = 0 OR NOT (user_id = ANY($1::varchar[])))
	AND is_stale = FALSE
`

	commandTag, err := r.db.Exec(ctx, query, activeUserIDs, windowStart, windowEnd, snapshotTime)
	if err != nil {
		return 0, fmt.Errorf("mark stale snapshots: %w", err)
	}

	return commandTag.RowsAffected(), nil
}

func upsertSnapshot(ctx context.Context, tx pgx.Tx, snapshot model.RiskSnapshotUpsertInput) error {
	topModelsJSON, err := json.Marshal(snapshot.TopModels)
	if err != nil {
		return fmt.Errorf("marshal top models: %w", err)
	}

	riskTagsJSON, err := json.Marshal(snapshot.RiskTags)
	if err != nil {
		return fmt.Errorf("marshal risk tags: %w", err)
	}

	const query = `
INSERT INTO risk_user_snapshot (
	user_id,
	risk_score,
	risk_level,
	request_count_1h,
	request_count_24h,
	cost_1h,
	cost_24h,
	unique_ip_24h,
	top_models_json,
	risk_tags_json,
	last_event_at,
	window_start,
	window_end,
	refreshed_at,
	is_stale
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10::jsonb, $11, $12, $13, $14, $15
)
ON CONFLICT (user_id) DO UPDATE SET
	risk_score = EXCLUDED.risk_score,
	risk_level = EXCLUDED.risk_level,
	request_count_1h = EXCLUDED.request_count_1h,
	request_count_24h = EXCLUDED.request_count_24h,
	cost_1h = EXCLUDED.cost_1h,
	cost_24h = EXCLUDED.cost_24h,
	unique_ip_24h = EXCLUDED.unique_ip_24h,
	top_models_json = EXCLUDED.top_models_json,
	risk_tags_json = EXCLUDED.risk_tags_json,
	last_event_at = EXCLUDED.last_event_at,
	window_start = EXCLUDED.window_start,
	window_end = EXCLUDED.window_end,
	refreshed_at = EXCLUDED.refreshed_at,
	is_stale = EXCLUDED.is_stale,
	updated_at = NOW()
`

	if _, err := tx.Exec(
		ctx,
		query,
		snapshot.UserID,
		snapshot.RiskScore,
		snapshot.RiskLevel,
		snapshot.RequestCount1h,
		snapshot.RequestCount24h,
		snapshot.Cost1h,
		snapshot.Cost24h,
		snapshot.UniqueIP24h,
		string(topModelsJSON),
		string(riskTagsJSON),
		snapshot.LastEventAt,
		snapshot.WindowStart,
		snapshot.WindowEnd,
		snapshot.RefreshedAt,
		snapshot.IsStale,
	); err != nil {
		return fmt.Errorf("upsert risk snapshot: %w", err)
	}

	return nil
}

func upsertRiskEvent(ctx context.Context, tx pgx.Tx, event model.RiskEventCreateInput) (int64, error) {
	detailJSON, err := json.Marshal(event.Detail)
	if err != nil {
		return 0, fmt.Errorf("marshal risk event detail: %w", err)
	}

	const query = `
INSERT INTO risk_events (
	user_id,
	api_key_id,
	event_fingerprint,
	event_type,
	status,
	severity,
	score_delta,
	title,
	detail_json,
	first_seen_at,
	last_seen_at,
	hit_count,
	occurred_at
) VALUES (
	$1, NULLIF($2, ''), $3, $4, $5, $6, $7, $8, $9::jsonb, $10, $11, $12, $13
)
ON CONFLICT (event_fingerprint) DO UPDATE SET
	api_key_id = EXCLUDED.api_key_id,
	status = 'open',
	severity = EXCLUDED.severity,
	score_delta = EXCLUDED.score_delta,
	title = EXCLUDED.title,
	detail_json = EXCLUDED.detail_json,
	last_seen_at = EXCLUDED.last_seen_at,
	hit_count = risk_events.hit_count + 1,
	occurred_at = EXCLUDED.occurred_at
RETURNING id
`

	var eventID int64
	if err := tx.QueryRow(
		ctx,
		query,
		event.UserID,
		event.APIKeyID,
		event.EventFingerprint,
		event.EventType,
		event.Status,
		event.Severity,
		event.ScoreDelta,
		event.Title,
		string(detailJSON),
		event.FirstSeenAt,
		event.LastSeenAt,
		event.HitCount,
		event.OccurredAt,
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("upsert risk event: %w", err)
	}

	return eventID, nil
}

func upsertRuleHits(ctx context.Context, tx pgx.Tx, eventID int64, hits []model.RiskRuleHitCreateInput) error {
	const query = `
INSERT INTO risk_rule_hits (
	event_id,
	rule_code,
	metric_value,
	threshold_value,
	extra_json
) VALUES (
	$1, $2, $3, $4, $5::jsonb
)
ON CONFLICT (event_id, rule_code) DO UPDATE SET
	metric_value = EXCLUDED.metric_value,
	threshold_value = EXCLUDED.threshold_value,
	extra_json = EXCLUDED.extra_json,
	created_at = NOW()
`

	for _, hit := range hits {
		extraJSON, err := json.Marshal(hit.Extra)
		if err != nil {
			return fmt.Errorf("marshal rule hit extra: %w", err)
		}

		if _, err := tx.Exec(
			ctx,
			query,
			eventID,
			hit.RuleCode,
			hit.MetricValue,
			hit.ThresholdValue,
			string(extraJSON),
		); err != nil {
			return fmt.Errorf("upsert risk rule hit: %w", err)
		}
	}

	return nil
}

func resolveInactiveEvents(ctx context.Context, tx pgx.Tx, userID string, activeFingerprints []string, resolvedAt time.Time) error {
	const query = `
UPDATE risk_events
SET
	status = 'resolved',
	last_seen_at = GREATEST(last_seen_at, $3)
WHERE
	user_id = $1
	AND status IN ('open', 'acknowledged')
	AND (
		COALESCE(array_length($2::varchar[], 1), 0) = 0
		OR NOT (event_fingerprint = ANY($2::varchar[]))
	)
`

	if _, err := tx.Exec(ctx, query, userID, activeFingerprints, resolvedAt); err != nil {
		return fmt.Errorf("resolve inactive events: %w", err)
	}

	return nil
}

type signTrendRow struct {
	SignUsers       int64
	SignRewardTotal float64
}

type paymentTrendRow struct {
	PaidOrders int64
	PaidAmount float64
}

type usageTrendRow struct {
	UsageRequests    int64
	UsageActiveUsers int64
}

type operationsUserTimeRow struct {
	UserID     int64
	OccurredAt time.Time
}

func (r *RiskRepository) querySignTrendMap(ctx context.Context, days int) (map[string]signTrendRow, error) {
	const query = `
SELECT
	TO_CHAR(sign_date, 'YYYY-MM-DD') AS date_key,
	COUNT(DISTINCT sub2api_user_id) FILTER (WHERE grant_status = 'success') AS sign_users,
	COALESCE(SUM(total_reward) FILTER (WHERE grant_status = 'success'), 0)::float8 AS sign_reward_total
FROM sign_checkins
WHERE sign_date >= CURRENT_DATE - ($1::int - 1)
GROUP BY sign_date
ORDER BY sign_date ASC
`

	rows, err := r.signDB.Query(ctx, query, days)
	if err != nil {
		return nil, fmt.Errorf("query sign trend map: %w", err)
	}
	defer rows.Close()

	items := make(map[string]signTrendRow, days)
	for rows.Next() {
		var key string
		var item signTrendRow
		if err := rows.Scan(&key, &item.SignUsers, &item.SignRewardTotal); err != nil {
			return nil, fmt.Errorf("scan sign trend row: %w", err)
		}
		items[key] = item
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sign trend rows: %w", err)
	}

	return items, nil
}

func (r *RiskRepository) queryPaymentTrendMap(ctx context.Context, days int) (map[string]paymentTrendRow, error) {
	const query = `
SELECT
	TO_CHAR(paid_at AT TIME ZONE 'UTC' AT TIME ZONE 'Asia/Shanghai', 'YYYY-MM-DD') AS date_key,
	COUNT(*) AS paid_orders,
	COALESCE(SUM(pay_amount), 0)::float8 AS paid_amount
FROM payment_orders
WHERE paid_at >= date_trunc('day', NOW()) - ($1::int - 1) * INTERVAL '1 day'
GROUP BY date_key
ORDER BY date_key ASC
`

	rows, err := r.db.Query(ctx, query, days)
	if err != nil {
		return nil, fmt.Errorf("query payment trend map: %w", err)
	}
	defer rows.Close()

	items := make(map[string]paymentTrendRow, days)
	for rows.Next() {
		var key string
		var item paymentTrendRow
		if err := rows.Scan(&key, &item.PaidOrders, &item.PaidAmount); err != nil {
			return nil, fmt.Errorf("scan payment trend row: %w", err)
		}
		items[key] = item
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate payment trend rows: %w", err)
	}

	return items, nil
}

func (r *RiskRepository) queryUsageTrendMap(ctx context.Context, days int) (map[string]usageTrendRow, error) {
	const query = `
SELECT
	TO_CHAR(created_at AT TIME ZONE 'UTC' AT TIME ZONE 'Asia/Shanghai', 'YYYY-MM-DD') AS date_key,
	COUNT(*) AS usage_requests,
	COUNT(DISTINCT user_id) AS usage_active_users
FROM usage_logs
WHERE created_at >= date_trunc('day', NOW()) - ($1::int - 1) * INTERVAL '1 day'
GROUP BY date_key
ORDER BY date_key ASC
`

	rows, err := r.db.Query(ctx, query, days)
	if err != nil {
		return nil, fmt.Errorf("query usage trend map: %w", err)
	}
	defer rows.Close()

	items := make(map[string]usageTrendRow, days)
	for rows.Next() {
		var key string
		var item usageTrendRow
		if err := rows.Scan(&key, &item.UsageRequests, &item.UsageActiveUsers); err != nil {
			return nil, fmt.Errorf("scan usage trend row: %w", err)
		}
		items[key] = item
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate usage trend rows: %w", err)
	}

	return items, nil
}

func (r *RiskRepository) queryRecentSignRows(ctx context.Context) ([]operationsUserTimeRow, error) {
	const query = `
SELECT sub2api_user_id, created_at
FROM sign_checkins
WHERE grant_status = 'success'
  AND created_at >= NOW() - INTERVAL '24 hours'
`

	rows, err := r.signDB.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query recent sign rows: %w", err)
	}
	defer rows.Close()

	items := make([]operationsUserTimeRow, 0, 64)
	for rows.Next() {
		var item operationsUserTimeRow
		if err := rows.Scan(&item.UserID, &item.OccurredAt); err != nil {
			return nil, fmt.Errorf("scan recent sign row: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent sign rows: %w", err)
	}

	return items, nil
}

func (r *RiskRepository) queryRecentPaidRows(ctx context.Context) ([]operationsUserTimeRow, error) {
	const query = `
SELECT user_id, paid_at
FROM payment_orders
WHERE paid_at >= NOW() - INTERVAL '24 hours'
`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query recent paid rows: %w", err)
	}
	defer rows.Close()

	items := make([]operationsUserTimeRow, 0, 64)
	for rows.Next() {
		var item operationsUserTimeRow
		if err := rows.Scan(&item.UserID, &item.OccurredAt); err != nil {
			return nil, fmt.Errorf("scan recent paid row: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent paid rows: %w", err)
	}

	return items, nil
}

func buildRecentDateKeys(days int) []string {
	if days <= 0 {
		return []string{}
	}

	keys := make([]string, 0, days)
	now := nowInShanghai()
	for offset := days - 1; offset >= 0; offset-- {
		keys = append(keys, now.AddDate(0, 0, -offset).Format("2006-01-02"))
	}

	return keys
}

func nowInShanghai() time.Time {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.Now()
	}
	return time.Now().In(location)
}

func normalizeSourceTable(sourceTable string) (string, error) {
	tableName := strings.TrimSpace(sourceTable)
	if tableName == "" {
		return "", fmt.Errorf("risk source table is not configured")
	}

	if !validSourceIdentifier.MatchString(tableName) {
		return "", fmt.Errorf("invalid risk source table name")
	}

	return tableName, nil
}

func buildUserFilter(filter model.UserListFilter) (string, []any) {
	conditions := make([]string, 0, 2)
	args := make([]any, 0, 2)

	if userID := strings.TrimSpace(filter.UserID); userID != "" {
		args = append(args, "%"+userID+"%")
		conditions = append(conditions, fmt.Sprintf("user_id ILIKE $%d", len(args)))
	}

	if riskLevel := strings.TrimSpace(filter.RiskLevel); riskLevel != "" {
		args = append(args, riskLevel)
		conditions = append(conditions, fmt.Sprintf("risk_level = $%d", len(args)))
	}

	if !filter.IncludeStale {
		conditions = append(conditions, "is_stale = FALSE")
	}

	if len(conditions) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

func buildEventFilter(filter model.EventListFilter) (string, []any) {
	conditions := make([]string, 0, 3)
	args := make([]any, 0, 3)

	if userID := strings.TrimSpace(filter.UserID); userID != "" {
		args = append(args, "%"+userID+"%")
		conditions = append(conditions, fmt.Sprintf("user_id ILIKE $%d", len(args)))
	}

	if severity := strings.TrimSpace(filter.Severity); severity != "" {
		args = append(args, severity)
		conditions = append(conditions, fmt.Sprintf("severity = $%d", len(args)))
	}

	if eventType := strings.TrimSpace(filter.EventType); eventType != "" {
		args = append(args, eventType)
		conditions = append(conditions, fmt.Sprintf("event_type = $%d", len(args)))
	}

	if status := strings.TrimSpace(filter.Status); status != "" {
		args = append(args, status)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}

	if len(conditions) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

func offset(page int, pageSize int) int {
	return (page - 1) * pageSize
}

func decodeStringSlice(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}

	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return []string{}
	}

	return values
}

func decodeDetailMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}

	var values map[string]any
	if err := json.Unmarshal(raw, &values); err != nil {
		return map[string]any{}
	}

	return values
}
