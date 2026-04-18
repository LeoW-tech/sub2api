package accountdedup

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunLogsInAndWritesPreviewReport(t *testing.T) {
	tempDir := t.TempDir()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/auth/login":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    0,
				"message": "success",
				"data": map[string]any{
					"access_token": "jwt-from-login",
					"token_type":   "Bearer",
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/admin/accounts":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    0,
				"message": "success",
				"data": map[string]any{
					"items": []map[string]any{
						{
							"id":         100,
							"name":       "dup-old",
							"platform":   "openai",
							"type":       "oauth",
							"created_at": time.Date(2026, 4, 19, 8, 0, 0, 0, time.UTC).Format(time.RFC3339),
							"credentials": map[string]any{
								"refresh_token": "rt-100",
							},
						},
						{
							"id":         101,
							"name":       "dup-new",
							"platform":   "openai",
							"type":       "oauth",
							"created_at": time.Date(2026, 4, 19, 9, 0, 0, 0, time.UTC).Format(time.RFC3339),
							"credentials": map[string]any{
								"refresh_token": "rt-100",
							},
						},
					},
					"total":     2,
					"page":      1,
					"page_size": 1000,
					"pages":     1,
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	result, err := Run(context.Background(), RunOptions{
		BaseURL:    server.URL,
		Email:      "admin@example.com",
		Password:   "secret",
		OutputDir:  tempDir,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if result.ReportPath == "" {
		t.Fatalf("expected report path to be set")
	}
	if filepath.Dir(result.ReportPath) != tempDir {
		t.Fatalf("expected report file in %q, got %q", tempDir, result.ReportPath)
	}
	if _, statErr := os.Stat(result.ReportPath); statErr != nil {
		t.Fatalf("expected report file to exist: %v", statErr)
	}
	if result.Report.Summary.TotalAccounts != 2 {
		t.Fatalf("expected 2 total accounts, got %d", result.Report.Summary.TotalAccounts)
	}
	if result.Report.Summary.DuplicateGroups != 1 {
		t.Fatalf("expected 1 duplicate group, got %d", result.Report.Summary.DuplicateGroups)
	}
	if result.Report.Summary.AccountsToDelete != 1 {
		t.Fatalf("expected 1 account to delete, got %d", result.Report.Summary.AccountsToDelete)
	}
}
