package accountdedup

import (
	"context"
	"encoding/json"
	"net/url"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func BuildPreview(accounts []Account) PreviewReport {
	report := PreviewReport{
		Mode:            "preview",
		DuplicateGroups: []DuplicateGroup{},
		SkippedAccounts: []SkippedAccount{},
		Deleted:         []DeleteResult{},
		Failed:          []DeleteResult{},
		Summary: Summary{
			TotalAccounts: len(accounts),
		},
	}

	type groupedAccount struct {
		account    Account
		matchBasis string
		matchValue string
	}

	grouped := make(map[string][]groupedAccount)
	for _, account := range accounts {
		matchBasis, matchValue, skipReason := classifyAccount(account)
		if skipReason != "" {
			report.SkippedAccounts = append(report.SkippedAccounts, SkippedAccount{
				Account: account,
				Reason:  skipReason,
			})
			continue
		}
		groupKey := buildGroupKey(account.Platform, account.Type, matchBasis, matchValue)
		grouped[groupKey] = append(grouped[groupKey], groupedAccount{
			account:    account,
			matchBasis: matchBasis,
			matchValue: matchValue,
		})
	}

	for _, bucket := range grouped {
		if len(bucket) < 2 {
			continue
		}
		sort.SliceStable(bucket, func(i, j int) bool {
			return compareAccountsForKeep(bucket[i].account, bucket[j].account) < 0
		})
		keep := bucket[0].account
		deleteAccounts := make([]Account, 0, len(bucket)-1)
		for i := 1; i < len(bucket); i++ {
			deleteAccounts = append(deleteAccounts, bucket[i].account)
		}
		sort.SliceStable(deleteAccounts, func(i, j int) bool {
			return compareAccountsForDelete(deleteAccounts[i], deleteAccounts[j]) < 0
		})

		report.DuplicateGroups = append(report.DuplicateGroups, DuplicateGroup{
			Platform:   keep.Platform,
			Type:       keep.Type,
			MatchBasis: bucket[0].matchBasis,
			MatchValue: bucket[0].matchValue,
			Keep:       keep,
			Delete:     deleteAccounts,
		})
	}

	sort.SliceStable(report.DuplicateGroups, func(i, j int) bool {
		left := report.DuplicateGroups[i]
		right := report.DuplicateGroups[j]
		if left.Platform != right.Platform {
			return left.Platform < right.Platform
		}
		if left.Type != right.Type {
			return left.Type < right.Type
		}
		if left.MatchBasis != right.MatchBasis {
			return left.MatchBasis < right.MatchBasis
		}
		return left.MatchValue < right.MatchValue
	})
	sort.SliceStable(report.SkippedAccounts, func(i, j int) bool {
		return report.SkippedAccounts[i].Account.ID < report.SkippedAccounts[j].Account.ID
	})

	for _, group := range report.DuplicateGroups {
		report.Summary.AccountsToDelete += len(group.Delete)
	}
	report.Summary.DuplicateGroups = len(report.DuplicateGroups)
	report.Summary.SkippedAccounts = len(report.SkippedAccounts)

	return report
}

func ApplyDeletionPlan(ctx context.Context, client *Client, report PreviewReport) ApplyResult {
	result := ApplyResult{
		Deleted: []DeleteResult{},
		Failed:  []DeleteResult{},
	}
	if client == nil {
		return result
	}

	for _, group := range report.DuplicateGroups {
		for _, account := range group.Delete {
			if err := client.DeleteAccount(ctx, account.ID); err != nil {
				result.Failed = append(result.Failed, DeleteResult{
					Account: account,
					Error:   err.Error(),
				})
				continue
			}
			result.Deleted = append(result.Deleted, DeleteResult{
				Account: account,
			})
		}
	}

	return result
}

func classifyAccount(account Account) (matchBasis, matchValue, skipReason string) {
	switch account.Type {
	case service.AccountTypeAPIKey:
		apiKey := normalizeString(account.Credentials["api_key"])
		if apiKey == "" {
			return "", "", "apikey 账号缺少 api_key"
		}
		baseURL := normalizeBaseURL(normalizeString(account.Credentials["base_url"]))
		return "api_key+base_url", apiKey + "\n" + baseURL, ""
	case service.AccountTypeOAuth, service.AccountTypeSetupToken:
		for _, key := range []string{
			"refresh_token",
			"chatgpt_account_id",
			"chatgpt_user_id",
			"anthropic_user_id",
			"claude_user_id",
			"email",
		} {
			value := normalizeCredentialKey(key, account.Credentials[key])
			if value != "" {
				return key, value, ""
			}
		}

		projectID := normalizeString(account.Credentials["project_id"])
		clientID := normalizeString(account.Credentials["client_id"])
		baseURL := normalizeBaseURL(normalizeString(account.Credentials["base_url"]))
		if projectID != "" && clientID != "" && baseURL != "" {
			return "project_id+client_id+base_url", projectID + "\n" + clientID + "\n" + baseURL, ""
		}
		return "", "", "oauth/setup-token 账号缺少可用于自动判重的身份字段"
	case service.AccountTypeUpstream:
		if len(account.Credentials) == 0 {
			return "", "", "upstream 账号缺少 credentials"
		}
		return "full_credentials", canonicalizeCredentials(account.Credentials), ""
	default:
		return "", "", "未支持的账号类型，跳过自动删除"
	}
}

func compareAccountsForKeep(left, right Account) int {
	if left.CreatedAt.After(right.CreatedAt) {
		return -1
	}
	if left.CreatedAt.Before(right.CreatedAt) {
		return 1
	}
	if left.ID > right.ID {
		return -1
	}
	if left.ID < right.ID {
		return 1
	}
	return 0
}

func compareAccountsForDelete(left, right Account) int {
	if left.CreatedAt.Before(right.CreatedAt) {
		return -1
	}
	if left.CreatedAt.After(right.CreatedAt) {
		return 1
	}
	if left.ID < right.ID {
		return -1
	}
	if left.ID > right.ID {
		return 1
	}
	return 0
}

func buildGroupKey(platform, accountType, matchBasis, matchValue string) string {
	return platform + "\x00" + accountType + "\x00" + matchBasis + "\x00" + matchValue
}

func normalizeCredentialKey(key string, value any) string {
	normalized := normalizeString(value)
	if normalized == "" {
		return ""
	}
	if key == "email" {
		return strings.ToLower(normalized)
	}
	return normalized
}

func normalizeString(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func normalizeBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return strings.TrimRight(raw, "/")
	}

	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	if parsed.Path == "/" {
		parsed.Path = ""
	} else {
		parsed.Path = strings.TrimRight(parsed.Path, "/")
	}
	return parsed.String()
}

func canonicalizeCredentials(credentials map[string]any) string {
	normalized := normalizeJSONValue(credentials)
	raw, err := json.Marshal(normalized)
	if err != nil {
		return ""
	}
	return string(raw)
}

func normalizeJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			result[key] = normalizeJSONValue(item)
		}
		return result
	case []any:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			result = append(result, normalizeJSONValue(item))
		}
		return result
	case string:
		if strings.Contains(strings.ToLower(typed), "://") {
			return normalizeBaseURL(typed)
		}
		return strings.TrimSpace(typed)
	default:
		return typed
	}
}
