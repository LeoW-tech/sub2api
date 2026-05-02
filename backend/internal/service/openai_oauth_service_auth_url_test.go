package service

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/stretchr/testify/require"
)

type openaiOAuthClientAuthURLStub struct{}

func (s *openaiOAuthClientAuthURLStub) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI, proxyURL, clientID string) (*openai.TokenResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *openaiOAuthClientAuthURLStub) RefreshToken(ctx context.Context, refreshToken, proxyURL string) (*openai.TokenResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *openaiOAuthClientAuthURLStub) RefreshTokenWithClientID(ctx context.Context, refreshToken, proxyURL string, clientID string) (*openai.TokenResponse, error) {
	return nil, errors.New("not implemented")
}

func TestOpenAIOAuthService_GenerateAuthURL_OpenAIKeepsCodexFlow(t *testing.T) {
	svc := NewOpenAIOAuthService(nil, &openaiOAuthClientAuthURLStub{})
	defer svc.Stop()

	result, err := svc.GenerateAuthURL(context.Background(), nil, "", PlatformOpenAI, nil)
	require.NoError(t, err)
	require.NotEmpty(t, result.AuthURL)
	require.NotEmpty(t, result.SessionID)

	parsed, err := url.Parse(result.AuthURL)
	require.NoError(t, err)
	q := parsed.Query()
	require.Equal(t, openai.ClientID, q.Get("client_id"))
	require.Equal(t, "true", q.Get("codex_cli_simplified_flow"))

	session, ok := svc.sessionStore.Get(result.SessionID)
	require.True(t, ok)
	require.Equal(t, openai.ClientID, session.ClientID)
}

func TestOpenAIOAuthService_GenerateAuthURL_SavesPendingCreate(t *testing.T) {
	svc := NewOpenAIOAuthService(nil, &openaiOAuthClientAuthURLStub{})
	defer svc.Stop()

	loadFactor := 3
	rateMultiplier := 1.2
	autoPause := true
	expiresAt := int64(1893456000)
	pendingCreate := &openai.OAuthPendingCreateAccount{
		Name:                "OpenAI Plus",
		Notes:               "created from helper",
		Concurrency:         10,
		LoadFactor:          &loadFactor,
		Priority:            50,
		RateMultiplier:      &rateMultiplier,
		GroupIDs:            []int64{8},
		ExpiresAt:           &expiresAt,
		AutoPauseOnExpired:  &autoPause,
		Extra:               map[string]any{"openai_compact_mode": "force_on"},
		CredentialOverrides: map[string]any{"model_mapping": map[string]any{"gpt-5": "gpt-5"}},
	}

	result, err := svc.GenerateAuthURL(context.Background(), nil, "", PlatformOpenAI, pendingCreate)
	require.NoError(t, err)
	require.NotEmpty(t, result.AuthURL)

	parsed, err := url.Parse(result.AuthURL)
	require.NoError(t, err)
	require.Equal(t, openai.DefaultRedirectURI, parsed.Query().Get("redirect_uri"))

	session, ok := svc.sessionStore.Get(result.SessionID)
	require.True(t, ok)
	require.NotNil(t, session.PendingCreate)
	require.Equal(t, pendingCreate.Name, session.PendingCreate.Name)
	require.Equal(t, pendingCreate.Priority, session.PendingCreate.Priority)
	require.Equal(t, pendingCreate.GroupIDs, session.PendingCreate.GroupIDs)
	require.Equal(t, pendingCreate.CredentialOverrides, session.PendingCreate.CredentialOverrides)
}
