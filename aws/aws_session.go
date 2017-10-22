package aws

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	deploy "github.com/guilherme-santos/deploy-ecs"
)

type AWSSession struct {
	Client      client.ConfigProvider
	Environment *deploy.Environment
}

func NewAWSSession(env *deploy.Environment) *AWSSession {
	awsSession := &AWSSession{
		Environment: env,
	}

	var err error

	awsSession.Client, err = session.NewSession(&aws.Config{
		Region: aws.String(env.Region),
	})
	if err != nil {
		fmt.Println("Cannot get AWS Session:", err)
		os.Exit(1)
	}

	return awsSession
}

func (sess *AWSSession) GetAuthorizationToken(registryID string) (user string, token string, endpoint string) {
	svc := ecr.New(sess.Client)

	params := &ecr.GetAuthorizationTokenInput{
		RegistryIds: []*string{
			aws.String(registryID),
		},
	}

	resp, err := svc.GetAuthorizationToken(params)
	checkErr("GetAuthorizationToken", err)

	tokenEncoded, _ := base64.StdEncoding.DecodeString(*resp.AuthorizationData[0].AuthorizationToken)
	userToken := strings.Split(string(tokenEncoded), ":")

	user = userToken[0]
	token = userToken[1]
	endpoint = *resp.AuthorizationData[0].ProxyEndpoint

	return
}
