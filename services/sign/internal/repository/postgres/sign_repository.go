package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"sub2api-sign/internal/model"
)

type SignRepository struct {
	db *pgxpool.Pool
}

func NewSignRepository(db *pgxpool.Pool) *SignRepository {
	return &SignRepository{db: db}
}

func (r *SignRepository) GetCheckinByDate(ctx context.Context, userID int64, signDate string) (model.CheckinRecord, bool, error) {
	const query = `
SELECT
	id,
	sub2api_user_id,
	sign_date::text,
	base_reward,
	bonus_reward,
	total_reward,
	reward_code,
	grant_method,
	grant_status,
	idempotency_key,
	ip,
	ua,
	COALESCE(failure_reason, ''),
	COALESCE(a.attempt_count, 0) AS attempt_count,
	created_at,
	updated_at
FROM sign_checkins
LEFT JOIN (
	SELECT checkin_id, COUNT(*) AS attempt_count
	FROM sign_reward_attempts
	GROUP BY checkin_id
) a ON a.checkin_id = sign_checkins.id
WHERE sub2api_user_id = $1 AND sign_date = $2
LIMIT 1`

	var item model.CheckinRecord
	err := r.db.QueryRow(ctx, query, userID, signDate).Scan(
		&item.ID,
		&item.Sub2APIUserID,
		&item.SignDate,
		&item.BaseReward,
		&item.BonusReward,
		&item.TotalReward,
		&item.RewardCode,
		&item.GrantMethod,
		&item.GrantStatus,
		&item.IdempotencyKey,
		&item.IP,
		&item.UA,
		&item.FailureReason,
		&item.AttemptCount,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.CheckinRecord{}, false, nil
	}

	if err != nil {
		return model.CheckinRecord{}, false, err
	}

	return item, true, nil
}

