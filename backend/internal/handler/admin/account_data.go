package admin

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"log/slog"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	dataType       = "sub2api-data"
	legacyDataType = "sub2api-bundle"
	dataVersion    = 1
	dataPageCap    = 1000
)

type DataPayload struct {
	Type       string        `json:"type,omitempty"`
	Version    int           `json:"version,omitempty"`
	ExportedAt string        `json:"exported_at"`
	Proxies    []DataProxy   `json:"proxies"`
	Accounts   []DataAccount `json:"accounts"`
}

type DataProxy struct {
	ProxyKey         string `json:"proxy_key"`
	ProxyExternalKey string `json:"proxy_external_key,omitempty"`
	Name             string `json:"name"`
	Protocol         string `json:"protocol"`
	Host             string `json:"host"`
	Port             int    `json:"port"`
	Username         string `json:"username,omitempty"`
	Password         string `json:"password,omitempty"`
	Status           string `json:"status"`
	ExitIP           string `json:"exit_ip,omitempty"`
	ExitIPCheckedAt  *int64 `json:"exit_ip_checked_at,omitempty"`
}

type DataAccount struct {
	Name               string         `json:"name"`
	Notes              *string        `json:"notes,omitempty"`
	Platform           string         `json:"platform"`
	Type               string         `json:"type"`
	Credentials        map[string]any `json:"credentials"`
	Extra              map[string]any `json:"extra,omitempty"`
	ProxyKey           *string        `json:"proxy_key,omitempty"`
	ProxyExternalKey   *string        `json:"proxy_external_key,omitempty"`
	ProxyName          *string        `json:"proxy_name,omitempty"`
	ExitIP             *string        `json:"exit_ip,omitempty"`
	Concurrency        *int           `json:"concurrency,omitempty"`
	LoadFactor         *int           `json:"load_factor,omitempty"`
	Priority           int            `json:"priority"`
	RateMultiplier     *float64       `json:"rate_multiplier,omitempty"`
	GroupIDs           []int64        `json:"group_ids"`
	ExpiresAt          *int64         `json:"expires_at,omitempty"`
	AutoPauseOnExpired *bool          `json:"auto_pause_on_expired,omitempty"`
}

type DataImportRequest struct {
	Data                  DataPayload `json:"data"`
	SkipDefaultGroupBind  *bool       `json:"skip_default_group_bind"`
	DefaultConcurrency    *int        `json:"default_concurrency"`
	DefaultLoadFactor     *int        `json:"default_load_factor"`
	BindAllEligibleGroups *bool       `json:"bind_all_eligible_groups"`
}

type DataImportResult struct {
	ProxyCreated   int               `json:"proxy_created"`
	ProxyReused    int               `json:"proxy_reused"`
	ProxyFailed    int               `json:"proxy_failed"`
	AccountCreated int               `json:"account_created"`
	AccountFailed  int               `json:"account_failed"`
	Errors         []DataImportError `json:"errors,omitempty"`
}

type DataImportError struct {
	Kind     string `json:"kind"`
	Name     string `json:"name,omitempty"`
	ProxyKey string `json:"proxy_key,omitempty"`
	Message  string `json:"message"`
}

func buildProxyKey(protocol, host string, port int, username, password string) string {
	return fmt.Sprintf("%s|%s|%d|%s|%s", strings.TrimSpace(protocol), strings.TrimSpace(host), port, strings.TrimSpace(username), strings.TrimSpace(password))
}

