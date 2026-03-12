package cmd

import (
	"io/fs"

	"github.com/spf13/cobra"
)

var templateFS fs.FS

func SetTemplateFS(f fs.FS) {
	templateFS = f
}

var createRepoCmd = &cobra.Command{
	Use:   "create-repo",
	Short: "Create a new GitHub repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		visibility, _ := cmd.Flags().GetString("visibility")
		description, _ := cmd.Flags().GetString("description")
		autoInit, _ := cmd.Flags().GetBool("auto-init")

		params := map[string]string{
			"name":        name,
			"visibility":  visibility,
			"description": description,
		}
		if !autoInit {
			params["auto_init"] = "false"
		}

		return runAction("create-repo", params)
	},
}

func init() {
	createRepoCmd.Flags().String("name", "", "repository name (required)")
	createRepoCmd.Flags().String("visibility", "public", "public or private")
	createRepoCmd.Flags().String("description", "", "repository description")
	createRepoCmd.Flags().Bool("auto-init", true, "initialize with README")
	_ = createRepoCmd.MarkFlagRequired("name")
	rootCmd.AddCommand(createRepoCmd)
}
