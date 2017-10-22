package cobra

import (
	"github.com/spf13/cobra"
)

func NewRollbackCommand(cmd *Command) {
	cobraCmd := &cobra.Command{
		Use:   "rollback",
		Short: "Rollback service to last revision",
	}

	var wait bool

	cobraCmd.Flags().BoolVar(&wait, "wait", false, "wait until service are stable")

	cobraCmd.PreRun = func(cobraCmd *cobra.Command, args []string) {
		cmd.CheckService()
		cmd.CheckEnvironment()
	}

	cobraCmd.Run = func(cobraCmd *cobra.Command, args []string) {
		services := cmd.AWSSession.ListTaskDefinitionStartedWith(cmd.ServiceName)

		for _, service := range services {
			cmd.AWSSession.Rollback(service)
		}

		if wait && len(services) > 0 {
			cmd.AWSSession.WaitUntilServicesStable(services)
		}
	}

	cmd.AddCommand(cobraCmd)
}
