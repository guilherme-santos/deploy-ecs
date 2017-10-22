package cobra

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/guilherme-santos/deploy-ecs/shell"
	"github.com/spf13/cobra"
)

func NewDeployCommand(cmd *Command) {
	cobraCmd := &cobra.Command{
		Use:   "deploy",
		Short: "Build an image and update service to use new revision",
	}

	var (
		revision    int64
		tagOrBranch string
		rebuild     bool
		wait        bool
	)

	cobraCmd.Flags().Int64Var(&revision, "revision", 0, "revision number, if not present will use last one")
	cobraCmd.Flags().StringVarP(&tagOrBranch, "tag", "t", "", "tag or branch name to build and deploy")
	cobraCmd.Flags().BoolVar(&rebuild, "rebuild", false, "force rebuild image even it already cached")
	cobraCmd.Flags().BoolVar(&wait, "wait", false, "wait until services are stable")

	cobraCmd.PreRun = func(cobraCmd *cobra.Command, args []string) {
		cmd.CheckService()
		cmd.CheckEnvironment()
	}

	cobraCmd.RunE = func(cobraCmd *cobra.Command, args []string) error {
		var deployTag, deployRevision bool

		if !strings.EqualFold("", tagOrBranch) {
			deployTag = true
		}
		if revision != 0 {
			deployRevision = true
		}

		if !(deployTag || deployRevision) {
			return errors.New("command needs an argument: --tag OR --revision")
		}

		if deployTag && deployRevision {
			return errors.New("command needs just one argument: --tag OR --revision")
		}

		var taskDefinitions map[string]string

		if deployRevision {
			taskDefinitions = make(map[string]string)
			taskDefinitions[cmd.ServiceName] = cmd.AWSSession.GetTaskDefinitionArn(cmd.ServiceName, revision)
		} else {
			if !shell.IsGitInstalled() {
				fmt.Println("The program 'git' is currently not installed.")
				os.Exit(1)
			}
			if !shell.IsDockerInstalled() {
				fmt.Println("The program 'docker' is currently not installed.")
				os.Exit(1)
			}

			image := generateDockerImage(cmd, tagOrBranch, rebuild)
			taskDefinitions = updateImageOfTaskDefinitions(cmd, image)
		}

		doDeploy(cmd, taskDefinitions, wait)

		return nil
	}

	cmd.AddCommand(cobraCmd)
}

func generateDockerImage(cmd *Command, tagOrBranch string, rebuild bool) string {
	fmt.Printf("After build the image we'll deploy to '%s' environment. Type CTRL+C to abort\n", cmd.Environment.ClusterName)
	time.Sleep(5 * time.Second)
	fmt.Println("")

	tempDir := shell.CreateTempDir(cmd.Service.Name + "-" + tagOrBranch)
	defer shell.RemoveTempDir(tempDir)

	shell.CloneRepository(tempDir, cmd.Service.Respository, tagOrBranch)

	if !shell.IsValidTag(tempDir, tagOrBranch) {
		// It's a branch
		hash := shell.GetLastCommitHash(tempDir)
		if len(tagOrBranch) > 9 {
			tagOrBranch = tagOrBranch[:9]
		}

		tagOrBranch += "-" + hash

		fmt.Println("Branch detected! Your docker image will be tagged as:", tagOrBranch)
	}

	shell.DockerBuild(tempDir, cmd.Service.Name, tagOrBranch, rebuild)
	fmt.Println("")

	return cmd.AWSSession.PushImageToAws(cmd.Service.Name, tagOrBranch)
}

func updateImageOfTaskDefinitions(cmd *Command, image string) map[string]string {
	services := cmd.AWSSession.ListTaskDefinitionStartedWith(cmd.ServiceName)
	if len(services) == 0 {
		fmt.Printf("No task-definition started with '%s'.\n", cmd.ServiceName)
		os.Exit(1)
	}

	taskDefinitions := make(map[string]string)

	for _, service := range services {
		taskDefinition := cmd.AWSSession.UpdateTaskDefinition(service, 0, map[string]string{
			"image": image,
		})

		taskDefinitions[service] = taskDefinition
	}

	return taskDefinitions
}

func doDeploy(cmd *Command, taskDefinitions map[string]string, wait bool) {
	servicesToMonitor := make([]string, 0, len(taskDefinitions))

	for service, taskDefinition := range taskDefinitions {
		cmd.AWSSession.Deploy(service, taskDefinition)
		servicesToMonitor = append(servicesToMonitor, service)
	}

	if wait && len(servicesToMonitor) > 0 {
		cmd.AWSSession.WaitUntilServicesStable(servicesToMonitor)
	}
}
