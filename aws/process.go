package aws

import (
	"fmt"
	"strings"
	"time"

	"github.com/NeowayLabs/logger"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	deploy "github.com/guilherme-santos/deploy-ecs"
	"github.com/guilherme-santos/deploy-ecs/ssh"
)

func (sess *AWSSession) ListProcess(services []string, showAll bool) {
	tasks := make([]*ecs.Task, 0)

	for _, service := range services {
		serviceTasks := DescribeTasksByService(sess.Client, sess.Environment.ClusterName, service, showAll)
		tasks = append(tasks, serviceTasks...)
	}

	if len(tasks) == 0 {
		fmt.Println("No task was found to this service!")
		return
	}

	fmt.Println("TASK ID                                  REVISION   UPTIME     PUBLIC DNS")

	for _, task := range tasks {
		taskDefinitionArn := *task.TaskDefinitionArn
		taskArn := *task.TaskArn
		taskID := taskArn[strings.LastIndex(taskArn, "/")+1:]
		taskRevision := getRevisionFromTaskDefinition(taskDefinitionArn)
		status := *task.LastStatus

		var (
			uptime        string
			stoppedReason string
		)

		if strings.EqualFold("RUNNING", status) {
			if task.StartedAt != nil {
				startedAt := *task.StartedAt
				uptime = formatUptime(time.Since(startedAt))
			} else {
				uptime = "PENDING"
			}
		} else if strings.EqualFold("STOPPED", status) {
			uptime = status
			stoppedReason = *task.StoppedReason
		} else {
			uptime = status
		}

		entry := GetTaskFromCache(taskID)
		if !entry.HasRemoteHost() {
			instance, err := DescribeContainerInstances(sess.Client, sess.Environment.ClusterName, *task.ContainerInstanceArn)
			if err != nil {
				fmt.Printf("%-38s   %-8s   %-8s   Error: %s\n", taskID, taskRevision, uptime, err)
				continue
			}

			entry.TaskArn = taskArn
			entry.ContainerInstanceArn = *task.ContainerInstanceArn
			entry.RemoteHost = *instance.PublicDnsName
			SaveTaskToCache(taskID, entry)
		}

		fmt.Printf("%-38s   %-8s   %-8s   %s\n", taskID, taskRevision, uptime, entry.RemoteHost)
		if !entry.HasContainer() {
			containers, err := ssh.GetContainers(sess.Environment, entry.RemoteHost, entry.TaskArn)
			if err != nil {
				fmt.Println("  - Error getting containers:", err)
				continue
			}
			if len(containers) == 0 {
				fmt.Println("  - No container was found")
				continue
			}

			entry.Containers = containers
			SaveTaskToCache(taskID, entry)
		}

		fmt.Printf("  CONTAINER NAME                         CONTAINER ID")
		if !strings.EqualFold("", stoppedReason) {
			fmt.Printf("   STOPPED REASON")
		}
		fmt.Println("")

		for _, container := range entry.Containers {
			fmt.Printf("    %-34s   %s", container.Name, container.DockerID[:12])
			fmt.Print("   ", stoppedReason)
			fmt.Println("")
		}
	}
}

func (sess *AWSSession) GetLogs(taskID, nameOrContainerID, tail string, follow bool) {
	entry := GetTaskFromCache(taskID)
	if !entry.HasRemoteHost() {
		tasks := DescribeTasks(sess.Client, sess.Environment.ClusterName, []string{taskID})
		if len(tasks) == 0 {
			logger.Warn("No task was found with task id: %s", taskID)
			return
		}

		task := tasks[0]

		instance, err := DescribeContainerInstances(sess.Client, sess.Environment.ClusterName, *task.ContainerInstanceArn)
		checkErr("DescribeContainerInstances", err)

		entry.TaskArn = *task.TaskArn
		entry.ContainerInstanceArn = *task.ContainerInstanceArn
		entry.RemoteHost = *instance.PublicDnsName
		SaveTaskToCache(taskID, entry)
	}
	if !entry.HasContainer() {
		containers, err := ssh.GetContainers(sess.Environment, entry.RemoteHost, entry.TaskArn)
		if err != nil {
			fmt.Println(err)
			return
		}
		if len(containers) == 0 {
			fmt.Println("No container was found")
			return
		}

		entry.Containers = containers
		SaveTaskToCache(taskID, entry)
	}

	var container deploy.Container

	if strings.EqualFold("", nameOrContainerID) {
		if len(entry.Containers) > 1 {
			fmt.Println("We have more than one container running over this task, inform name or container id")
			return
		}

		container = entry.Containers[0]
	} else {
		for _, c := range entry.Containers {
			if strings.HasPrefix(c.DockerID, nameOrContainerID) || strings.EqualFold(nameOrContainerID, c.Name) {
				container = c
			}
		}
	}

	if strings.EqualFold("", container.DockerID) {
		fmt.Println("No container was found with name or id:", nameOrContainerID)
		return
	}

	ssh.DockerLogs(sess.Environment, entry.RemoteHost, container.DockerID, tail, follow)
}

