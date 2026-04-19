package accountdedup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultBaseURL = "http://127.0.0.1:8081"

func Run(ctx context.Context, opts RunOptions) (RunResult, error) {
	baseURL := strings.TrimSpace(opts.BaseURL)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	client := NewClient(baseURL, opts.AccessToken, opts.HTTPClient)
	if strings.TrimSpace(opts.AccessToken) == "" {
		if strings.TrimSpace(opts.Email) == "" || opts.Password == "" {
			return RunResult{}, fmt.Errorf("must provide either --access-token or both --email and --password")
		}
		if err := client.Login(ctx, opts.Email, opts.Password); err != nil {
			return RunResult{}, err
		}
	}

	accounts, err := client.ListAccounts(ctx)
	if err != nil {
		return RunResult{}, err
	}

	report := BuildPreview(accounts)
	report.GeneratedAt = time.Now().UTC()
	report.BaseURL = baseURL

	outputDir := strings.TrimSpace(opts.OutputDir)
	if outputDir == "" {
		outputDir, err = defaultOutputDir(report.GeneratedAt)
		if err != nil {
			return RunResult{}, err
		}
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return RunResult{}, err
	}

	fileName := "account-dedup-preview.json"
	if opts.Apply {
		if opts.Confirm == nil {
			return RunResult{}, fmt.Errorf("--apply requires an interactive confirmation handler")
		}
		ok, confirmErr := opts.Confirm(buildConfirmPrompt(baseURL, report.Summary.AccountsToDelete))
		if confirmErr != nil {
			return RunResult{}, confirmErr
		}
		if !ok {
			return RunResult{}, fmt.Errorf("operation cancelled")
		}

		applyResult := ApplyDeletionPlan(ctx, client, report)
		report.Mode = "apply"
		report.Deleted = applyResult.Deleted
		report.Failed = applyResult.Failed
		report.Summary.DeletedAccounts = len(applyResult.Deleted)
		report.Summary.FailedDeletions = len(applyResult.Failed)
		fileName = "account-dedup-apply.json"
	}

	reportPath := filepath.Join(outputDir, fileName)
	if err := writeReport(reportPath, report); err != nil {
		return RunResult{}, err
	}
	printSummary(opts.Stdout, reportPath, report)

	return RunResult{
		Report:     report,
		ReportPath: reportPath,
	}, nil
}

func buildConfirmPrompt(baseURL string, deleteCount int) string {
	return fmt.Sprintf("目标环境: %s\n待删除账号数: %d\n规则: 同 platform+type 内查重，保留 created_at 最新，时间相同保留 ID 更大。\n输入 yes 继续执行删除: ", baseURL, deleteCount)
}

func writeReport(path string, report PreviewReport) error {
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o600)
}

func printSummary(stdout io.Writer, reportPath string, report PreviewReport) {
	if stdout == nil {
		return
	}
	_, _ = fmt.Fprintf(stdout,
		"模式: %s\n账号总数: %d\n重复组数: %d\n待删除账号数: %d\n跳过账号数: %d\n已删除: %d\n删除失败: %d\n报告: %s\n",
		report.Mode,
		report.Summary.TotalAccounts,
		report.Summary.DuplicateGroups,
		report.Summary.AccountsToDelete,
		report.Summary.SkippedAccounts,
		report.Summary.DeletedAccounts,
		report.Summary.FailedDeletions,
		reportPath,
	)
}

func defaultOutputDir(now time.Time) (string, error) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return "", err
	}
	timestamp := now.UTC().Format("20060102-150405")
	return filepath.Join(repoRoot, "runtime", "backups", timestamp), nil
}

func findRepoRoot() (string, error) {
	current, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if dirExists(filepath.Join(current, "runtime")) && dirExists(filepath.Join(current, "backend")) {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("failed to locate repository root from cwd")
		}
		current = parent
	}
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
