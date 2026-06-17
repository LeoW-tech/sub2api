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
	handler := NewOpenAIOAuthHandler(oauthSvc, adminSvc, nil)
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
	handler := NewOpenAIOAuthHandler(oauthSvc, adminSvc, nil)
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
