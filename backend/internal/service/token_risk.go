package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	TokenRiskLevelLow      = "low"
	TokenRiskLevelMedium   = "medium"
	TokenRiskLevelHigh     = "high"
	TokenRiskLevelCritical = "critical"

	TokenRiskStatusOpen          = "open"
	TokenRiskStatusHandled       = "handled"
	TokenRiskStatusFalsePositive = "false_positive"
	TokenRiskStatusWatching      = "watching"
)

type TokenRiskRepository interface {
	UpsertTokenRiskEvent(ctx context.Context, event *TokenRiskEvent) (*TokenRiskEvent, error)
	ListTokenRiskEvents(ctx context.Context, filter TokenRiskEventFilter) ([]*TokenRiskEvent, int64, error)
	GetTokenRiskEvent(ctx context.Context, id int64) (*TokenRiskEvent, error)
	ListRelatedContentModerationLogs(ctx context.Context, event *TokenRiskEvent, limit int) ([]*TokenRiskRelatedContentLog, error)
	UpdateTokenRiskEventStatus(ctx context.Context, id int64, status string, falsePositive bool, actorUserID int64) error
	CreateTokenRiskAction(ctx context.Context, action *TokenRiskAction) (*TokenRiskAction, error)
	ListTokenRiskActions(ctx context.Context, eventID int64) ([]*TokenRiskAction, error)
	UpsertTokenRiskWatchlist(ctx context.Context, item *TokenRiskWatchlistItem) (*TokenRiskWatchlistItem, error)
	DeactivateTokenRiskWatchlist(ctx context.Context, id int64, actorUserID int64) error
	ListTokenRiskWatchlist(ctx context.Context, activeOnly bool) ([]*TokenRiskWatchlistItem, error)
	GetTokenRiskSummary(ctx context.Context, since time.Time) (*TokenRiskSummary, error)
}

type TokenRiskEvent struct {
	ID                 int64      `json:"id"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	LastSeenAt         time.Time  `json:"last_seen_at"`
	SourceLogID        *int64     `json:"source_log_id,omitempty"`
	UserID             *int64     `json:"user_id,omitempty"`
	APIKeyID           *int64     `json:"api_key_id,omitempty"`
	TokenType          string     `json:"token_type"`
	TokenHash          string     `json:"token_hash"`
	TokenPrefix        string     `json:"token_prefix"`
	TokenSuffix        string     `json:"token_suffix"`
	APIKeySummary      string     `json:"api_key_summary"`
	ClientIP           string     `json:"client_ip"`
	UserAgent          string     `json:"user_agent"`
	Method             string     `json:"method"`
	Path               string     `json:"path"`
	StatusCode         int        `json:"status_code"`
	Result             string     `json:"result"`
	FailureReason      string     `json:"failure_reason"`
	RiskScore          int        `json:"risk_score"`
	RiskLevel          string     `json:"risk_level"`
	RiskCategories     []string   `json:"risk_categories"`
	MatchedRules       []string   `json:"matched_rules"`
	RecommendedActions []string   `json:"recommended_actions"`
	Explanation        string     `json:"explanation"`
	Status             string     `json:"status"`
	FalsePositive      bool       `json:"false_positive"`
	HandledByUserID    *int64     `json:"handled_by_user_id,omitempty"`
	HandledAt          *time.Time `json:"handled_at,omitempty"`
	Count5m            int64      `json:"count_5m,omitempty"`
	Count1h            int64      `json:"count_1h,omitempty"`
	Count24h           int64      `json:"count_24h,omitempty"`
	DistinctIP24h      int64      `json:"distinct_ip_24h,omitempty"`
}

type TokenRiskAction struct {
	ID          int64          `json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	EventID     int64          `json:"event_id"`
	ActorUserID int64          `json:"actor_user_id"`
	Action      string         `json:"action"`
	Note        string         `json:"note"`
	Result      string         `json:"result"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type TokenRiskEventDetail struct {
	Event              *TokenRiskEvent               `json:"event"`
	Actions            []*TokenRiskAction            `json:"actions"`
	RelatedContentLogs []*TokenRiskRelatedContentLog `json:"related_content_logs"`
	RelatedActivity    TokenRiskRelatedActivity      `json:"related_activity"`
	HumanExplanation   TokenRiskHumanExplanation     `json:"human_explanation"`
}

type TokenRiskRelatedActivity struct {
	Count5m       int64 `json:"count_5m"`
	Count1h       int64 `json:"count_1h"`
	Count24h      int64 `json:"count_24h"`
	DistinctIP24h int64 `json:"distinct_ip_24h"`
}

type TokenRiskHumanExplanation struct {
	Summary              string   `json:"summary"`
	Reasons              []string `json:"reasons"`
	RecommendedNextSteps []string `json:"recommended_next_steps"`
	ContentAvailability  string   `json:"content_availability"`
}

type TokenRiskRelatedContentLog struct {
	ID              int64     `json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	RequestID       string    `json:"request_id"`
	UserID          *int64    `json:"user_id,omitempty"`
	APIKeyID        *int64    `json:"api_key_id,omitempty"`
	Endpoint        string    `json:"endpoint"`
	Provider        string    `json:"provider"`
	Model           string    `json:"model"`
	Action          string    `json:"action"`
	Flagged         bool      `json:"flagged"`
	HighestCategory string    `json:"highest_category"`
	HighestScore    float64   `json:"highest_score"`
	InputExcerpt    string    `json:"input_excerpt"`
	ViolationCount  int       `json:"violation_count"`
	AutoBanned      bool      `json:"auto_banned"`
}

