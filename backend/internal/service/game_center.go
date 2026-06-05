package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"regexp"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	GameCenterRangeToday = "today"
	GameCenterRange7d    = "7d"
	GameCenterRange30d   = "30d"
	GameCenterRangeAll   = "all"

	GameCenterStakeFree = "free"
	GameCenterStakePaid = "paid"

	gameCenterDailyFreeLimit         = 2
	gameCenterDailyPaidLimit         = 5
	gameCenterMaxAmount              = 1000000
	gameCenterEmbedTokenMinSecretLen = 16
)

var (
	ErrGameCenterInvalidInput = infraerrors.BadRequest("GAME_CENTER_INVALID_INPUT", "invalid game center input")
	ErrGameCenterDailyLimit   = infraerrors.TooManyRequests("GAME_CENTER_DAILY_LIMIT", "daily game play limit reached")

	gameCenterSafeKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.:-]{0,63}$`)
)

type GameCenterRepository interface {
	GetGameCenterPlayByRoundID(ctx context.Context, roundID string) (*GameCenterPlay, error)
	CreateGameCenterPlay(ctx context.Context, play *GameCenterPlay) (*GameCenterPlay, bool, error)
	CountGameCenterPlays(ctx context.Context, filter GameCenterPlayCountFilter) (int, error)
	GetGameCenterLeaderboard(ctx context.Context, filter GameCenterLeaderboardFilter) ([]GameCenterLeaderboardItem, error)
	GetGameCenterUserStats(ctx context.Context, userID int64, since *time.Time) (*GameCenterUserStats, error)
	GetGameCenterUserRank(ctx context.Context, userID int64, filter GameCenterLeaderboardFilter) (int, error)
}

type GameCenterPlay struct {
	ID           int64          `json:"id"`
	UserID       int64          `json:"user_id"`
	GameKey      string         `json:"game_key"`
	RoundID      string         `json:"round_id"`
	StakeType    string         `json:"stake_type"`
	CostAmount   float64        `json:"cost_amount"`
	RewardAmount float64        `json:"reward_amount"`
	NetAmount    float64        `json:"net_amount"`
	PlayedAt     time.Time      `json:"played_at"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	Duplicate    bool           `json:"duplicate,omitempty"`
}

type GameCenterRecordPlayInput struct {
	GameKey      string         `json:"game_key"`
	RoundID      string         `json:"round_id"`
	StakeType    string         `json:"stake_type"`
	CostAmount   float64        `json:"cost_amount"`
	RewardAmount float64        `json:"reward_amount"`
	NetAmount    float64        `json:"net_amount"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type GameCenterPlayCountFilter struct {
	UserID    int64
	GameKey   string
	StakeType string
	Since     time.Time
}

type GameCenterLeaderboardFilter struct {
	GameKey string
	Range   string
	Since   *time.Time
	Limit   int
}

type GameCenterLeaderboardItem struct {
	Rank             int        `json:"rank"`
	UserID           int64      `json:"user_id"`
	Username         string     `json:"username"`
	AvatarURL        string     `json:"avatar_url"`
	NetAmount        float64    `json:"net_amount"`
	PlayCount        int        `json:"play_count"`
	WinRate          float64    `json:"win_rate"`
	LastPlayedAt     *time.Time `json:"last_played_at,omitempty"`
	PositiveNetCount int        `json:"-"`
}

type GameCenterLeaderboardResponse struct {
	Range   string                      `json:"range"`
	GameKey string                      `json:"game_key,omitempty"`
	Limit   int                         `json:"limit"`
	Items   []GameCenterLeaderboardItem `json:"items"`
}

type GameCenterUserStats struct {
	UserID         int64                          `json:"user_id"`
	TodayPlayCount int                            `json:"today_play_count"`
	TodayNetAmount float64                        `json:"today_net_amount"`
	TodayFreeCount map[string]int                 `json:"today_free_count"`
	TodayPaidCount map[string]int                 `json:"today_paid_count"`
	Remaining      map[string]GameCenterRemaining `json:"remaining"`
}

type GameCenterRemaining struct {
	Free int `json:"free"`
	Paid int `json:"paid"`
}

type GameCenterMeResponse struct {
	Stats     *GameCenterUserStats `json:"stats"`
	TodayRank int                  `json:"today_rank"`
}

type GameCenterEmbedSession struct {
	EmbedToken string    `json:"embed_token"`
	ExpiresAt  time.Time `json:"expires_at"`
	UserID     int64     `json:"user_id"`
	Username   string    `json:"username"`
}

type GameCenterEmbedSessionVerifyResult struct {
	Valid     bool      `json:"valid"`
	ExpiresAt time.Time `json:"expires_at"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	Balance   float64   `json:"balance"`
	Status    string    `json:"status"`
}

