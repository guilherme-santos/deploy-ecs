package cobra

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/user"
	"strings"

	deploy "github.com/guilherme-santos/deploy-ecs"
	"github.com/spf13/cobra"
)

func NewConfigEnvironmentsCommand(rootCmd *Command, cmd *cobra.Command) {
	cobraCmd := &cobra.Command{
		Use:   "environments",
		Short: "Manage your environments",
	}

	cobraCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all environments",
		Run: func(cobraCmd *cobra.Command, args []string) {
			if len(rootCmd.Config.Environments) == 0 {
				fmt.Println("No environments found")
				return
			}
			fmt.Println("List of environments:")

			for _, env := range rootCmd.Config.Environments {
				var (
					bastion   string
					isDefault string
				)

				if env.HasBastion() {
					bastion = "yes"
				} else {
					bastion = "no"
				}
				if strings.EqualFold(rootCmd.Config.DefaultEnvironment, env.ClusterName) {
					isDefault = "(default)"
				}

				fmt.Printf("  - cluster name: %s, region: %s, bastion: %s %s\n", env.ClusterName, env.Region, bastion, isDefault)
			}
		},
	})

	cobraCmd.AddCommand(&cobra.Command{
		Use:   "set-default <name>",
		Short: "Set specific envinroment as default",
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("command needs an argument: <name>")
			}

			if rootCmd.Config.GetEnvironment(args[0]) == nil {
				rootCmd.SilenceUsage = true

				listCmd := cobraCmd.Parent().CommandPath() + " list"
				return fmt.Errorf("Environment '%s' was not found, to list type:\n  - %s", args[0], listCmd)
			}

			rootCmd.Config.DefaultEnvironment = args[0]
			rootCmd.SaveConfig()

			fmt.Printf("Set environment '%s' as default\n", args[0])
			return nil
		},
	})

	cobraCmd.AddCommand(&cobra.Command{
		Use:   "add",
		Short: "Add new environment",
		Run: func(cobraCmd *cobra.Command, args []string) {
			scanner := bufio.NewScanner(os.Stdin)

			env := deploy.Environment{}

			fmt.Println("Hi, let's create a new environment...")

			env.ClusterName = askString(scanner, "AWS Cluster Name", "")
			env.Region = askString(scanner, "AWS Region", "")

			if askBool(scanner, "Do you have a bastion machine or you can access directly your ECS machines? (y/n)", nil) {
				fmt.Println("I need some information to access your bastion machine...")
				env.Bastion.Host = askString(scanner, "Bastion Host", "")
				env.Bastion.Port = askString(scanner, "Bastion Port", "22")

				var username string
				currentUser, err := user.Current()
				if err == nil {
					username = currentUser.Username
				}

				env.Bastion.User = askString(scanner, "Bastion User", username)
				env.Bastion.KeyPair = askString(scanner, "Bastion KeyPair", "id_rsa")
			}

			fmt.Println("Now, some information to access ec2 machine...")

			env.ECSHost.User = askString(scanner, "ECS User", "ec2-user")
			env.ECSHost.KeyPair = askString(scanner, "ECS KeyPair", "id_rsa")

			if len(rootCmd.Config.Environments) == 0 {
				rootCmd.Config.DefaultEnvironment = env.ClusterName
			}
			rootCmd.Config.Environments = append(rootCmd.Config.Environments, env)

			rootCmd.SaveConfig()

			fmt.Printf("Environment '%s' was created!\n", env.ClusterName)
		},
	})

	cobraCmd.AddCommand(&cobra.Command{
		Use:   "edit <name>",
		Short: "Edit a specific environment",
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("command needs an argument: <name>")
			}

			env := rootCmd.Config.GetEnvironment(args[0])
			if env == nil {
				rootCmd.SilenceUsage = true

				listCmd := cobraCmd.Parent().CommandPath() + " list"
				return fmt.Errorf("Environment '%s' was not found, to list type:\n  - %s", args[0], listCmd)
			}

			scanner := bufio.NewScanner(os.Stdin)

			fmt.Printf("Hey, let's edit '%s' environment...\n", args[0])

			env.ClusterName = askString(scanner, "AWS Cluster Name", env.ClusterName)
			env.Region = askString(scanner, "AWS Region", env.Region)

			hasBastion := env.HasBastion()
			if askBool(scanner, "Do you have a bastion machine or you can access directly your ECS machines? (y/n)", &hasBastion) {
				fmt.Println("Updating bastion information, it'll be used to access your bastion machine...")
				env.Bastion.Host = askString(scanner, "Bastion Host", env.Bastion.Host)
				env.Bastion.Port = askString(scanner, "Bastion Port", env.Bastion.Port)
				env.Bastion.User = askString(scanner, "Bastion User", env.Bastion.User)
				env.Bastion.KeyPair = askString(scanner, "Bastion KeyPair", env.Bastion.KeyPair)
			}

			fmt.Println("Now, some information to access ec2 machine...")

			env.ECSHost.User = askString(scanner, "ECS User", env.ECSHost.User)
			env.ECSHost.KeyPair = askString(scanner, "ECS KeyPair", env.ECSHost.KeyPair)

			rootCmd.SaveConfig()

			fmt.Printf("Environment '%s' was updated!\n", env.ClusterName)
			return nil
		},
	})

	cobraCmd.AddCommand(&cobra.Command{
		Use:   "del <name>",
		Short: "Delete a specific envinroment",
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("command needs an argument: <name>")
			}

			env := rootCmd.Config.GetEnvironment(args[0])
			if env == nil {
				rootCmd.SilenceUsage = true

				listCmd := cobraCmd.Parent().CommandPath() + " list"
				return fmt.Errorf("Environment '%s' was not found, to list type:\n  - %s", args[0], listCmd)
			}

			environments := make([]deploy.Environment, 0, len(rootCmd.Config.Environments)-1)
			for _, environment := range rootCmd.Config.Environments {
				if environment == *env {
					continue
				}

				environments = append(environments, environment)
			}

			rootCmd.Config.Environments = environments
			if len(rootCmd.Config.Environments) == 1 {
				rootCmd.Config.DefaultEnvironment = env.ClusterName
			} else if len(rootCmd.Config.Environments) == 0 {
				rootCmd.Config.DefaultEnvironment = ""
			}

			rootCmd.SaveConfig()

			return nil
		},
	})

	cmd.AddCommand(cobraCmd)
}

