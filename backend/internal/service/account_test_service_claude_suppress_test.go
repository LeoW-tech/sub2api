//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAccountTestService_Claude403SuppressesPermanentErrorWhenContextRequestsProbeOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := newTestContext()
	ctx.Request = ctx.Request.WithContext(accountTestSuppressStatusMutation(ctx.Request.Context()))

	repo := &claudeAccountTestRepo{}
	upstream := &queuedHTTPUpstream{responses: []*http.Response{
		newJSONResponse(http.StatusForbidden, `{"error":"forbidden"}`),
	}}
	svc := &AccountTestService{accountRepo: repo, httpUpstream: upstream}
	account := &Account{
		ID:          91,
		Platform:    PlatformAnthropic,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "claude-token"},
	}

	err := svc.testClaudeAccountConnection(ctx, account, "claude-sonnet-4-5")

	require.Error(t, err)
	require.Zero(t, repo.setErrorID)
	require.Empty(t, repo.setErrorMsg)
	require.Equal(t, StatusActive, account.Status)
}

func TestAccountTestService_ClaudeVertex403SuppressesPermanentErrorWhenContextRequestsProbeOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := newTestContext()
	ctx.Request = ctx.Request.WithContext(accountTestSuppressStatusMutation(ctx.Request.Context()))

	repo := &claudeAccountTestRepo{}
	cache := newClaudeTokenCacheStub()
	account := &Account{
		ID:          92,
		Platform:    PlatformAnthropic,
		Type:        AccountTypeServiceAccount,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{
			"project_id": "test-project",
			"location":   "us-central1",
			"service_account_json": map[string]any{
				"type":           "service_account",
				"project_id":     "test-project",
				"private_key_id": "kid",
				"private_key":    "unused",
				"client_email":   "svc@test-project.iam.gserviceaccount.com",
			},
		},
	}
	key, err := parseVertexServiceAccountKey(account)
	require.NoError(t, err)
	require.NoError(t, cache.SetAccessToken(context.Background(), vertexServiceAccountCacheKey(account, key), "vertex-token", time.Hour))

	upstream := &queuedHTTPUpstream{responses: []*http.Response{
		newJSONResponse(http.StatusForbidden, `{"error":"forbidden"}`),
	}}
	svc := &AccountTestService{
		accountRepo:         repo,
		claudeTokenProvider: &ClaudeTokenProvider{tokenCache: cache},
		httpUpstream:        upstream,
	}

	err = svc.testClaudeVertexServiceAccountConnection(ctx, ctx.Request.Context(), account, "claude-sonnet-4-5")

	require.Error(t, err)
	require.Zero(t, repo.setErrorID)
	require.Empty(t, repo.setErrorMsg)
	require.Equal(t, StatusActive, account.Status)
	require.Len(t, upstream.requests, 1)
	body, readErr := io.ReadAll(upstream.requests[0].Body)
	require.NoError(t, readErr)
	require.Contains(t, strings.TrimSpace(string(body)), "anthropic_version")
}

type claudeAccountTestRepo struct {
	mockAccountRepoForGemini
	setErrorID  int64
	setErrorMsg string
}

func (r *claudeAccountTestRepo) SetError(_ context.Context, id int64, errorMsg string) error {
	r.setErrorID = id
	r.setErrorMsg = errorMsg
	return nil
}
