package model

import "time"

const (
	CheckinStatusPending = "pending"
	CheckinStatusSuccess = "success"
	CheckinStatusFailed  = "failed"

	GrantMethodCreateAndRedeem = "create_and_redeem"
	GrantMethodBalanceAdd      = "balance_add"
)

type SessionContext struct {
	SessionID  string    `json:"session_id"`
	UserID     int64     `json:"user_id"`
	Username   string    `json:"username"`
	Email      string    `json:"email"`
	Balance    float64   `json:"balance"`
	Theme      string    `json:"theme"`
	Lang       string    `json:"lang"`
	UIMode     string    `json:"ui_mode"`
	UserToken  string    `json:"-"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"user_agent"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
}

type Sub2APIUser struct {
	ID       int64
	Username string
	Email    string
	Balance  float64
	Role     string
	Status   string
}

type BridgeResolveUserRequest struct {
	Token string `json:"token"`
}

type BridgeResolvedUser struct {
	ID       int64    `json:"id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Balance  *float64 `json:"balance"`
	Role     string   `json:"role"`
	Status   string   `json:"status"`
}

type BridgeResolveUserResponse struct {
	Data BridgeResolvedUser `json:"data"`
}

type BridgeGrantRewardRequest struct {
	UserID         int64   `json:"user_id"`
	Amount         float64 `json:"amount"`
	RewardCode     string  `json:"reward_code"`
	IdempotencyKey string  `json:"idempotency_key"`
	Note           string  `json:"note"`
}

type BootstrapRequest struct {
	UserID string
	Token  string
	Theme  string
	Lang   string
	UIMode string
	IP     string
	UA     string
}

type BootstrapResponse struct {
	AppName               string        `json:"app_name"`
	UserID                int64         `json:"user_id"`
	Username              string        `json:"username"`
	Theme                 string        `json:"theme"`
	Lang                  string        `json:"lang"`
	UIMode                string        `json:"ui_mode"`
	SignedToday           bool          `json:"signed_today"`
	Streak                int           `json:"streak"`
	TodayRewardMin        string        `json:"today_reward_min"`
	TodayRewardMax        string        `json:"today_reward_max"`
	CurrentBalanceDisplay string        `json:"current_balance_display"`
	RecentSignDays        []string      `json:"recent_sign_days"`
	MonthSignDays         []string      `json:"month_sign_days"`
	History               []HistoryItem `json:"history"`
}

type StatusResponse struct {
	SignedToday           bool          `json:"signed_today"`
	Streak                int           `json:"streak"`
	TodayRewardMin        string        `json:"today_reward_min"`
	TodayRewardMax        string        `json:"today_reward_max"`
	CurrentBalanceDisplay string        `json:"current_balance_display"`
	RecentSignDays        []string      `json:"recent_sign_days"`
	MonthSignDays         []string      `json:"month_sign_days"`
	History               []HistoryItem `json:"history"`
}

type CheckinResponse struct {
	SignedToday           bool   `json:"signed_today"`
	AlreadySigned         bool   `json:"already_signed"`
	SignDate              string `json:"sign_date"`
	BaseReward            string `json:"base_reward"`
	BonusReward           string `json:"bonus_reward"`
	TotalReward           string `json:"total_reward"`
	RewardCode            string `json:"reward_code"`
	GrantStatus           string `json:"grant_status"`
	GrantMethod           string `json:"grant_method"`
	Streak                int    `json:"streak"`
	CurrentBalanceDisplay string `json:"current_balance_display"`
}

type HistoryItem struct {
	ID          int64     `json:"id"`
	SignDate    string    `json:"sign_date"`
	BaseReward  string    `json:"base_reward"`
	BonusReward string    `json:"bonus_reward"`
	TotalReward string    `json:"total_reward"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type CheckinRecord struct {
	ID             int64
	Sub2APIUserID  int64
	SignDate       string
	BaseReward     float64
	BonusReward    float64
	TotalReward    float64
	RewardCode     string
	GrantMethod    string
	GrantStatus    string
	IdempotencyKey string
	IP             string
	UA             string
	FailureReason  string
	AttemptCount   int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type RewardAttempt struct {
	ID              int64
	CheckinID       int64
	RewardCode      string
	IdempotencyKey  string
	RequestPayload  string
	ResponsePayload string
	HTTPStatus      int
	Success         bool
	CreatedAt       time.Time
}

type RiskEvent struct {
	ID            int64     `json:"id"`
	Sub2APIUserID int64     `json:"sub2api_user_id"`
	RiskType      string    `json:"risk_type"`
	RiskScore     int       `json:"risk_score"`
	Detail        string    `json:"detail"`
	CreatedAt     time.Time `json:"created_at"`
}

type UserStats struct {
	Sub2APIUserID int64
	LastSignDate  string
	CurrentStreak int
	TotalReward   float64
	UpdatedAt     time.Time
}

type RewardTier struct {
	Min    float64
	Max    float64
	Weight int
}
