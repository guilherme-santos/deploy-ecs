package shell

import (
	"fmt"
	"os"
	"strings"
)

func IsGitInstalled() bool {
	_, err := RunCommand("git --version")
	return err == nil
}

func IsValidTag(dir, tag string) bool {
	cmd := fmt.Sprintf("git -C %s tag -l %s", dir, tag)
	resp, err := RunCommand(cmd)
	if err != nil {
		fmt.Printf("Cannot validate if '%s' is a valid tag:\n- %s\n", tag, err)
		os.Exit(1)
	}

	return !strings.EqualFold("", resp)
}

func GetGitOrigin() string {
	origin, err := RunCommand("git remote get-url origin")
	if err != nil {
		fmt.Println("Cannot get URL from remote origin:\n-", err)
		os.Exit(1)
	}

	return origin
}

func GetLastCommitHash(dir string) string {
	cmd := fmt.Sprintf("git -C %s rev-parse --short HEAD", dir)
	hash, err := RunCommand(cmd)
	if err != nil {
		fmt.Println("Cannot get last commit hash:\n-", err)
		os.Exit(1)
	}

	return hash
}

func CloneRepository(dir, url, tag string) {
	fmt.Println("Cloning repository...")

	cmd := fmt.Sprintf("git clone -q -b %s --single-branch --depth 1 %s %s", tag, url, dir)
	_, err := RunCommand(cmd)
	if err != nil {
		fmt.Println("Cannot clone project:\n-", err)
		os.Exit(1)
	}
}
