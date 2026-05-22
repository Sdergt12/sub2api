package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type tokenRiskRepository struct {
	db *sql.DB
}

func NewTokenRiskRepository(db *sql.DB) service.TokenRiskRepository {
	return &tokenRiskRepository{db: db}
}

func (r *tokenRiskRepository) UpsertTokenRiskEvent(ctx context.Context, event *service.TokenRiskEvent) (*service.TokenRiskEvent, error) {
	if r == nil || r.db == nil || event == nil {
		return nil, fmt.Errorf("nil token risk repository")
	}
	now := time.Now().UTC()
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}
	if event.UpdatedAt.IsZero() {
		event.UpdatedAt = now
	}
	if event.LastSeenAt.IsZero() {
		event.LastSeenAt = event.CreatedAt
	}
	if event.Status == "" {
		event.Status = service.TokenRiskStatusOpen
	}
	categories, _ := json.Marshal(event.RiskCategories)
	rules, _ := json.Marshal(event.MatchedRules)
	actions, _ := json.Marshal(event.RecommendedActions)

	query := `
INSERT INTO token_risk_events (
  created_at, updated_at, last_seen_at, source_log_id, user_id, api_key_id,
  token_type, token_hash, token_prefix, token_suffix, api_key_summary,
  client_ip, user_agent, method, path, status_code, result, failure_reason,
  risk_score, risk_level, risk_categories, matched_rules, recommended_actions,
  explanation, status, false_positive
) VALUES (
  $1, $2, $3, $4, $5, $6,
  $7, $8, $9, $10, $11,
  $12, $13, $14, $15, $16, $17, $18,
  $19, $20, $21::jsonb, $22::jsonb, $23::jsonb,
  $24, $25, $26
)
ON CONFLICT (source_log_id) DO UPDATE SET
  updated_at = NOW(),
  last_seen_at = GREATEST(token_risk_events.last_seen_at, EXCLUDED.last_seen_at),
  risk_score = GREATEST(token_risk_events.risk_score, EXCLUDED.risk_score),
  risk_level = EXCLUDED.risk_level,
  risk_categories = EXCLUDED.risk_categories,
  matched_rules = EXCLUDED.matched_rules,
  recommended_actions = EXCLUDED.recommended_actions,
  explanation = EXCLUDED.explanation
RETURNING id, created_at, updated_at, last_seen_at`
	var sourceLogID any
	if event.SourceLogID != nil && *event.SourceLogID > 0 {
		sourceLogID = *event.SourceLogID
	}
	err := r.db.QueryRowContext(ctx, query,
		event.CreatedAt, event.UpdatedAt, event.LastSeenAt, sourceLogID,
		nullInt64(event.UserID), nullInt64(event.APIKeyID),
		event.TokenType, event.TokenHash, event.TokenPrefix, event.TokenSuffix, event.APIKeySummary,
		event.ClientIP, event.UserAgent, event.Method, event.Path, event.StatusCode, event.Result, event.FailureReason,
		event.RiskScore, event.RiskLevel, string(categories), string(rules), string(actions),
		event.Explanation, event.Status, event.FalsePositive,
	).Scan(&event.ID, &event.CreatedAt, &event.UpdatedAt, &event.LastSeenAt)
	if err != nil {
		return nil, err
	}
	return event, nil
}

