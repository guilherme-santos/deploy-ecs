package cobra

import (
	"errors"

	"github.com/spf13/cobra"
)

func NewExecCommand(cmd *Command) {
	cobraCmd := &cobra.Command{
		Use:   "exec <task-id>  [name or container_id] <command>",
		Short: "Execute command from specific task",
	}

	cobraCmd.PreRun = func(cobraCmd *cobra.Command, args []string) {
		cmd.CheckEnvironment()
	}

	cobraCmd.RunE = func(cobraCmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return errors.New("command needs two arguments: <task-id> [name or container_id] <command>")
		}

		var (
			nameOrContainerID string
			command           string
		)
		if len(args) > 2 {
			nameOrContainerID = args[1]
			command = args[2]
		} else {
			command = args[1]
		}

		cmd.AWSSession.Exec(args[0], nameOrContainerID, command)
		return nil
	}

	cmd.AddCommand(cobraCmd)
}
