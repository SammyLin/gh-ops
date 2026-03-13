package cmd

import (
	"fmt"

	"github.com/SammyLin/gh-ops/internal/auth"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear cached GitHub OAuth token",
	RunE: func(cmd *cobra.Command, args []string) error {
		store := auth.NewTokenStore(auth.DefaultTokenPath())
		if err := store.Clear(); err != nil {
			return fmt.Errorf("failed to clear token: %w", err)
		}
		fmt.Println("Logged out. Cached token removed.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