func (r *SignRepository) InsertPendingCheckin(ctx context.Context, record model.CheckinRecord) (model.CheckinRecord, error) {
	const query = `
INSERT INTO sign_checkins (
	sub2api_user_id,
	sign_date,
	base_reward,
	bonus_reward,
	total_reward,
	reward_code,
	grant_method,
	grant_status,
	idempotency_key,
	ip,
	ua,
	failure_reason
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(
		ctx,
		query,
		record.Sub2APIUserID,
		record.SignDate,
		record.BaseReward,
		record.BonusReward,
		record.TotalReward,
		record.RewardCode,
		record.GrantMethod,
		record.GrantStatus,
		record.IdempotencyKey,
		record.IP,
		record.UA,
		record.FailureReason,
	).Scan(&record.ID, &record.CreatedAt, &record.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return model.CheckinRecord{}, fmt.Errorf("checkin already exists: %w", err)
		}
		return model.CheckinRecord{}, err
	}

	return record, nil
}

func (r *SignRepository) AddRewardAttempt(ctx context.Context, attempt model.RewardAttempt) error {
	const query = `
INSERT INTO sign_reward_attempts (
	checkin_id,
	reward_code,
	idempotency_key,
	request_payload,
	response_payload,
	http_status,
	success
) VALUES ($1,$2,$3,$4,$5,$6,$7)`

	_, err := r.db.Exec(
		ctx,
		query,
		attempt.CheckinID,
		attempt.RewardCode,
		attempt.IdempotencyKey,
		attempt.RequestPayload,
		attempt.ResponsePayload,
		attempt.HTTPStatus,
		attempt.Success,
	)

	return err
}

func (r *SignRepository) ListRetryCandidates(ctx context.Context, limit int) ([]model.CheckinRecord, error) {
	const query = `
SELECT
	c.id,
	c.sub2api_user_id,
	c.sign_date::text,
	c.base_reward,
	c.bonus_reward,
	c.total_reward,
	c.reward_code,
	c.grant_method,
	c.grant_status,
	c.idempotency_key,
	c.ip,
	c.ua,
	COALESCE(c.failure_reason, ''),
	COALESCE(a.attempt_count, 0) AS attempt_count,
	c.created_at,
	c.updated_at
FROM sign_checkins c
LEFT JOIN (
	SELECT checkin_id, COUNT(*) AS attempt_count
	FROM sign_reward_attempts
	GROUP BY checkin_id
) a ON a.checkin_id = c.id
WHERE c.grant_status IN ('failed', 'pending')
ORDER BY c.updated_at ASC, c.id ASC
LIMIT $1`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.CheckinRecord, 0, limit)
	for rows.Next() {
		var item model.CheckinRecord
		if err := rows.Scan(
			&item.ID,
			&item.Sub2APIUserID,
			&item.SignDate,
			&item.BaseReward,
			&item.BonusReward,
			&item.TotalReward,
			&item.RewardCode,
			&item.GrantMethod,
			&item.GrantStatus,
			&item.IdempotencyKey,
			&item.IP,
			&item.UA,
			&item.FailureReason,
			&item.AttemptCount,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *SignRepository) MarkCheckinSuccess(ctx context.Context, record model.CheckinRecord, streak int) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer rollbackQuietly(ctx, tx)

	if _, err := tx.Exec(
		ctx,
		`UPDATE sign_checkins SET grant_status = $2, updated_at = NOW(), failure_reason = '' WHERE id = $1`,
		record.ID,
		model.CheckinStatusSuccess,
	); err != nil {
		return err
	}

	if _, err := tx.Exec(
		ctx,
		`
INSERT INTO sign_user_stats (sub2api_user_id, last_sign_date, current_streak, total_reward, updated_at)
VALUES ($1,$2,$3,$4,NOW())
ON CONFLICT (sub2api_user_id) DO UPDATE SET
	last_sign_date = EXCLUDED.last_sign_date,
	current_streak = EXCLUDED.current_streak,
	total_reward = sign_user_stats.total_reward + EXCLUDED.total_reward,
	updated_at = NOW()`,
		record.Sub2APIUserID,
		record.SignDate,
		streak,
		record.TotalReward,
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *SignRepository) MarkCheckinFailed(ctx context.Context, checkinID int64, failureReason string) error {
	_, err := r.db.Exec(
		ctx,
		`UPDATE sign_checkins SET grant_status = $2, failure_reason = $3, updated_at = NOW() WHERE id = $1`,
		checkinID,
		model.CheckinStatusFailed,
		failureReason,
	)

	return err
}

func (r *SignRepository) ListHistory(ctx context.Context, userID int64, limit int) ([]model.HistoryItem, error) {
	const query = `
SELECT id, sign_date::text, base_reward, bonus_reward, total_reward, grant_status, created_at
FROM sign_checkins
WHERE sub2api_user_id = $1
ORDER BY sign_date DESC, id DESC
LIMIT $2`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.HistoryItem, 0, limit)
	for rows.Next() {
		var item model.HistoryItem
		var baseReward float64
		var bonusReward float64
		var totalReward float64
		if err := rows.Scan(
			&item.ID,
			&item.SignDate,
			&baseReward,
			&bonusReward,
			&totalReward,
			&item.Status,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}

		item.BaseReward = fmt.Sprintf("%.2f", baseReward)
		item.BonusReward = fmt.Sprintf("%.2f", bonusReward)
		item.TotalReward = fmt.Sprintf("%.2f", totalReward)
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *SignRepository) ListRecentSignDays(ctx context.Context, userID int64, days int, now time.Time) ([]string, error) {
	const query = `
SELECT sign_date::text
FROM sign_checkins
WHERE sub2api_user_id = $1
  AND grant_status = 'success'
  AND sign_date >= $2
ORDER BY sign_date DESC`

	since := now.AddDate(0, 0, -days+1).Format("2006-01-02")

	rows, err := r.db.Query(ctx, query, userID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]string, 0, days)
	for rows.Next() {
		var signDate string
		if err := rows.Scan(&signDate); err != nil {
			return nil, err
		}
		items = append(items, signDate)
	}

	return items, rows.Err()
}

func (r *SignRepository) GetUserStats(ctx context.Context, userID int64) (model.UserStats, bool, error) {
	const query = `
SELECT sub2api_user_id, last_sign_date::text, current_streak, total_reward, updated_at
FROM sign_user_stats
WHERE sub2api_user_id = $1
LIMIT 1`

	var item model.UserStats
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&item.Sub2APIUserID,
		&item.LastSignDate,
		&item.CurrentStreak,
		&item.TotalReward,
		&item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.UserStats{}, false, nil
	}

	return item, err == nil, err
}

func (r *SignRepository) AddRiskEvent(ctx context.Context, userID int64, riskType string, riskScore int, detail string) error {
	_, err := r.db.Exec(
		ctx,
		`INSERT INTO sign_risk_events (sub2api_user_id, risk_type, risk_score, detail) VALUES ($1,$2,$3,$4)`,
		userID,
		riskType,
		riskScore,
		detail,
	)
	return err
}

func rollbackQuietly(ctx context.Context, tx pgx.Tx) {
	if tx == nil {
		return
	}

	_ = tx.Rollback(ctx)
}
