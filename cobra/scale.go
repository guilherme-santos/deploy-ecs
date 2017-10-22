package cobra

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func NewScaleCommand(cmd *Command) {
	cobraCmd := &cobra.Command{
		Use:   "scale <number-of-tasks>",
		Short: "Scale service to number of task",
	}

	var wait bool

	cobraCmd.Flags().BoolVar(&wait, "wait", false, "wait until service is stopped")

	cobraCmd.PreRun = func(cobraCmd *cobra.Command, args []string) {
		cmd.CheckService()
		cmd.CheckEnvironment()
	}

	cobraCmd.RunE = func(cobraCmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("command needs an argument: <number-of-tasks>")
		}

		numberOfTasks, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			return fmt.Errorf("cannot convert <number-of-tasks> as integer: %s", err)
		}

		cmd.AWSSession.Scale(cmd.ServiceName, numberOfTasks)

		if wait {
			cmd.AWSSession.WaitUntilServicesStable([]string{cmd.ServiceName})
		}

		return nil
	}

	cmd.AddCommand(cobraCmd)
}
