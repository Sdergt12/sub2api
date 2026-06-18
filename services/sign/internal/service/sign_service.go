package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	mathrand "math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"sub2api-sign/internal/config"
	"sub2api-sign/internal/model"
)

type signRepository interface {
	GetCheckinByDate(ctx context.Context, userID int64, signDate string) (model.CheckinRecord, bool, error)
	InsertPendingCheckin(ctx context.Context, record model.CheckinRecord) (model.CheckinRecord, error)
	AddRewardAttempt(ctx context.Context, attempt model.RewardAttempt) error
	MarkCheckinSuccess(ctx context.Context, record model.CheckinRecord, streak int) error
	MarkCheckinFailed(ctx context.Context, checkinID int64, failureReason string) error
	ListHistory(ctx context.Context, userID int64, limit int) ([]model.HistoryItem, error)
	ListRecentSignDays(ctx context.Context, userID int64, days int, now time.Time) ([]string, error)
	GetUserStats(ctx context.Context, userID int64) (model.UserStats, bool, error)
	AddRiskEvent(ctx context.Context, userID int64, riskType string, riskScore int, detail string) error
	ListRetryCandidates(ctx context.Context, limit int) ([]model.CheckinRecord, error)
}

type upstreamClient interface {
	ResolveUser(ctx context.Context, token string) (model.Sub2APIUser, error)
	GrantReward(ctx context.Context, userID int64, amount float64, rewardCode string, idemKey string, note string) (int, string, error)
}

type SignService struct {
	cfg                config.Config
	repo               signRepository
	redis              *redis.Client
	upstream           upstreamClient
	runtimeMu          sync.Mutex
	runtimeCachedUntil time.Time
	runtimeCached      signRuntimeConfig
}

func NewSignService(cfg config.Config, repo signRepository, redisClient *redis.Client, upstream upstreamClient) *SignService {
	return &SignService{
		cfg:      cfg,
		repo:     repo,
		redis:    redisClient,
		upstream: upstream,
	}
}

func (s *SignService) ResolveUserForBridge(ctx context.Context, token string) (model.Sub2APIUser, error) {
	return s.upstream.ResolveUser(ctx, token)
}

// GrantRewardForBridge 复用签到服务的上游兑奖链路，供外挂服务通过内部桥接发奖。
// 这里故意不做额外业务判断，只负责把请求安全地转发到本机可达的主站接口。
func (s *SignService) GrantRewardForBridge(ctx context.Context, req model.BridgeGrantRewardRequest) (int, string, error) {
	return s.upstream.GrantReward(ctx, req.UserID, req.Amount, req.RewardCode, req.IdempotencyKey, req.Note)
}

type HTTPError struct {
	Status  int
	Message string
}

func (e *HTTPError) Error() string {
	return e.Message
}

func (s *SignService) Bootstrap(ctx context.Context, req model.BootstrapRequest) (model.BootstrapResponse, string, error) {
	if strings.TrimSpace(req.Token) == "" {
		return model.BootstrapResponse{}, "", &HTTPError{Status: http.StatusUnauthorized, Message: "缺少主站令牌"}
	}

	user, err := s.upstream.ResolveUser(ctx, req.Token)
	if err != nil {
		return model.BootstrapResponse{}, "", &HTTPError{Status: http.StatusUnauthorized, Message: "主站登录态校验失败"}
	}

	if strings.TrimSpace(req.UserID) != "" && req.UserID != fmt.Sprintf("%d", user.ID) {
		// 这里记录 user_id 与 token 反查身份不一致的情况，便于排查 iframe 参数伪造。
		_ = s.repo.AddRiskEvent(ctx, user.ID, "user_identity_mismatch", 80, fmt.Sprintf("query_user_id=%s, resolved_user_id=%d", req.UserID, user.ID))
	}

	sessionID, err := randomToken(24)
	if err != nil {
		return model.BootstrapResponse{}, "", err
	}

	now := s.now()
	session := model.SessionContext{
		SessionID:  sessionID,
		UserID:     user.ID,
		Username:   user.Username,
		Email:      user.Email,
		Balance:    user.Balance,
		Theme:      normalizeString(req.Theme, "light"),
		Lang:       normalizeString(req.Lang, "zh"),
		UIMode:     normalizeString(req.UIMode, "embedded"),
		UserToken:  req.Token,
		IP:         req.IP,
		UserAgent:  req.UA,
		CreatedAt:  now,
		ExpiresAt:  now.Add(s.cfg.SessionTTL),
		LastSeenAt: now,
	}

	if err := s.saveSession(ctx, session); err != nil {
		return model.BootstrapResponse{}, "", err
	}

	status, err := s.getStatusBySession(ctx, session)
	if err != nil {
		return model.BootstrapResponse{}, "", err
	}

	return model.BootstrapResponse{
		AppName:               s.cfg.AppName,
		UserID:                user.ID,
		Username:              user.Username,
		Theme:                 session.Theme,
		Lang:                  session.Lang,
		UIMode:                session.UIMode,
		SignedToday:           status.SignedToday,
		Streak:                status.Streak,
		TodayRewardMin:        status.TodayRewardMin,
		TodayRewardMax:        status.TodayRewardMax,
		CurrentBalanceDisplay: status.CurrentBalanceDisplay,
		RecentSignDays:        status.RecentSignDays,
		MonthSignDays:         status.MonthSignDays,
		History:               status.History,
	}, sessionID, nil
}

