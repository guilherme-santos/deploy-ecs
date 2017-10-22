package aws

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecs"
)

func checkErr(method string, err error) {
	if err != nil {
		fmt.Printf("Cannot call %s: %s\n", method, err)
		os.Exit(1)
	}
}

func getRevisionFromTaskDefinition(arn string) string {
	return arn[strings.LastIndex(arn, ":")+1:]
}

func formatUptime(d time.Duration) string {
	var result string

	seconds := d.Seconds()
	hours := int64(seconds / 3600)
	seconds -= float64(hours * 3600)
	minutes := int64(seconds / 60)
	seconds -= float64(minutes * 60)

	if hours > 0 {
		result += fmt.Sprintf("%dh", hours)
	}
	if minutes > 0 {
		result += fmt.Sprintf("%dm", minutes)
	}
	if hours == 0 && seconds > 0 {
		result += fmt.Sprintf("%.0fs", seconds)
	}

	return result
}

func DescribeRepository(sess client.ConfigProvider, service string) *ecr.Repository {
	fmt.Printf("# Looking for AWS repository of '%s'...\n", service)

	svc := ecr.New(sess)

	resp, err := svc.DescribeRepositories(&ecr.DescribeRepositoriesInput{
		RepositoryNames: []*string{
			aws.String(service),
		},
	})
	checkErr("DescribeRepositories", err)

	if len(resp.Repositories) == 0 {
		fmt.Println("No AWS repositoriy was found to service:", service)
		os.Exit(1)
	}

	return resp.Repositories[0]
}

func RegisterTaskDefinition(client client.ConfigProvider, service string, containerDefinitions []*ecs.ContainerDefinition) *ecs.TaskDefinition {
	svc := ecs.New(client)

	params := &ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: containerDefinitions,
		Family:               aws.String(service),
	}

	resp, err := svc.RegisterTaskDefinition(params)
	checkErr("RegisterTaskDefinition", err)

	fmt.Printf("New revision to '%s' was created, number: %d\n", service, *resp.TaskDefinition.Revision)

	return resp.TaskDefinition
}

func ListTaskDefinitions(client client.ConfigProvider, service string, limit int64) []string {
	fmt.Printf("# Listing task definitions of '%s'...\n", service)

	svc := ecs.New(client)

	params := &ecs.ListTaskDefinitionsInput{
		FamilyPrefix: aws.String(service),
		Sort:         aws.String("DESC"),
	}
	if limit > 0 {
		params.MaxResults = aws.Int64(limit)
	}

	resp, err := svc.ListTaskDefinitions(params)
	checkErr("ListTaskDefinitions", err)

	arns := make([]string, len(resp.TaskDefinitionArns))
	for k, arn := range resp.TaskDefinitionArns {
		arns[k] = *arn
	}

	return arns
}

func DescribeTasks(client client.ConfigProvider, cluster string, taskIDs []string) []*ecs.Task {
	svc := ecs.New(client)

	params := &ecs.DescribeTasksInput{
		Cluster: aws.String(cluster),
		Tasks: func(tasks []string) []*string {
			tasksIDs := make([]*string, len(tasks))
			for k, v := range tasks {
				tasksIDs[k] = aws.String(v)
			}
			return tasksIDs
		}(taskIDs),
	}

	resp, err := svc.DescribeTasks(params)
	checkErr("DescribeTasks", err)

	return resp.Tasks
}

func DescribeTasksByService(client client.ConfigProvider, cluster, service string, showAll bool) []*ecs.Task {
	tasks := ListRunningTasks(client, cluster, service)
	if showAll {
		stoppedTasks := ListStoppedTasks(client, cluster, service)
		tasks = append(tasks, stoppedTasks...)
	}

	if len(tasks) == 0 {
		return nil
	}

	return DescribeTasks(client, cluster, tasks)
}

func DescribeTaskDefinition(client client.ConfigProvider, service string, revision int64) *ecs.TaskDefinition {
	svc := ecs.New(client)

	taskDefinitionName := service
	if revision > 0 {
		taskDefinitionName += fmt.Sprintf(":%d", revision)
	}

	params := &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(taskDefinitionName),
	}

	resp, err := svc.DescribeTaskDefinition(params)
	checkErr("DescribeTaskDefinition", err)

	return resp.TaskDefinition
}

func ListRunningTasks(client client.ConfigProvider, cluster, service string) []string {
	svc := ecs.New(client)

	params := &ecs.ListTasksInput{
		Cluster:       aws.String(cluster),
		ServiceName:   aws.String(service),
		DesiredStatus: aws.String("RUNNING"),
	}

	resp, err := svc.ListTasks(params)
	checkErr("ListTasks", err)

	tasks := make([]string, len(resp.TaskArns))
	for k, taskArn := range resp.TaskArns {
		tasks[k] = *taskArn
	}

	return tasks
}

func ListStoppedTasks(client client.ConfigProvider, cluster, service string) []string {
	svc := ecs.New(client)

	params := &ecs.ListTasksInput{
		Cluster:       aws.String(cluster),
		ServiceName:   aws.String(service),
		DesiredStatus: aws.String("STOPPED"),
		MaxResults:    aws.Int64(1),
	}

	resp, err := svc.ListTasks(params)
	checkErr("ListTasks", err)

	tasks := make([]string, len(resp.TaskArns))
	for k, taskArn := range resp.TaskArns {
		tasks[k] = *taskArn
	}

	return tasks
}

func DescribeContainerInstances(client client.ConfigProvider, cluster, containerInstanceArn string) (*ec2.Instance, error) {
	ecsSvc := ecs.New(client)

	params := &ecs.DescribeContainerInstancesInput{
		Cluster: aws.String(cluster),
		ContainerInstances: []*string{
			aws.String(containerInstanceArn),
		},
	}

	resp, err := ecsSvc.DescribeContainerInstances(params)
	checkErr("DescribeContainerInstances", err)

	if len(resp.ContainerInstances) == 0 {
		fmt.Println("Cannot find any instance to", containerInstanceArn)
		os.Exit(1)
	}

	for _, containerInstance := range resp.ContainerInstances {
		ec2Svc := ec2.New(client)

		params := &ec2.DescribeInstancesInput{
			InstanceIds: []*string{
				containerInstance.Ec2InstanceId,
			},
		}

		resp, err := ec2Svc.DescribeInstances(params)
		if err != nil {
			return nil, err
		}

		if len(resp.Reservations) == 0 {
			return nil, errors.New("no instaces was found in 'reservation' field")
		}
		if len(resp.Reservations[0].Instances) == 0 {
			return nil, errors.New("no instaces was found")
		}

		return resp.Reservations[0].Instances[0], nil
	}

	return nil, nil
}
