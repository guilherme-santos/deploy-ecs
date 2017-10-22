package deploy

import "strings"

type (
	Config struct {
		DefaultEnvironment string
		Environments       []Environment
		GitHub             GitHubConfig
	}

	Environment struct {
		ClusterName string
		Region      string
		Bastion     ServerConfig
		ECSHost     ServerConfig
	}

	ServerConfig struct {
		Host    string
		Port    string
		User    string
		KeyPair string
	}

	GitHubConfig struct {
		Token             string
		DefaultRepository string
	}

	Container struct {
		DockerID string
		Name     string
	}
)

func (env *Environment) HasBastion() bool {
	return !strings.EqualFold("", env.Bastion.Host)
}

func (config *Config) GetAvailableEnvironments() []string {
	environments := make([]string, len(config.Environments))

	for k, env := range config.Environments {
		environments[k] = env.ClusterName
	}

	return environments
}

func (config *Config) GetEnvironment(environment string) *Environment {
	for k, env := range config.Environments {
		if strings.EqualFold(environment, env.ClusterName) {
			return &config.Environments[k]
		}
	}

	return nil
}
