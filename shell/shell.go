package shell

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strings"
)

func GetGoCommand(command string) *exec.Cmd {
	params := strings.Split(command, " ")
	if len(params) == 0 {
		return nil
	}

	if len(params) == 1 {
		return exec.Command(params[0])
	}

	startPos := -1
	args := make([]string, 0, len(params))
	for k, v := range params {
		if strings.HasPrefix(v, "\"") {
			startPos = k
			continue
		} else if strings.HasSuffix(v, "\"") && startPos > -1 {
			cmd := strings.Join(params[startPos:k+1], " ")
			cmd = cmd[1 : len(cmd)-1] // Remove quotes
			args = append(args, cmd)
			startPos = -1
			continue
		} else if startPos > -1 {
			continue
		}

		args = append(args, v)
	}

	return exec.Command(args[0], args[1:]...)
}

func RunCommand(command string) (string, error) {
	cmd := GetGoCommand(command)
	if cmd == nil {
		return "", errors.New("command is not valid")
	}

	resp, err := cmd.Output()
	if err != nil {
		var errMsg string

		if stdErr, ok := err.(*exec.ExitError); ok {
			errMsg = string(stdErr.Stderr)
			errMsg = strings.TrimSpace(errMsg)
		} else {
			errMsg = err.Error()
		}

		return "", errors.New(errMsg)
	}

	return strings.TrimSpace(string(resp)), nil
}

func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println("Cannot get local IP:", err)
		os.Exit(1)
	}

	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	fmt.Println("No IP address was found")
	os.Exit(1)
	return ""
}

func CreateTempDir(prefix string) string {
	dir, err := ioutil.TempDir("", fmt.Sprintf("aws-ecs-%s-deploy-", prefix))
	if err != nil {
		fmt.Println("Cannot create temporary folder:", err)
		os.Exit(1)
	}

	return dir
}

func RemoveTempDir(dir string) {
	os.RemoveAll(dir)
}
