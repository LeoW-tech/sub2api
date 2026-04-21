package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type crsSyncTestAccountRepo struct {
	created        []*Account
	updated        []*Account
	existingByCRS  map[string]*Account
	existingCRSIDs map[string]int64
}

var _ AccountRepository = (*crsSyncTestAccountRepo)(nil)

func (r *crsSyncTestAccountRepo) Create(_ context.Context, account *Account) error {
	cloned := *account
	r.created = append(r.created, &cloned)
	if r.existingByCRS == nil {
		r.existingByCRS = make(map[string]*Account)
	}
	if crsID, _ := cloned.Extra["crs_account_id"].(string); crsID != "" {
		r.existingByCRS[crsID] = &cloned
	}
	return nil
}

func (r *crsSyncTestAccountRepo) GetByCRSAccountID(_ context.Context, crsAccountID string) (*Account, error) {
	if r.existingByCRS == nil {
		return nil, nil
	}
	return r.existingByCRS[crsAccountID], nil
}

func (r *crsSyncTestAccountRepo) ListCRSAccountIDs(_ context.Context) (map[string]int64, error) {
	if r.existingCRSIDs == nil {
		return map[string]int64{}, nil
	}
	return r.existingCRSIDs, nil
}

func (r *crsSyncTestAccountRepo) Update(_ context.Context, account *Account) error {
	cloned := *account
	r.updated = append(r.updated, &cloned)
	return nil
}

func (r *crsSyncTestAccountRepo) GetByID(context.Context, int64) (*Account, error) {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) GetByIDs(context.Context, []int64) ([]*Account, error) {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) ExistsByID(context.Context, int64) (bool, error) {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) FindByExtraField(context.Context, string, any) ([]Account, error) {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) Delete(context.Context, int64) error { panic("unexpected") }
func (r *crsSyncTestAccountRepo) List(context.Context, pagination.PaginationParams) ([]Account, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, string, int64, string, string) ([]Account, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) ListByGroup(context.Context, int64) ([]Account, error) {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) ListActive(context.Context) ([]Account, error) {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) ListByPlatform(context.Context, string) ([]Account, error) {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) UpdateLastUsed(context.Context, int64) error { panic("unexpected") }
func (r *crsSyncTestAccountRepo) BatchUpdateLastUsed(context.Context, map[int64]time.Time) error {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) SetError(context.Context, int64, string) error { panic("unexpected") }
func (r *crsSyncTestAccountRepo) ClearError(context.Context, int64) error       { panic("unexpected") }
func (r *crsSyncTestAccountRepo) SetSchedulable(context.Context, int64, bool) error {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) AutoPauseExpiredAccounts(context.Context, time.Time) (int64, error) {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) BindGroups(context.Context, int64, []int64) error {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) ListSchedulable(context.Context) ([]Account, error) {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) ListSchedulableByGroupID(context.Context, int64) ([]Account, error) {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) ListSchedulableByPlatform(context.Context, string) ([]Account, error) {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]Account, error) {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) ListSchedulableByPlatforms(context.Context, []string) ([]Account, error) {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) ListSchedulableByGroupIDAndPlatforms(context.Context, int64, []string) ([]Account, error) {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) ListSchedulableUngroupedByPlatform(context.Context, string) ([]Account, error) {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) ListSchedulableUngroupedByPlatforms(context.Context, []string) ([]Account, error) {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) SetRateLimited(context.Context, int64, time.Time) error {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) SetModelRateLimit(context.Context, int64, string, time.Time) error {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) SetOverloaded(context.Context, int64, time.Time) error {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) ClearTempUnschedulable(context.Context, int64) error {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) PauseAccountsByProxyNetwork(context.Context, int64) ([]int64, error) {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) ResumeAccountsByProxyNetwork(context.Context, int64) ([]int64, error) {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) PauseAccountByNetwork(context.Context, int64) (bool, error) {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) RestoreAccountFromNetworkPause(context.Context, int64) (bool, error) {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) ClearNetworkAutoPause(context.Context, int64) error {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) ClearRateLimit(context.Context, int64) error {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) ClearAntigravityQuotaScopes(context.Context, int64) error {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) ClearModelRateLimits(context.Context, int64) error {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) UpdateSessionWindow(context.Context, int64, *time.Time, *time.Time, string) error {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) UpdateExtra(context.Context, int64, map[string]any) error {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) BulkUpdate(context.Context, []int64, AccountBulkUpdate) (int64, error) {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) IncrementQuotaUsed(context.Context, int64, float64) error {
	panic("unexpected")
}
func (r *crsSyncTestAccountRepo) ResetQuotaUsed(context.Context, int64) error {
	panic("unexpected")
}

