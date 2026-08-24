//go:build unit

package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func TestGetCheckoutInfoFiltersModelScopesForNonAntigravityPlans(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newPaymentHandlerCheckoutInfoTestClient(t)

	openAIGroup, err := client.Group.Create().
		SetName("openai-subscription").
		SetPlatform(service.PlatformOpenAI).
		SetRateMultiplier(1).
		SetSupportedModelScopes([]string{"claude", "gemini_text", "gemini_image"}).
		Save(ctx)
	require.NoError(t, err)

	antigravityGroup, err := client.Group.Create().
		SetName("antigravity-subscription").
		SetPlatform(service.PlatformAntigravity).
		SetRateMultiplier(1).
		SetSupportedModelScopes([]string{"claude", "gemini_image"}).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.SubscriptionPlan.Create().
		SetGroupID(openAIGroup.ID).
		SetName("OpenAI Codex").
		SetPrice(68).
		SetValidityDays(30).
		SetValidityUnit("day").
		SetForSale(true).
		SetSortOrder(1).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.SubscriptionPlan.Create().
		SetGroupID(antigravityGroup.ID).
		SetName("Antigravity").
		SetPrice(88).
		SetValidityDays(30).
		SetValidityUnit("day").
		SetForSale(true).
		SetSortOrder(2).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.PaymentProviderInstance.Create().
		SetProviderKey(payment.TypeAlipay).
		SetName("Alipay").
		SetConfig("{}").
		SetSupportedTypes(payment.TypeAlipay).
		SetEnabled(true).
		Save(ctx)
	require.NoError(t, err)

	h := NewPaymentHandler(
		nil,
		service.NewPaymentConfigService(client, &paymentCheckoutInfoSettingRepoStub{}, []byte("0123456789abcdef0123456789abcdef")),
	)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/payment/checkout-info", nil)

	h.GetCheckoutInfo(c)

	require.Equal(t, http.StatusOK, recorder.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Plans []struct {
				Name                 string   `json:"name"`
				GroupPlatform        string   `json:"group_platform"`
				SupportedModelScopes []string `json:"supported_model_scopes"`
			} `json:"plans"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data.Plans, 2)

	require.Equal(t, "OpenAI Codex", resp.Data.Plans[0].Name)
	require.Equal(t, service.PlatformOpenAI, resp.Data.Plans[0].GroupPlatform)
	require.Empty(t, resp.Data.Plans[0].SupportedModelScopes)

	require.Equal(t, "Antigravity", resp.Data.Plans[1].Name)
	require.Equal(t, service.PlatformAntigravity, resp.Data.Plans[1].GroupPlatform)
	require.Equal(t, []string{"claude", "gemini_image"}, resp.Data.Plans[1].SupportedModelScopes)
}

func newPaymentHandlerCheckoutInfoTestClient(t *testing.T) *dbent.Client {
	t.Helper()

	dbName := "file:" + strings.NewReplacer("/", "_", " ", "_").Replace(t.Name()) + "?mode=memory&cache=shared"
	db, err := sql.Open("sqlite", dbName)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}

type paymentCheckoutInfoSettingRepoStub struct{}

func (s *paymentCheckoutInfoSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	return nil, nil
}
func (s *paymentCheckoutInfoSettingRepoStub) GetValue(context.Context, string) (string, error) {
	return "", nil
}
func (s *paymentCheckoutInfoSettingRepoStub) Set(context.Context, string, string) error {
	return nil
}
func (s *paymentCheckoutInfoSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	values := make(map[string]string, len(keys))
	for _, key := range keys {
		values[key] = ""
	}
	return values, nil
}
func (s *paymentCheckoutInfoSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	return nil
}
func (s *paymentCheckoutInfoSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}
func (s *paymentCheckoutInfoSettingRepoStub) Delete(context.Context, string) error {
	return nil
}
