package aws

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
)

func (sess *AWSSession) GetTaskDefinition(service string, revision int64) {
	fmt.Printf("# Getting task definition of '%s'", service)
	if revision != 0 {
		fmt.Printf(" revision[%d]", revision)
	}
	fmt.Println(":")

	taskDefinition := DescribeTaskDefinition(sess.Client, service, revision)
	fmt.Println(taskDefinition.String())
}

func (sess *AWSSession) GetTaskDefinitionArn(service string, revision int64) string {
	taskDefinition := DescribeTaskDefinition(sess.Client, service, revision)
	return *taskDefinition.TaskDefinitionArn
}

func (sess *AWSSession) UpdateTaskDefinition(service string, revision int64, changes map[string]string) string {
	taskDefinition := DescribeTaskDefinition(sess.Client, service, revision)

	var hasChanged bool

	for k, containerDefinition := range taskDefinition.ContainerDefinitions {
		for name, value := range changes {
			switch name {
			case "cmd":
				fallthrough
			case "command":
				if len(containerDefinition.Command) == 1 {
					if !strings.EqualFold(*containerDefinition.Command[0], value) {
						taskDefinition.ContainerDefinitions[k].SetCommand([]*string{aws.String(value)})
						hasChanged = true
					}
				}
			case "entrypoint":
				if len(containerDefinition.EntryPoint) == 1 {
					if !strings.EqualFold(*containerDefinition.EntryPoint[0], value) {
						taskDefinition.ContainerDefinitions[k].SetEntryPoint([]*string{aws.String(value)})
						hasChanged = true
					}
				}
			case "image":
				if !strings.EqualFold(*containerDefinition.Image, value) {
					taskDefinition.ContainerDefinitions[k].SetImage(value)
					hasChanged = true
				}
			case "cpu":
				cpu, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					fmt.Println("Value to be set on CPU property is not a valid integer:", err)
					os.Exit(1)
				}

				if containerDefinition.Cpu == nil || cpu != *containerDefinition.Cpu {
					taskDefinition.ContainerDefinitions[k].SetCpu(cpu)
					hasChanged = true
				}
			case "memory":
				memory, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					fmt.Println("Value to be set on Memory property is not a valid integer:", err)
					os.Exit(1)
				}

				if containerDefinition.Memory == nil || memory != *containerDefinition.Memory {
					taskDefinition.ContainerDefinitions[k].SetMemory(memory)
					hasChanged = true
				}
			case "memory-reservation":
				memoryReservation, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					fmt.Println("Value to be set on MemoryReservation property is not a valid integer:", err)
					os.Exit(1)
				}

				if containerDefinition.MemoryReservation == nil || memoryReservation != *containerDefinition.MemoryReservation {
					taskDefinition.ContainerDefinitions[k].SetMemoryReservation(memoryReservation)
					hasChanged = true
				}
			default:
				fmt.Printf("Setting attribute[%s] was not implemented or is unknown", name)
				os.Exit(1)
			}
		}
	}

	if !hasChanged {
		fmt.Println("Nothing to update, current task definition:", *taskDefinition.Revision)
		return *taskDefinition.TaskDefinitionArn
	}

	taskDefinition = RegisterTaskDefinition(sess.Client, service, taskDefinition.ContainerDefinitions)
	return *taskDefinition.TaskDefinitionArn
}

func (sess *AWSSession) ListTaskDefinitionStartedWith(startedBy string) []string {
	svc := ecs.New(sess.Client)

	params := &ecs.ListTaskDefinitionFamiliesInput{
		FamilyPrefix: aws.String(startedBy),
		Status:       aws.String("ACTIVE"),
	}

	resp, err := svc.ListTaskDefinitionFamilies(params)
	checkErr("ListTaskDefinitionFamilies", err)

	services := make([]string, 0, len(resp.Families))
	for _, service := range resp.Families {
		services = append(services, *service)
	}

	return services
}

func (sess *AWSSession) ListRevisions(service string) {
	arns := ListTaskDefinitions(sess.Client, service, 10)
	if len(arns) == 0 {
		fmt.Println("No revision was found to this service!")
		return
	}

	fmt.Println("REVISION   DOCKER IMAGE")
	for _, arn := range arns {
		rev := arn[strings.LastIndex(arn, ":")+1:]

		fmt.Printf("%-8s   ", rev)

		svc := ecs.New(sess.Client)
		params := &ecs.DescribeTaskDefinitionInput{
			TaskDefinition: aws.String(arn),
		}
		resp, err := svc.DescribeTaskDefinition(params)
		checkErr("DescribeTaskDefinition", err)

		fmt.Println(*resp.TaskDefinition.ContainerDefinitions[0].Image)
	}
}