type gameCenterEmbedTokenPayload struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	ExpiresAt int64  `json:"exp"`
	Nonce     string `json:"nonce"`
}

type GameCenterService struct {
	repo        GameCenterRepository
	userService *UserService
}

func NewGameCenterService(repo GameCenterRepository, userServices ...*UserService) *GameCenterService {
	var userService *UserService
	if len(userServices) > 0 {
		userService = userServices[0]
	}
	return &GameCenterService{repo: repo, userService: userService}
}

func (s *GameCenterService) RecordPlay(ctx context.Context, userID int64, input GameCenterRecordPlayInput) (*GameCenterPlay, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil, ErrGameCenterInvalidInput
	}
	normalized, err := normalizeGameCenterPlayInput(input)
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.GetGameCenterPlayByRoundID(ctx, normalized.RoundID)
	if err == nil {
		if existing.UserID != userID {
			return nil, ErrGameCenterInvalidInput
		}
		existing.Duplicate = true
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	since := startOfUTCDate(time.Now().UTC())
	count, err := s.repo.CountGameCenterPlays(ctx, GameCenterPlayCountFilter{
		UserID:    userID,
		GameKey:   normalized.GameKey,
		StakeType: normalized.StakeType,
		Since:     since,
	})
	if err != nil {
		return nil, err
	}
	if normalized.StakeType == GameCenterStakeFree && count >= gameCenterDailyFreeLimit {
		return nil, ErrGameCenterDailyLimit
	}
	if normalized.StakeType == GameCenterStakePaid && count >= gameCenterDailyPaidLimit {
		return nil, ErrGameCenterDailyLimit
	}
	play := &GameCenterPlay{
		UserID:       userID,
		GameKey:      normalized.GameKey,
		RoundID:      normalized.RoundID,
		StakeType:    normalized.StakeType,
		CostAmount:   normalized.CostAmount,
		RewardAmount: normalized.RewardAmount,
		NetAmount:    normalized.NetAmount,
		PlayedAt:     time.Now().UTC(),
		Metadata:     trimGameCenterMetadata(normalized.Metadata),
	}
	created, inserted, err := s.repo.CreateGameCenterPlay(ctx, play)
	if err != nil {
		return nil, err
	}
	created.Duplicate = !inserted
	return created, nil
}

func (s *GameCenterService) Leaderboard(ctx context.Context, gameKey, rangeValue string, limit int) (*GameCenterLeaderboardResponse, error) {
	if s == nil || s.repo == nil {
		return nil, ErrGameCenterInvalidInput
	}
	filter, err := BuildGameCenterLeaderboardFilter(gameKey, rangeValue, limit)
	if err != nil {
		return nil, err
	}
	items, err := s.repo.GetGameCenterLeaderboard(ctx, filter)
	if err != nil {
		return nil, err
	}
	return &GameCenterLeaderboardResponse{
		Range:   filter.Range,
		GameKey: filter.GameKey,
		Limit:   filter.Limit,
		Items:   items,
	}, nil
}

