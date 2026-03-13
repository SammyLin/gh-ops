package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/SammyLin/gh-ops/internal/auth"
	"github.com/SammyLin/gh-ops/internal/config"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Set up GitHub OAuth credentials and authenticate",
	RunE: func(cmd *cobra.Command, args []string) error {
		tokenStore := auth.NewTokenStore(auth.DefaultTokenPath())

		// Check for cached token
		cached, _ := tokenStore.Load()
		if cached != nil {
			fmt.Printf("Already logged in as %s\n", cached.GitHubUser)
			return nil
		}

		// Try loading config
		cfg, err := config.Load(configPath)
		if err != nil || cfg.GitHub.ClientID == "" {
			// Guide user to create OAuth App
			fmt.Println("No GitHub OAuth credentials found. Let's set them up.")
			fmt.Println()
			fmt.Println("Step 1: Create a GitHub OAuth App")
			fmt.Println()
			fmt.Println("  1. Open: https://github.com/settings/developers")
			fmt.Println("  2. Click \"New OAuth App\"")
			fmt.Println("  3. Fill in:")
			fmt.Println("     - Application name: gh-ops")
			fmt.Println("     - Homepage URL: https://github.com/SammyLin/gh-ops")
			fmt.Println("     - Callback URL: http://localhost (not used)")
			fmt.Println("  4. Click \"Register application\"")
			fmt.Println("  5. Copy the Client ID")
			fmt.Println("  6. Click \"Generate a new client secret\" and copy it")
			fmt.Println()

			_ = openBrowser("https://github.com/settings/developers")

			reader := bufio.NewReader(os.Stdin)

			fmt.Print("Enter Client ID: ")
			clientID, _ := reader.ReadString('\n')
			clientID = strings.TrimSpace(clientID)
			if clientID == "" {
				return fmt.Errorf("client ID is required")
			}

			fmt.Print("Enter Client Secret: ")
			clientSecret, _ := reader.ReadString('\n')
			clientSecret = strings.TrimSpace(clientSecret)

			// Save to environment for this session
			_ = os.Setenv("GITHUB_CLIENT_ID", clientID)
			if clientSecret != "" {
				_ = os.Setenv("GITHUB_CLIENT_SECRET", clientSecret)
			}

			fmt.Println()
			fmt.Println("Step 2: Save these as environment variables in your shell profile:")
			fmt.Println()
			fmt.Printf("  export GITHUB_CLIENT_ID=%s\n", clientID)
			if clientSecret != "" {
				fmt.Printf("  export GITHUB_CLIENT_SECRET=%s\n", clientSecret)
			}
			fmt.Println()

			// Reload config with new env vars
			cfg, err = config.Load(configPath)
			if err != nil || cfg.GitHub.ClientID == "" {
				cfg = &config.Config{}
				cfg.GitHub.ClientID = clientID
			}
		}

		// Device Flow auth
		fmt.Println("Step 3: Authenticate with GitHub")
		fmt.Println()

		deviceResp, err := auth.RequestDeviceCode(auth.GitHubDeviceCodeURL, cfg.GitHub.ClientID, "repo")
		if err != nil {
			return err
		}

		fmt.Printf("  Open this URL:   %s\n", deviceResp.VerificationURI)
		fmt.Printf("  Enter this code: %s\n\n", deviceResp.UserCode)
		fmt.Println("Waiting for authorization...")
		_ = openBrowser(deviceResp.VerificationURI)

		accessToken, err := auth.PollForToken(auth.GitHubTokenURL, cfg.GitHub.ClientID, deviceResp.DeviceCode, deviceResp.Interval)
		if err != nil {
			return err
		}

		ghUser, err := auth.FetchGitHubUser(context.Background(), accessToken)
		if err != nil {
			return fmt.Errorf("failed to get user info: %w", err)
		}

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