func printQuestion(question, defaultValue string) {
	fmt.Print(question)
	if !strings.EqualFold("", defaultValue) {
		fmt.Printf(" [%s]", defaultValue)
	}
	fmt.Print(": ")
}

func askString(scanner *bufio.Scanner, question, defaultValue string) string {
	printQuestion(question, defaultValue)

	var (
		result          string
		hasDefaultValue bool
	)

	if !strings.EqualFold("", defaultValue) {
		hasDefaultValue = true
	}

	for scanner.Scan() {
		result = scanner.Text()

		if !strings.EqualFold("", result) {
			return result
		}

		if hasDefaultValue {
			return defaultValue
		}

		printQuestion(question, "")
	}

	return ""
}

func askBool(scanner *bufio.Scanner, question string, defaultValue *bool) bool {
	var (
		result          string
		hasDefaultValue bool
	)

	hasDefaultValue = defaultValue != nil
	if defaultValue == nil {
		printQuestion(question, "")
	} else if *defaultValue == true {
		printQuestion(question, "y")
	} else {
		printQuestion(question, "n")
	}

	for scanner.Scan() {
		result = scanner.Text()

		if strings.EqualFold("y", result) || strings.EqualFold("yes", result) {
			return true
		}

		if strings.EqualFold("n", result) || strings.EqualFold("no", result) {
			return false
		}

		if hasDefaultValue {
			return *defaultValue
		}

		printQuestion(question, "")
	}

	return false
}
