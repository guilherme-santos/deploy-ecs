package shell

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"strings"
)

func IsDockerInstalled() bool {
	_, err := RunCommand("docker --version")
	return err == nil
}

func DockerBuild(dir, service, tag string, rebuild bool) {
	// Verify if project has a Makefile
	makefile := path.Join(dir, "Makefile")
	if _, err := os.Stat(makefile); err == nil {
		// This project has a makefile, let's call default target
		fmt.Printf("Calling Makefile with target 'pre-docker-build'...\n")

		var stderr bytes.Buffer

		goCmd := GetGoCommand(fmt.Sprintf("/bin/sh -c \"cd %s; make pre-docker-build\"", dir))
		goCmd.Stdout = os.Stdout
		goCmd.Stderr = &stderr

		err := goCmd.Run()
		if err != nil {
			errMsg := stderr.String()
			errMsg = strings.TrimSpace(errMsg)
			fmt.Println("Error executing 'make pre-docker-image':\n-", errMsg)
		}
	}

	fmt.Printf("Building docker image '%s:%s'...\n", service, tag)

	cmd := fmt.Sprintf("docker build -t %s:%s", service, tag)
	if rebuild {
		cmd += " --no-cache"
	}
	cmd += " " + dir

	var stderr bytes.Buffer

	goCmd := GetGoCommand(cmd)
	goCmd.Stdout = os.Stdout
	goCmd.Stderr = &stderr

	err := goCmd.Run()
	if err != nil {
		errMsg := stderr.String()
		errMsg = strings.TrimSpace(errMsg)
		fmt.Println("Cannot build docker image:\n-", errMsg)
		os.Exit(1)
	}
}