type TokenRiskWatchlistItem struct {
	ID           int64     `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	SubjectType  string    `json:"subject_type"`
	SubjectValue string    `json:"subject_value"`
	Reason       string    `json:"reason"`
	ActorUserID  int64     `json:"actor_user_id"`
	Active       bool      `json:"active"`
}

type TokenRiskEventFilter struct {
	Since        *time.Time
	Until        *time.Time
	RiskLevel    string
	RiskCategory string
	TokenType    string
	Status       string
	UserID       *int64
	APIKeyID     *int64
	Query        string
	Page         int
	PageSize     int
}

type TokenRiskSummary struct {
	Total           int64                  `json:"total"`
	Open            int64                  `json:"open"`
	Handled         int64                  `json:"handled"`
	FalsePositive   int64                  `json:"false_positive"`
	High            int64                  `json:"high"`
	Critical        int64                  `json:"critical"`
	DistinctUsers   int64                  `json:"distinct_users"`
	DistinctTokens  int64                  `json:"distinct_tokens"`
	DistinctAPIKeys int64                  `json:"distinct_api_keys"`
	ByLevel         map[string]int64       `json:"by_level"`
	ByCategory      map[string]int64       `json:"by_category"`
	TopUsers        []TokenRiskSubjectStat `json:"top_users"`
	TopTokens       []TokenRiskSubjectStat `json:"top_tokens"`
	TopAPIKeys      []TokenRiskSubjectStat `json:"top_api_keys"`
	RecentHighRisk  []*TokenRiskEvent      `json:"recent_high_risk"`
}

type TokenRiskSubjectStat struct {
	Subject string `json:"subject"`
	UserID  *int64 `json:"user_id,omitempty"`
	Count   int64  `json:"count"`
	Score   int64  `json:"score"`
}

type TokenRiskAnalyzeContext struct {
	Count5m       int64
	Count1h       int64
	Count24h      int64
	DistinctIP24h int64
}

type TokenRiskService struct {
	repo                TokenRiskRepository
	opsRepo             OpsRepository
	announcementService *AnnouncementService
	userService         *UserService
	apiKeyService       *APIKeyService
}

func NewTokenRiskService(repo TokenRiskRepository, opsRepo OpsRepository, announcementService *AnnouncementService, userService *UserService, apiKeyService *APIKeyService) *TokenRiskService {
	return &TokenRiskService{
		repo:                repo,
		opsRepo:             opsRepo,
		announcementService: announcementService,
		userService:         userService,
		apiKeyService:       apiKeyService,
	}
}

func (s *TokenRiskService) IngestFromOpsLog(ctx context.Context, log *OpsSystemLog) (*TokenRiskEvent, error) {
	if s == nil || s.repo == nil || log == nil || log.Component != "audit.token" {
		return nil, nil
	}
	event := BuildTokenRiskEventFromOpsLog(log, TokenRiskAnalyzeContext{})
	if event == nil {
		return nil, nil
	}
	return s.repo.UpsertTokenRiskEvent(ctx, event)
}

func (s *TokenRiskService) BackfillFromOpsLogs(ctx context.Context, since time.Time, limit int) (int64, error) {
	if s == nil || s.repo == nil || s.opsRepo == nil {
		return 0, nil
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	filter := &OpsSystemLogFilter{
		StartTime: &since,
		Component: "audit.token",
		Page:      1,
		PageSize:  limit,
	}
	list, err := s.opsRepo.ListSystemLogs(ctx, filter)
	if err != nil {
		return 0, err
	}
	var count int64
	for _, item := range list.Logs {
		if item == nil {
			continue
		}
		if _, err := s.IngestFromOpsLog(ctx, item); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (s *TokenRiskService) ListEvents(ctx context.Context, filter TokenRiskEventFilter) ([]*TokenRiskEvent, int64, error) {
	if s == nil || s.repo == nil {
		return nil, 0, nil
	}
	items, total, err := s.repo.ListTokenRiskEvents(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	for _, item := range items {
		enrichTokenRiskEventRuntime(item)
	}
	return items, total, nil
}

func (s *TokenRiskService) GetEvent(ctx context.Context, id int64) (*TokenRiskEvent, []*TokenRiskAction, error) {
	if s == nil || s.repo == nil {
		return nil, nil, infraerrors.ServiceUnavailable("TOKEN_RISK_NOT_READY", "token risk service is not ready")
	}
	event, err := s.repo.GetTokenRiskEvent(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	actions, err := s.repo.ListTokenRiskActions(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return event, actions, nil
}

func (s *TokenRiskService) GetEventDetail(ctx context.Context, id int64) (*TokenRiskEventDetail, error) {
	event, actions, err := s.GetEvent(ctx, id)
	if err != nil {
		return nil, err
	}
	enrichTokenRiskEventRuntime(event)

	relatedLogs := []*TokenRiskRelatedContentLog{}
	if s != nil && s.repo != nil {
		relatedLogs, err = s.repo.ListRelatedContentModerationLogs(ctx, event, 5)
		if err != nil {
			return nil, err
		}
	}
	return &TokenRiskEventDetail{
		Event:              event,
		Actions:            actions,
		RelatedContentLogs: relatedLogs,
		RelatedActivity: TokenRiskRelatedActivity{
			Count5m:       event.Count5m,
			Count1h:       event.Count1h,
			Count24h:      event.Count24h,
			DistinctIP24h: event.DistinctIP24h,
		},
		HumanExplanation: buildTokenRiskHumanExplanation(event, relatedLogs),
	}, nil
}

func (s *TokenRiskService) Summary(ctx context.Context, since time.Time) (*TokenRiskSummary, error) {
	if s == nil || s.repo == nil {
		return &TokenRiskSummary{}, nil
	}
	return s.repo.GetTokenRiskSummary(ctx, since)
}

func (s *TokenRiskService) ListWatchlist(ctx context.Context, activeOnly bool) ([]*TokenRiskWatchlistItem, error) {
	if s == nil || s.repo == nil {
		return nil, infraerrors.ServiceUnavailable("TOKEN_RISK_NOT_READY", "token risk service is not ready")
	}
	return s.repo.ListTokenRiskWatchlist(ctx, activeOnly)
}

func (s *TokenRiskService) AddWatchlist(ctx context.Context, actorUserID int64, subjectType, subjectValue, reason string) (*TokenRiskWatchlistItem, error) {
	if s == nil || s.repo == nil {
		return nil, infraerrors.ServiceUnavailable("TOKEN_RISK_NOT_READY", "token risk service is not ready")
	}
	subjectType = strings.TrimSpace(subjectType)
	subjectValue = strings.TrimSpace(subjectValue)
	if actorUserID <= 0 || subjectType == "" || subjectValue == "" {
		return nil, infraerrors.BadRequest("TOKEN_RISK_WATCHLIST_INVALID", "invalid watchlist item")
	}
	switch subjectType {
	case "user", "token_hash", "api_key", "ip":
	default:
		return nil, infraerrors.BadRequest("TOKEN_RISK_WATCHLIST_TYPE_INVALID", "unsupported watchlist subject type")
	}
	return s.repo.UpsertTokenRiskWatchlist(ctx, &TokenRiskWatchlistItem{
		SubjectType:  subjectType,
		SubjectValue: truncateTokenRiskString(subjectValue, 256),
		Reason:       truncateTokenRiskString(reason, 1000),
		ActorUserID:  actorUserID,
		Active:       true,
	})
}

func (s *TokenRiskService) RemoveWatchlist(ctx context.Context, id int64, actorUserID int64) error {
	if s == nil || s.repo == nil {
		return infraerrors.ServiceUnavailable("TOKEN_RISK_NOT_READY", "token risk service is not ready")
	}
	if id <= 0 || actorUserID <= 0 {
		return infraerrors.BadRequest("TOKEN_RISK_WATCHLIST_INVALID", "invalid watchlist item")
	}
	return s.repo.DeactivateTokenRiskWatchlist(ctx, id, actorUserID)
}

func (s *TokenRiskService) ApplyAction(ctx context.Context, eventID int64, actorUserID int64, actionName string, note string, confirm bool) (*TokenRiskAction, error) {
	if s == nil || s.repo == nil {
		return nil, infraerrors.ServiceUnavailable("TOKEN_RISK_NOT_READY", "token risk service is not ready")
	}
	actionName = strings.TrimSpace(actionName)
	if eventID <= 0 || actorUserID <= 0 || actionName == "" {
		return nil, infraerrors.BadRequest("TOKEN_RISK_ACTION_INVALID", "invalid token risk action")
	}
	event, err := s.repo.GetTokenRiskEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}

	result := "recorded"
	metadata := map[string]any{}
	switch actionName {
	case "mark_handled":
		err = s.repo.UpdateTokenRiskEventStatus(ctx, eventID, TokenRiskStatusHandled, false, actorUserID)
		result = "handled"
	case "mark_false_positive":
		err = s.repo.UpdateTokenRiskEventStatus(ctx, eventID, TokenRiskStatusFalsePositive, true, actorUserID)
		result = "false_positive"
	case "watch_user":
		if event.UserID == nil || *event.UserID <= 0 {
			err = infraerrors.BadRequest("TOKEN_RISK_NO_USER", "event has no user id")
			break
		}
		_, err = s.repo.UpsertTokenRiskWatchlist(ctx, &TokenRiskWatchlistItem{
			SubjectType:  "user",
			SubjectValue: strconv.FormatInt(*event.UserID, 10),
			Reason:       note,
			ActorUserID:  actorUserID,
			Active:       true,
		})
		result = "watching"
	case "watch_token":
		if event.TokenHash == "" {
			err = infraerrors.BadRequest("TOKEN_RISK_NO_TOKEN_HASH", "event has no token hash")
			break
		}
		_, err = s.repo.UpsertTokenRiskWatchlist(ctx, &TokenRiskWatchlistItem{
			SubjectType:  "token_hash",
			SubjectValue: event.TokenHash,
			Reason:       note,
			ActorUserID:  actorUserID,
			Active:       true,
		})
		result = "watching"
	case "force_relogin":
		if !confirm {
			err = infraerrors.BadRequest("TOKEN_RISK_CONFIRM_REQUIRED", "high-risk action requires confirmation")
			break
		}
		if event.UserID == nil || *event.UserID <= 0 || s.userService == nil {
			err = infraerrors.BadRequest("TOKEN_RISK_NO_USER", "event has no user id")
			break
		}
		err = s.userService.UpdateStatus(ctx, *event.UserID, StatusActive)
		metadata["note"] = "force relogin is audit-only unless token_version enforcement is configured"
		result = "recorded"
	case "send_warning", "send_reminder":
		if event.UserID == nil || *event.UserID <= 0 || s.announcementService == nil {
			err = infraerrors.BadRequest("TOKEN_RISK_NO_USER", "event has no user id")
			break
		}
		title := "账户安全提醒"
		if actionName == "send_warning" {
			title = "账户异常使用警告"
		}
		_, err = s.announcementService.CreateDirect(ctx, &CreateDirectAnnouncementInput{
			TargetUserID: *event.UserID,
			Title:        title,
			Content:      buildTokenRiskNoticeContentSafe(event),
			NotifyMode:   "popup",
			ActorID:      &actorUserID,
		})
		result = "sent"
	default:
		err = infraerrors.BadRequest("TOKEN_RISK_ACTION_UNSUPPORTED", "unsupported token risk action")
	}
	if err != nil {
		return nil, err
	}

	action := &TokenRiskAction{
		EventID:     eventID,
		ActorUserID: actorUserID,
		Action:      actionName,
		Note:        truncateTokenRiskString(note, 1000),
		Result:      result,
		Metadata:    metadata,
	}
	return s.repo.CreateTokenRiskAction(ctx, action)
}

func BuildTokenRiskEventFromOpsLog(log *OpsSystemLog, ctx TokenRiskAnalyzeContext) *TokenRiskEvent {
	if log == nil || log.Extra == nil {
		return nil
	}
	tokenType := extraString(log.Extra, "token_type")
	result := extraString(log.Extra, "result")
	failure := extraString(log.Extra, "failure_reason")
	path := extraString(log.Extra, "path")
	statusCode := extraInt(log.Extra, "status_code")
	categories := map[string]bool{}
	rules := map[string]bool{}
	score := 0

	add := func(category, rule string, delta int) {
		if category != "" {
			categories[category] = true
		}
		if rule != "" {
			rules[rule] = true
		}
		score += delta
	}

	rule := extraString(log.Extra, "rule")
	if rule != "" {
		rules[rule] = true
	}

	normalizedFailure := strings.ToLower(failure + " " + rule + " " + result)
	normalizedPath := strings.ToLower(path)
	if strings.Contains(normalizedFailure, "invalid_authorization") || strings.Contains(normalizedFailure, "invalid token") {
		add("auth_invalid", "invalid_authorization_header", 25)
	}
	if strings.Contains(normalizedFailure, "expired") {
		add("auth_expired", "expired_token", 25)
	}
	if strings.Contains(normalizedFailure, "signature") || strings.Contains(normalizedFailure, "forged") || strings.Contains(normalizedFailure, "malformed") {
		add("auth_forged", "token_signature_or_shape_invalid", 45)
	}
	if strings.Contains(normalizedFailure, "permission") || strings.Contains(normalizedFailure, "forbidden") || statusCode == 403 {
		add("permission_violation", "permission_denied", 35)
	}
	if strings.HasPrefix(normalizedPath, "/api/v1/admin") || strings.Contains(normalizedPath, "/admin/") {
		if tokenType == "jwt" || tokenType == "api_key" {
			add("admin_api_probe", "non_admin_token_admin_path", 45)
		} else {
			add("config_tamper", "admin_sensitive_path", 20)
		}
	}
	if strings.Contains(normalizedPath, "register") || strings.Contains(normalizedPath, "invite") || strings.Contains(normalizedPath, "redeem") {
		add("batch_register", "registration_related_path", 25)
	}
	if strings.Contains(normalizedPath, "registrar") || strings.Contains(normalizedPath, "signup") {
		add("registrar_abuse", "registrar_related_path", 25)
	}
	if strings.Contains(normalizedPath, "settings") || strings.Contains(normalizedPath, "config") || strings.Contains(normalizedPath, "risk-control") {
		add("config_tamper", "sensitive_config_path", 35)
	}
	if strings.Contains(normalizedPath, "balance") || strings.Contains(normalizedPath, "reward") || strings.Contains(normalizedPath, "affiliate") {
		add("balance_or_reward_abuse", "balance_reward_path", 35)
	}
	if strings.Contains(normalizedPath, "game") || strings.Contains(normalizedPath, "claim") {
		add("game_abuse", "game_reward_path", 30)
	}
	if strings.Contains(normalizedPath, "embed") && (strings.Contains(normalizedFailure, "k") || strings.Contains(normalizedFailure, "token")) {
		add("embedded_bypass", "embedded_auth_bypass", 45)
	}
	if strings.Contains(normalizedFailure, "insufficient_balance") {
		if ctx.Count5m >= 5 || ctx.Count1h >= 20 {
			add("insufficient_balance_abuse", "insufficient_balance_repeated", 30)
		} else {
			add("balance_or_reward_abuse", "insufficient_balance_single", 15)
		}
	}
	if ctx.Count5m >= 30 || ctx.Count1h >= 120 {
		add("high_frequency", "high_frequency_window", 35)
	}
	if ctx.DistinctIP24h >= 4 && (tokenType == "api_key" || tokenType == "admin_api_key") {
		add("api_key_sharing", "multi_ip_api_key_usage", 40)
	}
	if looksLikePathScan(normalizedPath) {
		add("suspicious_path_scan", "sensitive_path_scan", 25)
	}
	if looksAdultOrGrey(normalizedFailure + " " + normalizedPath) {
		add("grey_industry", "sensitive_business_keyword", 35)
	}

	if statusCode >= 500 {
		score += 5
	}
	if result == "denied" {
		score += 10
	}
	if score == 0 {
		score = 10
		add("auth_invalid", "token_audit_observed", 0)
	}
	if score > 100 {
		score = 100
	}

	categoryList := mapKeys(categories)
	ruleList := mapKeys(rules)
	level := riskLevelFromScore(score)
	sourceLogID := log.ID
	event := &TokenRiskEvent{
		CreatedAt:          log.CreatedAt,
		UpdatedAt:          log.CreatedAt,
		LastSeenAt:         log.CreatedAt,
		SourceLogID:        &sourceLogID,
		UserID:             log.UserID,
		TokenType:          tokenType,
		TokenHash:          extraString(log.Extra, "token_hash"),
		TokenPrefix:        extraString(log.Extra, "token_prefix"),
		TokenSuffix:        extraString(log.Extra, "token_suffix"),
		APIKeySummary:      maskAPIKeySummary(extraString(log.Extra, "token_prefix"), extraString(log.Extra, "token_suffix")),
		ClientIP:           extraString(log.Extra, "client_ip"),
		UserAgent:          truncateTokenRiskString(extraString(log.Extra, "user_agent"), 512),
		Method:             extraString(log.Extra, "method"),
		Path:               path,
		StatusCode:         statusCode,
		Result:             result,
		FailureReason:      failure,
		RiskScore:          score,
		RiskLevel:          level,
		RiskCategories:     categoryList,
		MatchedRules:       ruleList,
		RecommendedActions: recommendedActions(level, categoryList),
		Explanation:        buildRiskExplanation(categoryList, ruleList, score),
		Status:             TokenRiskStatusOpen,
		Count5m:            ctx.Count5m,
		Count1h:            ctx.Count1h,
		Count24h:           ctx.Count24h,
		DistinctIP24h:      ctx.DistinctIP24h,
	}
	if apiKeyID := extraInt64(log.Extra, "api_key_id"); apiKeyID > 0 {
		event.APIKeyID = &apiKeyID
	}
	return event
}

func enrichTokenRiskEventRuntime(event *TokenRiskEvent) {
	if event == nil {
		return
	}
	if containsString(event.MatchedRules, "insufficient_balance") && !containsString(event.MatchedRules, "insufficient_balance_repeated") {
		event.MatchedRules = replaceString(event.MatchedRules, "insufficient_balance", "insufficient_balance_single")
	}
	// 历史事件写入时可能没有聚合上下文；列表和详情读取时按当前聚合指标补齐风险分类。
	repeatedInsufficientBalance := event.Count5m >= 5 || event.Count1h >= 20
	if containsString(event.MatchedRules, "insufficient_balance_single") && repeatedInsufficientBalance {
		event.RiskCategories = appendUniqueString(removeString(event.RiskCategories, "balance_or_reward_abuse"), "insufficient_balance_abuse")
		event.MatchedRules = appendUniqueString(removeString(event.MatchedRules, "insufficient_balance_single"), "insufficient_balance_repeated")
		if event.RiskScore < 60 {
			event.RiskScore = 60
		}
	}
	if (event.TokenType == "api_key" || event.TokenType == "admin_api_key") && event.DistinctIP24h >= 4 {
		event.RiskCategories = appendUniqueString(event.RiskCategories, "api_key_sharing")
		event.MatchedRules = appendUniqueString(event.MatchedRules, "multi_ip_api_key_usage")
		if event.RiskScore < 75 {
			event.RiskScore = 75
		}
	}
	if event.Count5m >= 30 || event.Count1h >= 120 {
		event.RiskCategories = appendUniqueString(event.RiskCategories, "high_frequency")
		event.MatchedRules = appendUniqueString(event.MatchedRules, "rpm_anomaly_window")
		if event.RiskScore < 65 {
			event.RiskScore = 65
		}
	}
	event.RiskLevel = riskLevelFromScore(event.RiskScore)
	event.RiskCategories = uniqueSortedStrings(event.RiskCategories)
	event.MatchedRules = uniqueSortedStrings(event.MatchedRules)
	event.RecommendedActions = recommendedActions(event.RiskLevel, event.RiskCategories)
	event.Explanation = buildRiskExplanation(event.RiskCategories, event.MatchedRules, event.RiskScore)
}

func riskLevelFromScore(score int) string {
	switch {
	case score >= 85:
		return TokenRiskLevelCritical
	case score >= 60:
		return TokenRiskLevelHigh
	case score >= 25:
		return TokenRiskLevelMedium
	default:
		return TokenRiskLevelLow
	}
}

func recommendedActions(level string, categories []string) []string {
	actions := map[string]bool{"mark_handled": true, "mark_false_positive": true}
	for _, c := range categories {
		switch c {
		case "balance_or_reward_abuse":
			actions["send_reminder"] = true
		case "permission_violation", "admin_api_probe", "auth_forged", "embedded_bypass":
			actions["watch_user"] = true
			actions["force_relogin"] = true
			actions["send_warning"] = true
		case "api_key_sharing", "high_frequency", "insufficient_balance_abuse":
			actions["watch_token"] = true
			actions["send_reminder"] = true
		default:
			actions["watch_user"] = true
		}
	}
	if level == TokenRiskLevelCritical {
		actions["send_warning"] = true
	}
	return mapKeys(actions)
}

func buildRiskExplanation(categories, rules []string, score int) string {
	return fmt.Sprintf("风险分 %d；分类：%s；规则：%s", score, strings.Join(localizeTokenRiskList(categories, tokenRiskCategoryLabels), "、"), strings.Join(localizeTokenRiskList(rules, tokenRiskRuleLabels), "、"))
}

func buildTokenRiskHumanExplanation(event *TokenRiskEvent, contentLogs []*TokenRiskRelatedContentLog) TokenRiskHumanExplanation {
	if event == nil {
		return TokenRiskHumanExplanation{}
	}
	summary := fmt.Sprintf("%s %s 返回 HTTP %d，主体 %s。", tokenRiskFirstNonEmpty(event.Method, "-"), tokenRiskFirstNonEmpty(event.Path, "-"), event.StatusCode, tokenRiskSubjectSummary(event))
	if event.StatusCode == 0 {
		summary = fmt.Sprintf("%s %s 被记录为 %s，主体 %s。", tokenRiskFirstNonEmpty(event.Method, "-"), tokenRiskFirstNonEmpty(event.Path, "-"), tokenRiskFirstNonEmpty(event.Result, "异常"), tokenRiskSubjectSummary(event))
	}

	reasons := []string{fmt.Sprintf("风险分 %d，等级 %s。", event.RiskScore, tokenRiskLevelLabel(event.RiskLevel))}
	for _, category := range event.RiskCategories {
		if label := tokenRiskCategoryLabels[category]; label != "" {
			reasons = append(reasons, label)
		}
	}
	if event.Count5m > 0 || event.Count1h > 0 || event.Count24h > 0 {
		rpm := float64(event.Count5m) / 5
		reasons = append(reasons, fmt.Sprintf("同主体近期频率：5 分钟 %d 次（约 %.1f RPM），1 小时 %d 次，24 小时 %d 次，24 小时来源 IP %d 个。", event.Count5m, rpm, event.Count1h, event.Count24h, event.DistinctIP24h))
	}
	if event.FailureReason != "" {
		reasons = append(reasons, "失败原因："+event.FailureReason)
	}

	contentAvailability := "该事件没有可关联的内容审核摘要。/v1/models 等无正文接口本身不会产生 prompt 内容；历史未记录的请求内容无法恢复。"
	if len(contentLogs) > 0 {
		contentAvailability = fmt.Sprintf("找到 %d 条相关内容审核记录，仅展示脱敏摘要、分类和分数，不展示完整请求原文。", len(contentLogs))
	}
	return TokenRiskHumanExplanation{
		Summary:              summary,
		Reasons:              uniqueOrderedStrings(reasons),
		RecommendedNextSteps: recommendedStepTexts(event),
		ContentAvailability:  contentAvailability,
	}
}

func buildTokenRiskNoticeContentSafe(event *TokenRiskEvent) string {
	if event == nil {
		return "系统检测到账户存在异常使用风险，请检查近期操作。"
	}
	return fmt.Sprintf("系统检测到账户近期存在异常使用风险。请求：%s %s；风险：%s。请确认 API key/token 未被共享或泄露。", event.Method, event.Path, strings.Join(localizeTokenRiskList(event.RiskCategories, tokenRiskCategoryLabels), "、"))
}

func recommendedStepTexts(event *TokenRiskEvent) []string {
	if event == nil {
		return nil
	}
	steps := []string{}
	if containsString(event.MatchedRules, "insufficient_balance_single") || containsString(event.RiskCategories, "balance_or_reward_abuse") {
		steps = append(steps, "先检查用户余额、API key 权限、分组和模型权限；单次失败通常是配置或余额问题。")
	}
	if containsString(event.RiskCategories, "insufficient_balance_abuse") {
		steps = append(steps, "余额不足仍持续重试，检查客户端重试策略；必要时观察或暂停该 API key。")
	}
	if containsString(event.RiskCategories, "permission_violation") {
		steps = append(steps, "核对该 token/API key 是否有访问该接口、分组或模型的权限。")
	}
	if containsString(event.RiskCategories, "admin_api_probe") {
		steps = append(steps, "普通主体触达管理接口，优先观察用户并检查是否存在越权探测。")
	}
	if containsString(event.RiskCategories, "high_frequency") {
		steps = append(steps, "请求频率异常，检查客户端循环重试、脚本调用或共享 key。")
	}
	if containsString(event.RiskCategories, "api_key_sharing") {
		steps = append(steps, "同一 API key 出现多来源 IP，建议确认是否被共享、倒卖或泄露。")
	}
	if containsString(event.RiskCategories, "embedded_bypass") {
		steps = append(steps, "embedded 鉴权异常，检查 k、token、src_host 和入口来源。")
	}
	if containsString(event.RiskCategories, "grey_industry") {
		steps = append(steps, "疑似敏感业务风险，只查看脱敏摘要和分类，必要时发送警告并加入观察。")
	}
	if len(steps) == 0 {
		steps = append(steps, "先查看同用户、同 IP、同 token hash 的历史行为，再决定标记已处理、误报或观察。")
	}
	return uniqueOrderedStrings(steps)
}

func mapKeys(input map[string]bool) []string {
	out := make([]string, 0, len(input))
	for k, ok := range input {
		if ok && strings.TrimSpace(k) != "" {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

var tokenRiskCategoryLabels = map[string]string{
	"auth_invalid":               "鉴权无效：Authorization 头、token 或 API key 无法通过校验。",
	"auth_expired":               "token 已过期：需要用户重新登录或重新生成凭据。",
	"auth_forged":                "疑似伪造或篡改：token 结构、签名或形态异常。",
	"permission_violation":       "权限不足：当前主体无权访问该接口、分组、模型或资源。",
	"admin_api_probe":            "管理接口探测：普通主体访问管理接口或敏感后台路径。",
	"high_frequency":             "高频请求：同主体在短时间窗口内请求次数异常。",
	"batch_register":             "注册相关异常：请求触达注册、邀请或兑换相关路径。",
	"registrar_abuse":            "注册机相关异常：请求触达注册机或批量注册相关路径。",
	"config_tamper":              "敏感配置操作：请求触达设置、配置或风控管理路径。",
	"balance_or_reward_abuse":    "余额/权限异常：可能是余额不足、分组不可用或奖励/余额相关操作异常。",
	"game_abuse":                 "游戏中心风险：请求触达游戏、领奖或积分奖励路径。",
	"embedded_bypass":            "embedded 绕过风险：嵌入页鉴权参数异常或疑似绕过。",
	"api_key_sharing":            "API key 共享风险：同一 key 在多个来源 IP/UA 使用。",
	"adult_content":              "疑似色情内容风险：仅展示脱敏摘要和分类，不展示原文。",
	"grey_industry":              "疑似灰产业务风险：仅展示脱敏摘要和分类，不展示原文。",
	"abnormal_geo_or_ua":         "来源环境异常：IP、地区或 User-Agent 出现异常变化。",
	"insufficient_balance_abuse": "余额不足持续重试：余额不足后仍高频调用，可能是客户端异常或 key 被滥用。",
	"suspicious_path_scan":       "敏感路径扫描：请求命中常见扫描、调试或后台探测路径。",
}

var tokenRiskRuleLabels = map[string]string{
	"invalid_authorization_header":     "鉴权头无效",
	"expired_token":                    "token 已过期",
	"token_signature_or_shape_invalid": "token 签名或结构异常",
	"permission_denied":                "权限被拒绝",
	"non_admin_token_admin_path":       "非管理员主体访问管理路径",
	"admin_sensitive_path":             "管理敏感路径",
	"registration_related_path":        "注册/邀请相关路径",
	"registrar_related_path":           "注册机相关路径",
	"sensitive_config_path":            "敏感配置路径",
	"balance_reward_path":              "余额/奖励相关路径",
	"game_reward_path":                 "游戏/领奖相关路径",
	"embedded_auth_bypass":             "embedded 鉴权绕过",
	"insufficient_balance_single":      "单次余额不足",
	"insufficient_balance_repeated":    "余额不足持续重试",
	"high_frequency_window":            "短时间高频",
	"multi_ip_api_key_usage":           "多 IP 使用同一 API key",
	"sensitive_path_scan":              "敏感路径扫描",
	"sensitive_business_keyword":       "敏感业务关键词",
	"token_audit_observed":             "Token 审计观察",
}

func localizeTokenRiskList(values []string, labels map[string]string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if label := labels[value]; label != "" {
			out = append(out, label)
		} else {
			out = append(out, value)
		}
	}
	return out
}

func tokenRiskLevelLabel(level string) string {
	switch level {
	case TokenRiskLevelCritical:
		return "严重"
	case TokenRiskLevelHigh:
		return "高"
	case TokenRiskLevelMedium:
		return "中"
	default:
		return "低"
	}
}

func tokenRiskSubjectSummary(event *TokenRiskEvent) string {
	if event == nil {
		return "-"
	}
	if event.APIKeySummary != "" {
		return "API key " + event.APIKeySummary
	}
	if event.TokenPrefix != "" || event.TokenSuffix != "" {
		return strings.Trim(event.TokenPrefix+"..."+event.TokenSuffix, ".")
	}
	if event.TokenHash != "" {
		return "hash=" + truncateTokenRiskString(event.TokenHash, 12) + "..."
	}
	return "-"
}

func tokenRiskFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func replaceString(values []string, old string, replacement string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == old {
			out = append(out, replacement)
		} else {
			out = append(out, value)
		}
	}
	return out
}

func removeString(values []string, remove string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != remove {
			out = append(out, value)
		}
	}
	return out
}

func appendUniqueString(values []string, value string) []string {
	if value == "" || containsString(values, value) {
		return values
	}
	return append(values, value)
}

func uniqueSortedStrings(values []string) []string {
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			seen[value] = true
		}
	}
	return mapKeys(seen)
}

func uniqueOrderedStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func extraString(extra map[string]any, key string) string {
	v := extra[key]
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case json.Number:
		return t.String()
	default:
		return ""
	}
}

func extraInt(extra map[string]any, key string) int {
	v := extraString(extra, key)
	i, _ := strconv.Atoi(v)
	return i
}

func extraInt64(extra map[string]any, key string) int64 {
	v := extraString(extra, key)
	i, _ := strconv.ParseInt(v, 10, 64)
	return i
}

func maskAPIKeySummary(prefix, suffix string) string {
	prefix = strings.TrimSpace(prefix)
	suffix = strings.TrimSpace(suffix)
	if prefix == "" && suffix == "" {
		return ""
	}
	return prefix + "..." + suffix
}

func truncateTokenRiskString(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max]
}

func looksLikePathScan(path string) bool {
	needles := []string{"/.env", "wp-admin", "phpmyadmin", "../", "%2e%2e", "/admin/login", "/debug", "/actuator"}
	for _, n := range needles {
		if strings.Contains(path, n) {
			return true
		}
	}
	return false
}

func looksAdultOrGrey(value string) bool {
	// 只做防御性分类，不保存敏感原文。
	needles := []string{"adult", "porn", "博彩", "彩票", "裸聊", "色情", "灰产", "接码", "批量注册", "薅羊毛"}
	for _, n := range needles {
		if strings.Contains(value, strings.ToLower(n)) {
			return true
		}
	}
	return false
}