func (s *SignService) GetStatus(ctx context.Context, sessionID string) (model.StatusResponse, error) {
	session, err := s.loadSession(ctx, sessionID)
	if err != nil {
		return model.StatusResponse{}, err
	}

	return s.getStatusBySession(ctx, session)
}

func (s *SignService) Checkin(ctx context.Context, sessionID string) (model.CheckinResponse, error) {
	session, err := s.loadSession(ctx, sessionID)
	if err != nil {
		return model.CheckinResponse{}, err
	}

	if err := s.enforceRateLimit(ctx, session); err != nil {
		return model.CheckinResponse{}, err
	}

	now := s.now()
	signDate := now.Format("2006-01-02")
	lockKey := fmt.Sprintf("sign:lock:%d:%s", session.UserID, signDate)
	locked, err := s.redis.SetNX(ctx, lockKey, "1", 15*time.Second).Result()
	if err != nil {
		return model.CheckinResponse{}, err
	}
	if !locked {
		return model.CheckinResponse{}, &HTTPError{Status: http.StatusConflict, Message: "签到处理中，请稍后再试"}
	}
	defer s.redis.Del(ctx, lockKey)

	existing, found, err := s.repo.GetCheckinByDate(ctx, session.UserID, signDate)
	if err != nil {
		return model.CheckinResponse{}, err
	}
	if found && existing.GrantStatus == model.CheckinStatusSuccess {
		return s.buildAlreadySignedResponse(ctx, session, existing)
	}
	if found {
		// 当天已有 pending/failed 记录时，复用原 reward_code 和金额重试发奖。
		// 这样用户刷新或再次点击不会产生第二笔奖励，也不用等后台补偿任务。
		return s.retryExistingCheckin(ctx, session, existing)
	}

	runtimeCfg := s.loadSignRuntimeConfig(ctx)
	streak, bonus := s.calculateStreakAndBonus(ctx, session.UserID, signDate, runtimeCfg)
	baseReward := sampleReward(runtimeCfg.RewardTiers)
	totalReward := roundMoney(baseReward + bonus)
	rewardCode := fmt.Sprintf("sign:%d:%s", session.UserID, signDate)

	record := model.CheckinRecord{
		Sub2APIUserID:  session.UserID,
		SignDate:       signDate,
		BaseReward:     baseReward,
		BonusReward:    bonus,
		TotalReward:    totalReward,
		RewardCode:     rewardCode,
		GrantMethod:    s.cfg.GrantMode,
		GrantStatus:    model.CheckinStatusPending,
		IdempotencyKey: fmt.Sprintf("sign:%d:%s:1", session.UserID, signDate),
		IP:             session.IP,
		UA:             session.UserAgent,
	}

	record, err = s.repo.InsertPendingCheckin(ctx, record)
	if err != nil {
		// 并发插入竞争统一回查当日记录，避免双击造成重复发奖。
		if existing, found, getErr := s.repo.GetCheckinByDate(ctx, session.UserID, signDate); getErr == nil && found {
			if existing.GrantStatus == model.CheckinStatusSuccess {
				return s.buildAlreadySignedResponse(ctx, session, existing)
			}
			return s.retryExistingCheckin(ctx, session, existing)
		}
		return model.CheckinResponse{}, err
	}

	if err := s.performGrant(ctx, record, 1); err != nil {
		return model.CheckinResponse{}, &HTTPError{Status: http.StatusBadGateway, Message: "奖励发放失败，请稍后重试"}
	}

	if err := s.repo.MarkCheckinSuccess(ctx, record, streak); err != nil {
		return model.CheckinResponse{}, err
	}

	session.Balance = roundMoney(session.Balance + totalReward)
	session.LastSeenAt = s.now()
	if err := s.saveSession(ctx, session); err != nil {
		return model.CheckinResponse{}, err
	}

	return s.buildGrantedResponse(session, signDate, baseReward, bonus, totalReward, rewardCode, record.GrantMethod, streak, false), nil
}

