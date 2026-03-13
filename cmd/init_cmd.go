package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
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

func hasPswCLI() bool {
	_, err := exec.LookPath("psw-cli")
	return err == nil
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

		reader := bufio.NewReader(os.Stdin)

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

		fmt.Print("Enter Client ID: ")
		clientID, _ := reader.ReadString('\n')
		clientID = strings.TrimSpace(clientID)
		if clientID == "" {
			return fmt.Errorf("client ID is required")
		}

		fmt.Print("Enter Client Secret: ")
		clientSecret, _ := reader.ReadString('\n')
		clientSecret = strings.TrimSpace(clientSecret)

		// Choose storage method
		usePsw := false
		vaultName := "gh-ops"

		if hasPswCLI() {
			fmt.Println()
			fmt.Println("psw-cli detected. How would you like to store credentials?")
			fmt.Println("  1. Plain text (in ~/.gh-ops/config.yaml)")
			fmt.Println("  2. psw-cli (encrypted vault)")
			fmt.Println()
			fmt.Print("Choose [1/2]: ")
			choice, _ := reader.ReadString('\n')
			choice = strings.TrimSpace(choice)

			if choice == "2" {
				usePsw = true
				fmt.Print("Enter vault name [gh-ops]: ")
				v, _ := reader.ReadString('\n')
				v = strings.TrimSpace(v)
				if v != "" {
					vaultName = v
				}
			}
		}

		if usePsw {
			if err := storeToPswCLI(clientID, clientSecret, vaultName); err != nil {
				return err
			}
		}

		// Write config file
		if err := os.MkdirAll(defaultConfigDir(), 0700); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		var configContent string
		if usePsw {
			pswPath, _ := exec.LookPath("psw-cli")
			configContent = fmt.Sprintf(`server:
  port: 9091
  base_url: http://127.0.0.1:9091

github:
  client_id:
    source: exec
    command: "%s get GITHUB_CLIENT_ID -v %s --raw"
  client_secret:
    source: exec
    command: "%s get GITHUB_CLIENT_SECRET -v %s --raw"

allowed_actions:
  - create-repo
  - merge-pr
  - create-tag
  - add-collaborator

audit:
  db_path: ./audit.db
`, pswPath, vaultName, pswPath, vaultName)
		} else {
			configContent = fmt.Sprintf(`server:
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
		}

		if err := os.WriteFile(cfgPath, []byte(configContent), 0600); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}

		fmt.Println()
		fmt.Printf("Config saved to %s\n", cfgPath)
		if usePsw {
			fmt.Printf("Credentials encrypted in psw-cli vault \"%s\"\n", vaultName)
		}
		fmt.Println("You can now run gh-ops commands (e.g. gh-ops create-repo --name my-repo)")

		return nil
	},
}

func storeToPswCLI(clientID, clientSecret, vaultName string) error {
	// Check if vault exists, create if not
	checkCmd := exec.Command("psw-cli", "vault", "list")
	output, err := checkCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list psw-cli vaults: %w", err)
	}

	if !strings.Contains(string(output), vaultName) {
		createCmd := exec.Command("psw-cli", "vault", "create", vaultName, "--expire", "365d")
		if err := createCmd.Run(); err != nil {
			return fmt.Errorf("failed to create psw-cli vault %q: %w", vaultName, err)
		}
		fmt.Printf("Created psw-cli vault \"%s\"\n", vaultName)
	}

	// Store client ID
	setID := exec.Command("psw-cli", "set", "GITHUB_CLIENT_ID", clientID, "-v", vaultName)
	if err := setID.Run(); err != nil {
		return fmt.Errorf("failed to store client ID in psw-cli: %w", err)
	}

	// Store client secret
	if clientSecret != "" {
		setSecret := exec.Command("psw-cli", "set", "GITHUB_CLIENT_SECRET", clientSecret, "-v", vaultName)
		if err := setSecret.Run(); err != nil {
			return fmt.Errorf("failed to store client secret in psw-cli: %w", err)
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
}
