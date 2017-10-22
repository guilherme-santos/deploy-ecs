package cobra

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func NewTaskDefinitionCommand(cmd *Command) {
	cobraCmd := &cobra.Command{
		Use:   "task-definition",
		Short: "Manage task definition of a service",
	}

	var (
		revision   int64
		sets       []string
		deploy     bool
		waitDeploy bool
	)

	cobraCmd.Flags().Int64Var(&revision, "revision", 0, "revision number, if not present will use last one")
	cobraCmd.Flags().StringArrayVar(&sets, "set", nil, "property=value to be updated (can be used multiple times)")
	cobraCmd.Flags().BoolVar(&deploy, "deploy", false, "change task-definition and redeploy")
	cobraCmd.Flags().BoolVar(&waitDeploy, "wait", false, "should be used with --deploy flag")

	cobraCmd.PreRun = func(cobraCmd *cobra.Command, args []string) {
		cmd.CheckService()
		cmd.CheckEnvironment()
	}

	cobraCmd.Run = func(cobraCmd *cobra.Command, args []string) {
		if len(sets) == 0 {
			getTaskDefinition(cmd, revision)
			return
		}

		updateTaskDefinition(cmd, revision, sets, deploy, waitDeploy)
	}

	cmd.AddCommand(cobraCmd)
}

func getTaskDefinition(cmd *Command, revision int64) {
	cmd.AWSSession.GetTaskDefinition(cmd.ServiceName, revision)
}

func updateTaskDefinition(cmd *Command, revision int64, sets []string, deploy, waitDeploy bool) {
	changes := make(map[string]string)
	for _, change := range sets {
		parts := strings.SplitN(change, "=", 2)

		var value string
		if len(parts) > 1 {
			value = parts[1]
		}

		changes[parts[0]] = value
	}

	arn := cmd.AWSSession.UpdateTaskDefinition(cmd.ServiceName, revision, changes)
	if deploy {
		revision := arn[strings.LastIndex(arn, ":")+1:]

		deployArgs := []string{"deploy", "--service", cmd.ServiceName, "--revision", fmt.Sprint(revision)}
		if waitDeploy {
			deployArgs = append(deployArgs, "--wait")
		}

		cmd.SetArgs(deployArgs)
		if err := cmd.Execute(); err != nil {
			fmt.Printf("Cannot deploy new revision to '%s': %s", cmd.ServiceName, err.Error())
			os.Exit(1)
		}
	}
}
