package ssh

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"strings"

	deploy "github.com/guilherme-santos/deploy-ecs"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

var DefaultSSHPort = "22"

type SSHConfig struct {
	server      deploy.ServerConfig
	client      *ssh.Client
	currentUser string
	homeDir     string
}

func NewLocalSSHConfig(server deploy.ServerConfig) *SSHConfig {
	currentUser, err := user.Current()
	if err != nil {
		fmt.Println("Cannot get current user:", err)
		os.Exit(1)
	}

	return &SSHConfig{
		server:      server,
		currentUser: currentUser.Username,
		homeDir:     currentUser.HomeDir,
	}
}

func NewRemoteSSHConfig(client *ssh.Client, server deploy.ServerConfig) *SSHConfig {
	config := &SSHConfig{
		server:      server,
		client:      client,
		currentUser: server.User,
	}

	config.homeDir, _ = config.runCommand("echo $HOME")

	return config
}

func (config *SSHConfig) runCommand(command string) (string, error) {
	sess, err := config.client.NewSession()
	if err != nil {
		fmt.Println("Cannot get session:", err)
		os.Exit(1)
	}

	defer sess.Close()

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)
	sess.Stdout = &stdout
	sess.Stderr = &stderr

	err = sess.Run(command)
	if err != nil {
		errMsg := stderr.String()
		errMsg = strings.TrimSpace(errMsg)
		return "", errors.New(errMsg)
	}

	return strings.TrimSpace(stdout.String()), nil
}

func (config *SSHConfig) GetUser() string {
	if !strings.EqualFold("", config.server.User) {
		return config.server.User
	}

	return config.currentUser
}

func (config *SSHConfig) GetHost() string {
	if !strings.EqualFold("", config.server.Host) {
		return config.server.Host
	}

	return ""
}

func (config *SSHConfig) GetPort() string {
	if !strings.EqualFold("", config.server.Port) {
		return config.server.Port
	}

	return DefaultSSHPort
}

func (config *SSHConfig) GetURL() string {
	return config.GetHost() + ":" + config.GetPort()
}

func (config *SSHConfig) GetAuthMethods() []ssh.AuthMethod {
	authMethods := make([]ssh.AuthMethod, 0, 5)

	if config.client == nil {
		// It's a local connection
		sshAgentSock := os.Getenv("SSH_AUTH_SOCK")
		if !strings.EqualFold("", sshAgentSock) {
			agentClient, err := net.Dial("unix", sshAgentSock)
			if err != nil {
				fmt.Println("Cannot connect to ssh-agent:", err)
				os.Exit(1)
			}

			signers := agent.NewClient(agentClient).Signers
			authMethods = append(authMethods, ssh.PublicKeysCallback(signers))
		}
	}

	keyPairs := make([]string, 0, 3)
	if !strings.EqualFold("", config.server.KeyPair) {
		keyPairs = append(keyPairs, config.server.KeyPair)
	}
	keyPairs = append(keyPairs, "id_rsa", "id_dsa")

	for _, keyPair := range keyPairs {
		if !strings.HasPrefix(keyPair, "/") {
			keyPair = config.homeDir + "/.ssh/" + keyPair
		}

		var (
			pem []byte
			err error
		)

		if config.client == nil {
			pem, err = ioutil.ReadFile(keyPair)
		} else {
			content, _ := config.runCommand("cat " + keyPair)
			contentBuf := bytes.NewBufferString(content)
			pem, err = ioutil.ReadAll(contentBuf)
		}

		if err != nil {
			continue
		}

		key, err := ssh.ParsePrivateKey(pem)
		if err != nil {
			continue
		}

		authMethods = append(authMethods, ssh.PublicKeys(key))
	}

	return authMethods
}

func (config *SSHConfig) GetSSHClientConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User:            config.GetUser(),
		Auth:            config.GetAuthMethods(),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}
