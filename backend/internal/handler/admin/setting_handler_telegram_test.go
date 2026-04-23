package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type telegramServiceStub struct {
	lastConfig *service.TelegramNotificationConfig
	lastText   string
	err        error
}

type failingGetAllRepoStub struct {
	settingHandlerRepoStub
	err error
}

func (s *failingGetAllRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	return nil, s.err
}

func (s *telegramServiceStub) SendTestTextWithConfig(_ context.Context, cfg *service.TelegramNotificationConfig, text string) error {
	if cfg != nil {
		copied := *cfg
		copied.ChatIDs = append([]string(nil), cfg.ChatIDs...)
		copied.ProxyURLs = append([]string(nil), cfg.ProxyURLs...)
		s.lastConfig = &copied
	}
	s.lastText = text
	return s.err
}

func TestSettingHandler_SendTestTelegram_FallsBackToSavedConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{
		values: map[string]string{
			service.SettingKeySiteName:          "Sub2API Stable",
			service.SettingKeyTelegramBotToken:  "saved-token",
			service.SettingKeyTelegramChatIDs:   "1820493278\n-1001234567890",
			service.SettingKeyTelegramProxyURLs: "http://host.docker.internal:58080,http://host.docker.internal:58081",
		},
	}
	svc := service.NewSettingService(repo, &config.Config{})
	telegram := &telegramServiceStub{}
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil, nil)
	handler.telegramService = telegram

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/settings/send-test-telegram", bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.SendTestTelegram(c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, telegram.lastConfig)
	require.Equal(t, "saved-token", telegram.lastConfig.BotToken)
	require.Equal(t, []string{"1820493278", "-1001234567890"}, telegram.lastConfig.ChatIDs)
	require.Equal(t, []string{"http://host.docker.internal:58080", "http://host.docker.internal:58081"}, telegram.lastConfig.ProxyURLs)
	require.Contains(t, telegram.lastText, "[Sub2API Stable] Telegram 测试消息")
}

func TestSettingHandler_SendTestTelegram_ReturnsBadRequestWithServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{
		values: map[string]string{
			service.SettingKeyTelegramBotToken: "saved-token",
		},
	}
	svc := service.NewSettingService(repo, &config.Config{})
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil, nil)
	handler.telegramService = &telegramServiceStub{err: errors.New("telegram send failed: 1820493278: telegram api status 400: Bad Request: chat not found")}

	body := map[string]any{
		"telegram_chat_ids": "1820493278",
	}
	rawBody, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/settings/send-test-telegram", bytes.NewReader(rawBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.SendTestTelegram(c)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "Failed to send test Telegram message: telegram send failed: 1820493278: telegram api status 400: Bad Request: chat not found", resp.Message)
}

func TestSettingHandler_SendTestTelegram_ReturnsServerErrorWhenSavedSettingsFallbackFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &failingGetAllRepoStub{
		settingHandlerRepoStub: settingHandlerRepoStub{
			values: map[string]string{},
		},
		err: errors.New("database unavailable"),
	}
	svc := service.NewSettingService(repo, &config.Config{})
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil, nil)
	handler.telegramService = &telegramServiceStub{}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/settings/send-test-telegram", bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.SendTestTelegram(c)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "Failed to load saved Telegram settings", resp.Message)
}
