package cobra

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"strings"

	deploy "github.com/guilherme-santos/deploy-ecs"
	"github.com/spf13/cobra"
	ini "gopkg.in/ini.v1"
)

func NewConfigCommand(cmd *Command) {
	cobraCmd := &cobra.Command{
		Use:   "config",
		Short: "Configure several attributes as environments and access token",
	}

	NewConfigEnvironmentsCommand(cmd, cobraCmd)
	NewConfigGitHubCommand(cmd, cobraCmd)

	cmd.AddCommand(cobraCmd)
}

func getConfigFilename() string {
	user, _ := user.Current()
	return path.Join(user.HomeDir, ".deploy-ecs")
}

func (cmd *Command) LoadConfig() error {
	configFile := getConfigFilename()

	config, err := ini.Load(configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Printf("cannot load '%s': %s\n", configFile, err)
			os.Exit(1)
		}

		config = ini.Empty()
	}

	cmd.Config = &deploy.Config{}

	environmentsSec, err := config.GetSection("environment")
	if err == nil {
		if key, err := environmentsSec.GetKey("default"); err == nil {
			cmd.Config.DefaultEnvironment = key.String()
		}

		for _, sec := range environmentsSec.ChildSections() {
			env := deploy.Environment{
				ClusterName: strings.TrimPrefix(sec.Name(), "environment."),
			}

			if key, err := sec.GetKey("region"); err == nil {
				env.Region = key.String()
			}
			if key, err := sec.GetKey("bastion_host"); err == nil {
				env.Bastion.Host = key.String()

				if key, err := sec.GetKey("bastion_port"); err == nil {
					env.Bastion.Port = key.String()
				}
				if key, err := sec.GetKey("bastion_user"); err == nil {
					env.Bastion.User = key.String()
				}
				if key, err := sec.GetKey("bastion_key_pair"); err == nil {
					env.Bastion.KeyPair = key.String()
				}
			}
			if key, err := sec.GetKey("ecs_user"); err == nil {
				env.ECSHost.User = key.String()
			}
			if key, err := sec.GetKey("ecs_key_pair"); err == nil {
				env.ECSHost.KeyPair = key.String()
			}

			cmd.Config.Environments = append(cmd.Config.Environments, env)
		}
	}

	githubSec, err := config.GetSection("github")
	if err == nil {
		if key, err := githubSec.GetKey("token"); err == nil {
			cmd.Config.GitHub.Token = key.String()
		}
		if key, err := githubSec.GetKey("repository"); err == nil {
			cmd.Config.GitHub.DefaultRepository = key.String()
		}
	}

	return nil
}

func (cmd *Command) SaveConfig() error {
	configFile := getConfigFilename()

	iniConfig := ini.Empty()

	environmentsSec, _ := iniConfig.NewSection("environment")
	environmentsSec.NewKey("default", cmd.Config.DefaultEnvironment)

	for _, env := range cmd.Config.Environments {
		environmentsSec, _ := iniConfig.NewSection("environment." + env.ClusterName)
		environmentsSec.NewKey("region", env.Region)
		if env.HasBastion() {
			environmentsSec.NewKey("bastion_host", env.Bastion.Host)
			environmentsSec.NewKey("bastion_port", env.Bastion.Port)
			environmentsSec.NewKey("bastion_user", env.Bastion.User)
			environmentsSec.NewKey("bastion_key_pair", env.Bastion.KeyPair)
		}
		environmentsSec.NewKey("ecs_user", env.ECSHost.User)
		environmentsSec.NewKey("ecs_key_pair", env.ECSHost.KeyPair)
	}

	githubSec, _ := iniConfig.NewSection("github")
	githubSec.NewKey("token", cmd.Config.GitHub.Token)
	githubSec.NewKey("repository", cmd.Config.GitHub.DefaultRepository)

	err := iniConfig.SaveTo(configFile)
	if err != nil {
		return fmt.Errorf("cannot save '%s': %s", configFile, err)
	}

	return nil
}
