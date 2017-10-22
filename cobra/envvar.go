package cobra

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func NewEnvvarCommand(cmd *Command) {
	cobraCmd := &cobra.Command{
		Use:     "env",
		Short:   "Manage env vars from a service",
		Aliases: []string{"envvar"},
	}

	var (
		revision    int64
		formatJson  bool
		gets        []string
		sets        []string
		unsets      []string
		deploy      bool
		waitDeploy  bool
		allServices bool
	)

	cobraCmd.Flags().Int64Var(&revision, "revision", 0, "revision number, if not present will use last one")
	cobraCmd.Flags().BoolVar(&formatJson, "json", false, "If envvars should be formated as json")
	cobraCmd.Flags().StringArrayVar(&gets, "get", nil, "key to be read (can be used multiple times)")
	cobraCmd.Flags().StringArrayVar(&sets, "set", nil, "key=value to be updated (can be used multiple times)")
	cobraCmd.Flags().StringArrayVar(&unsets, "unset", nil, "key to be removed (can be used multiple times)")
	cobraCmd.Flags().BoolVar(&deploy, "deploy", false, "change env var and redeploy")
	cobraCmd.Flags().BoolVar(&waitDeploy, "wait", false, "should be used with --deploy flag")
	cobraCmd.Flags().BoolVar(&allServices, "all", false, "update this current service and all its children <service>-*, should be used just when you're setting envvars")

	cobraCmd.PreRun = func(cobraCmd *cobra.Command, args []string) {
		cmd.CheckService()
		cmd.CheckEnvironment()
	}

	cobraCmd.RunE = func(cobraCmd *cobra.Command, args []string) error {
		services := map[string]int64{}

		if allServices {
			if revision != 0 {
				return errors.New("Cannot use --all passing a revision, remove it to use last revision")
			}

			for _, service := range cmd.AWSSession.ListTaskDefinitionStartedWith(cmd.ServiceName) {
				services[service] = 0
			}
		} else {
			services[cmd.ServiceName] = 0
		}

		stdin, _ := os.Stdin.Stat()
		if (stdin.Mode()&os.ModeCharDevice) != os.ModeCharDevice && stdin.Size() > 0 {
			if len(sets) > 0 || len(unsets) > 0 {
				return errors.New("Cannot update envvar: use stdin or --set / --unset not both")
			}

			for service := range services {
				changes, unsets := cmd.AWSSession.DiffEnvvarFromFile(cmd.ServiceName, revision, os.Stdin, formatJson)
				services[service] = cmd.AWSSession.UpdateEnvvar(cmd.ServiceName, revision, changes, unsets)
			}
		} else if len(sets) == 0 && len(unsets) == 0 {
			if allServices {
				return errors.New("Cannot use --all to get envvars")
			}

			cmd.AWSSession.GetEnvvar(cmd.ServiceName, revision, gets, formatJson)
			return nil
		} else {
			for service := range services {
				services[service] = updateEnvvar(cmd, service, revision, sets, unsets)
			}
		}

		if !deploy {
			return nil
		}

		for service, revision := range services {
			if revision > 0 {
				deployArgs := []string{"deploy", "--service", service, "--revision", fmt.Sprint(revision)}
				if waitDeploy {
					deployArgs = append(deployArgs, "--wait")
				}

				cmd.SetArgs(deployArgs)
				if err := cmd.Execute(); err != nil {
					return fmt.Errorf("Cannot deploy new revision to '%s': %s", service, err.Error())
				}
			}
		}

		return nil
	}

	cmd.AddCommand(cobraCmd)
}

func updateEnvvar(cmd *Command, serviceName string, revision int64, sets []string, unsets []string) int64 {
	changes := make(map[string]string)
	for _, change := range sets {
		parts := strings.SplitN(change, "=", 2)

		var value string
		if len(parts) > 1 {
			value = parts[1]
		}

		changes[parts[0]] = value
	}

	removes := make(map[string]struct{})
	for _, field := range unsets {
		removes[field] = struct{}{}
	}

	return cmd.AWSSession.UpdateEnvvar(serviceName, revision, changes, removes)
}
