package accountdedup

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestBuildPreviewGroupsAPIKeyAccountsByAPIKeyAndBaseURL(t *testing.T) {
	now := time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC)

	report := BuildPreview([]Account{
		{
			ID:        1,
			Name:      "older",
			Platform:  service.PlatformAnthropic,
			Type:      service.AccountTypeAPIKey,
			CreatedAt: now.Add(-2 * time.Hour),
			Credentials: map[string]any{
				"api_key":  " sk-ant-001 ",
				"base_url": "https://api.example.com/",
			},
		},
		{
			ID:        2,
			Name:      "newer",
			Platform:  service.PlatformAnthropic,
			Type:      service.AccountTypeAPIKey,
			CreatedAt: now,
			Credentials: map[string]any{
				"api_key":  "sk-ant-001",
				"base_url": "https://api.example.com",
			},
		},
	})

	if len(report.DuplicateGroups) != 1 {
		t.Fatalf("expected 1 duplicate group, got %d", len(report.DuplicateGroups))
	}

	group := report.DuplicateGroups[0]
	if group.MatchBasis != "api_key+base_url" {
		t.Fatalf("expected match basis api_key+base_url, got %q", group.MatchBasis)
	}
	if group.Keep.ID != 2 {
		t.Fatalf("expected newest account to be kept, got %d", group.Keep.ID)
	}
	if len(group.Delete) != 1 || group.Delete[0].ID != 1 {
		t.Fatalf("expected account 1 to be deleted, got %#v", group.Delete)
	}
}

func TestBuildPreviewGroupsOAuthAccountsByIdentityPriority(t *testing.T) {
	now := time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC)

	report := BuildPreview([]Account{
		{
			ID:        11,
			Name:      "oauth-old",
			Platform:  service.PlatformOpenAI,
			Type:      service.AccountTypeOAuth,
			CreatedAt: now.Add(-2 * time.Hour),
			Credentials: map[string]any{
				"refresh_token":      "rt-openai-1",
				"chatgpt_account_id": "chatgpt-account-should-be-ignored",
			},
		},
		{
			ID:        12,
			Name:      "oauth-new",
			Platform:  service.PlatformOpenAI,
			Type:      service.AccountTypeOAuth,
			CreatedAt: now,
			Credentials: map[string]any{
				"refresh_token": "rt-openai-1",
			},
		},
	})

	if len(report.DuplicateGroups) != 1 {
		t.Fatalf("expected 1 duplicate group, got %d", len(report.DuplicateGroups))
	}

	group := report.DuplicateGroups[0]
	if group.MatchBasis != "refresh_token" {
		t.Fatalf("expected refresh_token basis, got %q", group.MatchBasis)
	}
	if group.Keep.ID != 12 {
		t.Fatalf("expected account 12 kept, got %d", group.Keep.ID)
	}
}

func TestBuildPreviewGroupsSetupTokenAccountsByProjectClientAndBaseURL(t *testing.T) {
	now := time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC)

	report := BuildPreview([]Account{
		{
			ID:        21,
			Name:      "setup-old",
			Platform:  service.PlatformGemini,
			Type:      service.AccountTypeSetupToken,
			CreatedAt: now.Add(-2 * time.Hour),
			Credentials: map[string]any{
				"project_id": "proj-1",
				"client_id":  "client-1",
				"base_url":   "https://gemini.example.com/",
			},
		},
		{
			ID:        22,
			Name:      "setup-new",
			Platform:  service.PlatformGemini,
			Type:      service.AccountTypeSetupToken,
			CreatedAt: now,
			Credentials: map[string]any{
				"project_id": "proj-1",
				"client_id":  "client-1",
				"base_url":   "https://gemini.example.com",
			},
		},
	})

	if len(report.DuplicateGroups) != 1 {
		t.Fatalf("expected 1 duplicate group, got %d", len(report.DuplicateGroups))
	}

	group := report.DuplicateGroups[0]
	if group.MatchBasis != "project_id+client_id+base_url" {
		t.Fatalf("expected project/client/base_url basis, got %q", group.MatchBasis)
	}
	if group.Keep.ID != 22 {
		t.Fatalf("expected account 22 kept, got %d", group.Keep.ID)
	}
}

func TestBuildPreviewGroupsUpstreamAccountsByCanonicalCredentials(t *testing.T) {
	now := time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC)

	report := BuildPreview([]Account{
		{
			ID:        31,
			Name:      "upstream-old",
			Platform:  service.PlatformOpenAI,
			Type:      service.AccountTypeUpstream,
			CreatedAt: now.Add(-2 * time.Hour),
			Credentials: map[string]any{
				"api_key":  " upstream-key ",
				"base_url": "https://upstream.example.com/",
			},
		},
		{
			ID:        32,
			Name:      "upstream-new",
			Platform:  service.PlatformOpenAI,
			Type:      service.AccountTypeUpstream,
			CreatedAt: now,
			Credentials: map[string]any{
				"base_url": "https://upstream.example.com",
				"api_key":  "upstream-key",
			},
		},
	})

	if len(report.DuplicateGroups) != 1 {
		t.Fatalf("expected 1 duplicate group, got %d", len(report.DuplicateGroups))
	}

	group := report.DuplicateGroups[0]
	if group.MatchBasis != "full_credentials" {
		t.Fatalf("expected full_credentials basis, got %q", group.MatchBasis)
	}
	if group.Keep.ID != 32 {
		t.Fatalf("expected account 32 kept, got %d", group.Keep.ID)
	}
}

func TestBuildPreviewSkipsAccountsWithoutEnoughIdentity(t *testing.T) {
	now := time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC)

	report := BuildPreview([]Account{
		{
			ID:        41,
			Name:      "oauth-no-identity",
			Platform:  service.PlatformOpenAI,
			Type:      service.AccountTypeOAuth,
			CreatedAt: now,
			Credentials: map[string]any{
				"access_token": "access-only",
			},
		},
	})

	if len(report.DuplicateGroups) != 0 {
		t.Fatalf("expected no duplicate groups, got %d", len(report.DuplicateGroups))
	}
	if len(report.SkippedAccounts) != 1 {
		t.Fatalf("expected one skipped account, got %d", len(report.SkippedAccounts))
	}
	if report.SkippedAccounts[0].Account.ID != 41 {
		t.Fatalf("expected skipped account 41, got %d", report.SkippedAccounts[0].Account.ID)
	}
}

func TestBuildPreviewKeepsHigherIDWhenCreatedAtMatches(t *testing.T) {
	now := time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC)

	report := BuildPreview([]Account{
		{
			ID:        51,
			Name:      "older-id",
			Platform:  service.PlatformOpenAI,
			Type:      service.AccountTypeAPIKey,
			CreatedAt: now,
			Credentials: map[string]any{
				"api_key": "same-key",
			},
		},
		{
			ID:        52,
			Name:      "newer-id",
			Platform:  service.PlatformOpenAI,
			Type:      service.AccountTypeAPIKey,
			CreatedAt: now,
			Credentials: map[string]any{
				"api_key": "same-key",
			},
		},
	})

	if len(report.DuplicateGroups) != 1 {
		t.Fatalf("expected 1 duplicate group, got %d", len(report.DuplicateGroups))
	}
	if report.DuplicateGroups[0].Keep.ID != 52 {
		t.Fatalf("expected higher ID to be kept, got %d", report.DuplicateGroups[0].Keep.ID)
	}
}