func (r *tokenRiskRepository) ListTokenRiskEvents(ctx context.Context, filter service.TokenRiskEventFilter) ([]*service.TokenRiskEvent, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("nil token risk repository")
	}
	page, pageSize := normalizeTokenRiskPage(filter.Page, filter.PageSize)
	where, args := buildTokenRiskWhere(filter)

	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM token_risk_events e "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)
	query := `
SELECT
  e.id, e.created_at, e.updated_at, e.last_seen_at, e.source_log_id,
  e.user_id, e.api_key_id, e.token_type, e.token_hash, e.token_prefix, e.token_suffix,
  e.api_key_summary, e.client_ip, e.user_agent, e.method, e.path, e.status_code,
  e.result, e.failure_reason, e.risk_score, e.risk_level,
  e.risk_categories::text, e.matched_rules::text, e.recommended_actions::text,
  e.explanation, e.status, e.false_positive, e.handled_by_user_id, e.handled_at,
  COALESCE((SELECT COUNT(*) FROM token_risk_events w WHERE token_risk_same_subject(w, e) AND w.created_at >= NOW() - INTERVAL '5 minutes'), 0),
  COALESCE((SELECT COUNT(*) FROM token_risk_events w WHERE token_risk_same_subject(w, e) AND w.created_at >= NOW() - INTERVAL '1 hour'), 0),
  COALESCE((SELECT COUNT(*) FROM token_risk_events w WHERE token_risk_same_subject(w, e) AND w.created_at >= NOW() - INTERVAL '24 hours'), 0),
  COALESCE((SELECT COUNT(DISTINCT NULLIF(w.client_ip, '')) FROM token_risk_events w WHERE token_risk_same_subject(w, e) AND w.created_at >= NOW() - INTERVAL '24 hours'), 0)
FROM token_risk_events e
` + where + `
ORDER BY e.created_at DESC, e.id DESC
LIMIT $` + itoa(len(args)-1) + ` OFFSET $` + itoa(len(args))
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]*service.TokenRiskEvent, 0, pageSize)
	for rows.Next() {
		item, err := scanTokenRiskEvent(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *tokenRiskRepository) GetTokenRiskEvent(ctx context.Context, id int64) (*service.TokenRiskEvent, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil token risk repository")
	}
	query := `
SELECT
  e.id, e.created_at, e.updated_at, e.last_seen_at, e.source_log_id,
  e.user_id, e.api_key_id, e.token_type, e.token_hash, e.token_prefix, e.token_suffix,
  e.api_key_summary, e.client_ip, e.user_agent, e.method, e.path, e.status_code,
  e.result, e.failure_reason, e.risk_score, e.risk_level,
  e.risk_categories::text, e.matched_rules::text, e.recommended_actions::text,
  e.explanation, e.status, e.false_positive, e.handled_by_user_id, e.handled_at,
  COALESCE((SELECT COUNT(*) FROM token_risk_events w WHERE token_risk_same_subject(w, e) AND w.created_at >= NOW() - INTERVAL '5 minutes'), 0),
  COALESCE((SELECT COUNT(*) FROM token_risk_events w WHERE token_risk_same_subject(w, e) AND w.created_at >= NOW() - INTERVAL '1 hour'), 0),
  COALESCE((SELECT COUNT(*) FROM token_risk_events w WHERE token_risk_same_subject(w, e) AND w.created_at >= NOW() - INTERVAL '24 hours'), 0),
  COALESCE((SELECT COUNT(DISTINCT NULLIF(w.client_ip, '')) FROM token_risk_events w WHERE token_risk_same_subject(w, e) AND w.created_at >= NOW() - INTERVAL '24 hours'), 0)
FROM token_risk_events e
WHERE e.id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanTokenRiskEvent(row)
}

func (r *tokenRiskRepository) UpdateTokenRiskEventStatus(ctx context.Context, id int64, status string, falsePositive bool, actorUserID int64) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil token risk repository")
	}
	_, err := r.db.ExecContext(ctx, `
UPDATE token_risk_events
SET status = $2, false_positive = $3, handled_by_user_id = $4, handled_at = NOW(), updated_at = NOW()
WHERE id = $1`, id, status, falsePositive, actorUserID)
	return err
}

func (r *tokenRiskRepository) CreateTokenRiskAction(ctx context.Context, action *service.TokenRiskAction) (*service.TokenRiskAction, error) {
	if r == nil || r.db == nil || action == nil {
		return nil, fmt.Errorf("nil token risk repository")
	}
	metadata, _ := json.Marshal(action.Metadata)
	err := r.db.QueryRowContext(ctx, `
INSERT INTO token_risk_actions (event_id, actor_user_id, action, note, result, metadata)
VALUES ($1, $2, $3, $4, $5, $6::jsonb)
RETURNING id, created_at`,
		action.EventID, action.ActorUserID, action.Action, action.Note, action.Result, string(metadata),
	).Scan(&action.ID, &action.CreatedAt)
	if err != nil {
		return nil, err
	}
	return action, nil
}

func (r *tokenRiskRepository) ListTokenRiskActions(ctx context.Context, eventID int64) ([]*service.TokenRiskAction, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil token risk repository")
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT id, created_at, event_id, actor_user_id, action, note, result, metadata::text
FROM token_risk_actions
WHERE event_id = $1
ORDER BY created_at DESC, id DESC`, eventID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := []*service.TokenRiskAction{}
	for rows.Next() {
		item := &service.TokenRiskAction{}
		var raw string
		if err := rows.Scan(&item.ID, &item.CreatedAt, &item.EventID, &item.ActorUserID, &item.Action, &item.Note, &item.Result, &raw); err != nil {
			return nil, err
		}
		item.Metadata = map[string]any{}
		_ = json.Unmarshal([]byte(raw), &item.Metadata)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *tokenRiskRepository) UpsertTokenRiskWatchlist(ctx context.Context, item *service.TokenRiskWatchlistItem) (*service.TokenRiskWatchlistItem, error) {
	if r == nil || r.db == nil || item == nil {
		return nil, fmt.Errorf("nil token risk repository")
	}
	err := r.db.QueryRowContext(ctx, `
INSERT INTO token_risk_watchlist (subject_type, subject_value, reason, actor_user_id, active)
VALUES ($1, $2, $3, $4, TRUE)
ON CONFLICT (subject_type, subject_value) DO UPDATE SET
  updated_at = NOW(), reason = EXCLUDED.reason, actor_user_id = EXCLUDED.actor_user_id, active = TRUE
RETURNING id, created_at, updated_at, active`,
		item.SubjectType, item.SubjectValue, item.Reason, item.ActorUserID,
	).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt, &item.Active)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *tokenRiskRepository) DeactivateTokenRiskWatchlist(ctx context.Context, id int64, actorUserID int64) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil token risk repository")
	}
	_, err := r.db.ExecContext(ctx, `UPDATE token_risk_watchlist SET active = FALSE, updated_at = NOW(), actor_user_id = $2 WHERE id = $1`, id, actorUserID)
	return err
}

