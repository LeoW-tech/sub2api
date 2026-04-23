//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestScheduledTestRunnerService_TryRecoverAccount_EnablesSchedulable(t *testing.T) {
	repo := &rateLimitClearRepoStub{
		getByIDAccount: &Account{
			ID:          88,
			Status:      StatusDisabled,
			Schedulable: false,
			Extra:       map[string]any{},
		},
	}
	rateLimitSvc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	runner := NewScheduledTestRunnerService(nil, nil, nil, rateLimitSvc, nil)

	runner.tryRecoverAccount(context.Background(), 88, 7)

	require.Equal(t, 1, repo.getByIDCalls)
	require.Equal(t, 1, repo.setSchedulableCalls)
	require.NotNil(t, repo.setSchedulableValue)
	require.True(t, *repo.setSchedulableValue)
}
