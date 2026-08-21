package admin

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// OpenAIOAuthHandler handles OpenAI OAuth-related operations
type OpenAIOAuthHandler struct {
	openaiOAuthService *service.OpenAIOAuthService
	adminService       service.AdminService
	quotaService       openAIQuotaService
	rateLimitService   openAIAccountStateRecoverer
}

type openAIQuotaService interface {
	QueryUsage(ctx context.Context, accountID int64) (*service.OpenAIQuotaUsage, error)
	CacheResetCreditsSnapshot(ctx context.Context, accountID int64, credits *service.OpenAIRateLimitResetCredits) error
	ResetCredit(ctx context.Context, accountID int64) (*service.OpenAIQuotaResetResult, error)
}

type openAIAccountStateRecoverer interface {
	RecoverAccountState(ctx context.Context, accountID int64, options service.AccountRecoveryOptions) (*service.SuccessfulTestRecoveryResult, error)
}

const (
	openAIQuotaResetWarningCacheRefreshFailed    = "reset_credit_cache_refresh_failed"
	openAIQuotaResetWarningAccountRecoveryFailed = "account_state_recovery_failed"
	openAIQuotaResetWarningAccountRefreshFailed  = "account_state_refresh_failed"
)

// openAIQuotaResetPostProcessTimeout bounds the work performed AFTER the
// (non-refundable) reset credit has already been consumed upstream. The whole
// request must stay comfortably inside the panel HTTP client timeout, otherwise
// the browser aborts a mutation that already succeeded and the operator retries
// it — spending a second credit.
const openAIQuotaResetPostProcessTimeout = 8 * time.Second

type openAIQuotaResetResponse struct {
	service.OpenAIQuotaResetResult
	Quota                 *service.OpenAIQuotaUsage `json:"quota,omitempty"`
	Account               *dto.Account              `json:"account,omitempty"`
	CacheRefreshed        bool                      `json:"cache_refreshed"`
	AccountStateRecovered bool                      `json:"account_state_recovered"`
	WarningCode           string                    `json:"warning_code,omitempty"`
}

// openAIQuotaRefreshResponse is the reset-credit-persisting variant of the quota
// query. The usage payload is embedded so the shape stays identical to the plain
// query; cache_persisted reports whether the snapshot write succeeded, because a
// failed display-cache write must never discard a successful upstream read.
type openAIQuotaRefreshResponse struct {
	service.OpenAIQuotaUsage
	CachePersisted bool `json:"cache_persisted"`
}

// openAIQuotaResetPostProcessContext detaches the post-reset bookkeeping from the
// client connection. The credit is already spent at that point, so account-state
// recovery must complete even if the operator closes the tab (mirrors
// systemUpdateContext, added for the same reason in #4504).
func openAIQuotaResetPostProcessContext(ctx context.Context) (context.Context, context.CancelFunc) {
	base := context.Background()
	if ctx != nil {
		base = context.WithoutCancel(ctx)
	}
	return context.WithTimeout(base, openAIQuotaResetPostProcessTimeout)
}

func oauthPlatformFromPath(c *gin.Context) string {
	return service.PlatformOpenAI
}

// NewOpenAIOAuthHandler creates a new OpenAI OAuth handler
func NewOpenAIOAuthHandler(
	openaiOAuthService *service.OpenAIOAuthService,
	adminService service.AdminService,
	quotaService *service.OpenAIQuotaService,
	rateLimitService *service.RateLimitService,
) *OpenAIOAuthHandler {
	h := &OpenAIOAuthHandler{
		openaiOAuthService: openaiOAuthService,
		adminService:       adminService,
	}
	// Assign through explicit nil checks: storing a nil *Service in an interface
	// field yields a non-nil interface, which would silently defeat the
	// `== nil` capability guards below and panic instead of returning 400.
	if quotaService != nil {
		h.quotaService = quotaService
	}
	if rateLimitService != nil {
		h.rateLimitService = rateLimitService
	}
	return h
}

