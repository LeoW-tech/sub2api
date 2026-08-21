package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type openaiOAuthClientHandlerStub struct{}

func (s *openaiOAuthClientHandlerStub) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI, proxyURL, clientID string) (*openai.TokenResponse, error) {
	return &openai.TokenResponse{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresIn:    3600,
	}, nil
}

func (s *openaiOAuthClientHandlerStub) RefreshToken(ctx context.Context, refreshToken, proxyURL string) (*openai.TokenResponse, error) {
	return nil, nil
}

func (s *openaiOAuthClientHandlerStub) RefreshTokenWithClientID(ctx context.Context, refreshToken, proxyURL string, clientID string) (*openai.TokenResponse, error) {
	return nil, nil
}

func TestOpenAIOAuthHandler_CompletePendingCreateCreatesAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	oauthSvc := service.NewOpenAIOAuthService(nil, &openaiOAuthClientHandlerStub{})
	defer oauthSvc.Stop()

	rateMultiplier := 1.5
	loadFactor := 2
	expiresAt := time.Now().Add(24 * time.Hour).Unix()
	autoPause := true
	generated, err := oauthSvc.GenerateAuthURL(context.Background(), nil, "", service.PlatformOpenAI, &openai.OAuthPendingCreateAccount{
		Name:               "OpenAI Plus",
		Notes:              "created by callback",
		Concurrency:        12,
		LoadFactor:         &loadFactor,
		Priority:           50,
		RateMultiplier:     &rateMultiplier,
		GroupIDs:           []int64{8},
		ExpiresAt:          &expiresAt,
		AutoPauseOnExpired: &autoPause,
		Extra: map[string]any{
			"openai_compact_mode": "force_on",
		},
		CredentialOverrides: map[string]any{
			"model_mapping": map[string]any{"gpt-5": "gpt-5-mini"},
		},
	})
	require.NoError(t, err)

	parsedAuthURL, err := url.Parse(generated.AuthURL)
	require.NoError(t, err)
	state := parsedAuthURL.Query().Get("state")
	require.NotEmpty(t, state)

	router := gin.New()
	handler := NewOpenAIOAuthHandler(oauthSvc, adminSvc, nil, nil)
	router.POST("/openai/complete-pending-create", handler.CompletePendingCreate)

	body, err := json.Marshal(map[string]string{
		"code":  "auth-code",
		"state": state,
	})
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/openai/complete-pending-create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, adminSvc.createdAccounts, 1)
	created := adminSvc.createdAccounts[0]
	require.Equal(t, "OpenAI Plus", created.Name)
	require.NotNil(t, created.Notes)
	require.Equal(t, "created by callback", *created.Notes)
	require.Equal(t, service.PlatformOpenAI, created.Platform)
	require.Equal(t, "oauth", created.Type)
	require.Equal(t, 12, created.Concurrency)
	require.Equal(t, 50, created.Priority)
	require.Equal(t, []int64{8}, created.GroupIDs)
	require.Equal(t, &rateMultiplier, created.RateMultiplier)
	require.Equal(t, &loadFactor, created.LoadFactor)
	require.Equal(t, &expiresAt, created.ExpiresAt)
	require.Equal(t, &autoPause, created.AutoPauseOnExpired)
	require.False(t, created.SkipDefaultGroupBind)
	require.Equal(t, "access-token", created.Credentials["access_token"])
	require.Equal(t, "refresh-token", created.Credentials["refresh_token"])
	require.Equal(t, map[string]any{"gpt-5": "gpt-5-mini"}, created.Credentials["model_mapping"])
	require.Equal(t, "force_on", created.Extra["openai_compact_mode"])
}

func TestOpenAIOAuthHandler_CompletePendingCreateSkipsDefaultGroupWhenNoGroupSelected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	oauthSvc := service.NewOpenAIOAuthService(nil, &openaiOAuthClientHandlerStub{})
	defer oauthSvc.Stop()

	generated, err := oauthSvc.GenerateAuthURL(context.Background(), nil, "", service.PlatformOpenAI, &openai.OAuthPendingCreateAccount{
		Name:        "OpenAI no group",
		Concurrency: 10,
		Priority:    50,
		GroupIDs:    []int64{},
	})
	require.NoError(t, err)

	parsedAuthURL, err := url.Parse(generated.AuthURL)
	require.NoError(t, err)
	state := parsedAuthURL.Query().Get("state")
	require.NotEmpty(t, state)

	router := gin.New()
	handler := NewOpenAIOAuthHandler(oauthSvc, adminSvc, nil, nil)
	router.POST("/openai/complete-pending-create", handler.CompletePendingCreate)

	body, err := json.Marshal(map[string]string{
		"code":  "auth-code",
		"state": state,
	})
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/openai/complete-pending-create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, adminSvc.createdAccounts, 1)
	require.Empty(t, adminSvc.createdAccounts[0].GroupIDs)
	require.True(t, adminSvc.createdAccounts[0].SkipDefaultGroupBind)
}

func TestOpenAIOAuthHandler_RefreshAccountSubscriptionsDryRunFiltersAndLimits(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{
		{
			ID:       1,
			Name:     "plus-missing",
			Platform: service.PlatformOpenAI,
			Type:     service.AccountTypeOAuth,
			Credentials: map[string]any{
				"access_token": "access-token",
				"plan_type":    "plus",
			},
		},
		{
			ID:       2,
			Name:     "free-missing",
			Platform: service.PlatformOpenAI,
			Type:     service.AccountTypeOAuth,
			Credentials: map[string]any{
				"access_token": "access-token",
				"plan_type":    "free",
			},
		},
		{
			ID:       3,
			Name:     "plus-complete",
			Platform: service.PlatformOpenAI,
			Type:     service.AccountTypeOAuth,
			Credentials: map[string]any{
				"access_token":            "access-token",
				"plan_type":               "plus",
				"subscription_expires_at": "2026-07-23T16:23:04Z",
			},
		},
		{
			ID:       4,
			Name:     "unknown-missing",
			Platform: service.PlatformOpenAI,
			Type:     service.AccountTypeOAuth,
			Credentials: map[string]any{
				"refresh_token": "refresh-token",
			},
		},
	}

	router := gin.New()
	handler := NewOpenAIOAuthHandler(nil, adminSvc, nil, nil)
	router.POST("/openai/accounts/refresh-subscriptions", handler.RefreshAccountSubscriptions)

	body, err := json.Marshal(map[string]any{"dry_run": true, "limit": 1})
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/openai/accounts/refresh-subscriptions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, adminSvc.lastListAccounts.calls)
	require.Equal(t, service.PlatformOpenAI, adminSvc.lastListAccounts.platform)
	require.Equal(t, service.AccountTypeOAuth, adminSvc.lastListAccounts.accountType)

	var payload struct {
		Data struct {
			Results []OpenAIRefreshSubscriptionResult `json:"results"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Len(t, payload.Data.Results, 1)
	require.Equal(t, int64(1), payload.Data.Results[0].AccountID)
	require.Equal(t, "skipped", payload.Data.Results[0].Status)
	require.Equal(t, "dry_run", payload.Data.Results[0].Reason)
}