func (sess *AWSSession) Exec(taskID, nameOrContainerID, command string) {
	entry := GetTaskFromCache(taskID)
	if !entry.HasRemoteHost() {
		tasks := DescribeTasks(sess.Client, sess.Environment.ClusterName, []string{taskID})
		if len(tasks) == 0 {
			logger.Warn("No task was found with task id: %s", taskID)
			return
		}

		task := tasks[0]

		instance, err := DescribeContainerInstances(sess.Client, sess.Environment.ClusterName, *task.ContainerInstanceArn)
		checkErr("DescribeContainerInstances", err)

		entry.TaskArn = *task.TaskArn
		entry.ContainerInstanceArn = *task.ContainerInstanceArn
		entry.RemoteHost = *instance.PublicDnsName
		SaveTaskToCache(taskID, entry)
	}

	if !entry.HasContainer() {
		containers, err := ssh.GetContainers(sess.Environment, entry.RemoteHost, entry.TaskArn)
		if err != nil {
			fmt.Println(err)
			return
		}
		if len(containers) == 0 {
			fmt.Println("No container was found")
			return
		}

		entry.Containers = containers
		SaveTaskToCache(taskID, entry)
	}

	var container deploy.Container

	if strings.EqualFold("", nameOrContainerID) {
		if len(entry.Containers) > 1 {
			fmt.Println("We have more than one container running over this task, inform name or container id")
			return
		}

		container = entry.Containers[0]
	} else {
		for _, c := range entry.Containers {
			if strings.HasPrefix(c.DockerID, nameOrContainerID) || strings.EqualFold(nameOrContainerID, c.Name) {
				container = c
			}
		}
	}

	if strings.EqualFold("", container.DockerID) {
		if len(entry.Containers) > 1 {
			fmt.Println("No container was found with name or id:", nameOrContainerID)
			return
		}

		// In this case nameOrContainerID was not a container ID it's part of command name with some options
		command = nameOrContainerID + " " + command
		container = entry.Containers[0]
	}

	ssh.DockerExec(sess.Environment, entry.RemoteHost, container.DockerID, command)
}

func (sess *AWSSession) Kill(service, taskID string) {
	fmt.Printf("Killing service '%s' on cluster '%s'...\n", service, sess.Environment.ClusterName)

	svc := ecs.New(sess.Client)

	params := &ecs.StopTaskInput{
		Cluster: aws.String(sess.Environment.ClusterName),
		Task:    aws.String(taskID),
		Reason:  aws.String("Killed by user using deploy-ecs"),
	}

	_, err := svc.StopTask(params)
	checkErr("StopTask", err)
}

func (sess *AWSSession) Scale(service string, numberOfTasks int64) {
	fmt.Printf("Scalling service '%s' on cluster '%s' to %d...\n", service, sess.Environment.ClusterName, numberOfTasks)

	svc := ecs.New(sess.Client)

	params := &ecs.UpdateServiceInput{
		Cluster:      aws.String(sess.Environment.ClusterName),
		Service:      aws.String(service),
		DesiredCount: aws.Int64(numberOfTasks),
	}

	_, err := svc.UpdateService(params)
	checkErr("UpdateService", err)
}
