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
	return s.repo.ListTokenRiskEvents(ctx, filter)
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
			Content:      buildTokenRiskNoticeContent(event),
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
		add("insufficient_balance_abuse", "insufficient_balance", 30)
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
	return fmt.Sprintf("risk_score=%d; categories=%s; rules=%s", score, strings.Join(categories, ","), strings.Join(rules, ","))
}

func buildTokenRiskNoticeContent(event *TokenRiskEvent) string {
	if event == nil {
		return "系统检测到账户存在异常使用风险，请检查近期操作。"
	}
	return fmt.Sprintf("系统检测到账户近期存在异常使用风险，分类：%s，路径：%s。为保护账户安全，请确认 API key/token 未被共享或泄露。", strings.Join(event.RiskCategories, ","), event.Path)
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
