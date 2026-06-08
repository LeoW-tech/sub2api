package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/collection"
)

func TestProvideTimingWheelService_ReturnsError(t *testing.T) {
	original := newTimingWheel
	t.Cleanup(func() { newTimingWheel = original })

	newTimingWheel = func(_ time.Duration, _ int, _ collection.Execute) (*collection.TimingWheel, error) {
		return nil, errors.New("boom")
	}

	svc, err := ProvideTimingWheelService()
	if err == nil {
		t.Fatalf("期望返回 error，但得到 nil")
	}
	if svc != nil {
		t.Fatalf("期望返回 nil svc，但得到非空")
	}
}

func TestProvideTimingWheelService_Success(t *testing.T) {
	svc, err := ProvideTimingWheelService()
	if err != nil {
		t.Fatalf("期望 err 为 nil，但得到: %v", err)
	}
	if svc == nil {
		t.Fatalf("期望 svc 非空，但得到 nil")
	}
	svc.Stop()
}

func TestProvideSettingServiceSetsBuildVersionForPublicInjection(t *testing.T) {
	repo := &wireSettingRepoStub{values: map[string]string{}}
	svc := ProvideSettingService(repo, nil, nil, nil, BuildInfo{Version: "0.1.135"})

	payload, err := svc.GetPublicSettingsForInjection(t.Context())
	if err != nil {
		t.Fatalf("GetPublicSettingsForInjection() error = %v", err)
	}

	settings, ok := payload.(*PublicSettingsInjectionPayload)
	if !ok {
		t.Fatalf("payload type = %T, want *PublicSettingsInjectionPayload", payload)
	}
	if settings.Version != "0.1.135" {
		t.Fatalf("settings.Version = %q, want %q", settings.Version, "0.1.135")
	}
}

type wireSettingRepoStub struct {
	values map[string]string
}

func (s *wireSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *wireSettingRepoStub) GetValue(context.Context, string) (string, error) {
	return "", ErrSettingNotFound
}

func (s *wireSettingRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *wireSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *wireSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *wireSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *wireSettingRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}
