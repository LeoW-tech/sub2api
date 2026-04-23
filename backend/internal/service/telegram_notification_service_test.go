//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitTelegramList(t *testing.T) {
	got := splitTelegramList(" 123 \n456,789;123 ")
	require.Equal(t, []string{"123", "456", "789"}, got)
}

func TestTelegramNotificationService_SendText_UsesNextProxyOnFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/botsecret/sendMessage", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), `"chat_id":"123456"`)
		require.Contains(t, string(body), `"text":"hello"`)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	svc := &TelegramNotificationService{
		settingService: &SettingService{
			settingRepo: &settingRepoStubForTelegram{
				values: map[string]string{
					SettingKeyTelegramEnabled:   "true",
					SettingKeyTelegramBotToken:  "secret",
					SettingKeyTelegramChatIDs:   "123456",
					SettingKeyTelegramProxyURLs: "http://bad-proxy:8080,http://good-proxy:8080",
				},
			},
		},
		baseURL: server.URL,
		clientFactory: func(proxyURL string) (*http.Client, error) {
			if strings.Contains(proxyURL, "bad-proxy") {
				return nil, context.DeadlineExceeded
			}
			return server.Client(), nil
		},
	}

	err := svc.SendText(context.Background(), "hello")
	require.NoError(t, err)
}

type settingRepoStubForTelegram struct {
	values map[string]string
}

func (s *settingRepoStubForTelegram) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}

func (s *settingRepoStubForTelegram) Get(_ context.Context, key string) (*Setting, error) {
	value, err := s.GetValue(context.Background(), key)
	if err != nil {
		return nil, err
	}
	return &Setting{Key: key, Value: value}, nil
}
func (s *settingRepoStubForTelegram) Set(context.Context, string, string) error { panic("unexpected") }
func (s *settingRepoStubForTelegram) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		result[key] = s.values[key]
	}
	return result, nil
}
func (s *settingRepoStubForTelegram) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected")
}
func (s *settingRepoStubForTelegram) GetAll(_ context.Context) (map[string]string, error) {
	return s.values, nil
}
func (s *settingRepoStubForTelegram) Delete(context.Context, string) error { panic("unexpected") }
