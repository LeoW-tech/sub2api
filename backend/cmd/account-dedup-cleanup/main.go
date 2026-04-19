package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/accountdedup"
)

func main() {
	baseURL := flag.String("base-url", "", "Admin base URL, default http://127.0.0.1:8081")
	accessToken := flag.String("access-token", "", "Admin access token")
	email := flag.String("email", "", "Admin email used for /api/v1/auth/login when no access token is provided")
	password := flag.String("password", "", "Admin password used for /api/v1/auth/login when no access token is provided")
	apply := flag.Bool("apply", false, "Apply deletions after interactive confirmation")
	outputDir := flag.String("output-dir", "", "Directory used to store preview/apply reports")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := accountdedup.Run(ctx, accountdedup.RunOptions{
		BaseURL:     *baseURL,
		AccessToken: *accessToken,
		Email:       *email,
		Password:    *password,
		OutputDir:   *outputDir,
		Apply:       *apply,
		Stdout:      os.Stdout,
		Confirm:     promptForConfirmation,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "完成，报告已写入 %s\n", result.ReportPath)
}

func promptForConfirmation(prompt string) (bool, error) {
	fmt.Fprint(os.Stdout, prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(line), "yes"), nil
}
