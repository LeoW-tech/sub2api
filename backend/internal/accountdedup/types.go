package accountdedup

import (
	"io"
	"net/http"
	"time"
)

type Account struct {
	ID          int64          `json:"id"`
	Name        string         `json:"name"`
	Platform    string         `json:"platform"`
	Type        string         `json:"type"`
	Credentials map[string]any `json:"credentials"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type DuplicateGroup struct {
	Platform   string    `json:"platform"`
	Type       string    `json:"type"`
	MatchBasis string    `json:"match_basis"`
	MatchValue string    `json:"match_value"`
	Keep       Account   `json:"keep"`
	Delete     []Account `json:"delete"`
}

type SkippedAccount struct {
	Account Account `json:"account"`
	Reason  string  `json:"reason"`
}

type DeleteResult struct {
	Account Account `json:"account"`
	Error   string  `json:"error,omitempty"`
}

type Summary struct {
	TotalAccounts    int `json:"total_accounts"`
	DuplicateGroups  int `json:"duplicate_groups"`
	AccountsToDelete int `json:"accounts_to_delete"`
	SkippedAccounts  int `json:"skipped_accounts"`
	DeletedAccounts  int `json:"deleted_accounts"`
	FailedDeletions  int `json:"failed_deletions"`
}

type PreviewReport struct {
	GeneratedAt     time.Time        `json:"generated_at"`
	BaseURL         string           `json:"base_url"`
	Mode            string           `json:"mode"`
	DuplicateGroups []DuplicateGroup `json:"duplicate_groups"`
	SkippedAccounts []SkippedAccount `json:"skipped_accounts"`
	Deleted         []DeleteResult   `json:"deleted"`
	Failed          []DeleteResult   `json:"failed"`
	Summary         Summary          `json:"summary"`
}

type ApplyResult struct {
	Deleted []DeleteResult `json:"deleted"`
	Failed  []DeleteResult `json:"failed"`
}

type RunOptions struct {
	BaseURL     string
	AccessToken string
	Email       string
	Password    string
	OutputDir   string
	Apply       bool
	HTTPClient  *http.Client
	Stdout      io.Writer
	Confirm     func(prompt string) (bool, error)
}

type RunResult struct {
	Report     PreviewReport `json:"report"`
	ReportPath string        `json:"report_path"`
}
