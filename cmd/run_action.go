package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/SammyLin/gh-ops/internal/actions"
	"github.com/SammyLin/gh-ops/internal/auth"
	"github.com/SammyLin/gh-ops/internal/config"
)

func runAction(actionName string, params map[string]string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !cfg.IsActionAllowed(actionName) {
		return fmt.Errorf("action %q is not in the allowed list", actionName)
	}

	// Check for cached token
	tokenStore := auth.NewTokenStore(auth.DefaultTokenPath())
	cached, _ := tokenStore.Load()

	if cached != nil {
		fmt.Printf("Using cached token for %s...\n", cached.GitHubUser)

		if !autoApprove {
			confirmed, err := waitForApproval(cfg, actionName, params, cached.GitHubUser)
			if err != nil {
				return fmt.Errorf("approval flow failed: %w", err)
			}
			if !confirmed {
				return fmt.Errorf("action cancelled by user")
			}
		}

		result, err := executeAction(cfg, actionName, params, cached.AccessToken, cached.GitHubUser)
		if err != nil {
			fmt.Printf("Cached token failed, re-authorizing...\n")
			_ = tokenStore.Clear()
		} else {
			fmt.Println(result)
			return nil
		}
	}

	return runDeviceFlow(cfg, actionName, params, tokenStore)
}

func executeAction(cfg *config.ResolvedConfig, actionName string, params map[string]string, accessToken, ghUser string) (string, error) {
	return actions.Execute(actionName, params, accessToken)
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return nil
	}
}