func (r *tokenRiskRepository) ListTokenRiskWatchlist(ctx context.Context, activeOnly bool) ([]*service.TokenRiskWatchlistItem, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil token risk repository")
	}
	where := ""
	if activeOnly {
		where = "WHERE active = TRUE"
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT id, created_at, updated_at, subject_type, subject_value, reason, actor_user_id, active
FROM token_risk_watchlist `+where+`
ORDER BY updated_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := []*service.TokenRiskWatchlistItem{}
	for rows.Next() {
		item := &service.TokenRiskWatchlistItem{}
		if err := rows.Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt, &item.SubjectType, &item.SubjectValue, &item.Reason, &item.ActorUserID, &item.Active); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *tokenRiskRepository) GetTokenRiskSummary(ctx context.Context, since time.Time) (*service.TokenRiskSummary, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil token risk repository")
	}
	out := &service.TokenRiskSummary{
		ByLevel:    map[string]int64{},
		ByCategory: map[string]int64{},
	}
	row := r.db.QueryRowContext(ctx, `
SELECT
  COUNT(*),
  COUNT(*) FILTER (WHERE status = 'open'),
  COUNT(*) FILTER (WHERE status = 'handled'),
  COUNT(*) FILTER (WHERE false_positive = TRUE),
  COUNT(*) FILTER (WHERE risk_level = 'high'),
  COUNT(*) FILTER (WHERE risk_level = 'critical'),
  COUNT(DISTINCT user_id) FILTER (WHERE user_id IS NOT NULL),
  COUNT(DISTINCT token_hash) FILTER (WHERE token_hash <> ''),
  COUNT(DISTINCT api_key_id) FILTER (WHERE api_key_id IS NOT NULL)
FROM token_risk_events
WHERE created_at >= $1`, since)
	if err := row.Scan(&out.Total, &out.Open, &out.Handled, &out.FalsePositive, &out.High, &out.Critical, &out.DistinctUsers, &out.DistinctTokens, &out.DistinctAPIKeys); err != nil {
		return nil, err
	}
	if err := r.fillSummaryMap(ctx, `SELECT risk_level, COUNT(*) FROM token_risk_events WHERE created_at >= $1 GROUP BY risk_level`, since, out.ByLevel); err != nil {
		return nil, err
	}
	if err := r.fillSummaryMap(ctx, `SELECT value, COUNT(*) FROM token_risk_events, jsonb_array_elements_text(risk_categories) AS value WHERE created_at >= $1 GROUP BY value`, since, out.ByCategory); err != nil {
		return nil, err
	}
	out.TopUsers, _ = r.topSubjectStats(ctx, since, "user")
	out.TopTokens, _ = r.topSubjectStats(ctx, since, "token")
	out.TopAPIKeys, _ = r.topSubjectStats(ctx, since, "api_key")
	out.RecentHighRisk, _, _ = r.ListTokenRiskEvents(ctx, service.TokenRiskEventFilter{Since: &since, RiskLevel: "high", Page: 1, PageSize: 8})
	return out, nil
}

func (r *tokenRiskRepository) fillSummaryMap(ctx context.Context, query string, since time.Time, target map[string]int64) error {
	rows, err := r.db.QueryContext(ctx, query, since)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var key string
		var count int64
		if err := rows.Scan(&key, &count); err != nil {
			return err
		}
		target[key] = count
	}
	return rows.Err()
}