// OpenAIGenerateAuthURLRequest represents the request for generating OpenAI auth URL
type OpenAIGenerateAuthURLRequest struct {
	ProxyID       *int64                            `json:"proxy_id"`
	RedirectURI   string                            `json:"redirect_uri"`
	PendingCreate *openai.OAuthPendingCreateAccount `json:"pending_create"`
}

// GenerateAuthURL generates OpenAI OAuth authorization URL
// POST /api/v1/admin/openai/generate-auth-url
func (h *OpenAIOAuthHandler) GenerateAuthURL(c *gin.Context) {
	var req OpenAIGenerateAuthURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body
		req = OpenAIGenerateAuthURLRequest{}
	}

	result, err := h.openaiOAuthService.GenerateAuthURL(
		c.Request.Context(),
		req.ProxyID,
		req.RedirectURI,
		oauthPlatformFromPath(c),
		req.PendingCreate,
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

// OpenAIExchangeCodeRequest represents the request for exchanging OpenAI auth code
type OpenAIExchangeCodeRequest struct {
	SessionID   string `json:"session_id" binding:"required"`
	Code        string `json:"code" binding:"required"`
	State       string `json:"state" binding:"required"`
	RedirectURI string `json:"redirect_uri"`
	ProxyID     *int64 `json:"proxy_id"`
}

// ExchangeCode exchanges OpenAI authorization code for tokens
// POST /api/v1/admin/openai/exchange-code
func (h *OpenAIOAuthHandler) ExchangeCode(c *gin.Context) {
	var req OpenAIExchangeCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	tokenInfo, err := h.openaiOAuthService.ExchangeCode(c.Request.Context(), &service.OpenAIExchangeCodeInput{
		SessionID:   req.SessionID,
		Code:        req.Code,
		State:       req.State,
		RedirectURI: req.RedirectURI,
		ProxyID:     req.ProxyID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, tokenInfo)
}

// OpenAICompletePendingCreateRequest represents an OpenAI OAuth callback that
// should complete a pending account creation captured before authorization.
type OpenAICompletePendingCreateRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
}

// CompletePendingCreate exchanges the callback code and creates the pending OpenAI account.
// POST /api/v1/admin/openai/complete-pending-create
func (h *OpenAIOAuthHandler) CompletePendingCreate(c *gin.Context) {
	var req OpenAICompletePendingCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	result, err := h.openaiOAuthService.ExchangePendingCreateByState(c.Request.Context(), req.Code, req.State)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	account, err := h.createOpenAIAccountFromPendingCreate(c, result.TokenInfo, result.PendingCreate)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.AccountFromService(account))
}

// OpenAIRefreshTokenRequest represents the request for refreshing OpenAI token
type OpenAIRefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
	RT           string `json:"rt"`
	ClientID     string `json:"client_id"`
	ProxyID      *int64 `json:"proxy_id"`
}

type OpenAICodexPATCreateRequest struct {
	AccessToken             string         `json:"access_token" binding:"required"`
	Name                    string         `json:"name"`
	Notes                   *string        `json:"notes"`
	GroupIDs                []int64        `json:"group_ids"`
	ProxyID                 *int64         `json:"proxy_id"`
	Concurrency             *int           `json:"concurrency"`
	Priority                *int           `json:"priority"`
	RateMultiplier          *float64       `json:"rate_multiplier"`
	LoadFactor              *int           `json:"load_factor"`
	ExpiresAt               *int64         `json:"expires_at"`
	AutoPauseOnExpired      *bool          `json:"auto_pause_on_expired"`
	CredentialExtras        map[string]any `json:"credential_extras"`
	Extra                   map[string]any `json:"extra"`
	SkipDefaultGroupBind    *bool          `json:"skip_default_group_bind"`
	ConfirmMixedChannelRisk *bool          `json:"confirm_mixed_channel_risk"`
}