func (h *AccountHandler) ExportData(c *gin.Context) {
	ctx := c.Request.Context()

	selectedIDs, err := parseAccountIDs(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	accounts, err := h.resolveExportAccounts(ctx, selectedIDs, c)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	includeProxies, err := parseIncludeProxies(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var proxies []service.Proxy
	if includeProxies {
		proxies, err = h.resolveExportProxies(ctx, accounts)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
	} else {
		proxies = []service.Proxy{}
	}

	proxyKeyByID := make(map[int64]string, len(proxies))
	proxyByID := make(map[int64]service.Proxy, len(proxies))
	dataProxies := make([]DataProxy, 0, len(proxies))
	for i := range proxies {
		p := proxies[i]
		key := buildProxyKey(p.Protocol, p.Host, p.Port, p.Username, p.Password)
		proxyKeyByID[p.ID] = key
		proxyByID[p.ID] = p
		var exitIPCheckedAt *int64
		if p.ExitIPCheckedAt != nil {
			ts := p.ExitIPCheckedAt.Unix()
			exitIPCheckedAt = &ts
		}
		dataProxies = append(dataProxies, DataProxy{
			ProxyKey:         key,
			ProxyExternalKey: p.ExternalKey,
			Name:             p.Name,
			Protocol:         p.Protocol,
			Host:             p.Host,
			Port:             p.Port,
			Username:         p.Username,
			Password:         p.Password,
			Status:           p.Status,
			ExitIP:           p.ExitIP,
			ExitIPCheckedAt:  exitIPCheckedAt,
		})
	}

	dataAccounts := make([]DataAccount, 0, len(accounts))
	for i := range accounts {
		acc := accounts[i]
		var proxyKey *string
		if acc.ProxyID != nil {
			if key, ok := proxyKeyByID[*acc.ProxyID]; ok {
				proxyKey = &key
			}
		}
		var proxyExternalKey *string
		var proxyName *string
		var exitIP *string
		if acc.ProxyID != nil {
			if proxyMeta, ok := proxyByID[*acc.ProxyID]; ok {
				if proxyMeta.ExternalKey != "" {
					proxyExternalKey = stringPtr(proxyMeta.ExternalKey)
				}
				if proxyMeta.Name != "" {
					proxyName = stringPtr(proxyMeta.Name)
				}
				if proxyMeta.ExitIP != "" {
					exitIP = stringPtr(proxyMeta.ExitIP)
				}
			}
		}
		var expiresAt *int64
		if acc.ExpiresAt != nil {
			v := acc.ExpiresAt.Unix()
			expiresAt = &v
		}
		dataAccounts = append(dataAccounts, DataAccount{
			Name:               acc.Name,
			Notes:              acc.Notes,
			Platform:           acc.Platform,
			Type:               acc.Type,
			Credentials:        acc.Credentials,
			Extra:              acc.Extra,
			ProxyKey:           proxyKey,
			ProxyExternalKey:   proxyExternalKey,
			ProxyName:          proxyName,
			ExitIP:             exitIP,
			Concurrency:        intValuePtr(acc.Concurrency),
			LoadFactor:         acc.LoadFactor,
			Priority:           acc.Priority,
			RateMultiplier:     acc.RateMultiplier,
			GroupIDs:           append([]int64{}, acc.GroupIDs...),
			ExpiresAt:          expiresAt,
			AutoPauseOnExpired: &acc.AutoPauseOnExpired,
		})
	}

	payload := DataPayload{
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Proxies:    dataProxies,
		Accounts:   dataAccounts,
	}

	response.Success(c, payload)
}

func (h *AccountHandler) ImportData(c *gin.Context) {
	var req DataImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if err := validateDataHeader(req.Data); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := validateDataImportRequest(req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	executeAdminIdempotentJSON(c, "admin.accounts.import_data", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		return h.importData(ctx, req)
	})
}

func (h *AccountHandler) importData(ctx context.Context, req DataImportRequest) (DataImportResult, error) {
	skipDefaultGroupBind := true
	if req.SkipDefaultGroupBind != nil {
		skipDefaultGroupBind = *req.SkipDefaultGroupBind
	}
	bindAllEligibleGroups := true
	if req.BindAllEligibleGroups != nil {
		bindAllEligibleGroups = *req.BindAllEligibleGroups
	}
	defaultConcurrency := 10
	if req.DefaultConcurrency != nil {
		defaultConcurrency = *req.DefaultConcurrency
	}
	defaultLoadFactor := 10
	if req.DefaultLoadFactor != nil {
		defaultLoadFactor = *req.DefaultLoadFactor
	}

	dataPayload := req.Data
	result := DataImportResult{}

	existingProxies, err := h.listAllProxies(ctx)
	if err != nil {
		return result, err
	}

	proxyKeyToID := make(map[string]int64, len(existingProxies))
	proxyExternalKeyToID := make(map[string]int64, len(existingProxies))
	proxyNameToIDs := make(map[string][]int64, len(existingProxies))
	for i := range existingProxies {
		p := existingProxies[i]
		key := buildProxyKey(p.Protocol, p.Host, p.Port, p.Username, p.Password)
		proxyKeyToID[key] = p.ID
		if trimmed := strings.TrimSpace(p.ExternalKey); trimmed != "" {
			proxyExternalKeyToID[trimmed] = p.ID
		}
		nameKey := strings.TrimSpace(p.Name)
		if nameKey != "" {
			proxyNameToIDs[nameKey] = append(proxyNameToIDs[nameKey], p.ID)
		}
	}

	for i := range dataPayload.Proxies {
		item := dataPayload.Proxies[i]
		key := item.ProxyKey
		if key == "" {
			key = buildProxyKey(item.Protocol, item.Host, item.Port, item.Username, item.Password)
		}
		if err := validateDataProxy(item); err != nil {
			result.ProxyFailed++
			result.Errors = append(result.Errors, DataImportError{
				Kind:     "proxy",
				Name:     item.Name,
				ProxyKey: key,
				Message:  err.Error(),
			})
			continue
		}
		normalizedStatus := normalizeProxyStatus(item.Status)
		if existingID, ok := resolveExistingProxyID(key, strings.TrimSpace(item.ProxyExternalKey), "", proxyKeyToID, proxyExternalKeyToID, proxyNameToIDs); ok {
			proxyKeyToID[key] = existingID
			if strings.TrimSpace(item.ProxyExternalKey) != "" {
				proxyExternalKeyToID[strings.TrimSpace(item.ProxyExternalKey)] = existingID
			}
			result.ProxyReused++
			updateInput := buildImportedProxyUpdate(item, normalizedStatus)
			if proxy, getErr := h.adminService.GetProxy(ctx, existingID); getErr == nil && proxy != nil && shouldUpdateImportedProxy(proxy, item, normalizedStatus) {
				_, _ = h.adminService.UpdateProxy(ctx, existingID, updateInput)
			}
			continue
		}

		created, createErr := h.adminService.CreateProxy(ctx, &service.CreateProxyInput{
			Name:            defaultProxyName(item.Name),
			ExternalKey:     strings.TrimSpace(item.ProxyExternalKey),
			Protocol:        item.Protocol,
			Host:            item.Host,
			Port:            item.Port,
			Username:        item.Username,
			Password:        item.Password,
			ExitIP:          strings.TrimSpace(item.ExitIP),
			ExitIPCheckedAt: unixPtrToTime(item.ExitIPCheckedAt),
		})
		if createErr != nil {
			result.ProxyFailed++
			result.Errors = append(result.Errors, DataImportError{
				Kind:     "proxy",
				Name:     item.Name,
				ProxyKey: key,
				Message:  createErr.Error(),
			})
			continue
		}
		proxyKeyToID[key] = created.ID
		if strings.TrimSpace(item.ProxyExternalKey) != "" {
			proxyExternalKeyToID[strings.TrimSpace(item.ProxyExternalKey)] = created.ID
		}
		result.ProxyCreated++

		if normalizedStatus != "" && normalizedStatus != created.Status {
			_, _ = h.adminService.UpdateProxy(ctx, created.ID, &service.UpdateProxyInput{
				Status: normalizedStatus,
			})
		}
	}

	eligibleGroupIDsByPlatform := map[string][]int64{}
	if bindAllEligibleGroups {
		platformSet := make(map[string]struct{})
		for i := range dataPayload.Accounts {
			platform := strings.TrimSpace(dataPayload.Accounts[i].Platform)
			if platform == "" {
				continue
			}
			platformSet[platform] = struct{}{}
		}
		for platform := range platformSet {
			groups, groupErr := h.adminService.GetAllGroupsByPlatform(ctx, platform)
			if groupErr != nil {
				return result, groupErr
			}
			ids := make([]int64, 0, len(groups))
			for i := range groups {
				group := groups[i]
				if group.Platform != platform || group.Status != service.StatusActive {
					continue
				}
				ids = append(ids, group.ID)
			}
			eligibleGroupIDsByPlatform[platform] = ids
		}
	}

	// 收集需要异步设置隐私的 Antigravity OAuth 账号
	var privacyAccounts []*service.Account

	for i := range dataPayload.Accounts {
		item := dataPayload.Accounts[i]
		if err := validateDataAccount(item); err != nil {
			result.AccountFailed++
			result.Errors = append(result.Errors, DataImportError{
				Kind:    "account",
				Name:    item.Name,
				Message: err.Error(),
			})
			continue
		}

		var proxyID *int64
		if resolvedID, err := resolveAccountProxyID(item, proxyKeyToID, proxyExternalKeyToID, proxyNameToIDs); err != nil {
			result.AccountFailed++
			result.Errors = append(result.Errors, DataImportError{
				Kind:     "account",
				Name:     item.Name,
				ProxyKey: derefOptionalString(item.ProxyKey),
				Message:  err.Error(),
			})
			continue
		} else if resolvedID != nil {
			proxyID = resolvedID
		}

		if proxyID != nil && item.ExitIP != nil {
			exitIP := strings.TrimSpace(*item.ExitIP)
			if exitIP != "" {
				_, _ = h.adminService.UpdateProxy(ctx, *proxyID, &service.UpdateProxyInput{
					ExitIP: &exitIP,
				})
			}
		}

		enrichCredentialsFromIDToken(&item)

		finalConcurrency := defaultConcurrency
		if item.Concurrency != nil {
			finalConcurrency = *item.Concurrency
		}

		var finalLoadFactor *int
		if item.LoadFactor != nil {
			finalLoadFactor = intValuePtr(*item.LoadFactor)
		} else {
			finalLoadFactor = intValuePtr(defaultLoadFactor)
		}

		accountGroupIDs := []int64(nil)
		handledGroupBinding := false
		if item.GroupIDs != nil {
			accountGroupIDs = append([]int64(nil), item.GroupIDs...)
			handledGroupBinding = true
		} else if bindAllEligibleGroups {
			accountGroupIDs = append([]int64(nil), eligibleGroupIDsByPlatform[item.Platform]...)
			handledGroupBinding = true
		}

		accountSkipDefaultGroupBind := skipDefaultGroupBind
		if handledGroupBinding {
			accountSkipDefaultGroupBind = true
		}

		accountInput := &service.CreateAccountInput{
			Name:                 item.Name,
			Notes:                item.Notes,
			Platform:             item.Platform,
			Type:                 item.Type,
			Credentials:          item.Credentials,
			Extra:                item.Extra,
			ProxyID:              proxyID,
			Concurrency:          finalConcurrency,
			Priority:             item.Priority,
			RateMultiplier:       item.RateMultiplier,
			LoadFactor:           finalLoadFactor,
			GroupIDs:             accountGroupIDs,
			ExpiresAt:            item.ExpiresAt,
			AutoPauseOnExpired:   item.AutoPauseOnExpired,
			SkipDefaultGroupBind: accountSkipDefaultGroupBind,
		}

		created, err := h.adminService.CreateAccount(ctx, accountInput)
		if err != nil {
			result.AccountFailed++
			result.Errors = append(result.Errors, DataImportError{
				Kind:    "account",
				Name:    item.Name,
				Message: err.Error(),
			})
			continue
		}
		// 收集 Antigravity OAuth 账号，稍后异步设置隐私
		if created.Platform == service.PlatformAntigravity && created.Type == service.AccountTypeOAuth {
			privacyAccounts = append(privacyAccounts, created)
		}
		result.AccountCreated++
	}

	// 异步设置 Antigravity 隐私，避免大量导入时阻塞请求
	if len(privacyAccounts) > 0 {
		adminSvc := h.adminService
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("import_antigravity_privacy_panic", "recover", r)
				}
			}()
			bgCtx := context.Background()
			for _, acc := range privacyAccounts {
				adminSvc.ForceAntigravityPrivacy(bgCtx, acc)
			}
			slog.Info("import_antigravity_privacy_done", "count", len(privacyAccounts))
		}()
	}

	return result, nil
}

