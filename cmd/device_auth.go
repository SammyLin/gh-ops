package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/SammyLin/gh-ops/internal/actions"
	"github.com/SammyLin/gh-ops/internal/audit"
	"github.com/SammyLin/gh-ops/internal/auth"
	"github.com/SammyLin/gh-ops/internal/config"
)

func runDeviceFlow(cfg *config.ResolvedConfig, actionName string, params map[string]string, tokenStore *auth.TokenStore) error {
	if cfg.GitHub.ClientID == "" {
		return fmt.Errorf("GITHUB_CLIENT_ID must be set (check your environment or config file)")
	}

	// Show action confirmation in terminal
	fmt.Printf("\n  Action:  %s\n", actionName)
	for k, v := range params {
		if v != "" {
			fmt.Printf("  %-14s %s\n", k+":", v)
		}
	}
	fmt.Println()

	// Request device code
	deviceResp, err := auth.RequestDeviceCode(auth.GitHubDeviceCodeURL, cfg.GitHub.ClientID, "repo")
	if err != nil {
		return err
	}

	// Show user code and open browser
	fmt.Printf("  Open this URL:   %s\n", deviceResp.VerificationURI)
	fmt.Printf("  Enter this code: %s\n\n", deviceResp.UserCode)
	fmt.Printf("Waiting for authorization...\n")
	_ = openBrowser(deviceResp.VerificationURI)

	// Poll for token
	accessToken, err := auth.PollForToken(auth.GitHubTokenURL, cfg.GitHub.ClientID, deviceResp.DeviceCode, deviceResp.Interval)
	if err != nil {
		return err
	}

	// Fetch GitHub user
	ghUser, err := auth.FetchGitHubUser(context.Background(), accessToken)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	// Cache token
	_ = tokenStore.Save(&auth.CachedToken{
		AccessToken: accessToken,
		GitHubUser:  ghUser,
		SavedAt:     time.Now().UTC(),
	})

	// Wait for approval
	if !autoApprove {
		confirmed, approvalErr := waitForApproval(cfg, actionName, params, ghUser)
		if approvalErr != nil {
			return fmt.Errorf("approval flow failed: %w", approvalErr)
		}
		if !confirmed {
			return fmt.Errorf("action cancelled by user")
		}
	}

	// Execute action
	result, err := actions.Execute(actionName, params, accessToken)

	// Audit log
	auditLogger, auditErr := audit.New(cfg.Audit.DBPath)
	if auditErr != nil {
		log.Printf("Warning: audit log disabled: %v", auditErr)
	} else {
		defer func() { _ = auditLogger.Close() }()
		logResult := "success"
		if err != nil {
			logResult = "error: " + err.Error()
		}
		_ = auditLogger.Log(audit.Entry{
			GitHubUser: ghUser,
			Action:     actionName,
			Parameters: params,
			Result:     logResult,
			IPAddress:  "localhost",
		})
	}

	if err != nil {
		return err
	}

	fmt.Printf("\n✅ %s\n", result)
	return nil
}
