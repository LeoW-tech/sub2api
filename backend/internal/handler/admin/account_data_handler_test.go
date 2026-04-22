package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type dataResponse struct {
	Code int         `json:"code"`
	Data dataPayload `json:"data"`
}

type dataPayload struct {
	Type     string        `json:"type"`
	Version  int           `json:"version"`
	Proxies  []dataProxy   `json:"proxies"`
	Accounts []dataAccount `json:"accounts"`
}

type dataProxy struct {
	ProxyKey         string `json:"proxy_key"`
	ProxyExternalKey string `json:"proxy_external_key"`
	Name             string `json:"name"`
	Protocol         string `json:"protocol"`
	Host             string `json:"host"`
	Port             int    `json:"port"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	Status           string `json:"status"`
	ExitIP           string `json:"exit_ip"`
}

type dataAccount struct {
	Name             string         `json:"name"`
	Platform         string         `json:"platform"`
	Type             string         `json:"type"`
	Credentials      map[string]any `json:"credentials"`
	Extra            map[string]any `json:"extra"`
	ProxyKey         *string        `json:"proxy_key"`
	ProxyExternalKey *string        `json:"proxy_external_key"`
	ProxyName        *string        `json:"proxy_name"`
	ExitIP           *string        `json:"exit_ip"`
	Concurrency      *int           `json:"concurrency"`
	LoadFactor       *int           `json:"load_factor"`
	Priority         int            `json:"priority"`
	GroupIDs         []int64        `json:"group_ids"`
}

func setupAccountDataRouter() (*gin.Engine, *stubAdminService) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	adminSvc := newStubAdminService()

	h := NewAccountHandler(
		adminSvc,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	router.GET("/api/v1/admin/accounts/data", h.ExportData)
	router.POST("/api/v1/admin/accounts/data", h.ImportData)
	router.GET("/api/v1/admin/accounts", h.List)
	return router, adminSvc
}

func TestExportDataIncludesSecrets(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()

	proxyID := int64(11)
	adminSvc.proxies = []service.Proxy{
		{
			ID:              proxyID,
			Name:            "proxy",
			ExternalKey:     "door-hk-w10",
			Protocol:        "http",
			Host:            "127.0.0.1",
			Port:            8080,
			Username:        "user",
			Password:        "pass",
			Status:          service.StatusActive,
			ExitIP:          "203.0.113.10",
			ExitIPCheckedAt: ptrTime(time.Unix(1_700_000_000, 0).UTC()),
		},
		{
			ID:       12,
			Name:     "orphan",
			Protocol: "https",
			Host:     "10.0.0.1",
			Port:     443,
			Username: "o",
			Password: "p",
			Status:   service.StatusActive,
		},
	}
	adminSvc.accounts = []service.Account{
		{
			ID:          21,
			Name:        "account",
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeOAuth,
			Credentials: map[string]any{"token": "secret"},
			Extra:       map[string]any{"note": "x"},
			ProxyID:     &proxyID,
			Concurrency: 3,
			LoadFactor:  intPtr(9),
			Priority:    50,
			GroupIDs:    []int64{101, 102},
			Status:      service.StatusDisabled,
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/data", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp dataResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Empty(t, resp.Data.Type)
	require.Equal(t, 0, resp.Data.Version)
	require.Len(t, resp.Data.Proxies, 1)
	require.Equal(t, "pass", resp.Data.Proxies[0].Password)
	require.Equal(t, "door-hk-w10", resp.Data.Proxies[0].ProxyExternalKey)
	require.Equal(t, "203.0.113.10", resp.Data.Proxies[0].ExitIP)
	require.Len(t, resp.Data.Accounts, 1)
	require.Equal(t, "secret", resp.Data.Accounts[0].Credentials["token"])
	require.NotNil(t, resp.Data.Accounts[0].Concurrency)
	require.Equal(t, 3, *resp.Data.Accounts[0].Concurrency)
	require.NotNil(t, resp.Data.Accounts[0].LoadFactor)
	require.Equal(t, 9, *resp.Data.Accounts[0].LoadFactor)
	require.Equal(t, []int64{101, 102}, resp.Data.Accounts[0].GroupIDs)
}

func TestExportDataWithoutProxies(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()

	proxyID := int64(11)
	adminSvc.proxies = []service.Proxy{
		{
			ID:       proxyID,
			Name:     "proxy",
			Protocol: "http",
			Host:     "127.0.0.1",
			Port:     8080,
			Username: "user",
			Password: "pass",
			Status:   service.StatusActive,
		},
	}
	adminSvc.accounts = []service.Account{
		{
			ID:          21,
			Name:        "account",
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeOAuth,
			Credentials: map[string]any{"token": "secret"},
			ProxyID:     &proxyID,
			Concurrency: 3,
			Priority:    50,
			Status:      service.StatusDisabled,
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/data?include_proxies=false", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp dataResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data.Proxies, 0)
	require.Len(t, resp.Data.Accounts, 1)
	require.Nil(t, resp.Data.Accounts[0].ProxyKey)
	require.NotNil(t, resp.Data.Accounts[0].GroupIDs)
	require.Empty(t, resp.Data.Accounts[0].GroupIDs)
}

func TestExportDataPassesAccountFiltersAndSort(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()
	adminSvc.accounts = []service.Account{
		{ID: 1, Name: "acc-1", Status: service.StatusActive},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/admin/accounts/data?platform=openai&type=oauth&status=active&group=12&privacy_mode=blocked&network_status=offline&ip=203.0.113.10&search=keyword&sort_by=priority&sort_order=desc",
		nil,
	)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Equal(t, 1, adminSvc.lastListAccounts.calls)
	require.Equal(t, "openai", adminSvc.lastListAccounts.platform)
	require.Equal(t, "oauth", adminSvc.lastListAccounts.accountType)
	require.Equal(t, "active", adminSvc.lastListAccounts.status)
	require.Equal(t, int64(12), adminSvc.lastListAccounts.groupID)
	require.Equal(t, "blocked", adminSvc.lastListAccounts.privacyMode)
	require.Equal(t, "offline", adminSvc.lastListAccounts.networkStatus)
	require.Equal(t, "203.0.113.10", adminSvc.lastListAccounts.exitIP)
	require.Equal(t, "keyword", adminSvc.lastListAccounts.search)
	require.Equal(t, "priority", adminSvc.lastListAccounts.sortBy)
	require.Equal(t, "desc", adminSvc.lastListAccounts.sortOrder)
}

func TestAccountListPassesIPFilter(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/admin/accounts?ip=203.0.113.10&search=keyword",
		nil,
	)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Equal(t, 1, adminSvc.lastListAccounts.calls)
	require.Equal(t, "203.0.113.10", adminSvc.lastListAccounts.exitIP)
	require.Equal(t, "keyword", adminSvc.lastListAccounts.search)
}

func TestExportDataSelectedIDsOverrideFilters(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/admin/accounts/data?ids=1,2&platform=openai&search=keyword&sort_by=priority&sort_order=desc",
		nil,
	)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp dataResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data.Accounts, 2)
	require.Equal(t, 0, adminSvc.lastListAccounts.calls)
}

func TestImportDataReusesProxyAndSkipsDefaultGroup(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()

	adminSvc.proxies = []service.Proxy{
		{
			ID:       1,
			Name:     "proxy",
			Protocol: "socks5",
			Host:     "1.2.3.4",
			Port:     1080,
			Username: "u",
			Password: "p",
			Status:   service.StatusActive,
		},
	}

	dataPayload := map[string]any{
		"data": map[string]any{
			"type":    dataType,
			"version": dataVersion,
			"proxies": []map[string]any{
				{
					"proxy_key": "socks5|1.2.3.4|1080|u|p",
					"name":      "proxy",
					"protocol":  "socks5",
					"host":      "1.2.3.4",
					"port":      1080,
					"username":  "u",
					"password":  "p",
					"status":    "active",
				},
			},
			"accounts": []map[string]any{
				{
					"name":        "acc",
					"platform":    service.PlatformOpenAI,
					"type":        service.AccountTypeOAuth,
					"credentials": map[string]any{"token": "x"},
					"proxy_key":   "socks5|1.2.3.4|1080|u|p",
					"concurrency": 3,
					"priority":    50,
				},
			},
		},
		"skip_default_group_bind": true,
	}

	body, _ := json.Marshal(dataPayload)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/data", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Len(t, adminSvc.createdProxies, 0)
	require.Len(t, adminSvc.createdAccounts, 1)
	require.True(t, adminSvc.createdAccounts[0].SkipDefaultGroupBind)
}

func TestImportDataAppliesDefaultsAndBindsAllEligibleGroups(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()

	adminSvc.groups = []service.Group{
		{ID: 11, Name: "openai-standard", Platform: service.PlatformOpenAI, Status: service.StatusActive},
		{ID: 12, Name: "openai-subscription", Platform: service.PlatformOpenAI, Status: service.StatusActive},
		{ID: 13, Name: "openai-inactive", Platform: service.PlatformOpenAI, Status: "inactive"},
		{ID: 21, Name: "anthropic-standard", Platform: service.PlatformAnthropic, Status: service.StatusActive},
	}

	dataPayload := map[string]any{
		"data": map[string]any{
			"type":    dataType,
			"version": dataVersion,
			"proxies": []map[string]any{},
			"accounts": []map[string]any{{
				"name":        "acc-defaults",
				"platform":    service.PlatformOpenAI,
				"type":        service.AccountTypeOAuth,
				"credentials": map[string]any{"token": "x"},
				"priority":    50,
			}},
		},
	}

	body, _ := json.Marshal(dataPayload)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/data", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Len(t, adminSvc.createdAccounts, 1)
	created := adminSvc.createdAccounts[0]
	require.Equal(t, 10, created.Concurrency)
	require.NotNil(t, created.LoadFactor)
	require.Equal(t, 10, *created.LoadFactor)
	require.Equal(t, []int64{11, 12}, created.GroupIDs)
}

func TestImportDataPreservesExplicitValuesOverDefaults(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()

	adminSvc.groups = []service.Group{
		{ID: 11, Name: "openai-standard", Platform: service.PlatformOpenAI, Status: service.StatusActive},
		{ID: 12, Name: "openai-subscription", Platform: service.PlatformOpenAI, Status: service.StatusActive},
	}

	dataPayload := map[string]any{
		"data": map[string]any{
			"type":    dataType,
			"version": dataVersion,
			"proxies": []map[string]any{},
			"accounts": []map[string]any{{
				"name":        "acc-explicit",
				"platform":    service.PlatformOpenAI,
				"type":        service.AccountTypeOAuth,
				"credentials": map[string]any{"token": "x"},
				"concurrency": 7,
				"load_factor": 8,
				"group_ids":   []int64{12},
				"priority":    50,
			}},
		},
		"default_concurrency":      10,
		"default_load_factor":      10,
		"bind_all_eligible_groups": true,
	}

	body, _ := json.Marshal(dataPayload)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/data", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Len(t, adminSvc.createdAccounts, 1)
	created := adminSvc.createdAccounts[0]
	require.Equal(t, 7, created.Concurrency)
	require.NotNil(t, created.LoadFactor)
	require.Equal(t, 8, *created.LoadFactor)
	require.Equal(t, []int64{12}, created.GroupIDs)
}

func TestImportDataExplicitEmptyGroupIDsSkipsDefaultBinding(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()

	adminSvc.groups = []service.Group{
		{ID: 11, Name: "openai-standard", Platform: service.PlatformOpenAI, Status: service.StatusActive},
	}

	dataPayload := map[string]any{
		"data": map[string]any{
			"type":    dataType,
			"version": dataVersion,
			"proxies": []map[string]any{},
			"accounts": []map[string]any{{
				"name":        "acc-empty-groups",
				"platform":    service.PlatformOpenAI,
				"type":        service.AccountTypeOAuth,
				"credentials": map[string]any{"token": "x"},
				"group_ids":   []int64{},
				"priority":    50,
			}},
		},
	}

	body, _ := json.Marshal(dataPayload)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/data", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Len(t, adminSvc.createdAccounts, 1)
	created := adminSvc.createdAccounts[0]
	require.Empty(t, created.GroupIDs)
	require.True(t, created.SkipDefaultGroupBind)
}

func TestImportDataFallsBackToLegacySkipWhenBindAllEligibleGroupsDisabled(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()

	adminSvc.groups = []service.Group{
		{ID: 11, Name: "openai-standard", Platform: service.PlatformOpenAI, Status: service.StatusActive},
	}

	dataPayload := map[string]any{
		"data": map[string]any{
			"type":    dataType,
			"version": dataVersion,
			"proxies": []map[string]any{},
			"accounts": []map[string]any{{
				"name":        "acc-legacy",
				"platform":    service.PlatformOpenAI,
				"type":        service.AccountTypeOAuth,
				"credentials": map[string]any{"token": "x"},
				"priority":    50,
			}},
		},
		"bind_all_eligible_groups": false,
		"skip_default_group_bind":  true,
	}

	body, _ := json.Marshal(dataPayload)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/data", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Len(t, adminSvc.createdAccounts, 1)
	created := adminSvc.createdAccounts[0]
	require.Nil(t, created.GroupIDs)
	require.True(t, created.SkipDefaultGroupBind)
}

func TestImportDataBindsAccountByProxyExternalKey(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()

	adminSvc.proxies = []service.Proxy{
		{
			ID:          88,
			Name:        "🇭🇰 香港 W10 | IEPL",
			ExternalKey: "door-hk-w10",
			Protocol:    "http",
			Host:        "host.docker.internal",
			Port:        58052,
			Status:      service.StatusActive,
		},
	}

	dataPayload := map[string]any{
		"data": map[string]any{
			"type":    dataType,
			"version": dataVersion,
			"proxies": []map[string]any{},
			"accounts": []map[string]any{{
				"name":               "acc-external-key",
				"platform":           service.PlatformOpenAI,
				"type":               service.AccountTypeOAuth,
				"credentials":        map[string]any{"token": "x"},
				"proxy_external_key": "door-hk-w10",
				"concurrency":        3,
				"priority":           50,
			}},
		},
	}

	body, _ := json.Marshal(dataPayload)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/data", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Len(t, adminSvc.createdAccounts, 1)
	require.NotNil(t, adminSvc.createdAccounts[0].ProxyID)
	require.Equal(t, int64(88), *adminSvc.createdAccounts[0].ProxyID)
}

func TestImportDataBindsAccountByProxyNameAndPersistsExitIP(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()

	adminSvc.proxies = []service.Proxy{
		{
			ID:       99,
			Name:     "🇭🇰 香港 W10 | IEPL",
			Protocol: "http",
			Host:     "host.docker.internal",
			Port:     58053,
			Status:   service.StatusActive,
		},
	}

	exitIP := "203.0.113.20"
	dataPayload := map[string]any{
		"data": map[string]any{
			"type":    dataType,
			"version": dataVersion,
			"proxies": []map[string]any{},
			"accounts": []map[string]any{{
				"name":        "acc-proxy-name",
				"platform":    service.PlatformOpenAI,
				"type":        service.AccountTypeOAuth,
				"credentials": map[string]any{"token": "x"},
				"proxy_name":  "🇭🇰 香港 W10 | IEPL",
				"exit_ip":     exitIP,
				"concurrency": 3,
				"priority":    50,
			}},
		},
	}

	body, _ := json.Marshal(dataPayload)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/data", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Len(t, adminSvc.createdAccounts, 1)
	require.NotNil(t, adminSvc.createdAccounts[0].ProxyID)
	require.Equal(t, int64(99), *adminSvc.createdAccounts[0].ProxyID)
	require.Len(t, adminSvc.updatedProxyIDs, 1)
	require.Equal(t, int64(99), adminSvc.updatedProxyIDs[0])
	require.Equal(t, exitIP, derefString(adminSvc.updatedProxies[0].ExitIP))
}

func ptrTime(v time.Time) *time.Time {
	return &v
}

func intPtr(v int) *int {
	return &v
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