func (h *AccountHandler) listAllProxies(ctx context.Context) ([]service.Proxy, error) {
	page := 1
	pageSize := dataPageCap
	var out []service.Proxy
	for {
		items, total, err := h.adminService.ListProxies(ctx, page, pageSize, "", "", "", "created_at", "desc")
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
		if len(out) >= int(total) || len(items) == 0 {
			break
		}
		page++
	}
	return out, nil
}

func (h *AccountHandler) listAccountsFiltered(ctx context.Context, platform, accountType, status, search string, groupID int64, privacyMode, networkStatus, exitIP, capacityStatus, sortBy, sortOrder string) ([]service.Account, error) {
	page := 1
	pageSize := dataPageCap
	var out []service.Account
	for {
		items, total, err := h.adminService.ListAccounts(ctx, page, pageSize, platform, accountType, status, search, groupID, privacyMode, networkStatus, exitIP, capacityStatus, sortBy, sortOrder)
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
		if len(out) >= int(total) || len(items) == 0 {
			break
		}
		page++
	}
	return out, nil
}

func (h *AccountHandler) resolveExportAccounts(ctx context.Context, ids []int64, c *gin.Context) ([]service.Account, error) {
	if len(ids) > 0 {
		accounts, err := h.adminService.GetAccountsByIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
		out := make([]service.Account, 0, len(accounts))
		for _, acc := range accounts {
			if acc == nil {
				continue
			}
			out = append(out, *acc)
		}
		return out, nil
	}

	platform := c.Query("platform")
	accountType := c.Query("type")
	status := c.Query("status")
	privacyMode := strings.TrimSpace(c.Query("privacy_mode"))
	networkStatus := strings.TrimSpace(c.Query("network_status"))
	exitIP := strings.TrimSpace(c.Query("ip"))
	capacityStatus := strings.TrimSpace(c.Query("capacity_status"))
	search := strings.TrimSpace(c.Query("search"))
	sortBy := c.DefaultQuery("sort_by", "name")
	sortOrder := c.DefaultQuery("sort_order", "asc")
	if len(search) > 100 {
		search = search[:100]
	}
	if capacityStatus != "" && capacityStatus != accountListCapacityConcurrentQueryValue {
		return nil, infraerrors.BadRequest("INVALID_CAPACITY_STATUS_FILTER", "invalid capacity status filter")
	}

	groupID := int64(0)
	if groupIDStr := c.Query("group"); groupIDStr != "" {
		if groupIDStr == accountListGroupUngroupedQueryValue {
			groupID = service.AccountListGroupUngrouped
		} else {
			parsedGroupID, parseErr := strconv.ParseInt(groupIDStr, 10, 64)
			if parseErr != nil || parsedGroupID <= 0 {
				return nil, infraerrors.BadRequest("INVALID_GROUP_FILTER", "invalid group filter")
			}
			groupID = parsedGroupID
		}
	}

	return h.listAccountsFiltered(ctx, platform, accountType, status, search, groupID, privacyMode, networkStatus, exitIP, capacityStatus, sortBy, sortOrder)
}

