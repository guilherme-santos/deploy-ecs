package cobra

import (
	"fmt"
	"os"
	"strings"

	deploy "github.com/guilherme-santos/deploy-ecs"
	"github.com/guilherme-santos/deploy-ecs/aws"
	"github.com/guilherme-santos/deploy-ecs/shell"
	"github.com/spf13/cobra"
)

type (
	Command struct {
		cobra.Command

		env string

		Service struct {
			Name        string
			Respository string
			Namespace   string
		}
		ServiceName string
		Version     string
		Config      *deploy.Config
		Environment *deploy.Environment
		AWSSession  *aws.AWSSession
	}
)

func NewCommand(version, build string) *Command {
	cmd := &Command{
		Command: cobra.Command{
			Use:   "deploy-ecs",
			Short: "Tool to deploy and manager AWS ECS configuration",
		},
		Version: version,
	}

	var versionFlag bool

	cmd.Run = func(cobraCmd *cobra.Command, args []string) {
		if versionFlag {
			fmt.Printf("%s version %s", cmd.Name(), cmd.Version)
			if !strings.EqualFold("", build) {
				fmt.Printf(", build %s", build)
			}
			fmt.Println("")
		} else {
			cobraCmd.Usage()
		}
	}

	cmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Print version information")

	cmd.PersistentFlags().StringVarP(&cmd.Service.Name, "service", "s", "", `Service name, some commands will affect <service-name>-*
		If not provide it will be extracted from Repository URL (see --repository)`)
	cmd.PersistentFlags().StringVarP(&cmd.Service.Namespace, "namespace", "n", "", `Namespace can be used to deploy same service more than once`)
	cmd.PersistentFlags().StringVarP(&cmd.Service.Respository, "repository", "r", "", `Repository URL, service name will be extract from url
		If not provide it will be used current directory, if was a git project`)

	cmd.LoadConfig()

	helper := fmt.Sprint("Environment name, options: ", cmd.getListEnvironments())
	cmd.PersistentFlags().StringVar(&cmd.env, "env", cmd.Config.DefaultEnvironment, helper)

	NewSelfUpdateCommand(cmd)
	NewConfigCommand(cmd)
	// NewServicesCommand(cmd)
	NewListRevisionsCommand(cmd)
	NewTaskDefinitionCommand(cmd)
	NewEnvvarCommand(cmd)
	NewProcessStatusCommand(cmd)
	NewLogsCommand(cmd)
	NewDeployCommand(cmd)
	NewRollbackCommand(cmd)
	NewExecCommand(cmd)
	NewKillCommand(cmd)
	NewScaleCommand(cmd)

	return cmd
}

func (cmd *Command) getListEnvironments() string {
	environments := cmd.Config.GetAvailableEnvironments()
	if len(environments) == 0 {
		return "_no environment found_"
	}

	return strings.Join(environments, ", ")
}

func (cmd *Command) getServiceNameWithNamespace() string {
	if !strings.EqualFold("", cmd.Service.Namespace) {
		return cmd.Service.Namespace + "-" + cmd.Service.Name
	}

	return cmd.Service.Name
}

func (cmd *Command) CheckEnvironment() {
	cmd.Environment = cmd.Config.GetEnvironment(cmd.env)
	if cmd.Environment == nil {
		fmt.Printf("Environment '%s' is not a valid, use: %s\n", cmd.env, cmd.getListEnvironments())
		os.Exit(1)
	}
	cmd.AWSSession = aws.NewAWSSession(cmd.Environment)
}

func (cmd *Command) CheckService() {
	if strings.EqualFold("", cmd.Service.Name) {
		if strings.EqualFold("", cmd.Service.Respository) {
			if !shell.IsGitInstalled() {
				fmt.Printf("The program 'git' is currently not installed.")
				os.Exit(1)
			}

			cmd.Service.Respository = shell.GetGitOrigin()
		}

		pos := strings.LastIndex(cmd.Service.Respository, "/")
		if pos > 0 {
			cmd.Service.Name = strings.TrimSuffix(cmd.Service.Respository[pos+1:], ".git")
		} else {
			fmt.Printf("Repository '%s' is not a valid repository\n", cmd.Service.Respository)
			os.Exit(1)
		}

		cmd.ServiceName = cmd.getServiceNameWithNamespace()

		return
	}

	if strings.EqualFold("", cmd.Service.Respository) {
		if strings.EqualFold("", cmd.Config.GitHub.DefaultRepository) {
			fmt.Println("Error: default repository was not configured use --repository or configure with follow command:")
			fmt.Println("\t- deploy-ecs config github set repository <git@github.com:YourCompany>")

			cmd.Usage()
			os.Exit(1)
		}

		cmd.Service.Respository = cmd.Config.GitHub.DefaultRepository + cmd.Service.Name
	}

	cmd.ServiceName = cmd.getServiceNameWithNamespace()
}
