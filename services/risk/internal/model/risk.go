package model

import "time"

const (
	RuleCodeBurstRequests = "burst_requests"
	RuleCodeCostSpike     = "cost_spike"
	RuleCodeUniqueIP      = "unique_ip"

	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityHigh     = "high"
	SeverityCritical = "critical"

	RiskLevelNormal  = "normal"
	RiskLevelWarning = "warning"
	RiskLevelHigh    = "high"

	EventStatusOpen         = "open"
	EventStatusAcknowledged = "acknowledged"
	EventStatusResolved     = "resolved"
	EventStatusIgnored      = "ignored"
)

type DashboardOverview struct {
	HighRiskUsers     int64      `json:"high_risk_users"`
	WarningUsers      int64      `json:"warning_users"`
	EventsToday       int64      `json:"events_today"`
	EventsUnprocessed int64      `json:"events_unprocessed"`
	StaleUsers        int64      `json:"stale_users"`
	LastRefreshAt     *time.Time `json:"last_refresh_at"`
	WindowStart       *time.Time `json:"window_start"`
	WindowEnd         *time.Time `json:"window_end"`
}

type OperationsSignSummary struct {
	TodaySuccessUsers int64      `json:"today_success_users"`
	TodaySuccessCount int64      `json:"today_success_count"`
	TodayTotalReward  float64    `json:"today_total_reward"`
	TodayAvgReward    float64    `json:"today_avg_reward"`
	TodayPendingCount int64      `json:"today_pending_count"`
	TodayFailedCount  int64      `json:"today_failed_count"`
	LastCheckinAt     *time.Time `json:"last_checkin_at"`
}

type OperationsPaymentSummary struct {
	TodayCreatedOrders  int64   `json:"today_created_orders"`
	TodayCreatedAmount  float64 `json:"today_created_amount"`
	TodayPaidOrders     int64   `json:"today_paid_orders"`
	TodayPaidAmount     float64 `json:"today_paid_amount"`
	TodayPaidUsers      int64   `json:"today_paid_users"`
	TodayCanceledOrders int64   `json:"today_canceled_orders"`
	AvgPaidOrderAmount  float64 `json:"avg_paid_order_amount"`
}

type OperationsConversionSummary struct {
	SignedUsers24h         int64   `json:"signed_users_24h"`
	PaidUsers24h           int64   `json:"paid_users_24h"`
	SignedThenPaidUsers24h int64   `json:"signed_then_paid_users_24h"`
	SignedToPaidRate24h    float64 `json:"signed_to_paid_rate_24h"`
}

type OperationsUsageSummary struct {
	Requests24h     int64   `json:"requests_24h"`
	ActiveUsers24h  int64   `json:"active_users_24h"`
	TotalBilled24h  float64 `json:"total_billed_24h"`
	TotalActual24h  float64 `json:"total_actual_24h"`
	AvgLatencyMs    float64 `json:"avg_latency_ms"`
	P90LatencyMs    float64 `json:"p90_latency_ms"`
	AvgFirstTokenMs float64 `json:"avg_first_token_ms"`
	P90FirstTokenMs float64 `json:"p90_first_token_ms"`
}

type OperationsTrendPoint struct {
	Date             string  `json:"date"`
	SignUsers        int64   `json:"sign_users"`
	SignRewardTotal  float64 `json:"sign_reward_total"`
	PaidOrders       int64   `json:"paid_orders"`
	PaidAmount       float64 `json:"paid_amount"`
	UsageRequests    int64   `json:"usage_requests"`
	UsageActiveUsers int64   `json:"usage_active_users"`
}

type OperationsUserRankItem struct {
	UserID       int64      `json:"user_id"`
	UserLabel    string     `json:"user_label"`
	MetricValue  float64    `json:"metric_value"`
	RequestCount int64      `json:"request_count,omitempty"`
	PaidOrders   int64      `json:"paid_orders,omitempty"`
	LastActiveAt *time.Time `json:"last_active_at,omitempty"`
}

