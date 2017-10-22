package aws

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
)

func inArray(element string, array []string) bool {
	if len(array) == 0 {
		return true
	}

	for _, v := range array {
		if strings.EqualFold(element, v) {
			return true
		}
	}

	return false
}

func readEnvvarFile(reader *bufio.Reader) map[string]string {
	envvars := make(map[string]string)

	for {
		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}

			fmt.Println("Cannot read from stdin:", err)
			os.Exit(1)
		}

		input = strings.TrimSpace(input)
		if strings.HasPrefix(input, "#") {
			continue
		}

		parts := strings.SplitN(input, "=", 2)

		var value string
		if len(parts) > 1 {
			value = parts[1]
		}

		envvars[parts[0]] = value
	}

	return envvars
}

func readEnvvarFileAsJson(reader *bufio.Reader) map[string]string {
	envvars := make(map[string]string)

	var content string

	for {
		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}

			fmt.Println("Cannot read from stdin:", err)
			os.Exit(1)
		}

		input = strings.TrimSpace(input)
		if strings.HasPrefix(input, "#") {
			continue
		}

		content += input + "\n"
	}

	var jsonFile map[string]interface{}

	err := json.Unmarshal([]byte(content), &jsonFile)
	if err != nil {
		fmt.Println("Cannot parse stdin as json:", err)
		os.Exit(1)
	}

	for k, v := range jsonFile {
		envvars[k] = fmt.Sprintf("%v", v)
	}

	return envvars
}

func (sess *AWSSession) GetEnvvar(service string, revision int64, gets []string, formatJson bool) {
	taskDefinition := DescribeTaskDefinition(sess.Client, service, revision)

	for _, def := range taskDefinition.ContainerDefinitions {
		envvars := make(map[string]string)
		for _, envvar := range def.Environment {
			if inArray(*envvar.Name, gets) {
				envvars[*envvar.Name] = *envvar.Value
			}
		}

		msg := fmt.Sprintf("# Getting env-vars of '%s'", service)
		if revision != 0 {
			msg += fmt.Sprintf(" revision[%d]", revision)
		}
		msg += ":"

		if formatJson {
			msg += "\n"
			j, _ := json.MarshalIndent(&envvars, "", "   ")
			msg += string(j)
		} else {
			keys := make([]string, 0, len(envvars))

			for k := range envvars {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, key := range keys {
				msg += fmt.Sprintf("\n%s=%s", key, envvars[key])
			}
		}

		fmt.Println(msg)
		return
	}
}

func (sess *AWSSession) DiffEnvvarFromFile(service string, revision int64, file *os.File, formatJson bool) (map[string]string, map[string]struct{}) {
	taskDefinition := DescribeTaskDefinition(sess.Client, service, revision)

	var changes map[string]string

	// Read envvar from file
	reader := bufio.NewReader(file)
	if formatJson {
		changes = readEnvvarFileAsJson(reader)
	} else {
		changes = readEnvvarFile(reader)
	}

	unsets := make(map[string]struct{}, 0)

	for _, def := range taskDefinition.ContainerDefinitions {
		for _, envvar := range def.Environment {
			if _, ok := changes[*envvar.Name]; !ok {
				if _, ok := unsets[*envvar.Name]; !ok {
					unsets[*envvar.Name] = struct{}{}
				}
			}
		}
	}

	return changes, unsets
}

func (sess *AWSSession) UpdateEnvvar(service string, revision int64, changes map[string]string, unsets map[string]struct{}) int64 {
	taskDefinition := DescribeTaskDefinition(sess.Client, service, revision)

	var hasChanged bool

	for k, def := range taskDefinition.ContainerDefinitions {
		envvars := make([]*ecs.KeyValuePair, 0, len(def.Environment))

		currentChanges := make(map[string]string, len(changes))
		for k, v := range changes {
			currentChanges[k] = v
		}

		for _, envvar := range def.Environment {
			newValue, ok := currentChanges[*envvar.Name]
			if ok {
				if !strings.EqualFold(*envvar.Value, newValue) {
					envvar.SetValue(newValue)
					hasChanged = true
				}

				delete(currentChanges, *envvar.Name)
			}

			if _, ok = unsets[*envvar.Name]; ok {
				hasChanged = true
				continue
			}

			envvars = append(envvars, envvar)
		}

		if len(currentChanges) > 0 {
			// Some fields that didn't exist need to be add
			hasChanged = true
		}

		for name, value := range currentChanges {
			envvars = append(envvars, &ecs.KeyValuePair{
				Name:  aws.String(name),
				Value: aws.String(value),
			})
		}

		taskDefinition.ContainerDefinitions[k].Environment = envvars
	}

	if !hasChanged {
		fmt.Println("Nothing to update in this task definition")
		return 0
	}

	taskDefinition = RegisterTaskDefinition(sess.Client, service, taskDefinition.ContainerDefinitions)
	return *taskDefinition.Revision
}
