package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/SammyLin/gh-ops/internal/auth"
	"github.com/SammyLin/gh-ops/internal/config"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with GitHub via Device Flow",
	RunE: func(cmd *cobra.Command, args []string) error {
		tokenStore := auth.NewTokenStore(auth.DefaultTokenPath())

		// Check for cached token
		cached, _ := tokenStore.Load()
		if cached != nil {
			fmt.Printf("Already logged in as %s\n", cached.GitHubUser)
			return nil
		}

		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if cfg.GitHub.ClientID == "" {
			return fmt.Errorf("GITHUB_CLIENT_ID must be set (check your environment or config file)")
		}

		// Request device code
		deviceResp, err := auth.RequestDeviceCode(auth.GitHubDeviceCodeURL, cfg.GitHub.ClientID, "repo")
		if err != nil {
			return err
		}

		fmt.Printf("\n  Open this URL:   %s\n", deviceResp.VerificationURI)
		fmt.Printf("  Enter this code: %s\n\n", deviceResp.UserCode)
		fmt.Println("Waiting for authorization...")
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

		fmt.Printf("\nLogged in as %s\n", ghUser)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