type OperationsOverview struct {
	Sign       OperationsSignSummary       `json:"sign"`
	Payment    OperationsPaymentSummary    `json:"payment"`
	Conversion OperationsConversionSummary `json:"conversion"`
	Usage      OperationsUsageSummary      `json:"usage"`
	Trends     []OperationsTrendPoint      `json:"trends"`
	TopPayers  []OperationsUserRankItem    `json:"top_payers"`
	TopCost    []OperationsUserRankItem    `json:"top_cost"`
}

type RiskUserSnapshot struct {
	ID              int64      `json:"id"`
	UserID          string     `json:"user_id"`
	RiskScore       int        `json:"risk_score"`
	RiskLevel       string     `json:"risk_level"`
	RequestCount1h  int64      `json:"request_count_1h"`
	RequestCount24h int64      `json:"request_count_24h"`
	Cost1h          float64    `json:"cost_1h"`
	Cost24h         float64    `json:"cost_24h"`
	UniqueIP24h     int        `json:"unique_ip_24h"`
	TopModels       []string   `json:"top_models"`
	RiskTags        []string   `json:"risk_tags"`
	LastEventAt     *time.Time `json:"last_event_at"`
	WindowStart     *time.Time `json:"window_start"`
	WindowEnd       *time.Time `json:"window_end"`
	RefreshedAt     *time.Time `json:"refreshed_at"`
	IsStale         bool       `json:"is_stale"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type RiskEvent struct {
	ID               int64          `json:"id"`
	UserID           string         `json:"user_id"`
	APIKeyID         string         `json:"api_key_id,omitempty"`
	EventFingerprint string         `json:"event_fingerprint"`
	EventType        string         `json:"event_type"`
	Status           string         `json:"status"`
	Severity         string         `json:"severity"`
	ScoreDelta       int            `json:"score_delta"`
	Title            string         `json:"title"`
	Detail           map[string]any `json:"detail"`
	FirstSeenAt      time.Time      `json:"first_seen_at"`
	LastSeenAt       time.Time      `json:"last_seen_at"`
	HitCount         int            `json:"hit_count"`
	OccurredAt       time.Time      `json:"occurred_at"`
	CreatedAt        time.Time      `json:"created_at"`
}

type UserListFilter struct {
	Page         int
	PageSize     int
	UserID       string
	RiskLevel    string
	IncludeStale bool
}

type EventListFilter struct {
	Page      int
	PageSize  int
	UserID    string
	Severity  string
	EventType string
	Status    string
}

type UserRiskAggregate struct {
	UserID          string
	APIKeyID        string
	RequestCount1h  int64
	RequestCount24h int64
	Cost1h          float64
	Cost24h         float64
	UniqueIP24h     int
	TopModels       []string
	WindowEnd       time.Time
}

type RuleHitInput struct {
	EventFingerprint string
	RuleCode         string
	MetricValue      float64
	ThresholdValue   float64
	Severity         string
	ScoreDelta       int
	Title            string
	Detail           map[string]any
}

type RiskSnapshotUpsertInput struct {
	UserID          string
	RiskScore       int
	RiskLevel       string
	RequestCount1h  int64
	RequestCount24h int64
	Cost1h          float64
	Cost24h         float64
	UniqueIP24h     int
	TopModels       []string
	RiskTags        []string
	LastEventAt     *time.Time
	WindowStart     time.Time
	WindowEnd       time.Time
	RefreshedAt     time.Time
	IsStale         bool
}

type RiskEventCreateInput struct {
	UserID           string
	APIKeyID         string
	EventFingerprint string
	EventType        string
	Status           string
	Severity         string
	ScoreDelta       int
	Title            string
	Detail           map[string]any
	FirstSeenAt      time.Time
	LastSeenAt       time.Time
	HitCount         int
	OccurredAt       time.Time
}

type RiskRuleHitCreateInput struct {
	EventID        int64
	RuleCode       string
	MetricValue    float64
	ThresholdValue float64
	Extra          map[string]any
}

type RiskRefreshStats struct {
	ScannedUsers    int `json:"scanned_users"`
	UpdatedUsers    int `json:"updated_users"`
	CreatedEvents   int `json:"created_events"`
	CreatedRuleHits int `json:"created_rule_hits"`
	StaleUsers      int `json:"stale_users"`
}

type RiskThresholds struct {
	BurstRequests1h  int
	CostSpikeRatio   float64
	CostSpikeFloor1h float64
	UniqueIP24h      int
}
