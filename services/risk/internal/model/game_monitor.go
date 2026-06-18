package model

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type StringLike string
type FloatLike float64
type BoolLike bool

func (v *StringLike) UnmarshalJSON(data []byte) error {
	if v == nil {
		return fmt.Errorf("stringlike target is nil")
	}
	if string(data) == "null" {
		*v = ""
		return nil
	}

	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		*v = StringLike(asString)
		return nil
	}

	var asNumber json.Number
	if err := json.Unmarshal(data, &asNumber); err == nil {
		*v = StringLike(asNumber.String())
		return nil
	}

	var asBool bool
	if err := json.Unmarshal(data, &asBool); err == nil {
		*v = StringLike(strconv.FormatBool(asBool))
		return nil
	}

	return fmt.Errorf("unsupported stringlike value: %s", string(data))
}

func (v *FloatLike) UnmarshalJSON(data []byte) error {
	if v == nil {
		return fmt.Errorf("floatlike target is nil")
	}
	if string(data) == "null" {
		*v = 0
		return nil
	}

	var asNumber float64
	if err := json.Unmarshal(data, &asNumber); err == nil {
		*v = FloatLike(asNumber)
		return nil
	}

	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		trimmed := asString
		if trimmed == "" {
			*v = 0
			return nil
		}
		parsed, parseErr := strconv.ParseFloat(trimmed, 64)
		if parseErr != nil {
			return fmt.Errorf("parse floatlike string %q: %w", trimmed, parseErr)
		}
		*v = FloatLike(parsed)
		return nil
	}

	return fmt.Errorf("unsupported floatlike value: %s", string(data))
}

func (v *BoolLike) UnmarshalJSON(data []byte) error {
	if v == nil {
		return fmt.Errorf("boollike target is nil")
	}
	if string(data) == "null" {
		*v = false
		return nil
	}

	var asBool bool
	if err := json.Unmarshal(data, &asBool); err == nil {
		*v = BoolLike(asBool)
		return nil
	}

	var asNumber int
	if err := json.Unmarshal(data, &asNumber); err == nil {
		*v = BoolLike(asNumber != 0)
		return nil
	}

	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		switch strings.ToLower(strings.TrimSpace(asString)) {
		case "", "0", "false", "no", "off":
			*v = false
			return nil
		case "1", "true", "yes", "on":
			*v = true
			return nil
		default:
			return fmt.Errorf("unsupported boollike string: %q", asString)
		}
	}

	return fmt.Errorf("unsupported boollike value: %s", string(data))
}

type GameMonitorOverview struct {
	DayKey            string     `json:"day_key"`
	TotalRounds       int64      `json:"total_rounds"`
	FreeRounds        int64      `json:"free_rounds"`
	PaidRounds        int64      `json:"paid_rounds"`
	GrossRewardAmount float64    `json:"gross_reward_amount"`
	EntryFeeAmount    float64    `json:"entry_fee_amount"`
	NetRewardAmount   float64    `json:"net_reward_amount"`
	PendingClaims     int64      `json:"pending_claims"`
	FailedClaims      int64      `json:"failed_claims"`
	CapHits           int64      `json:"cap_hits"`
	RefreshedAt       *time.Time `json:"refreshed_at"`
}

