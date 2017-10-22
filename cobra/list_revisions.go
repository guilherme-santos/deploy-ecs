package cobra

import (
	"github.com/spf13/cobra"
)

func NewListRevisionsCommand(cmd *Command) {
	cobraCmd := &cobra.Command{
		Use:   "list-revisions",
		Short: "List all availables revision from a service",
	}

	cobraCmd.PreRun = func(cobraCmd *cobra.Command, args []string) {
		cmd.CheckService()
		cmd.CheckEnvironment()
	}

	cobraCmd.Run = func(cobraCmd *cobra.Command, args []string) {
		cmd.AWSSession.ListRevisions(cmd.ServiceName)
	}

	cmd.AddCommand(cobraCmd)
}