func (s *SignService) retryExistingCheckin(ctx context.Context, session model.SessionContext, record model.CheckinRecord) (model.CheckinResponse, error) {
	attempt := record.AttemptCount + 1
	if attempt < 2 {
		attempt = 2
	}

	if err := s.performGrant(ctx, record, attempt); err != nil {
		return model.CheckinResponse{}, &HTTPError{Status: http.StatusBadGateway, Message: "奖励发放失败，请稍后重试"}
	}

	streak, _ := s.calculateStreakAndBonus(ctx, record.Sub2APIUserID, record.SignDate, s.loadSignRuntimeConfig(ctx))
	if err := s.repo.MarkCheckinSuccess(ctx, record, streak); err != nil {
		return model.CheckinResponse{}, err
	}

	session.Balance = roundMoney(session.Balance + record.TotalReward)
	session.LastSeenAt = s.now()
	if err := s.saveSession(ctx, session); err != nil {
		return model.CheckinResponse{}, err
	}

	return s.buildGrantedResponse(session, record.SignDate, record.BaseReward, record.BonusReward, record.TotalReward, record.RewardCode, record.GrantMethod, streak, false), nil
}

func (s *SignService) buildGrantedResponse(session model.SessionContext, signDate string, baseReward float64, bonus float64, totalReward float64, rewardCode string, grantMethod string, streak int, alreadySigned bool) model.CheckinResponse {
	return model.CheckinResponse{
		SignedToday:           true,
		AlreadySigned:         alreadySigned,
		SignDate:              signDate,
		BaseReward:            formatMoney(baseReward),
		BonusReward:           formatMoney(bonus),
		TotalReward:           formatMoney(totalReward),
		RewardCode:            rewardCode,
		GrantStatus:           model.CheckinStatusSuccess,
		GrantMethod:           grantMethod,
		Streak:                streak,
		CurrentBalanceDisplay: formatMoney(session.Balance),
	}
}

func (s *SignService) ListHistory(ctx context.Context, sessionID string) ([]model.HistoryItem, error) {
	session, err := s.loadSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	return s.repo.ListHistory(ctx, session.UserID, s.cfg.HistoryLimit)
}

func (s *SignService) RunRecoveryLoop(ctx context.Context) {
	if s.cfg.RecoveryInterval <= 0 {
		return
	}

	ticker := time.NewTicker(s.cfg.RecoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.RecoverFailedCheckins(ctx)
		}
	}
}

func (s *SignService) RecoverFailedCheckins(ctx context.Context) error {
	candidates, err := s.repo.ListRetryCandidates(ctx, s.cfg.RecoveryBatchSize)
	if err != nil {
		return err
	}

	for _, candidate := range candidates {
		attempt := candidate.AttemptCount + 1
		if attempt < 2 {
			attempt = 2
		}

		if err := s.performGrant(ctx, candidate, attempt); err != nil {
			continue
		}

		streak, _ := s.calculateStreakAndBonus(ctx, candidate.Sub2APIUserID, candidate.SignDate, s.loadSignRuntimeConfig(ctx))
		_ = s.repo.MarkCheckinSuccess(ctx, candidate, streak)
	}

	return nil
}

