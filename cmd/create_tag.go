package cmd

import (
	"github.com/spf13/cobra"
)

var createTagCmd = &cobra.Command{
	Use:   "create-tag",
	Short: "Create a git tag",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, _ := cmd.Flags().GetString("repo")
		tag, _ := cmd.Flags().GetString("tag")
		sha, _ := cmd.Flags().GetString("sha")
		message, _ := cmd.Flags().GetString("message")

		params := map[string]string{
			"repo":    repo,
			"tag":     tag,
			"sha":     sha,
			"message": message,
		}

		return runAction("create-tag", params)
	},
}

func init() {
	createTagCmd.Flags().String("repo", "", "repository in owner/repo format (required)")
	createTagCmd.Flags().String("tag", "", "tag name, e.g. v1.0.0 (required)")
	createTagCmd.Flags().String("sha", "", "commit SHA (default: HEAD of default branch)")
	createTagCmd.Flags().String("message", "", "tag message (creates annotated tag)")
	_ = createTagCmd.MarkFlagRequired("repo")
	_ = createTagCmd.MarkFlagRequired("tag")
	rootCmd.AddCommand(createTagCmd)
}