type GameMonitorUserSnapshot struct {
	ID                int64      `json:"id"`
	DayKey            string     `json:"day_key"`
	UserID            string     `json:"user_id"`
	Username          string     `json:"username"`
	PlayCount         int64      `json:"play_count"`
	FreePlayCount     int64      `json:"free_play_count"`
	PaidPlayCount     int64      `json:"paid_play_count"`
	GrossRewardAmount float64    `json:"gross_reward_amount"`
	EntryFeeAmount    float64    `json:"entry_fee_amount"`
	NetRewardAmount   float64    `json:"net_reward_amount"`
	PendingClaimCount int64      `json:"pending_claim_count"`
	FailedClaimCount  int64      `json:"failed_claim_count"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type GameMonitorRound struct {
	RoundID           string     `json:"round_id"`
	DayKey            string     `json:"day_key"`
	UserID            string     `json:"user_id"`
	Username          string     `json:"username"`
	GameID            string     `json:"game_id"`
	RiskMode          string     `json:"risk_mode"`
	Status            string     `json:"status"`
	ClaimStatus       string     `json:"claim_status"`
	IsPaidRound       BoolLike   `json:"is_paid_round"`
	ChoiceIndex       *int       `json:"choice_index"`
	EntryFeeAmount    float64    `json:"entry_fee_amount"`
	GrossRewardAmount float64    `json:"gross_reward_amount"`
	NetRewardAmount   float64    `json:"net_reward_amount"`
	CreatedAt         time.Time  `json:"created_at"`
	RevealedAt        *time.Time `json:"revealed_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type GameMonitorClaim struct {
	ClaimID           string     `json:"claim_id"`
	RoundID           string     `json:"round_id"`
	DayKey            string     `json:"day_key"`
	UserID            string     `json:"user_id"`
	Username          string     `json:"username"`
	GameID            string     `json:"game_id"`
	RiskMode          string     `json:"risk_mode"`
	ClaimKind         string     `json:"claim_kind"`
	ClaimStatus       string     `json:"claim_status"`
	Amount            float64    `json:"amount"`
	EntryFeeAmount    float64    `json:"entry_fee_amount"`
	GrossRewardAmount float64    `json:"gross_reward_amount"`
	NetRewardAmount   float64    `json:"net_reward_amount"`
	AttemptCount      int        `json:"attempt_count"`
	LastError         string     `json:"last_error"`
	CreatedAt         time.Time  `json:"created_at"`
	RedeemedAt        *time.Time `json:"redeemed_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type GameMonitorEvent struct {
	EventKey   string         `json:"event_key"`
	DayKey     string         `json:"day_key"`
	UserID     string         `json:"user_id"`
	Username   string         `json:"username"`
	GameID     string         `json:"game_id"`
	RoundID    string         `json:"round_id"`
	ClaimID    string         `json:"claim_id"`
	RiskMode   string         `json:"risk_mode"`
	EventType  string         `json:"event_type"`
	Severity   string         `json:"severity"`
	Message    string         `json:"message"`
	Detail     map[string]any `json:"detail"`
	OccurredAt time.Time      `json:"occurred_at"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type GameMonitorListFilter struct {
	Page        int
	PageSize    int
	DayKey      string
	UserID      string
	GameID      string
	RiskMode    string
	ClaimStatus string
	EventType   string
	SortBy      string
}

type GameMonitorRefreshInput struct {
	DayKey string `json:"day_key"`
}

type GameMonitorRefreshStats struct {
	DayKey             string `json:"day_key"`
	ImportedUsers      int    `json:"imported_users"`
	ImportedRounds     int    `json:"imported_rounds"`
	ImportedClaims     int    `json:"imported_claims"`
	ImportedEvents     int    `json:"imported_events"`
	RefreshedSnapshots int    `json:"refreshed_snapshots"`
}

type GameMonitorRoundUpsertInput struct {
	RoundID           string     `json:"round_id"`
	DayKey            string     `json:"day_key"`
	UserID            StringLike `json:"user_id"`
	Username          string     `json:"username"`
	GameID            string     `json:"game_id"`
	RiskMode          string     `json:"risk_mode"`
	Status            string     `json:"status"`
	ClaimStatus       string     `json:"claim_status"`
	IsPaidRound       BoolLike   `json:"is_paid_round"`
	ChoiceIndex       *int       `json:"choice_index"`
	EntryFeeAmount    FloatLike  `json:"entry_fee_amount"`
	GrossRewardAmount FloatLike  `json:"gross_reward_amount"`
	NetRewardAmount   FloatLike  `json:"net_reward_amount"`
	CreatedAt         time.Time  `json:"created_at"`
	RevealedAt        *time.Time `json:"revealed_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type GameMonitorClaimUpsertInput struct {
	ClaimID           string     `json:"claim_id"`
	RoundID           string     `json:"round_id"`
	DayKey            string     `json:"day_key"`
	UserID            StringLike `json:"user_id"`
	Username          string     `json:"username"`
	GameID            string     `json:"game_id"`
	RiskMode          string     `json:"risk_mode"`
	ClaimKind         string     `json:"claim_kind"`
	ClaimStatus       string     `json:"claim_status"`
	Amount            FloatLike  `json:"amount"`
	EntryFeeAmount    FloatLike  `json:"entry_fee_amount"`
	GrossRewardAmount FloatLike  `json:"gross_reward_amount"`
	NetRewardAmount   FloatLike  `json:"net_reward_amount"`
	AttemptCount      int        `json:"attempt_count"`
	LastError         string     `json:"last_error"`
	CreatedAt         time.Time  `json:"created_at"`
	RedeemedAt        *time.Time `json:"redeemed_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type GameMonitorEventUpsertInput struct {
	EventKey   string         `json:"event_key"`
	DayKey     string         `json:"day_key"`
	UserID     StringLike     `json:"user_id"`
	Username   string         `json:"username"`
	GameID     string         `json:"game_id"`
	RoundID    string         `json:"round_id"`
	ClaimID    string         `json:"claim_id"`
	RiskMode   string         `json:"risk_mode"`
	EventType  string         `json:"event_type"`
	Severity   string         `json:"severity"`
	Message    string         `json:"message"`
	Detail     map[string]any `json:"detail"`
	OccurredAt time.Time      `json:"occurred_at"`
}

type GameMonitorUserSnapshotUpsertInput struct {
	DayKey            string     `json:"day_key"`
	UserID            StringLike `json:"user_id"`
	Username          string     `json:"username"`
	PlayCount         int64      `json:"play_count"`
	FreePlayCount     int64      `json:"free_play_count"`
	PaidPlayCount     int64      `json:"paid_play_count"`
	GrossRewardAmount FloatLike  `json:"gross_reward_amount"`
	EntryFeeAmount    FloatLike  `json:"entry_fee_amount"`
	NetRewardAmount   FloatLike  `json:"net_reward_amount"`
	PendingClaimCount int64      `json:"pending_claim_count"`
	FailedClaimCount  int64      `json:"failed_claim_count"`
}
