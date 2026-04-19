package accountdedup

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientPreviewUsesListEndpointAndDoesNotDelete(t *testing.T) {
	var deleteCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/admin/accounts":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    0,
				"message": "success",
				"data": map[string]any{
					"items": []map[string]any{
						{
							"id":         1,
							"name":       "dup-1",
							"platform":   "openai",
							"type":       "oauth",
							"created_at": time.Date(2026, 4, 19, 8, 0, 0, 0, time.UTC).Format(time.RFC3339),
							"credentials": map[string]any{
								"refresh_token": "rt-1",
							},
						},
						{
							"id":         2,
							"name":       "dup-2",
							"platform":   "openai",
							"type":       "oauth",
							"created_at": time.Date(2026, 4, 19, 9, 0, 0, 0, time.UTC).Format(time.RFC3339),
							"credentials": map[string]any{
								"refresh_token": "rt-1",
							},
						},
					},
					"total":     2,
					"page":      1,
					"page_size": 1000,
					"pages":     1,
				},
			})
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/admin/accounts/"):
			deleteCalls++
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    0,
				"message": "success",
				"data": map[string]any{
					"message": "deleted",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", server.Client())

	accounts, err := client.ListAccounts(context.Background())
	if err != nil {
		t.Fatalf("ListAccounts returned error: %v", err)
	}

	report := BuildPreview(accounts)
	if len(report.DuplicateGroups) != 1 {
		t.Fatalf("expected one duplicate group, got %d", len(report.DuplicateGroups))
	}
	if deleteCalls != 0 {
		t.Fatalf("expected no delete calls during preview, got %d", deleteCalls)
	}
}

func TestClientApplyDeletesOnlyAccountsMarkedForRemoval(t *testing.T) {
	var deletedIDs []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/admin/accounts":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    0,
				"message": "success",
				"data": map[string]any{
					"items": []map[string]any{
						{
							"id":         10,
							"name":       "dup-10",
							"platform":   "anthropic",
							"type":       "apikey",
							"created_at": time.Date(2026, 4, 19, 8, 0, 0, 0, time.UTC).Format(time.RFC3339),
							"credentials": map[string]any{
								"api_key": "key-1",
							},
						},
						{
							"id":         11,
							"name":       "dup-11",
							"platform":   "anthropic",
							"type":       "apikey",
							"created_at": time.Date(2026, 4, 19, 9, 0, 0, 0, time.UTC).Format(time.RFC3339),
							"credentials": map[string]any{
								"api_key": "key-1",
							},
						},
					},
					"total":     2,
					"page":      1,
					"page_size": 1000,
					"pages":     1,
				},
			})
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/admin/accounts/"):
			deletedIDs = append(deletedIDs, strings.TrimPrefix(r.URL.Path, "/api/v1/admin/accounts/"))
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    0,
				"message": "success",
				"data": map[string]any{
					"message": "deleted",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", server.Client())

	accounts, err := client.ListAccounts(context.Background())
	if err != nil {
		t.Fatalf("ListAccounts returned error: %v", err)
	}
	report := BuildPreview(accounts)
	applyResult := ApplyDeletionPlan(context.Background(), client, report)

	if len(applyResult.Deleted) != 1 {
		t.Fatalf("expected one deleted account, got %d", len(applyResult.Deleted))
	}
	if len(applyResult.Failed) != 0 {
		t.Fatalf("expected no delete failures, got %d", len(applyResult.Failed))
	}
	if len(deletedIDs) != 1 || deletedIDs[0] != "10" {
		t.Fatalf("expected account 10 to be deleted, got %#v", deletedIDs)
	}
}
