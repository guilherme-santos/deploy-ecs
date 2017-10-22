package ssh

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"

	deploy "github.com/guilherme-santos/deploy-ecs"
)

func DockerLogs(env *deploy.Environment, remoteHost, containerID, tail string, follow bool) {
	client := Connect(env, remoteHost, true)
	defer client.Close()

	sess, err := client.NewSession()
	if err != nil {
		fmt.Println("Cannot get session:", err)
		os.Exit(1)
	}

	defer sess.Close()

	sess.Stdout = os.Stdout
	sess.Stderr = os.Stdout

	command := fmt.Sprintf("docker logs --tail %s %s", tail, containerID)
	if follow {
		command += " -f"
	}

	err = RunCommand(sess, command)
	if err != nil {
		fmt.Printf("Error running \"%s\":\n- %s\n", command, err)
		os.Exit(1)
	}
}

func DockerExec(env *deploy.Environment, remoteHost, containerID, command string) {
	client := Connect(env, remoteHost, true)
	defer client.Close()

	sess, err := client.NewSession()
	if err != nil {
		fmt.Println("Cannot get session:", err)
		os.Exit(1)
	}

	defer sess.Close()

	sess.Stdin = os.Stdin
	sess.Stdout = os.Stdout

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.ECHOCTL:       0,
		ssh.TTY_OP_ISPEED: 115200,
		ssh.TTY_OP_OSPEED: 115200,
	}

	termFD := int(os.Stdin.Fd())

	w, h, _ := terminal.GetSize(termFD)

	termState, _ := terminal.MakeRaw(termFD)
	defer terminal.Restore(termFD, termState)

	sess.RequestPty("xterm-256color", h, w, modes)

	command = fmt.Sprintf("docker exec -it %s %s", containerID, command)

	RunCommand(sess, command)
}

func GetContainers(env *deploy.Environment, remoteHost, taskArn string) ([]deploy.Container, error) {
	client := Connect(env, remoteHost, false)
	defer client.Close()

	sess, err := client.NewSession()
	if err != nil {
		fmt.Println("Cannot get session:", err)
		os.Exit(1)
	}

	defer sess.Close()

	var stdout bytes.Buffer
	sess.Stdout = &stdout

	command := fmt.Sprintf("curl -s 'http://localhost:51678/v1/tasks?taskarn=%s'", taskArn)

	err = RunCommand(sess, command)
	if err != nil {
		fmt.Printf("Error running \"%s\":\n- %s\n", command, err)
		os.Exit(1)
	}

	var agentResp struct {
		Containers []struct {
			DockerId string
			Name     string
		}
	}

	err = json.Unmarshal(stdout.Bytes(), &agentResp)
	if err != nil {
		fmt.Println("Cannot read response from ECS Agent:", err)
		os.Exit(1)
	}

	if len(agentResp.Containers) == 0 {
		return nil, errors.New("cannot find container id to this task-arn")
	}

	containers := make([]deploy.Container, len(agentResp.Containers))

	for k, container := range agentResp.Containers {
		containers[k] = deploy.Container{
			Name:     container.Name,
			DockerID: container.DockerId,
		}
	}

	return containers, nil
}
