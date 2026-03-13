package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func defaultConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, ".gh-ops")
}

func defaultConfigPath() string {
	return filepath.Join(defaultConfigDir(), "config.yaml")
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up GitHub OAuth App credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath := defaultConfigPath()

		// Check if already configured
		if _, err := os.Stat(cfgPath); err == nil {
			fmt.Printf("Config already exists at %s\n", cfgPath)
			fmt.Println("To reconfigure, delete it and run gh-ops init again.")
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

		// Write config file
		if err := os.MkdirAll(defaultConfigDir(), 0700); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		configContent := fmt.Sprintf(`server:
  port: 9091
  base_url: http://127.0.0.1:9091

github:
  client_id: "%s"
  client_secret: "%s"

allowed_actions:
  - create-repo
  - merge-pr
  - create-tag
  - add-collaborator

audit:
  db_path: ./audit.db
`, clientID, clientSecret)

		if err := os.WriteFile(cfgPath, []byte(configContent), 0600); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}

		fmt.Println()
		fmt.Printf("Config saved to %s\n", cfgPath)
		fmt.Println("Run gh-ops login to authenticate with GitHub.")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
