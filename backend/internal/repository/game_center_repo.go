package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type gameCenterRepository struct {
	db *sql.DB
}

func NewGameCenterRepository(db *sql.DB) service.GameCenterRepository {
	return &gameCenterRepository{db: db}
}

func (r *gameCenterRepository) GetGameCenterPlayByRoundID(ctx context.Context, roundID string) (*service.GameCenterPlay, error) {
	if r == nil || r.db == nil {
		return nil, service.ErrGameCenterInvalidInput
	}
	return r.getGameCenterPlayByRoundID(ctx, roundID)
}

func (r *gameCenterRepository) CreateGameCenterPlay(ctx context.Context, play *service.GameCenterPlay) (*service.GameCenterPlay, bool, error) {
	if r == nil || r.db == nil || play == nil {
		return nil, false, service.ErrGameCenterInvalidInput
	}
	metadata, err := json.Marshal(play.Metadata)
	if err != nil {
		return nil, false, err
	}
	const insertSQL = `
INSERT INTO game_center_plays (
  user_id, game_key, round_id, stake_type, cost_amount, reward_amount, net_amount, played_at, metadata
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb
)
ON CONFLICT (round_id) DO NOTHING
RETURNING id, user_id, game_key, round_id, stake_type,
  cost_amount::double precision, reward_amount::double precision, net_amount::double precision,
  played_at, metadata, created_at, updated_at`
	out := service.GameCenterPlay{}
	var rawMetadata []byte
	err = r.db.QueryRowContext(ctx, insertSQL,
		play.UserID, play.GameKey, play.RoundID, play.StakeType,
		play.CostAmount, play.RewardAmount, play.NetAmount, play.PlayedAt, string(metadata),
	).Scan(
		&out.ID, &out.UserID, &out.GameKey, &out.RoundID, &out.StakeType,
		&out.CostAmount, &out.RewardAmount, &out.NetAmount,
		&out.PlayedAt, &rawMetadata, &out.CreatedAt, &out.UpdatedAt,
	)
	if err == nil {
		out.Metadata = decodeGameCenterMetadata(rawMetadata)
		return &out, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, err
	}
	existing, err := r.getGameCenterPlayByRoundID(ctx, play.RoundID)
	if err != nil {
		return nil, false, err
	}
	return existing, false, nil
}

func (r *gameCenterRepository) CountGameCenterPlays(ctx context.Context, filter service.GameCenterPlayCountFilter) (int, error) {
	if r == nil || r.db == nil {
		return 0, service.ErrGameCenterInvalidInput
	}
	var count int
	err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)::integer
FROM game_center_plays
WHERE user_id = $1 AND game_key = $2 AND stake_type = $3 AND played_at >= $4`,
		filter.UserID, filter.GameKey, filter.StakeType, filter.Since,
	).Scan(&count)
	return count, err
}

func (r *gameCenterRepository) GetGameCenterLeaderboard(ctx context.Context, filter service.GameCenterLeaderboardFilter) ([]service.GameCenterLeaderboardItem, error) {
	if r == nil || r.db == nil {
		return nil, service.ErrGameCenterInvalidInput
	}
	whereSQL, args := gameCenterLeaderboardWhere(filter)
	args = append(args, filter.Limit)
	query := `
SELECT
  ROW_NUMBER() OVER (ORDER BY SUM(p.net_amount) DESC, COUNT(*) DESC, MAX(p.played_at) DESC, p.user_id ASC)::integer AS rank,
  p.user_id,
  COALESCE(NULLIF(u.username, ''), 'user-' || p.user_id::text) AS username,
  COALESCE(ua.url, '') AS avatar_url,
  SUM(p.net_amount)::double precision AS net_amount,
  COUNT(*)::integer AS play_count,
  SUM(CASE WHEN p.net_amount > 0 THEN 1 ELSE 0 END)::integer AS positive_net_count,
  MAX(p.played_at) AS last_played_at
FROM game_center_plays p
JOIN users u ON u.id = p.user_id
LEFT JOIN user_avatars ua ON ua.user_id = p.user_id
` + whereSQL + `
GROUP BY p.user_id, u.username, ua.url
ORDER BY net_amount DESC, play_count DESC, last_played_at DESC, p.user_id ASC
LIMIT $` + itoa(len(args)) + `
`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]service.GameCenterLeaderboardItem, 0)
	for rows.Next() {
		var item service.GameCenterLeaderboardItem
		var lastPlayedAt sql.NullTime
		if err := rows.Scan(
			&item.Rank, &item.UserID, &item.Username, &item.AvatarURL,
			&item.NetAmount, &item.PlayCount, &item.PositiveNetCount, &lastPlayedAt,
		); err != nil {
			return nil, err
		}
		if item.PlayCount > 0 {
			item.WinRate = float64(item.PositiveNetCount) / float64(item.PlayCount)
		}
		if lastPlayedAt.Valid {
			item.LastPlayedAt = &lastPlayedAt.Time
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *gameCenterRepository) GetGameCenterUserStats(ctx context.Context, userID int64, since *time.Time) (*service.GameCenterUserStats, error) {
	if r == nil || r.db == nil {
		return nil, service.ErrGameCenterInvalidInput
	}
	query := `