func (s *SignService) getStatusBySession(ctx context.Context, session model.SessionContext) (model.StatusResponse, error) {
	now := s.now()
	signDate := now.Format("2006-01-02")

	record, found, err := s.repo.GetCheckinByDate(ctx, session.UserID, signDate)
	if err != nil {
		return model.StatusResponse{}, err
	}

	stats, statsFound, err := s.repo.GetUserStats(ctx, session.UserID)
	if err != nil {
		return model.StatusResponse{}, err
	}

	streak := 0
	if statsFound {
		streak = stats.CurrentStreak
	}
	if found && record.GrantStatus == model.CheckinStatusSuccess && record.SignDate == signDate && streak == 0 {
		streak = 1
	}

	recentSignDays, err := s.repo.ListRecentSignDays(ctx, session.UserID, 7, now)
	if err != nil {
		return model.StatusResponse{}, err
	}

	// 本月概览只需要当月 1 号至今天的成功签到日期，前端负责渲染月历。
	monthSignDays, err := s.repo.ListRecentSignDays(ctx, session.UserID, now.Day(), now)
	if err != nil {
		return model.StatusResponse{}, err
	}

	history, err := s.repo.ListHistory(ctx, session.UserID, s.cfg.HistoryLimit)
	if err != nil {
		return model.StatusResponse{}, err
	}

	minReward, maxReward := rewardRange(s.loadSignRuntimeConfig(ctx).RewardTiers)

	return model.StatusResponse{
		SignedToday:           found && record.GrantStatus == model.CheckinStatusSuccess,
		Streak:                streak,
		TodayRewardMin:        formatMoney(minReward),
		TodayRewardMax:        formatMoney(maxReward),
		CurrentBalanceDisplay: formatMoney(session.Balance),
		RecentSignDays:        recentSignDays,
		MonthSignDays:         monthSignDays,
		History:               history,
	}, nil
}

func (s *SignService) buildAlreadySignedResponse(ctx context.Context, session model.SessionContext, record model.CheckinRecord) (model.CheckinResponse, error) {
	stats, statsFound, err := s.repo.GetUserStats(ctx, session.UserID)
	if err != nil {
		return model.CheckinResponse{}, err
	}

	streak := 1
	if statsFound && stats.CurrentStreak > 0 {
		streak = stats.CurrentStreak
	}

	return model.CheckinResponse{
		SignedToday:           true,
		AlreadySigned:         true,
		SignDate:              record.SignDate,
		BaseReward:            formatMoney(record.BaseReward),
		BonusReward:           formatMoney(record.BonusReward),
		TotalReward:           formatMoney(record.TotalReward),
		RewardCode:            record.RewardCode,
		GrantStatus:           record.GrantStatus,
		GrantMethod:           record.GrantMethod,
		Streak:                streak,
		CurrentBalanceDisplay: formatMoney(session.Balance),
	}, nil
}

func (s *SignService) calculateStreakAndBonus(ctx context.Context, userID int64, signDate string, runtimeCfg signRuntimeConfig) (int, float64) {
	stats, found, err := s.repo.GetUserStats(ctx, userID)
	if err != nil || !found {
		return 1, 0
	}

	lastDate, err := time.Parse("2006-01-02", stats.LastSignDate)
	if err != nil {
		return 1, 0
	}

	currentDate, err := time.Parse("2006-01-02", signDate)
	if err != nil {
		return 1, 0
	}

	diffDays := int(currentDate.Sub(lastDate).Hours() / 24)
	streak := 1
	if diffDays == 1 {
		streak = stats.CurrentStreak + 1
	}

	bonus := 0.0
	// 连签奖励是里程碑额外加成，不参与基础随机奖励 10.00 的上限。
	switch streak {
	case 3:
		bonus = runtimeCfg.BonusDay3
	case 7:
		bonus = runtimeCfg.BonusDay7
	case 15:
		bonus = runtimeCfg.BonusDay15
	case 30:
		bonus = runtimeCfg.BonusDay30
	}

	return streak, roundMoney(bonus)
}

