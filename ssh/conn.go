package ssh

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	deploy "github.com/guilherme-santos/deploy-ecs"
	"golang.org/x/crypto/ssh"
)

func localConnect(server deploy.ServerConfig, verbose bool) (*ssh.Client, error) {
	sshConfig := NewLocalSSHConfig(server)

	if verbose {
		fmt.Printf("Trying to connect to '%s@%s'...", sshConfig.GetUser(), sshConfig.GetURL())
	}

	client, err := ssh.Dial("tcp", sshConfig.GetURL(), sshConfig.GetSSHClientConfig())
	if err != nil {
		if verbose {
			fmt.Println(" FAIL")
		}
		return nil, err
	}

	if verbose {
		fmt.Println(" OK")
	}

	return client, nil
}

func Connect(env *deploy.Environment, remoteHost string, verbose bool) *ssh.Client {
	if !env.HasBastion() {
		client, err := localConnect(deploy.ServerConfig{
			Host:    remoteHost,
			User:    env.ECSHost.User,
			KeyPair: env.ECSHost.KeyPair,
		}, verbose)

		if err != nil {
			fmt.Println("Error connecting to remote server:\n-", err)
			os.Exit(1)
		}

		return client
	}

	client, err := localConnect(env.Bastion, verbose)
	if err != nil {
		fmt.Println("Error connecting to remote server:\n-", err)
		os.Exit(1)
	}

	// Try to connect to remoteHost over bastion
	sshConfig := NewRemoteSSHConfig(client, deploy.ServerConfig{
		Host:    remoteHost,
		User:    env.ECSHost.User,
		KeyPair: env.ECSHost.KeyPair,
	})

	if verbose {
		fmt.Printf("Trying to connect to '%s@%s'...", sshConfig.GetUser(), sshConfig.GetURL())
	}

	netConn, err := client.Dial("tcp", sshConfig.GetURL())
	if err != nil {
		if verbose {
			fmt.Println(" FAIL")
		}

		fmt.Println("Error connecting to remote server:\n-", err)
		os.Exit(1)
	}

	conn, chans, reqs, err := ssh.NewClientConn(netConn, sshConfig.GetHost(), sshConfig.GetSSHClientConfig())
	if err != nil {
		if verbose {
			fmt.Println(" FAIL")
		}

		fmt.Println("Error connecting to remote server:\n-", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Println(" OK")
	}

	return ssh.NewClient(conn, chans, reqs)
}

func RunCommand(session *ssh.Session, command string) error {
	var (
		stderr        bytes.Buffer
		captureStderr bool
	)

	if session.Stderr == nil {
		session.Stderr = &stderr
		captureStderr = true
	}

	err := session.Run(command)
	if err != nil {
		var errMsg string
		if captureStderr {
			errMsg = stderr.String()
			errMsg = strings.TrimSpace(errMsg)
		} else {
			errMsg = err.Error()
		}

		return errors.New(errMsg)
	}

	return nil
}
