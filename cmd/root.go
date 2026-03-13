package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var configPath string
var jsonOutput bool
var autoApprove bool

// Version is set at build time via ldflags.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "gh-ops",
	Short: "One-click GitHub operations via OAuth",
	Long:  "A CLI tool that executes GitHub operations with user OAuth authorization.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", defaultConfigPath(), "config file path")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
	rootCmd.PersistentFlags().BoolVar(&autoApprove, "auto-approve", false, "skip web confirmation")
}
