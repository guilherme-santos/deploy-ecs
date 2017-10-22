package cobra

import (
	"github.com/spf13/cobra"
)

func NewProcessStatusCommand(cmd *Command) {
	cobraCmd := &cobra.Command{
		Use:   "ps",
		Short: "List all task running to this service",
	}

	var showAll bool

	cobraCmd.Flags().BoolVarP(&showAll, "all", "a", false, "show all process (default shows just running)")

	cobraCmd.PreRun = func(cobraCmd *cobra.Command, args []string) {
		cmd.CheckService()
		cmd.CheckEnvironment()
	}

	cobraCmd.Run = func(cobraCmd *cobra.Command, args []string) {
		services := cmd.AWSSession.ListTaskDefinitionStartedWith(cmd.ServiceName)
		cmd.AWSSession.ListProcess(services, showAll)
	}

	cmd.AddCommand(cobraCmd)
}
