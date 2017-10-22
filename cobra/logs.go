package cobra

import (
	"errors"

	"github.com/spf13/cobra"
)

func NewLogsCommand(cmd *Command) {
	cobraCmd := &cobra.Command{
		Use:   "logs <task-id> [name or container_id]",
		Short: "Show log from specific task",
	}

	var (
		tail   string
		follow bool
	)

	cobraCmd.Flags().StringVar(&tail, "tail", "all", "Number of lines to show from the end of the logs")
	cobraCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")

	cobraCmd.PreRun = func(cobraCmd *cobra.Command, args []string) {
		cmd.CheckEnvironment()
	}

	cobraCmd.RunE = func(cobraCmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("command needs an argument: <task-id> [name or container_id]")
		}

		var nameOrContainerID string
		if len(args) > 1 {
			nameOrContainerID = args[1]
		}

		cmd.AWSSession.GetLogs(args[0], nameOrContainerID, tail, follow)
		return nil
	}

	cmd.AddCommand(cobraCmd)
}
