package cmd

import (
	"github.com/spf13/cobra"
)

var addCollaboratorCmd = &cobra.Command{
	Use:   "add-collaborator",
	Short: "Add a collaborator to a repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, _ := cmd.Flags().GetString("repo")
		user, _ := cmd.Flags().GetString("user")
		permission, _ := cmd.Flags().GetString("permission")

		params := map[string]string{
			"repo":       repo,
			"user":       user,
			"permission": permission,
		}

		return runAction("add-collaborator", params)
	},
}

func init() {
	addCollaboratorCmd.Flags().String("repo", "", "repository in owner/repo format (required)")
	addCollaboratorCmd.Flags().String("user", "", "GitHub username (required)")
	addCollaboratorCmd.Flags().String("permission", "push", "pull, push, or admin")
	_ = addCollaboratorCmd.MarkFlagRequired("repo")
	_ = addCollaboratorCmd.MarkFlagRequired("user")
	rootCmd.AddCommand(addCollaboratorCmd)
}
