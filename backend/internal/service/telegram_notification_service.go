package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
)

const telegramDefaultBaseURL = "https://api.telegram.org"

var telegramDefaultProxyURLs = []string{
	"http://host.docker.internal:58080",
	"http://host.docker.internal:58081",
	"http://host.docker.internal:58082",
}

type telegramClientFactory func(proxyURL string) (*http.Client, error)

type TelegramNotificationConfig struct {
	Enabled   bool
	BotToken  string
	ChatIDs   []string
	ProxyURLs []string
}

// TelegramNotificationService 仅提供最小化文本推送能力。
type TelegramNotificationService struct {
	settingService *SettingService
	baseURL        string
	clientFactory  telegramClientFactory
}

func NewTelegramNotificationService(settingService *SettingService) *TelegramNotificationService {
	return &TelegramNotificationService{
		settingService: settingService,
		baseURL:        telegramDefaultBaseURL,
		clientFactory:  defaultTelegramClientFactory,
	}
}

func defaultTelegramClientFactory(proxyURL string) (*http.Client, error) {
	return httpclient.GetClient(httpclient.Options{
		ProxyURL:              strings.TrimSpace(proxyURL),
		Timeout:               15 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	})
}

func (s *TelegramNotificationService) SendText(ctx context.Context, text string) error {
	if s == nil || s.settingService == nil {
		return nil
	}
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return fmt.Errorf("load telegram config: %w", err)
	}
	return s.sendTextWithConfig(ctx, cfg, text, false)
}

func (s *TelegramNotificationService) SendTestTextWithConfig(ctx context.Context, cfg *TelegramNotificationConfig, text string) error {
	if s == nil {
		return fmt.Errorf("telegram notification service unavailable")
	}
	return s.sendTextWithConfig(ctx, cfg, text, true)
}

func (s *TelegramNotificationService) sendTextWithConfig(ctx context.Context, cfg *TelegramNotificationConfig, text string, strict bool) error {
	if cfg == nil {
		if strict {
			return fmt.Errorf("telegram config is required")
		}
		return nil
	}

	botToken := strings.TrimSpace(cfg.BotToken)
	chatIDs := append([]string(nil), cfg.ChatIDs...)
	proxyURLs := append([]string(nil), cfg.ProxyURLs...)
	if len(proxyURLs) == 0 {
		proxyURLs = append([]string(nil), telegramDefaultProxyURLs...)
	}

	if strict {
		if botToken == "" {
			return fmt.Errorf("telegram bot token is required")
		}
		if len(chatIDs) == 0 {
			return fmt.Errorf("telegram chat id is required")
		}
	}
	if !strict && (!cfg.Enabled || botToken == "" || len(chatIDs) == 0) {
		return nil
	}

	var sendErrors []string
	for _, chatID := range chatIDs {
		if err := s.sendToChat(ctx, &TelegramNotificationConfig{
			Enabled:   cfg.Enabled,
			BotToken:  botToken,
			ChatIDs:   chatIDs,
			ProxyURLs: proxyURLs,
		}, chatID, text); err != nil {
			sendErrors = append(sendErrors, fmt.Sprintf("%s: %v", chatID, err))
		}
	}
	if len(sendErrors) > 0 {
		return fmt.Errorf("telegram send failed: %s", strings.Join(sendErrors, "; "))
	}
	return nil
}

func (s *TelegramNotificationService) loadConfig(ctx context.Context) (*TelegramNotificationConfig, error) {
	settings, err := s.settingService.GetAllSettings(ctx)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		return &TelegramNotificationConfig{}, nil
	}

	proxyURLs := splitTelegramList(settings.TelegramProxyURLs)
	if len(proxyURLs) == 0 {
		proxyURLs = append([]string(nil), telegramDefaultProxyURLs...)
	}

	return &TelegramNotificationConfig{
		Enabled:   settings.TelegramEnabled,
		BotToken:  strings.TrimSpace(settings.TelegramBotToken),
		ChatIDs:   splitTelegramList(settings.TelegramChatIDs),
		ProxyURLs: proxyURLs,
	}, nil
}

func splitTelegramList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ',' || r == ';'
	})
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func (s *TelegramNotificationService) sendToChat(ctx context.Context, cfg *TelegramNotificationConfig, chatID string, text string) error {
	if cfg == nil {
		return nil
	}
	var lastErr error
	for _, proxyURL := range cfg.ProxyURLs {
		if err := s.sendWithProxy(ctx, proxyURL, cfg.BotToken, chatID, text); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	if lastErr != nil {
		return lastErr
	}
	return s.sendWithProxy(ctx, "", cfg.BotToken, chatID, text)
}

func (s *TelegramNotificationService) sendWithProxy(ctx context.Context, proxyURL, botToken, chatID, text string) error {
	clientFactory := s.clientFactory
	if clientFactory == nil {
		clientFactory = defaultTelegramClientFactory
	}
	client, err := clientFactory(proxyURL)
	if err != nil {
		return err
	}

	body, err := json.Marshal(map[string]any{
		"chat_id":                  chatID,
		"text":                     text,
		"disable_web_page_preview": true,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(s.baseURL, "/")+"/bot"+botToken+"/sendMessage",
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram api status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var payload struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &payload); err == nil && !payload.OK {
			if payload.Description != "" {
				return fmt.Errorf("telegram api rejected message: %s", payload.Description)
			}
			return fmt.Errorf("telegram api rejected message")
		}
	}
	return nil
}
