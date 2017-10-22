package cobra

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func NewConfigGitHubCommand(rootCmd *Command, cmd *cobra.Command) {
	cobraCmd := &cobra.Command{
		Use:   "github",
		Short: "Set some config to comunicate with GitHub",
	}

	cobraCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all configuration related with github",
		Run: func(cobraCmd *cobra.Command, args []string) {
			fmt.Print("token=", rootCmd.Config.GitHub.Token, "\n")
			fmt.Print("repository=", rootCmd.Config.GitHub.DefaultRepository, "\n")
		},
	})

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get configuration from specific key",
	}
	getCmd.AddCommand(&cobra.Command{
		Use:   "token",
		Short: "Get GitHub token",
		Run: func(cobraCmd *cobra.Command, args []string) {
			fmt.Println(rootCmd.Config.GitHub.Token)
		},
	})
	getCmd.AddCommand(&cobra.Command{
		Use:   "repository",
		Short: "Get default GitHub respository",
		Run: func(cobraCmd *cobra.Command, args []string) {
			fmt.Println(rootCmd.Config.GitHub.DefaultRepository)
		},
	})
	cobraCmd.AddCommand(getCmd)

	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Set configuration from specific key",
	}
	setCmd.AddCommand(&cobra.Command{
		Use:   "token <value>",
		Short: "Set GitHub token",
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("command needs an argument: <value>")
			}

			rootCmd.Config.GitHub.Token = args[0]
			rootCmd.SaveConfig()

			return nil
		},
	})
	setCmd.AddCommand(&cobra.Command{
		Use:   "repository <value>",
		Short: "Set default GitHub respository",
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("command needs an argument: <value>")
			}

			rootCmd.Config.GitHub.DefaultRepository = args[0]
			if !strings.HasSuffix(rootCmd.Config.GitHub.DefaultRepository, "/") {
				rootCmd.Config.GitHub.DefaultRepository += "/"
			}

			rootCmd.SaveConfig()

			return nil
		},
	})
	cobraCmd.AddCommand(setCmd)

	cmd.AddCommand(cobraCmd)
}
