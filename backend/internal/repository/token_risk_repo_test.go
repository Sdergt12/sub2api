package repository

import (
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestBuildRelatedContentModerationLogMatchClausesPreferStrongIdentifiers(t *testing.T) {
	sourceLogID := int64(101)
	userID := int64(202)
	apiKeyID := int64(303)
	clauses, args := buildRelatedContentModerationLogMatchClauses(&service.TokenRiskEvent{
		SourceLogID: &sourceLogID,
		UserID:      &userID,
		APIKeyID:    &apiKeyID,
	})
	joined := strings.Join(clauses, " OR ")

	if len(args) != 2 {
		t.Fatalf("expected only source log id and api key id args, got %d: %#v", len(args), args)
	}
	if !strings.Contains(joined, "NULLIF(request_id, '')") || !strings.Contains(joined, "l.request_id <> ''") {
		t.Fatalf("request_id clause must reject empty request ids: %s", joined)
	}
	if strings.Contains(joined, "l.user_id") {
		t.Fatalf("user fallback must not be used when strong identifiers exist: %s", joined)
	}
	if !strings.Contains(joined, "l.api_key_id") {
		t.Fatalf("expected api key clause: %s", joined)
	}
}

func TestBuildRelatedContentModerationLogMatchClausesUserFallbackOnly(t *testing.T) {
	userID := int64(202)
	clauses, args := buildRelatedContentModerationLogMatchClauses(&service.TokenRiskEvent{
		UserID: &userID,
	})
	joined := strings.Join(clauses, " OR ")

	if len(args) != 1 || args[0] != userID {
		t.Fatalf("expected only user id arg, got %#v", args)
	}
	if joined != "l.user_id = $1" {
		t.Fatalf("expected user fallback clause, got %s", joined)
	}
}
