package service

import (
	"testing"
	"time"
)

func TestBuildTokenRiskEventFromOpsLogInsufficientBalanceSingleIsNotAbuse(t *testing.T) {
	event := BuildTokenRiskEventFromOpsLog(tokenRiskTestOpsLog("insufficient_balance"), TokenRiskAnalyzeContext{})
	if event == nil {
		t.Fatal("expected event")
	}
	if containsString(event.RiskCategories, "insufficient_balance_abuse") {
		t.Fatalf("single insufficient balance should not be abuse: %#v", event.RiskCategories)
	}
	if !containsString(event.RiskCategories, "balance_or_reward_abuse") {
		t.Fatalf("expected balance/config category, got %#v", event.RiskCategories)
	}
	if !containsString(event.MatchedRules, "insufficient_balance_single") {
		t.Fatalf("expected single rule, got %#v", event.MatchedRules)
	}
}

func TestBuildTokenRiskEventFromOpsLogInsufficientBalanceRepeatedIsAbuse(t *testing.T) {
	event := BuildTokenRiskEventFromOpsLog(tokenRiskTestOpsLog("insufficient_balance"), TokenRiskAnalyzeContext{Count5m: 5, Count1h: 20})
	if event == nil {
		t.Fatal("expected event")
	}
	if !containsString(event.RiskCategories, "insufficient_balance_abuse") {
		t.Fatalf("expected repeated insufficient balance abuse, got %#v", event.RiskCategories)
	}
	if !containsString(event.MatchedRules, "insufficient_balance_repeated") {
		t.Fatalf("expected repeated rule, got %#v", event.MatchedRules)
	}
}

func TestEnrichTokenRiskEventRuntimePromotesRepeatedLegacyInsufficientBalance(t *testing.T) {
	event := &TokenRiskEvent{
		RiskScore:      45,
		RiskLevel:      TokenRiskLevelMedium,
		RiskCategories: []string{"balance_or_reward_abuse"},
		MatchedRules:   []string{"insufficient_balance"},
		Count5m:        6,
	}
	enrichTokenRiskEventRuntime(event)
	if !containsString(event.RiskCategories, "insufficient_balance_abuse") {
		t.Fatalf("expected runtime promotion, got %#v", event.RiskCategories)
	}
	if containsString(event.MatchedRules, "insufficient_balance") {
		t.Fatalf("legacy rule should be normalized, got %#v", event.MatchedRules)
	}
	if event.RiskLevel != TokenRiskLevelHigh {
		t.Fatalf("expected high risk after promotion, got %s", event.RiskLevel)
	}
}

func TestBuildTokenRiskHumanExplanationNoContentForModelsEndpoint(t *testing.T) {
	event := &TokenRiskEvent{
		Method:        "GET",
		Path:          "/v1/models",
		StatusCode:    403,
		APIKeySummary: "sk-abc...1234",
		RiskScore:     50,
		RiskLevel:     TokenRiskLevelMedium,
		MatchedRules:  []string{"insufficient_balance_single"},
	}
	explanation := buildTokenRiskHumanExplanation(event, nil)
	if explanation.ContentAvailability == "" {
		t.Fatal("expected explicit content availability message")
	}
	if len(explanation.RecommendedNextSteps) == 0 {
		t.Fatal("expected recommended next steps")
	}
}

func tokenRiskTestOpsLog(failure string) *OpsSystemLog {
	userID := int64(1001)
	return &OpsSystemLog{
		ID:        1,
		CreatedAt: time.Now().UTC(),
		Component: "audit.token",
		UserID:    &userID,
		Extra: map[string]any{
			"token_type":     "api_key",
			"token_hash":     "hash",
			"token_prefix":   "sk-abc",
			"token_suffix":   "1234",
			"api_key_id":     int64(10),
			"result":         "denied",
			"failure_reason": failure,
			"status_code":    403,
			"method":         "GET",
			"path":           "/v1/models",
			"client_ip":      "203.0.113.10",
			"user_agent":     "test-agent",
		},
	}
}
