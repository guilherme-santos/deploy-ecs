package aws

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/guilherme-santos/deploy-ecs/shell"
)

func (sess *AWSSession) PushImageToAws(service, tag string) string {
	repository := DescribeRepository(sess.Client, service)
	repositoryURL := *repository.RepositoryUri
	registryID := *repository.RegistryId

	remoteImage := fmt.Sprintf("%s:%s", repositoryURL, tag)

	fmt.Printf("Pushing docker image '%s'...\n", repositoryURL)

	cmd := fmt.Sprintf("docker tag %s:%s %s", service, tag, remoteImage)
	_, err := shell.RunCommand(cmd)
	if err != nil {
		fmt.Println("Cannot tag docker image:", err)
		os.Exit(1)
	}

	user, token, endpoint := sess.GetAuthorizationToken(registryID)

	cmd = fmt.Sprintf("docker login -u %s -p %s %s", user, token, endpoint)
	_, err = shell.RunCommand(cmd)
	if err != nil {
		fmt.Println("Cannot login on AWS respository:", err)
		os.Exit(1)
	}

	fmt.Println("This operation can take several minutes...")

	_, err = shell.RunCommand("docker push " + remoteImage)
	if err != nil {
		fmt.Println("Cannot push docker image to AWS respository:", err)
		os.Exit(1)
	}

	return remoteImage
}

func (sess *AWSSession) Deploy(service, taskDefinition string) {
	revision := getRevisionFromTaskDefinition(taskDefinition)
	fmt.Printf("Updating service '%s' on cluster '%s' to revision[%s]...\n", service, sess.Environment.ClusterName, revision)

	svc := ecs.New(sess.Client)

	params := &ecs.UpdateServiceInput{
		Cluster:        aws.String(sess.Environment.ClusterName),
		Service:        aws.String(service),
		TaskDefinition: aws.String(taskDefinition),
	}

	_, err := svc.UpdateService(params)
	checkErr("UpdateService", err)
}

func (sess *AWSSession) Rollback(service string) {
	arns := ListTaskDefinitions(sess.Client, service, 2)

	if len(arns) < 2 {
		fmt.Println("No old revision was found to rollback")
		os.Exit(1)
	}

	// get one revision before last
	arn := arns[1]
	revision := getRevisionFromTaskDefinition(arn)
	fmt.Printf("Rollback service '%s' on cluster '%s' to revision[%s]...\n", service, sess.Environment.ClusterName, revision)

	sess.Deploy(service, arn)
}

func showProgress(wg *sync.WaitGroup, stop chan bool) {
	defer wg.Done()
	times := 0
	for {
		select {
		case <-stop:
			fmt.Println("")
			return
		default:
			times++
			fmt.Print(".")
			if times == 80 {
				fmt.Println("")
				times = 0
			}

			time.Sleep(1 * time.Second)
		}
	}
}

func (sess *AWSSession) WaitUntilTaskStopped(taskIDs []string) {
	fmt.Println("\nWait until following services are stopped:\n       -", strings.Join(taskIDs, "\n       - "))

	var wg sync.WaitGroup
	wg.Add(1)

	stop := make(chan bool)

	go showProgress(&wg, stop)

	svc := ecs.New(sess.Client)

	params := &ecs.DescribeTasksInput{
		Cluster: aws.String(sess.Environment.ClusterName),
		Tasks: func(taskIDs []string) []*string {
			awsTasks := make([]*string, len(taskIDs))
			for k, v := range taskIDs {
				awsTasks[k] = aws.String(v)
			}
			return awsTasks
		}(taskIDs),
	}

	err := svc.WaitUntilTasksStopped(params)

	stop <- true
	wg.Wait()

	checkErr("WaitUntilTasksStopped", err)
}

func (sess *AWSSession) WaitUntilServicesStable(services []string) {
	fmt.Println("\nWait until following services are stable:\n       -", strings.Join(services, "\n       - "))

	var wg sync.WaitGroup
	wg.Add(1)

	stop := make(chan bool)

	go showProgress(&wg, stop)

	svc := ecs.New(sess.Client)

	params := &ecs.DescribeServicesInput{
		Cluster: aws.String(sess.Environment.ClusterName),
		Services: func(services []string) []*string {
			awsServices := make([]*string, len(services))
			for k, v := range services {
				awsServices[k] = aws.String(v)
			}
			return awsServices
		}(services),
	}

	err := svc.WaitUntilServicesStable(params)

	stop <- true
	wg.Wait()

	checkErr("WaitUntilServicesStable", err)
}