// RefreshToken refreshes an OpenAI OAuth token
// POST /api/v1/admin/openai/refresh-token
func (h *OpenAIOAuthHandler) RefreshToken(c *gin.Context) {
	var req OpenAIRefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	refreshToken := strings.TrimSpace(req.RefreshToken)
	if refreshToken == "" {
		refreshToken = strings.TrimSpace(req.RT)
	}
	if refreshToken == "" {
		response.BadRequest(c, "refresh_token is required")
		return
	}

	var proxyURL string
	if req.ProxyID != nil {
		proxy, err := h.adminService.GetProxy(c.Request.Context(), *req.ProxyID)
		if err == nil && proxy != nil {
			proxyURL = proxy.URL()
		}
	}

	// 未指定 client_id 时，根据请求路径平台自动设置默认值，避免 repository 层盲猜
	clientID := strings.TrimSpace(req.ClientID)
	if clientID == "" {
		platform := oauthPlatformFromPath(c)
		clientID, _ = openai.OAuthClientConfigByPlatform(platform)
	}

	tokenInfo, err := h.openaiOAuthService.RefreshTokenWithClientID(c.Request.Context(), refreshToken, proxyURL, clientID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, tokenInfo)
}

// RefreshAccountToken refreshes token for a specific OpenAI account
// POST /api/v1/admin/openai/accounts/:id/refresh
func (h *OpenAIOAuthHandler) RefreshAccountToken(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	updatedAccount, err := h.refreshOpenAIAccountSubscriptionMetadata(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.AccountFromService(updatedAccount))
}

// RefreshAccountSubscription refreshes OpenAI OAuth subscription metadata for one account.
// POST /api/v1/admin/openai/accounts/:id/refresh-subscription
func (h *OpenAIOAuthHandler) RefreshAccountSubscription(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	updatedAccount, err := h.refreshOpenAIAccountSubscriptionMetadata(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.AccountFromService(updatedAccount))
}

type OpenAIRefreshSubscriptionsRequest struct {
	Limit  int  `json:"limit"`
	DryRun bool `json:"dry_run"`
}