SELECT game_key, stake_type, COUNT(*)::integer, COALESCE(SUM(net_amount), 0)::double precision
FROM game_center_plays
WHERE user_id = $1`
	args := []any{userID}
	if since != nil {
		args = append(args, *since)
		query += " AND played_at >= $2"
	}
	query += " GROUP BY game_key, stake_type"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	stats := &service.GameCenterUserStats{
		UserID:         userID,
		TodayFreeCount: map[string]int{},
		TodayPaidCount: map[string]int{},
		Remaining:      map[string]service.GameCenterRemaining{},
	}
	for rows.Next() {
		var gameKey, stakeType string
		var count int
		var net float64
		if err := rows.Scan(&gameKey, &stakeType, &count, &net); err != nil {
			return nil, err
		}
		stats.TodayPlayCount += count
		stats.TodayNetAmount += net
		remaining := stats.Remaining[gameKey]
		if remaining.Free == 0 {
			remaining.Free = gameCenterFreeLimit()
		}
		if remaining.Paid == 0 {
			remaining.Paid = gameCenterPaidLimit()
		}
		switch stakeType {
		case service.GameCenterStakeFree:
			stats.TodayFreeCount[gameKey] = count
			remaining.Free = max(0, gameCenterFreeLimit()-count)
		case service.GameCenterStakePaid:
			stats.TodayPaidCount[gameKey] = count
			remaining.Paid = max(0, gameCenterPaidLimit()-count)
		}
		stats.Remaining[gameKey] = remaining
	}
	return stats, rows.Err()
}

func (r *gameCenterRepository) GetGameCenterUserRank(ctx context.Context, userID int64, filter service.GameCenterLeaderboardFilter) (int, error) {
	if r == nil || r.db == nil {
		return 0, service.ErrGameCenterInvalidInput
	}
	whereSQL, args := gameCenterLeaderboardWhere(filter)
	args = append(args, userID)
	query := `
WITH ranked AS (
  SELECT
    p.user_id,
    ROW_NUMBER() OVER (ORDER BY SUM(p.net_amount) DESC, COUNT(*) DESC, MAX(p.played_at) DESC, p.user_id ASC)::integer AS rank
  FROM game_center_plays p
` + whereSQL + `
  GROUP BY p.user_id
)
SELECT rank FROM ranked WHERE user_id = $` + itoa(len(args)) + `
`
	var rank int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&rank)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return rank, err
}

func (r *gameCenterRepository) getGameCenterPlayByRoundID(ctx context.Context, roundID string) (*service.GameCenterPlay, error) {
	out := service.GameCenterPlay{}
	var rawMetadata []byte
	err := r.db.QueryRowContext(ctx, `
SELECT id, user_id, game_key, round_id, stake_type,
  cost_amount::double precision, reward_amount::double precision, net_amount::double precision,
  played_at, metadata, created_at, updated_at
FROM game_center_plays
WHERE round_id = $1`,
		roundID,
	).Scan(
		&out.ID, &out.UserID, &out.GameKey, &out.RoundID, &out.StakeType,
		&out.CostAmount, &out.RewardAmount, &out.NetAmount,
		&out.PlayedAt, &rawMetadata, &out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	out.Metadata = decodeGameCenterMetadata(rawMetadata)
	return &out, nil
}

func gameCenterLeaderboardWhere(filter service.GameCenterLeaderboardFilter) (string, []any) {
	conditions := make([]string, 0, 2)
	args := make([]any, 0, 2)
	if filter.GameKey != "" {
		args = append(args, filter.GameKey)
		conditions = append(conditions, "p.game_key = $"+itoa(len(args)))
	}
	if filter.Since != nil {
		args = append(args, *filter.Since)
		conditions = append(conditions, "p.played_at >= $"+itoa(len(args)))
	}
	if len(conditions) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}

func decodeGameCenterMetadata(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		return map[string]any{}
	}
	return out
}

func gameCenterFreeLimit() int { return 2 }

func gameCenterPaidLimit() int { return 5 }