func (r *tokenRiskRepository) topSubjectStats(ctx context.Context, since time.Time, kind string) ([]service.TokenRiskSubjectStat, error) {
	expr := "COALESCE(user_id::text, '')"
	where := "user_id IS NOT NULL"
	if kind == "token" {
		expr = "token_hash"
		where = "token_hash <> ''"
	}
	if kind == "api_key" {
		expr = "api_key_id::text"
		where = "api_key_id IS NOT NULL"
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT `+expr+` AS subject, COUNT(*), COALESCE(SUM(risk_score), 0)
FROM token_risk_events
WHERE created_at >= $1 AND `+where+`
GROUP BY subject
ORDER BY SUM(risk_score) DESC, COUNT(*) DESC
LIMIT 8`, since)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.TokenRiskSubjectStat{}
	for rows.Next() {
		var item service.TokenRiskSubjectStat
		if err := rows.Scan(&item.Subject, &item.Count, &item.Score); err != nil {
			return nil, err
		}
		if kind == "user" {
			var id int64
			if _, err := fmt.Sscan(item.Subject, &id); err == nil && id > 0 {
				item.UserID = &id
			}
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

type tokenRiskScanner interface {
	Scan(dest ...any) error
}

func scanTokenRiskEvent(row tokenRiskScanner) (*service.TokenRiskEvent, error) {
	item := &service.TokenRiskEvent{}
	var sourceLogID, userID, apiKeyID, handledBy sql.NullInt64
	var handledAt sql.NullTime
	var categoriesRaw, rulesRaw, actionsRaw string
	if err := row.Scan(
		&item.ID, &item.CreatedAt, &item.UpdatedAt, &item.LastSeenAt, &sourceLogID,
		&userID, &apiKeyID, &item.TokenType, &item.TokenHash, &item.TokenPrefix, &item.TokenSuffix,
		&item.APIKeySummary, &item.ClientIP, &item.UserAgent, &item.Method, &item.Path, &item.StatusCode,
		&item.Result, &item.FailureReason, &item.RiskScore, &item.RiskLevel,
		&categoriesRaw, &rulesRaw, &actionsRaw, &item.Explanation, &item.Status, &item.FalsePositive,
		&handledBy, &handledAt, &item.Count5m, &item.Count1h, &item.Count24h, &item.DistinctIP24h,
	); err != nil {
		return nil, err
	}
	if sourceLogID.Valid {
		item.SourceLogID = &sourceLogID.Int64
	}
	if userID.Valid {
		item.UserID = &userID.Int64
	}
	if apiKeyID.Valid {
		item.APIKeyID = &apiKeyID.Int64
	}
	if handledBy.Valid {
		item.HandledByUserID = &handledBy.Int64
	}
	if handledAt.Valid {
		item.HandledAt = &handledAt.Time
	}
	_ = json.Unmarshal([]byte(categoriesRaw), &item.RiskCategories)
	_ = json.Unmarshal([]byte(rulesRaw), &item.MatchedRules)
	_ = json.Unmarshal([]byte(actionsRaw), &item.RecommendedActions)
	return item, nil
}

func buildTokenRiskWhere(filter service.TokenRiskEventFilter) (string, []any) {
	clauses := []string{"1=1"}
	args := []any{}
	if filter.Since != nil {
		args = append(args, filter.Since.UTC())
		clauses = append(clauses, "e.created_at >= $"+itoa(len(args)))
	}
	if filter.Until != nil {
		args = append(args, filter.Until.UTC())
		clauses = append(clauses, "e.created_at <= $"+itoa(len(args)))
	}
	if v := strings.TrimSpace(filter.RiskLevel); v != "" {
		args = append(args, v)
		clauses = append(clauses, "e.risk_level = $"+itoa(len(args)))
	}
	if v := strings.TrimSpace(filter.RiskCategory); v != "" {
		args = append(args, v)
		clauses = append(clauses, "e.risk_categories ? $"+itoa(len(args)))
	}
	if v := strings.TrimSpace(filter.TokenType); v != "" {
		args = append(args, v)
		clauses = append(clauses, "e.token_type = $"+itoa(len(args)))
	}
	if v := strings.TrimSpace(filter.Status); v != "" {
		args = append(args, v)
		clauses = append(clauses, "e.status = $"+itoa(len(args)))
	}
	if filter.UserID != nil && *filter.UserID > 0 {
		args = append(args, *filter.UserID)
		clauses = append(clauses, "e.user_id = $"+itoa(len(args)))
	}
	if filter.APIKeyID != nil && *filter.APIKeyID > 0 {
		args = append(args, *filter.APIKeyID)
		clauses = append(clauses, "e.api_key_id = $"+itoa(len(args)))
	}
	if q := strings.TrimSpace(filter.Query); q != "" {
		args = append(args, "%"+q+"%")
		n := itoa(len(args))
		clauses = append(clauses, "(e.token_hash ILIKE $"+n+" OR e.client_ip ILIKE $"+n+" OR e.path ILIKE $"+n+" OR e.failure_reason ILIKE $"+n+")")
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func normalizeTokenRiskPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}