func (s *SignService) enforceRateLimit(ctx context.Context, session model.SessionContext) error {
	userKey := fmt.Sprintf("sign:rate:user:%d", session.UserID)
	ipKey := fmt.Sprintf("sign:rate:ip:%s", normalizeString(session.IP, "unknown"))

	if exceeded, err := s.incrementLimit(ctx, userKey, 10*time.Second, s.cfg.RateLimitPerUser10s); err != nil {
		return err
	} else if exceeded {
		_ = s.repo.AddRiskEvent(ctx, session.UserID, "rate_limit_user", 50, "签到请求过于频繁")
		return &HTTPError{Status: http.StatusTooManyRequests, Message: "请求过于频繁，请稍后再试"}
	}

	if exceeded, err := s.incrementLimit(ctx, ipKey, time.Minute, s.cfg.RateLimitPerIP1m); err != nil {
		return err
	} else if exceeded {
		_ = s.repo.AddRiskEvent(ctx, session.UserID, "rate_limit_ip", 80, fmt.Sprintf("ip=%s", session.IP))
		return &HTTPError{Status: http.StatusTooManyRequests, Message: "请求过于频繁，请稍后再试"}
	}

	return nil
}

func (s *SignService) incrementLimit(ctx context.Context, key string, ttl time.Duration, maxCount int) (bool, error) {
	count, err := s.redis.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if count == 1 {
		s.redis.Expire(ctx, key, ttl)
	}
	return count > int64(maxCount), nil
}

func (s *SignService) saveSession(ctx context.Context, session model.SessionContext) error {
	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("sign:session:%s", session.SessionID)
	return s.redis.Set(ctx, key, payload, s.cfg.SessionTTL).Err()
}

func (s *SignService) loadSession(ctx context.Context, sessionID string) (model.SessionContext, error) {
	if strings.TrimSpace(sessionID) == "" {
		return model.SessionContext{}, &HTTPError{Status: http.StatusUnauthorized, Message: "缺少签到会话"}
	}

	key := fmt.Sprintf("sign:session:%s", sessionID)
	payload, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return model.SessionContext{}, &HTTPError{Status: http.StatusUnauthorized, Message: "签到会话已过期，请重新打开页面"}
		}
		return model.SessionContext{}, err
	}

	var session model.SessionContext
	if err := json.Unmarshal([]byte(payload), &session); err != nil {
		return model.SessionContext{}, err
	}

	session.LastSeenAt = s.now()
	if err := s.saveSession(ctx, session); err != nil {
		return model.SessionContext{}, err
	}

	return session, nil
}

func (s *SignService) performGrant(ctx context.Context, record model.CheckinRecord, attempt int) error {
	idempotencyKey := fmt.Sprintf("sign:%d:%s:%d", record.Sub2APIUserID, record.SignDate, attempt)
	requestPayload, _ := json.Marshal(map[string]any{
		"user_id":         record.Sub2APIUserID,
		"reward_code":     record.RewardCode,
		"idempotency_key": idempotencyKey,
		"amount":          record.TotalReward,
		"grant_method":    record.GrantMethod,
		"attempt":         attempt,
	})

	httpStatus, responsePayload, grantErr := s.upstream.GrantReward(
		ctx,
		record.Sub2APIUserID,
		record.TotalReward,
		record.RewardCode,
		idempotencyKey,
		fmt.Sprintf("每日签到奖励 %s", record.SignDate),
	)

	_ = s.repo.AddRewardAttempt(ctx, model.RewardAttempt{
		CheckinID:       record.ID,
		RewardCode:      record.RewardCode,
		IdempotencyKey:  idempotencyKey,
		RequestPayload:  string(requestPayload),
		ResponsePayload: responsePayload,
		HTTPStatus:      httpStatus,
		Success:         grantErr == nil && httpStatus >= 200 && httpStatus < 300,
	})

	if grantErr != nil || httpStatus < 200 || httpStatus >= 300 {
		failureReason := strings.TrimSpace(responsePayload)
		if failureReason == "" && grantErr != nil {
			failureReason = grantErr.Error()
		}
		_ = s.repo.MarkCheckinFailed(ctx, record.ID, truncateString(failureReason, 500))
		return errors.New("grant failed")
	}

	return nil
}

func (s *SignService) now() time.Time {
	loc, err := time.LoadLocation(s.cfg.Timezone)
	if err != nil {
		return time.Now()
	}

	return time.Now().In(loc)
}

