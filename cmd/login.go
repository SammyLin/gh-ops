package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/SammyLin/gh-ops/internal/config"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Set up GitHub OAuth App credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if already configured
		cfg, err := config.Load(configPath)
		if err == nil && cfg.GitHub.ClientID != "" {
			fmt.Println("GitHub OAuth credentials already configured.")
			fmt.Printf("  Client ID: %s...%s\n", cfg.GitHub.ClientID[:4], cfg.GitHub.ClientID[len(cfg.GitHub.ClientID)-4:])
			fmt.Println()
			fmt.Println("To reconfigure, update your environment variables or config.yaml.")
			return nil
		}

		fmt.Println("Let's set up your GitHub OAuth App credentials.")
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

		fmt.Println()
		fmt.Println("Add these to your shell profile (e.g. ~/.zshrc or ~/.bashrc):")
		fmt.Println()
		fmt.Printf("  export GITHUB_CLIENT_ID=%s\n", clientID)
		if clientSecret != "" {
			fmt.Printf("  export GITHUB_CLIENT_SECRET=%s\n", clientSecret)
		}
		fmt.Println()
		fmt.Println("Then reload your shell or run the export commands above.")
		fmt.Println("After that, run any gh-ops command (e.g. gh-ops create-repo) — you'll be prompted to authorize with your GitHub account.")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
