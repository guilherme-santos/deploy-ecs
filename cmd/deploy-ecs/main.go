package main

import (
	"os"
	"strings"

	"github.com/guilherme-santos/deploy-ecs/cobra"
)

var (
	Version string
	Build   string
)

func main() {
	if strings.EqualFold("", Version) {
		Version = "dev"
	}

	rootCmd := cobra.NewCommand(Version, Build)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
