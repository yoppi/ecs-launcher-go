package launcher

import (
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
)

type AWSConfig struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
}

func NewSession(c *AWSConfig) *session.Session {
	creds, err := NewCredentials(c)
	if err != nil {
		panic(err)
	}

	config := aws.NewConfig()
	config.WithRegion(c.Region).WithCredentials(creds)

	return session.New(config)
}

func NewCredentials(c *AWSConfig) (*credentials.Credentials, error) {
	var (
		creds *credentials.Credentials
		err   error
	)

	creds, err = newStaticCredentials(c)
	if err != nil {
		creds, err = newEnvCredentials()
		if err != nil {
			creds, err = newRoleCredentials()
		}
	}

	return creds, err
}

func newStaticCredentials(c *AWSConfig) (*credentials.Credentials, error) {
	creds := credentials.NewStaticCredentials(c.AccessKeyID, c.SecretAccessKey, "")
	if _, err := creds.Get(); err != nil {
		return nil, err
	}
	return creds, nil
}

func newEnvCredentials() (*credentials.Credentials, error) {
	creds := credentials.NewEnvCredentials()
	_, err := creds.Get()
	return creds, err
}

func newRoleCredentials() (*credentials.Credentials, error) {
	metadata := ec2metadata.New(session.New(), &aws.Config{
		HTTPClient: http.DefaultClient,
	})
	creds := credentials.NewCredentials(&ec2rolecreds.EC2RoleProvider{
		Client: metadata,
	})
	_, err := creds.Get()
	return creds, err
}
