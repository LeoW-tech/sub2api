//go:build unit

package dto

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestGroupFromServiceAdmin_FiltersSupportedModelScopesForNonAntigravity(t *testing.T) {
	group := &service.Group{
		ID:                   1,
		Name:                 "openai-group",
		Platform:             service.PlatformOpenAI,
		SupportedModelScopes: []string{"claude", "gemini_text", "gemini_image"},
	}

	out := GroupFromServiceAdmin(group)

	require.NotNil(t, out)
	require.Empty(t, out.SupportedModelScopes)
}

func TestGroupFromServiceAdmin_PreservesSupportedModelScopesForAntigravity(t *testing.T) {
	group := &service.Group{
		ID:                   1,
		Name:                 "antigravity-group",
		Platform:             service.PlatformAntigravity,
		SupportedModelScopes: []string{"claude", "gemini_image"},
	}

	out := GroupFromServiceAdmin(group)

	require.NotNil(t, out)
	require.Equal(t, []string{"claude", "gemini_image"}, out.SupportedModelScopes)
}
