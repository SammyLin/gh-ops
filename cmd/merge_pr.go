package cmd

import (
	"github.com/spf13/cobra"
)

var mergePRCmd = &cobra.Command{
	Use:   "merge-pr",
	Short: "Merge a pull request",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, _ := cmd.Flags().GetString("repo")
		prNumber, _ := cmd.Flags().GetString("pr-number")
		mergeMethod, _ := cmd.Flags().GetString("merge-method")

		params := map[string]string{
			"repo":         repo,
			"pr_number":    prNumber,
			"merge_method": mergeMethod,
		}

		return runAction("merge-pr", params)
	},
}

func init() {
	mergePRCmd.Flags().String("repo", "", "repository in owner/repo format (required)")
	mergePRCmd.Flags().String("pr-number", "", "pull request number (required)")
	mergePRCmd.Flags().String("merge-method", "merge", "merge, squash, or rebase")
	_ = mergePRCmd.MarkFlagRequired("repo")
	_ = mergePRCmd.MarkFlagRequired("pr-number")
	rootCmd.AddCommand(mergePRCmd)
}
