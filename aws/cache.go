package aws

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	deploy "github.com/guilherme-santos/deploy-ecs"
)

type (
	CacheEntry struct {
		RemoteHost           string
		ContainerInstanceArn string
		TaskArn              string
		Containers           []deploy.Container
	}
)

func (entry CacheEntry) HasRemoteHost() bool {
	return !strings.EqualFold("", entry.RemoteHost)
}

func (entry CacheEntry) HasContainer() bool {
	if len(entry.Containers) == 0 {
		return false
	}

	for _, container := range entry.Containers {
		if container.DockerID == "" {
			return false
		}
	}

	return true
}

func GetTaskFromCache(taskID string) CacheEntry {
	filename := filepath.Join(os.TempDir(), "deploy-ecs", taskID+".json")

	var entry CacheEntry

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return entry
	}

	json.Unmarshal(content, &entry)
	return entry
}

func SaveTaskToCache(taskID string, entry CacheEntry) {
	tempDir := filepath.Join(os.TempDir(), "deploy-ecs")
	os.MkdirAll(tempDir, 0755)

	filename := filepath.Join(tempDir, taskID+".json")
	content, _ := json.Marshal(entry)
	ioutil.WriteFile(filename, content, 0644)
}