type OpenAIRefreshSubscriptionResult struct {
	AccountID int64  `json:"account_id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Reason    string `json:"reason,omitempty"`
}

// RefreshAccountSubscriptions backfills subscription metadata for existing OpenAI OAuth accounts.
// POST /api/v1/admin/openai/accounts/refresh-subscriptions
func (h *OpenAIOAuthHandler) RefreshAccountSubscriptions(c *gin.Context) {
	var req OpenAIRefreshSubscriptionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = OpenAIRefreshSubscriptionsRequest{}
	}
	limit := req.Limit
	if limit <= 0 || limit > 20 {
		limit = 20
	}

	results := make([]OpenAIRefreshSubscriptionResult, 0, limit)
	const pageSize = 100
	for page := 1; len(results) < limit; page++ {
		accounts, _, err := h.adminService.ListAccounts(c.Request.Context(), page, pageSize, service.PlatformOpenAI, service.AccountTypeOAuth, "", "", 0, "", "", "", "", "", "id", "ASC")
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		if len(accounts) == 0 {
			break
		}
		for i := range accounts {
			account := accounts[i]
			if !openAIAccountNeedsSubscriptionBackfill(&account) {
				continue
			}
			if len(results) >= limit {
				break
			}
			result := OpenAIRefreshSubscriptionResult{AccountID: account.ID, Name: account.Name}
			if req.DryRun {
				result.Status = "skipped"
				result.Reason = "dry_run"
				results = append(results, result)
				continue
			}
			if _, err := h.refreshOpenAIAccountSubscriptionMetadata(c.Request.Context(), account.ID); err != nil {
				result.Status = "failed"
				result.Reason = "refresh_failed"
			} else {
				result.Status = "updated"
			}
			results = append(results, result)
		}
		if len(accounts) < pageSize {
			break
		}
	}

	response.Success(c, gin.H{
		"limit":   limit,
		"dry_run": req.DryRun,
		"results": results,
	})
}

func (h *OpenAIOAuthHandler) refreshOpenAIAccountSubscriptionMetadata(ctx context.Context, accountID int64) (*service.Account, error) {
	account, err := h.adminService.GetAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account.Platform != service.PlatformOpenAI {
		return nil, serviceErrorBadRequest("OPENAI_OAUTH_INVALID_ACCOUNT", "Account platform does not match OAuth endpoint")
	}
	if account.Type != service.AccountTypeOAuth {
		return nil, serviceErrorBadRequest("OPENAI_OAUTH_INVALID_ACCOUNT_TYPE", "Cannot refresh non-OAuth account credentials")
	}
	if account.IsCredentialShadow() {
		return nil, serviceErrorBadRequest("OPENAI_OAUTH_SHADOW_ACCOUNT", "Cannot refresh spark shadow account; its credentials are managed by the parent account")
	}

	tokenInfo, err := h.openaiOAuthService.RefreshAccountToken(ctx, account)
	if err != nil {
		return nil, err
	}
	newCredentials := h.openaiOAuthService.BuildAccountCredentials(tokenInfo)
	for k, v := range account.Credentials {
		if _, exists := newCredentials[k]; !exists {
			newCredentials[k] = v
		}
	}
	newCredentials = service.NormalizeOpenAIPersonalAccessTokenCredentials(account, tokenInfo, newCredentials)
	return h.adminService.UpdateAccount(ctx, accountID, &service.UpdateAccountInput{Credentials: newCredentials})
}

func openAIAccountNeedsSubscriptionBackfill(account *service.Account) bool {
	if account == nil || account.Platform != service.PlatformOpenAI || account.Type != service.AccountTypeOAuth {
		return false
	}
	if strings.TrimSpace(account.GetCredential("subscription_expires_at")) != "" {
		return false
	}
	if strings.TrimSpace(account.GetCredential("access_token")) == "" && strings.TrimSpace(account.GetCredential("refresh_token")) == "" {
		return false
	}
	planType := strings.ToLower(strings.TrimSpace(account.GetCredential("plan_type")))
	return planType == "" || planType != "free"
}

func serviceErrorBadRequest(code, message string) error {
	return infraerrors.New(http.StatusBadRequest, code, message)
}

// CreateAccountFromOAuth creates a new OpenAI OAuth account from token info
// POST /api/v1/admin/openai/create-from-oauth
func (h *OpenAIOAuthHandler) CreateAccountFromOAuth(c *gin.Context) {
	var req struct {
		SessionID   string  `json:"session_id" binding:"required"`
		Code        string  `json:"code" binding:"required"`
		State       string  `json:"state" binding:"required"`
		RedirectURI string  `json:"redirect_uri"`
		ProxyID     *int64  `json:"proxy_id"`
		Name        string  `json:"name"`
		Concurrency int     `json:"concurrency"`
		Priority    int     `json:"priority"`
		GroupIDs    []int64 `json:"group_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Exchange code for tokens
	tokenInfo, err := h.openaiOAuthService.ExchangeCode(c.Request.Context(), &service.OpenAIExchangeCodeInput{
		SessionID:   req.SessionID,
		Code:        req.Code,
		State:       req.State,
		RedirectURI: req.RedirectURI,
		ProxyID:     req.ProxyID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// Build credentials from token info
	credentials := h.openaiOAuthService.BuildAccountCredentials(tokenInfo)

	platform := oauthPlatformFromPath(c)

	// Use email as default name if not provided
	name := req.Name
	if name == "" && tokenInfo.Email != "" {
		name = tokenInfo.Email
	}
	if name == "" {
		name = "OpenAI OAuth Account"
	}

	// Create account
	account, err := h.adminService.CreateAccount(c.Request.Context(), &service.CreateAccountInput{
		Name:        name,
		Platform:    platform,
		Type:        "oauth",
		Credentials: credentials,
		Extra:       nil,
		ProxyID:     req.ProxyID,
		Concurrency: req.Concurrency,
		Priority:    req.Priority,
		GroupIDs:    req.GroupIDs,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.AccountFromService(account))
}

func (h *OpenAIOAuthHandler) createOpenAIAccountFromPendingCreate(c *gin.Context, tokenInfo *service.OpenAITokenInfo, pendingCreate *openai.OAuthPendingCreateAccount) (*service.Account, error) {
	credentials := h.openaiOAuthService.BuildAccountCredentials(tokenInfo)
	for key, value := range pendingCreate.CredentialOverrides {
		credentials[key] = value
	}

	name := strings.TrimSpace(pendingCreate.Name)
	if name == "" && tokenInfo.Email != "" {
		name = tokenInfo.Email
	}
	if name == "" {
		name = "OpenAI OAuth Account"
	}

	var notes *string
	if strings.TrimSpace(pendingCreate.Notes) != "" {
		n := pendingCreate.Notes
		notes = &n
	}

	concurrency := pendingCreate.Concurrency
	if concurrency <= 0 {
		concurrency = 10
	}
	priority := pendingCreate.Priority
	if priority <= 0 {
		priority = 50
	}

	account, err := h.adminService.CreateAccount(c.Request.Context(), &service.CreateAccountInput{
		Name:                  name,
		Notes:                 notes,
		Platform:              service.PlatformOpenAI,
		Type:                  service.AccountTypeOAuth,
		Credentials:           credentials,
		Extra:                 pendingCreate.Extra,
		ProxyID:               pendingCreate.ProxyID,
		Concurrency:           concurrency,
		Priority:              priority,
		RateMultiplier:        pendingCreate.RateMultiplier,
		LoadFactor:            pendingCreate.LoadFactor,
		GroupIDs:              pendingCreate.GroupIDs,
		ExpiresAt:             pendingCreate.ExpiresAt,
		AutoPauseOnExpired:    pendingCreate.AutoPauseOnExpired,
		SkipDefaultGroupBind:  len(pendingCreate.GroupIDs) == 0,
		SkipMixedChannelCheck: true,
	})
	if err != nil {
		return nil, err
	}

	h.adminService.ForceOpenAIPrivacy(c.Request.Context(), account)
	return account, nil
}

// CreateAccountFromCodexPAT creates an OpenAI OAuth account from a Codex at-* personal access token.
// POST /api/v1/admin/openai/create-from-codex-pat
func (h *OpenAIOAuthHandler) CreateAccountFromCodexPAT(c *gin.Context) {
	var req OpenAICodexPATCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := service.ValidateOpenAILongContextBillingExtra(service.PlatformOpenAI, req.Extra); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if req.Concurrency != nil && *req.Concurrency < 0 {
		response.BadRequest(c, "concurrency must be >= 0")
		return
	}
	if req.Priority != nil && *req.Priority < 0 {
		response.BadRequest(c, "priority must be >= 0")
		return
	}
	if req.RateMultiplier != nil && *req.RateMultiplier < 0 {
		response.BadRequest(c, "rate_multiplier must be >= 0")
		return
	}
	if req.LoadFactor != nil && *req.LoadFactor > 10000 {
		response.BadRequest(c, "load_factor must be <= 10000")
		return
	}

	var proxyURL string
	if req.ProxyID != nil {
		proxy, err := h.adminService.GetProxy(c.Request.Context(), *req.ProxyID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		if proxy != nil {
			proxyURL = proxy.URL()
		}
	}

	tokenInfo, err := h.openaiOAuthService.ValidateCodexPersonalAccessToken(c.Request.Context(), req.AccessToken, proxyURL)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	credentials := mergeCodexImportMap(
		h.openaiOAuthService.BuildAccountCredentials(tokenInfo),
		sanitizeCodexImportCredentialExtras(req.CredentialExtras),
	)
	extra := mergeCodexImportMap(req.Extra, map[string]any{
		"import_source":       "codex_personal_access_token",
		"auth_provider":       "codex_personal_access_token",
		"imported_at":         time.Now().UTC().Format(time.RFC3339),
		"access_token_sha256": codexTokenFingerprint(req.AccessToken),
	})

	concurrency := 3
	if req.Concurrency != nil {
		concurrency = *req.Concurrency
	}
	priority := 50
	if req.Priority != nil {
		priority = *req.Priority
	}
	skipDefaultGroupBind := false
	if req.SkipDefaultGroupBind != nil {
		skipDefaultGroupBind = *req.SkipDefaultGroupBind
	}

	account, err := h.adminService.CreateAccount(c.Request.Context(), &service.CreateAccountInput{
		Name:                  buildOpenAICodexPATAccountName(req.Name, tokenInfo),
		Notes:                 req.Notes,
		Platform:              service.PlatformOpenAI,
		Type:                  service.AccountTypeOAuth,
		Credentials:           credentials,
		Extra:                 extra,
		ProxyID:               req.ProxyID,
		Concurrency:           concurrency,
		Priority:              priority,
		RateMultiplier:        req.RateMultiplier,
		LoadFactor:            req.LoadFactor,
		GroupIDs:              req.GroupIDs,
		ExpiresAt:             req.ExpiresAt,
		AutoPauseOnExpired:    req.AutoPauseOnExpired,
		SkipDefaultGroupBind:  skipDefaultGroupBind,
		SkipMixedChannelCheck: req.ConfirmMixedChannelRisk != nil && *req.ConfirmMixedChannelRisk,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.AccountFromService(account))
}

func buildOpenAICodexPATAccountName(name string, tokenInfo *service.OpenAITokenInfo) string {
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	if tokenInfo != nil {
		for _, candidate := range []string{tokenInfo.Email, tokenInfo.ChatGPTAccountID, tokenInfo.ChatGPTUserID} {
			if candidate = strings.TrimSpace(candidate); candidate != "" {
				return candidate
			}
		}
	}
	return "Codex PAT Account"
}

// QueryQuota queries the rate-limit / quota usage for an OpenAI account.
// GET /api/v1/admin/openai/accounts/:id/quota
func (h *OpenAIOAuthHandler) QueryQuota(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	if h.quotaService == nil {
		response.BadRequest(c, "openai quota service is not enabled")
		return
	}

	usage, err := h.quotaService.QueryUsage(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, usage)
}

// RefreshQuota queries the rate-limit / quota usage AND persists the reset-credit
// snapshot so the card can be rehydrated without an upstream round-trip.
// POST /api/v1/admin/openai/accounts/:id/quota/refresh
//
// It is a POST (not a GET with a side-effect flag) because it writes account
// state: the audit middleware only records mutating verbs, so a persisting GET
// would mutate the database without an audit trail.
func (h *OpenAIOAuthHandler) RefreshQuota(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	if h.quotaService == nil {
		response.BadRequest(c, "openai quota service is not enabled")
		return
	}

	usage, err := h.quotaService.QueryUsage(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if usage == nil {
		response.Error(c, http.StatusInternalServerError, "openai quota query returned an empty result")
		return
	}

	refreshResponse := openAIQuotaRefreshResponse{OpenAIQuotaUsage: *usage}
	// A failed snapshot write leaves the previous cache intact — report it as a
	// partial success instead of discarding the usage payload we just fetched,
	// which would leave the card without a credit count at all.
	if err := h.quotaService.CacheResetCreditsSnapshot(c.Request.Context(), accountID, usage.RateLimitResetCredits); err != nil {
		slog.Warn("openai_quota_reset_credit_cache_persist_failed", "account_id", accountID, "error", err)
		response.Success(c, refreshResponse)
		return
	}
	refreshResponse.CachePersisted = true
	response.Success(c, refreshResponse)
}

// CreateShadowRequest is the request body for CreateShadow.
type CreateShadowRequest struct {
	Name        string  `json:"name"`
	Priority    int     `json:"priority"`
	Concurrency int     `json:"concurrency"`
	GroupIDs    []int64 `json:"group_ids"`
}

// CreateShadow creates a spark-dimension shadow account for a parent OpenAI OAuth account.
// POST /api/v1/admin/accounts/:id/shadow
func (h *OpenAIOAuthHandler) CreateShadow(c *gin.Context) {
	parentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	var req CreateShadowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	shadow, err := h.adminService.CreateShadow(c.Request.Context(), parentID, service.ShadowOptions{
		Name:        req.Name,
		Priority:    req.Priority,
		Concurrency: req.Concurrency,
		GroupIDs:    req.GroupIDs,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.AccountFromServiceShallow(shadow))
}

// ResetQuota consumes one rate-limit reset credit for an OpenAI account.
// POST /api/v1/admin/openai/accounts/:id/reset-quota
func (h *OpenAIOAuthHandler) ResetQuota(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	if h.quotaService == nil {
		response.BadRequest(c, "openai quota service is not enabled")
		return
	}
	result, err := h.quotaService.ResetCredit(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if result == nil {
		response.Error(c, http.StatusInternalServerError, "openai quota reset returned an empty result")
		return
	}

	resetResponse := openAIQuotaResetResponse{OpenAIQuotaResetResult: *result}
	postCtx, cancelPost := openAIQuotaResetPostProcessContext(c.Request.Context())
	defer cancelPost()

	// Step 1 — unblocking the account is the whole point of consuming a credit
	// (#3672 / #3740), so it runs FIRST and is never gated on the display cache.
	// Recovery is DB-only and leaves the manual `schedulable` switch untouched.
	if h.rateLimitService == nil {
		resetResponse.WarningCode = openAIQuotaResetWarningAccountRecoveryFailed
		response.Success(c, resetResponse)
		return
	}
	if _, err := h.rateLimitService.RecoverAccountState(postCtx, accountID, service.AccountRecoveryOptions{
		InvalidateToken: true,
	}); err != nil {
		// Recovery failures are almost always storage-level; the remaining steps
		// share that dependency, so stop here instead of compounding the failure.
		slog.Warn("openai_quota_reset_account_recovery_failed", "account_id", accountID, "error", err)
		resetResponse.WarningCode = openAIQuotaResetWarningAccountRecoveryFailed
		response.Success(c, resetResponse)
		return
	}
	resetResponse.AccountStateRecovered = true

	// Step 2 — refresh the reset-credit display cache. A failure here is reported
	// but must not hide the recovered account row produced by step 3.
	usage, usageErr := h.quotaService.QueryUsage(postCtx, accountID)
	switch {
	case usageErr != nil || usage == nil:
		slog.Warn("openai_quota_reset_cache_refresh_failed", "account_id", accountID, "error", usageErr)
		resetResponse.WarningCode = openAIQuotaResetWarningCacheRefreshFailed
	default:
		if err := h.quotaService.CacheResetCreditsSnapshot(postCtx, accountID, usage.RateLimitResetCredits); err != nil {
			slog.Warn("openai_quota_reset_cache_refresh_failed", "account_id", accountID, "error", err)
			resetResponse.WarningCode = openAIQuotaResetWarningCacheRefreshFailed
		} else {
			resetResponse.Quota = usage
			resetResponse.CacheRefreshed = true
		}
	}

	// Step 3 — hand back the post-recovery account row so the list drops the
	// stale rate-limit badge without waiting for the next poll.
	account, err := h.adminService.GetAccount(postCtx, accountID)
	if err != nil {
		slog.Warn("openai_quota_reset_account_refresh_failed", "account_id", accountID, "error", err)
		if resetResponse.WarningCode == "" {
			resetResponse.WarningCode = openAIQuotaResetWarningAccountRefreshFailed
		}
		response.Success(c, resetResponse)
		return
	}
	resetResponse.Account = dto.AccountFromService(account)
	response.Success(c, resetResponse)
}