type crsSyncTestProxyRepo struct {
	active  []Proxy
	created []*Proxy
}

var _ ProxyRepository = (*crsSyncTestProxyRepo)(nil)

func (r *crsSyncTestProxyRepo) Create(_ context.Context, proxy *Proxy) error {
	cloned := *proxy
	cloned.ID = int64(len(r.active) + len(r.created) + 1)
	proxy.ID = cloned.ID
	r.created = append(r.created, &cloned)
	return nil
}

func (r *crsSyncTestProxyRepo) ListActive(_ context.Context) ([]Proxy, error) {
	return append([]Proxy(nil), r.active...), nil
}

func (r *crsSyncTestProxyRepo) GetByID(context.Context, int64) (*Proxy, error) {
	panic("unexpected")
}
func (r *crsSyncTestProxyRepo) ListByIDs(context.Context, []int64) ([]Proxy, error) {
	panic("unexpected")
}
func (r *crsSyncTestProxyRepo) Update(context.Context, *Proxy) error { panic("unexpected") }
func (r *crsSyncTestProxyRepo) Delete(context.Context, int64) error  { panic("unexpected") }
func (r *crsSyncTestProxyRepo) List(context.Context, pagination.PaginationParams) ([]Proxy, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (r *crsSyncTestProxyRepo) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string) ([]Proxy, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (r *crsSyncTestProxyRepo) ListWithFiltersAndAccountCount(context.Context, pagination.PaginationParams, string, string, string) ([]ProxyWithAccountCount, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (r *crsSyncTestProxyRepo) ListActiveWithAccountCount(context.Context) ([]ProxyWithAccountCount, error) {
	panic("unexpected")
}
func (r *crsSyncTestProxyRepo) ExistsByHostPortAuth(context.Context, string, int, string, string) (bool, error) {
	panic("unexpected")
}
func (r *crsSyncTestProxyRepo) CountAccountsByProxyID(context.Context, int64) (int64, error) {
	panic("unexpected")
}
func (r *crsSyncTestProxyRepo) ListAccountSummariesByProxyID(context.Context, int64) ([]ProxyAccountSummary, error) {
	panic("unexpected")
}

func newCRSTestServer(t *testing.T, exportBody any) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/web/auth/login":
			require.Equal(t, http.MethodPost, r.Method)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"token":   "test-admin-token",
			})
		case "/admin/sync/export-accounts":
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "Bearer test-admin-token", r.Header.Get("Authorization"))
			_ = json.NewEncoder(w).Encode(exportBody)
		default:
			http.NotFound(w, r)
		}
	}))
}

func newCRSSyncTestConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	return cfg
}

func TestResolveCRSProxyBindingByName_StrictMatchWithoutNormalization(t *testing.T) {
	index := buildCRSProxyNameIndex([]Proxy{
		{ID: 7, Name: "🇭🇰 香港W01"},
	})

	matched := resolveCRSProxyBindingByName("🇭🇰 香港W01", index)
	require.Equal(t, crsProxyMatchMatched, matched.MatchStatus)
	require.NotNil(t, matched.ProxyID)
	require.Equal(t, int64(7), *matched.ProxyID)

	notFound := resolveCRSProxyBindingByName(" 🇭🇰 香港W01 ", index)
	require.Equal(t, crsProxyMatchNotFound, notFound.MatchStatus)
	require.Nil(t, notFound.ProxyID)
	require.Len(t, notFound.Warnings, 1)
}