func (h *AccountHandler) resolveExportProxies(ctx context.Context, accounts []service.Account) ([]service.Proxy, error) {
	if len(accounts) == 0 {
		return []service.Proxy{}, nil
	}

	seen := make(map[int64]struct{})
	ids := make([]int64, 0)
	for i := range accounts {
		if accounts[i].ProxyID == nil {
			continue
		}
		id := *accounts[i].ProxyID
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return []service.Proxy{}, nil
	}

	return h.adminService.GetProxiesByIDs(ctx, ids)
}

func parseAccountIDs(c *gin.Context) ([]int64, error) {
	values := c.QueryArray("ids")
	if len(values) == 0 {
		raw := strings.TrimSpace(c.Query("ids"))
		if raw != "" {
			values = []string{raw}
		}
	}
	if len(values) == 0 {
		return nil, nil
	}

	ids := make([]int64, 0, len(values))
	for _, item := range values {
		for _, part := range strings.Split(item, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.ParseInt(part, 10, 64)
			if err != nil || id <= 0 {
				return nil, fmt.Errorf("invalid account id: %s", part)
			}
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func parseIncludeProxies(c *gin.Context) (bool, error) {
	raw := strings.TrimSpace(strings.ToLower(c.Query("include_proxies")))
	if raw == "" {
		return true, nil
	}
	switch raw {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return true, fmt.Errorf("invalid include_proxies value: %s", raw)
	}
}

func validateDataHeader(payload DataPayload) error {
	if payload.Type != "" && payload.Type != dataType && payload.Type != legacyDataType {
		return fmt.Errorf("unsupported data type: %s", payload.Type)
	}
	if payload.Version != 0 && payload.Version != dataVersion {
		return fmt.Errorf("unsupported data version: %d", payload.Version)
	}
	if payload.Proxies == nil {
		return errors.New("proxies is required")
	}
	if payload.Accounts == nil {
		return errors.New("accounts is required")
	}
	return nil
}

func validateDataProxy(item DataProxy) error {
	if strings.TrimSpace(item.Protocol) == "" {
		return errors.New("proxy protocol is required")
	}
	if strings.TrimSpace(item.Host) == "" {
		return errors.New("proxy host is required")
	}
	if item.Port <= 0 || item.Port > 65535 {
		return errors.New("proxy port is invalid")
	}
	switch item.Protocol {
	case "http", "https", "socks5", "socks5h":
	default:
		return fmt.Errorf("proxy protocol is invalid: %s", item.Protocol)
	}
	if item.Status != "" {
		normalizedStatus := normalizeProxyStatus(item.Status)
		if normalizedStatus != service.StatusActive && normalizedStatus != "inactive" {
			return fmt.Errorf("proxy status is invalid: %s", item.Status)
		}
	}
	return nil
}

func validateDataAccount(item DataAccount) error {
	if strings.TrimSpace(item.Name) == "" {
		return errors.New("account name is required")
	}
	if strings.TrimSpace(item.Platform) == "" {
		return errors.New("account platform is required")
	}
	if strings.TrimSpace(item.Type) == "" {
		return errors.New("account type is required")
	}
	if len(item.Credentials) == 0 {
		return errors.New("account credentials is required")
	}
	switch item.Type {
	case service.AccountTypeOAuth, service.AccountTypeSetupToken, service.AccountTypeAPIKey, service.AccountTypeUpstream:
	default:
		return fmt.Errorf("account type is invalid: %s", item.Type)
	}
	if item.RateMultiplier != nil && *item.RateMultiplier < 0 {
		return errors.New("rate_multiplier must be >= 0")
	}
	if item.Concurrency != nil && *item.Concurrency < 0 {
		return errors.New("concurrency must be >= 0")
	}
	if item.LoadFactor != nil {
		if *item.LoadFactor < 0 {
			return errors.New("load_factor must be >= 0")
		}
		if *item.LoadFactor > 10000 {
			return errors.New("load_factor must be <= 10000")
		}
	}
	if item.Priority < 0 {
		return errors.New("priority must be >= 0")
	}
	return nil
}

func validateDataImportRequest(req DataImportRequest) error {
	if req.DefaultConcurrency != nil && *req.DefaultConcurrency < 0 {
		return errors.New("default_concurrency must be >= 0")
	}
	if req.DefaultLoadFactor != nil {
		if *req.DefaultLoadFactor < 0 {
			return errors.New("default_load_factor must be >= 0")
		}
		if *req.DefaultLoadFactor > 10000 {
			return errors.New("default_load_factor must be <= 10000")
		}
	}
	return nil
}

func defaultProxyName(name string) string {
	if strings.TrimSpace(name) == "" {
		return "imported-proxy"
	}
	return name
}

func buildImportedProxyUpdate(item DataProxy, normalizedStatus string) *service.UpdateProxyInput {
	return &service.UpdateProxyInput{
		Name:        defaultProxyName(item.Name),
		Protocol:    item.Protocol,
		Host:        item.Host,
		Port:        item.Port,
		Username:    item.Username,
		Password:    item.Password,
		Status:      normalizedStatus,
		ExternalKey: nonEmptyStringPtr(item.ProxyExternalKey),
		ExitIP:      nonEmptyStringPtr(item.ExitIP),
	}
}

func shouldUpdateImportedProxy(existing *service.Proxy, item DataProxy, normalizedStatus string) bool {
	if existing == nil {
		return false
	}
	if defaultProxyName(item.Name) != existing.Name ||
		item.Protocol != existing.Protocol ||
		item.Host != existing.Host ||
		item.Port != existing.Port ||
		item.Username != existing.Username ||
		item.Password != existing.Password ||
		(strings.TrimSpace(item.ProxyExternalKey) != existing.ExternalKey) ||
		(strings.TrimSpace(item.ExitIP) != existing.ExitIP) {
		return true
	}
	return normalizedStatus != "" && normalizedStatus != existing.Status
}

func resolveExistingProxyID(
	proxyKey string,
	proxyExternalKey string,
	proxyName string,
	proxyKeyToID map[string]int64,
	proxyExternalKeyToID map[string]int64,
	proxyNameToIDs map[string][]int64,
) (int64, bool) {
	if proxyKey != "" {
		if id, ok := proxyKeyToID[proxyKey]; ok {
			return id, true
		}
	}
	if proxyExternalKey != "" {
		if id, ok := proxyExternalKeyToID[proxyExternalKey]; ok {
			return id, true
		}
	}
	if proxyName != "" {
		if ids := proxyNameToIDs[proxyName]; len(ids) == 1 {
			return ids[0], true
		}
	}
	return 0, false
}

func resolveAccountProxyID(
	item DataAccount,
	proxyKeyToID map[string]int64,
	proxyExternalKeyToID map[string]int64,
	proxyNameToIDs map[string][]int64,
) (*int64, error) {
	if item.ProxyKey != nil && strings.TrimSpace(*item.ProxyKey) != "" {
		if id, ok := proxyKeyToID[strings.TrimSpace(*item.ProxyKey)]; ok {
			return int64Ptr(id), nil
		}
		return nil, fmt.Errorf("proxy_key not found")
	}
	if item.ProxyExternalKey != nil && strings.TrimSpace(*item.ProxyExternalKey) != "" {
		if id, ok := proxyExternalKeyToID[strings.TrimSpace(*item.ProxyExternalKey)]; ok {
			return int64Ptr(id), nil
		}
		return nil, fmt.Errorf("proxy_external_key not found")
	}
	if item.ProxyName != nil && strings.TrimSpace(*item.ProxyName) != "" {
		ids := proxyNameToIDs[strings.TrimSpace(*item.ProxyName)]
		switch len(ids) {
		case 0:
			return nil, fmt.Errorf("proxy_name not found")
		case 1:
			return int64Ptr(ids[0]), nil
		default:
			return nil, fmt.Errorf("proxy_name is ambiguous")
		}
	}
	return nil, nil
}

func stringPtr(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	s := strings.TrimSpace(v)
	return &s
}

func int64Ptr(v int64) *int64 {
	return &v
}

func intValuePtr(v int) *int {
	return &v
}

func derefOptionalString(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}

func nonEmptyStringPtr(v string) *string {
	s := strings.TrimSpace(v)
	if s == "" {
		return nil
	}
	return &s
}

func unixPtrToTime(v *int64) *time.Time {
	if v == nil {
		return nil
	}
	ts := time.Unix(*v, 0).UTC()
	return &ts
}

// enrichCredentialsFromIDToken performs best-effort extraction of user info fields
// (email, plan_type, chatgpt_account_id, etc.) from id_token in credentials.
// Only applies to OpenAI OAuth accounts. Skips expired token errors silently.
// Existing credential values are never overwritten — only missing fields are filled.
func enrichCredentialsFromIDToken(item *DataAccount) {
	if item.Credentials == nil {
		return
	}
	// Only enrich OpenAI OAuth accounts
	platform := strings.ToLower(strings.TrimSpace(item.Platform))
	if platform != service.PlatformOpenAI {
		return
	}
	if strings.ToLower(strings.TrimSpace(item.Type)) != service.AccountTypeOAuth {
		return
	}

	idToken, _ := item.Credentials["id_token"].(string)
	if strings.TrimSpace(idToken) == "" {
		return
	}

	// DecodeIDToken skips expiry validation — safe for imported data
	claims, err := openai.DecodeIDToken(idToken)
	if err != nil {
		slog.Debug("import_enrich_id_token_decode_failed", "account", item.Name, "error", err)
		return
	}

	userInfo := claims.GetUserInfo()
	if userInfo == nil {
		return
	}

	// Fill missing fields only (never overwrite existing values)
	setIfMissing := func(key, value string) {
		if value == "" {
			return
		}
		if existing, _ := item.Credentials[key].(string); existing == "" {
			item.Credentials[key] = value
		}
	}

	setIfMissing("email", userInfo.Email)
	setIfMissing("plan_type", userInfo.PlanType)
	setIfMissing("chatgpt_account_id", userInfo.ChatGPTAccountID)
	setIfMissing("chatgpt_user_id", userInfo.ChatGPTUserID)
	setIfMissing("organization_id", userInfo.OrganizationID)
}

func normalizeProxyStatus(status string) string {
	normalized := strings.TrimSpace(strings.ToLower(status))
	switch normalized {
	case "":
		return ""
	case service.StatusActive:
		return service.StatusActive
	case "inactive", service.StatusDisabled:
		return "inactive"
	default:
		return normalized
	}
}