func (s *GameCenterService) Me(ctx context.Context, userID int64) (*GameCenterMeResponse, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil, ErrGameCenterInvalidInput
	}
	since := startOfUTCDate(time.Now().UTC())
	stats, err := s.repo.GetGameCenterUserStats(ctx, userID, &since)
	if err != nil {
		return nil, err
	}
	rank, err := s.repo.GetGameCenterUserRank(ctx, userID, GameCenterLeaderboardFilter{
		Range: GameCenterRangeToday,
		Since: &since,
		Limit: 100,
	})
	if err != nil {
		return nil, err
	}
	return &GameCenterMeResponse{Stats: stats, TodayRank: rank}, nil
}

func (s *GameCenterService) CreateEmbedSession(ctx context.Context, userID int64) (*GameCenterEmbedSession, error) {
	if s == nil || s.userService == nil || userID <= 0 {
		return nil, ErrGameCenterInvalidInput
	}
	user, err := s.userService.GetByID(ctx, userID)
	if err != nil || user == nil || !user.IsActive() {
		return nil, infraerrors.Unauthorized("GAME_CENTER_USER_INVALID", "game center user is not available")
	}
	expiresAt := time.Now().UTC().Add(5 * time.Minute)
	nonce, err := randomGameCenterNonce()
	if err != nil {
		return nil, err
	}
	payload := gameCenterEmbedTokenPayload{
		UserID:    user.ID,
		Username:  displayGameCenterUsername(user),
		Role:      user.Role,
		ExpiresAt: expiresAt.Unix(),
		Nonce:     nonce,
	}
	token, err := encodeGameCenterEmbedToken(payload)
	if err != nil {
		return nil, err
	}
	return &GameCenterEmbedSession{
		EmbedToken: token,
		ExpiresAt:  expiresAt,
		UserID:     user.ID,
		Username:   payload.Username,
	}, nil
}

func (s *GameCenterService) VerifyEmbedSession(ctx context.Context, token string) (*GameCenterEmbedSessionVerifyResult, error) {
	if s == nil || s.userService == nil {
		return nil, ErrGameCenterInvalidInput
	}
	payload, err := decodeGameCenterEmbedToken(token)
	if err != nil {
		return nil, infraerrors.Unauthorized("GAME_CENTER_EMBED_TOKEN_INVALID", "game center embed token is invalid")
	}
	expiresAt := time.Unix(payload.ExpiresAt, 0).UTC()
	if time.Now().UTC().After(expiresAt) {
		return nil, infraerrors.Unauthorized("GAME_CENTER_EMBED_TOKEN_EXPIRED", "game center embed token is expired")
	}
	user, err := s.userService.GetByID(ctx, payload.UserID)
	if err != nil || user == nil || !user.IsActive() {
		return nil, infraerrors.Unauthorized("GAME_CENTER_USER_INVALID", "game center user is not available")
	}
	return &GameCenterEmbedSessionVerifyResult{
		Valid:     true,
		ExpiresAt: expiresAt,
		UserID:    user.ID,
		Username:  displayGameCenterUsername(user),
		Role:      user.Role,
		Balance:   user.Balance,
		Status:    user.Status,
	}, nil
}

