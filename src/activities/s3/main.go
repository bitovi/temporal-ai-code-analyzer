package s3

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type AWSConfigInput struct {
	Id       string
	Secret   string
	Token    string
	Endpoint string
	Region   string
}

func getClient(endpoint *string, region *string, credentials *credentials.Credentials) (*s3.S3, error) {
	sess, err := session.NewSession(&aws.Config{
		Endpoint:         endpoint,
		Region:           region,
		Credentials:      credentials,
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	return s3.New(sess), nil
}

type CreateBucketInput struct {
	AWSConfigInput
	Bucket string
}

func CreateBucket(input CreateBucketInput) error {
	client, err := getClient(
		aws.String(input.Endpoint),
		aws.String(input.Region),
		credentials.NewStaticCredentials(input.Id, input.Secret, input.Token),
	)
	if err != nil {
		return fmt.Errorf("error getting S3 Client %w", err)
	}
	_, err = client.CreateBucket(&s3.CreateBucketInput{Bucket: &input.Bucket})
	if err != nil {
		return fmt.Errorf("error creating S3 Bucket %w", err)

	}
	return nil
}

type DeleteBucketInput struct {
	AWSConfigInput
	Bucket string
}

func DeleteBucket(input DeleteBucketInput) error {
	client, err := getClient(
		aws.String(input.Endpoint),
		aws.String(input.Region),
		credentials.NewStaticCredentials(input.Id, input.Secret, input.Token),
	)
	if err != nil {
		return fmt.Errorf("error getting S3 Client %w", err)
	}

	_, err = client.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: aws.String(input.Bucket),
	})
	if err != nil {
		return fmt.Errorf("error deleting S3 Bucket %w", err)
	}

	return nil
}
