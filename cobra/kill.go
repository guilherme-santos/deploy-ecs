package cobra

import (
	"errors"

	"github.com/spf13/cobra"
)

func NewKillCommand(cmd *Command) {
	cobraCmd := &cobra.Command{
		Use:   "kill <task-id> ...",
		Short: "Kill a specific task",
	}

	var wait bool

	cobraCmd.Flags().BoolVar(&wait, "wait", false, "wait until service is stopped")

	cobraCmd.PreRun = func(cobraCmd *cobra.Command, args []string) {
		cmd.CheckService()
		cmd.CheckEnvironment()
	}

	cobraCmd.RunE = func(cobraCmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("command needs an argument: <task-id> ...")
		}

		for _, task := range args {
			cmd.AWSSession.Kill(cmd.ServiceName, task)
		}

		if wait {
			cmd.AWSSession.WaitUntilTaskStopped(args)
		}

		return nil
	}

	cmd.AddCommand(cobraCmd)
}