func BuildGameCenterLeaderboardFilter(gameKey, rangeValue string, limit int) (GameCenterLeaderboardFilter, error) {
	key := strings.TrimSpace(gameKey)
	if key != "" && key != "all" && !gameCenterSafeKeyPattern.MatchString(key) {
		return GameCenterLeaderboardFilter{}, ErrGameCenterInvalidInput
	}
	if key == "all" {
		key = ""
	}
	r := strings.TrimSpace(strings.ToLower(rangeValue))
	if r == "" {
		r = GameCenterRangeToday
	}
	var since *time.Time
	now := time.Now().UTC()
	switch r {
	case GameCenterRangeToday:
		v := startOfUTCDate(now)
		since = &v
	case GameCenterRange7d:
		v := now.AddDate(0, 0, -7)
		since = &v
	case GameCenterRange30d:
		v := now.AddDate(0, 0, -30)
		since = &v
	case GameCenterRangeAll:
	default:
		return GameCenterLeaderboardFilter{}, ErrGameCenterInvalidInput
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return GameCenterLeaderboardFilter{GameKey: key, Range: r, Since: since, Limit: limit}, nil
}

func normalizeGameCenterPlayInput(input GameCenterRecordPlayInput) (GameCenterRecordPlayInput, error) {
	input.GameKey = strings.TrimSpace(input.GameKey)
	input.RoundID = strings.TrimSpace(input.RoundID)
	input.StakeType = strings.TrimSpace(strings.ToLower(input.StakeType))
	if !gameCenterSafeKeyPattern.MatchString(input.GameKey) || !gameCenterSafeKeyPattern.MatchString(input.RoundID) {
		return input, ErrGameCenterInvalidInput
	}
	if input.StakeType != GameCenterStakeFree && input.StakeType != GameCenterStakePaid {
		return input, ErrGameCenterInvalidInput
	}
	if !validGameCenterAmount(input.CostAmount) || !validGameCenterAmount(input.RewardAmount) || !validGameCenterAmount(input.NetAmount) {
		return input, ErrGameCenterInvalidInput
	}
	if math.Abs((input.RewardAmount-input.CostAmount)-input.NetAmount) > 0.000001 {
		return input, ErrGameCenterInvalidInput
	}
	return input, nil
}

func validGameCenterAmount(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && math.Abs(value) <= gameCenterMaxAmount
}

func startOfUTCDate(t time.Time) time.Time {
	utc := t.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

func trimGameCenterMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, min(len(metadata), 20))
	for k, v := range metadata {
		key := strings.TrimSpace(k)
		if key == "" || len(key) > 64 {
			continue
		}
		// 元数据只保留非敏感短字段，避免 token、k 参数、Cookie 等被日志/数据库长期保存。
		lower := strings.ToLower(key)
		if strings.Contains(lower, "token") || strings.Contains(lower, "secret") || lower == "k" || strings.Contains(lower, "cookie") {
			continue
		}
		switch typed := v.(type) {
		case string:
			if len(typed) > 256 {
				typed = typed[:256]
			}
			out[key] = typed
		case float64, bool, int, int64:
			out[key] = typed
		}
		if len(out) >= 20 {
			break
		}
	}
	return out
}

func displayGameCenterUsername(user *User) string {
	if user == nil {
		return ""
	}
	if trimmed := strings.TrimSpace(user.Username); trimmed != "" {
		return trimmed
	}
	return fmt.Sprintf("user-%d", user.ID)
}

func gameCenterEmbedSecret() string {
	if value := strings.TrimSpace(os.Getenv("SUB2API_GAME_CENTER_EMBED_SECRET")); value != "" {
		return value
	}
	return strings.TrimSpace(os.Getenv("SUB2API_INTERNAL_CONFIG_TOKEN"))
}

func randomGameCenterNonce() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

func encodeGameCenterEmbedToken(payload gameCenterEmbedTokenPayload) (string, error) {
	secret := gameCenterEmbedSecret()
	if len(secret) < gameCenterEmbedTokenMinSecretLen {
		return "", infraerrors.InternalServer("GAME_CENTER_EMBED_SECRET_NOT_CONFIGURED", "game center embed secret is not configured")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	body := base64.RawURLEncoding.EncodeToString(raw)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(body))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return body + "." + signature, nil
}

func decodeGameCenterEmbedToken(token string) (gameCenterEmbedTokenPayload, error) {
	var payload gameCenterEmbedTokenPayload
	secret := gameCenterEmbedSecret()
	if len(secret) < gameCenterEmbedTokenMinSecretLen {
		return payload, infraerrors.InternalServer("GAME_CENTER_EMBED_SECRET_NOT_CONFIGURED", "game center embed secret is not configured")
	}
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return payload, ErrGameCenterInvalidInput
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(parts[0]))
	expected := mac.Sum(nil)
	actual, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(actual, expected) {
		return payload, ErrGameCenterInvalidInput
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return payload, err
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return payload, err
	}
	if payload.UserID <= 0 || payload.ExpiresAt <= 0 || strings.TrimSpace(payload.Nonce) == "" {
		return payload, ErrGameCenterInvalidInput
	}
	return payload, nil
}