type signRuntimeConfig struct {
	RewardTiers []model.RewardTier `json:"reward_tiers"`
	BonusDay3   float64            `json:"bonus_day3"`
	BonusDay7   float64            `json:"bonus_day7"`
	BonusDay15  float64            `json:"bonus_day15"`
	BonusDay30  float64            `json:"bonus_day30"`
}

func (s *SignService) defaultSignRuntimeConfig() signRuntimeConfig {
	return signRuntimeConfig{
		RewardTiers: s.cfg.RewardTiers,
		BonusDay3:   s.cfg.BonusDay3,
		BonusDay7:   s.cfg.BonusDay7,
		BonusDay15:  s.cfg.BonusDay15,
		BonusDay30:  s.cfg.BonusDay30,
	}
}

func (s *SignService) loadSignRuntimeConfig(ctx context.Context) signRuntimeConfig {
	now := time.Now()
	s.runtimeMu.Lock()
	if !s.runtimeCachedUntil.IsZero() && now.Before(s.runtimeCachedUntil) && len(s.runtimeCached.RewardTiers) > 0 {
		cached := s.runtimeCached
		s.runtimeMu.Unlock()
		return cached
	}
	s.runtimeMu.Unlock()

	cfg := s.fetchSignRuntimeConfig(ctx)

	s.runtimeMu.Lock()
	s.runtimeCached = cfg
	s.runtimeCachedUntil = now.Add(30 * time.Second)
	s.runtimeMu.Unlock()
	return cfg
}

func (s *SignService) fetchSignRuntimeConfig(ctx context.Context) signRuntimeConfig {
	fallback := s.defaultSignRuntimeConfig()
	baseURL := strings.TrimRight(s.cfg.Sub2APIBaseURL, "/")
	token := strings.TrimSpace(s.cfg.Sub2APIInternalConfigToken)
	if baseURL == "" || token == "" {
		return fallback
	}

	reqCtx, cancel := context.WithTimeout(ctx, s.cfg.RequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, baseURL+"/api/v1/external/runtime-config/sign", nil)
	if err != nil {
		return fallback
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Internal-Config-Token", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fallback
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fallback
	}

	var envelope struct {
		Code int               `json:"code"`
		Data signRuntimeConfig `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fallback
	}
	if envelope.Code != 0 || len(envelope.Data.RewardTiers) == 0 {
		return fallback
	}
	if !validSignRuntimeConfig(envelope.Data) {
		return fallback
	}
	return envelope.Data
}

func validSignRuntimeConfig(cfg signRuntimeConfig) bool {
	totalWeight := 0
	for _, tier := range cfg.RewardTiers {
		if tier.Min < 0 || tier.Max < tier.Min || tier.Weight <= 0 {
			return false
		}
		totalWeight += tier.Weight
	}
	return totalWeight > 0 && cfg.BonusDay3 >= 0 && cfg.BonusDay7 >= 0 && cfg.BonusDay15 >= 0 && cfg.BonusDay30 >= 0
}

func rewardRange(tiers []model.RewardTier) (float64, float64) {
	minValue := tiers[0].Min
	maxValue := tiers[0].Max
	for _, tier := range tiers {
		if tier.Min < minValue {
			minValue = tier.Min
		}
		if tier.Max > maxValue {
			maxValue = tier.Max
		}
	}
	return minValue, maxValue
}

func sampleReward(tiers []model.RewardTier) float64 {
	totalWeight := 0
	for _, tier := range tiers {
		totalWeight += tier.Weight
	}

	selected := mathrand.Intn(totalWeight) + 1
	accumulated := 0
	for _, tier := range tiers {
		accumulated += tier.Weight
		if selected <= accumulated {
			value := tier.Min + mathrand.Float64()*(tier.Max-tier.Min)
			return roundMoney(value)
		}
	}

	return roundMoney(tiers[len(tiers)-1].Max)
}

func randomToken(byteLength int) (string, error) {
	value := make([]byte, byteLength)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}

	return hex.EncodeToString(value), nil
}

func roundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}

func formatMoney(value float64) string {
	return fmt.Sprintf("%.2f", roundMoney(value))
}

func truncateString(value string, maxLength int) string {
	if len(value) <= maxLength {
		return value
	}
	return value[:maxLength]
}

func normalizeString(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