func TestSyncFromCRS_UsesProxyNameBeforeLegacyProxySync(t *testing.T) {
	accountRepo := &crsSyncTestAccountRepo{}
	proxyRepo := &crsSyncTestProxyRepo{
		active: []Proxy{
			{ID: 7, Name: "🇭🇰 香港W01", Protocol: "http", Host: "local-proxy", Port: 8080, Status: StatusActive},
		},
	}
	server := newCRSTestServer(t, map[string]any{
		"success": true,
		"data": map[string]any{
			"openaiResponsesAccounts": []map[string]any{
				{
					"kind":        "openai-responses",
					"id":          "acc-matched",
					"name":        "Matched",
					"platform":    PlatformOpenAI,
					"proxy_name":  "🇭🇰 香港W01",
					"schedulable": true,
					"isActive":    true,
					"status":      StatusActive,
					"priority":    10,
					"proxy": map[string]any{
						"protocol": "http",
						"host":     "should-not-create",
						"port":     9001,
					},
					"credentials": map[string]any{
						"api_key": "sk-matched",
					},
				},
				{
					"kind":        "openai-responses",
					"id":          "acc-unmatched",
					"name":        "Unmatched",
					"platform":    PlatformOpenAI,
					"proxy_name":  "🇭🇰 香港W02",
					"schedulable": true,
					"isActive":    true,
					"status":      StatusActive,
					"priority":    20,
					"proxy": map[string]any{
						"protocol": "http",
						"host":     "also-should-not-create",
						"port":     9002,
					},
					"credentials": map[string]any{
						"api_key": "sk-unmatched",
					},
				},
			},
		},
	})
	defer server.Close()

	svc := NewCRSSyncService(accountRepo, proxyRepo, nil, nil, nil, newCRSSyncTestConfig())
	result, err := svc.SyncFromCRS(context.Background(), SyncFromCRSInput{
		BaseURL:     server.URL,
		Username:    "admin",
		Password:    "password",
		SyncProxies: true,
	})
	require.NoError(t, err)
	require.Equal(t, 2, result.Created)
	require.Equal(t, 0, result.Failed)
	require.Equal(t, 1, result.ProxyMatched)
	require.Equal(t, 1, result.ProxyUnmatched)
	require.Len(t, result.Items, 2)
	require.Len(t, proxyRepo.created, 0, "proxy_name 存在时不应再回退到 legacy 代理创建")
	require.Len(t, accountRepo.created, 2)
	require.NotNil(t, accountRepo.created[0].ProxyID)
	require.Equal(t, int64(7), *accountRepo.created[0].ProxyID)
	require.Nil(t, accountRepo.created[1].ProxyID)
	require.Equal(t, "🇭🇰 香港W01", result.Items[0].ProxyName)
	require.NotNil(t, result.Items[0].MatchedProxyID)
	require.Equal(t, int64(7), *result.Items[0].MatchedProxyID)
	require.Contains(t, result.Items[1].Warnings[0], "proxy_name not found")
}

func TestPreviewFromCRS_ReportsProxyMatchStatus(t *testing.T) {
	accountRepo := &crsSyncTestAccountRepo{
		existingCRSIDs: map[string]int64{
			"acc-existing": 11,
		},
	}
	proxyRepo := &crsSyncTestProxyRepo{
		active: []Proxy{
			{ID: 21, Name: "dup", Status: StatusActive},
			{ID: 22, Name: "dup", Status: StatusActive},
		},
	}
	server := newCRSTestServer(t, map[string]any{
		"success": true,
		"data": map[string]any{
			"openaiResponsesAccounts": []map[string]any{
				{
					"kind":        "openai-responses",
					"id":          "acc-existing",
					"name":        "Existing",
					"platform":    PlatformOpenAI,
					"proxy_name":  "dup",
					"schedulable": true,
					"isActive":    true,
					"status":      StatusActive,
					"credentials": map[string]any{"api_key": "sk-existing"},
				},
				{
					"kind":        "openai-responses",
					"id":          "acc-new",
					"name":        "New",
					"platform":    PlatformOpenAI,
					"schedulable": true,
					"isActive":    true,
					"status":      StatusActive,
					"credentials": map[string]any{"api_key": "sk-new"},
				},
			},
		},
	})
	defer server.Close()

	svc := NewCRSSyncService(accountRepo, proxyRepo, nil, nil, nil, newCRSSyncTestConfig())
	result, err := svc.PreviewFromCRS(context.Background(), SyncFromCRSInput{
		BaseURL:  server.URL,
		Username: "admin",
		Password: "password",
	})
	require.NoError(t, err)
	require.Equal(t, 0, result.ProxyMatched)
	require.Equal(t, 1, result.ProxyUnmatched)
	require.Len(t, result.ExistingAccounts, 1)
	require.Len(t, result.NewAccounts, 1)
	require.Equal(t, crsProxyMatchConflict, result.ExistingAccounts[0].ProxyMatchStatus)
	require.Equal(t, "dup", result.ExistingAccounts[0].ProxyName)
	require.Contains(t, result.ExistingAccounts[0].Warnings[0], "proxy_name is ambiguous")
	require.Equal(t, crsProxyMatchMissing, result.NewAccounts[0].ProxyMatchStatus)
}
