package s3

import (
	"bytes"
	"fmt"
	"os"

	"bitovi.com/code-analyzer/src/utils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	AWSEndpoint         = os.Getenv("AWS_CONFIG_ENDPOINT")
	AWSRegion           = os.Getenv("AWS_CONFIG_REGION")
	AWSCredentialsId    = os.Getenv("AWS_CONFIG_CREDENTIALS_ID")
	AWSCredentialsKey   = os.Getenv("AWS_CONFIG_CREDENTIALS_KEY")
	AWSCredentialsToken = os.Getenv("AWS_CONFIG_CREDENTIALS_TOKEN")
)

func getClient() (*s3.S3, error) {
	sess, err := session.NewSession(&aws.Config{
		Endpoint:         &AWSEndpoint,
		Region:           &AWSRegion,
		Credentials:      credentials.NewStaticCredentials(AWSCredentialsId, AWSCredentialsKey, AWSCredentialsToken),
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	return s3.New(sess), nil
}

type CreateBucketInput struct {
	Bucket string
}

func CreateBucket(input CreateBucketInput) error {
	if utils.ChaosExists("aws") {
		return fmt.Errorf("error creating S3 bucket -- AWS is totally down")
	}
	client, err := getClient()
	if err != nil {
		return fmt.Errorf("error getting S3 Client %w", err)
	}
	_, err = client.CreateBucket(&s3.CreateBucketInput{Bucket: aws.String(input.Bucket)})
	if err != nil {
		return fmt.Errorf("error creating S3 Bucket %w", err)

	}
	return nil
}

type DeleteBucketInput struct {
	Bucket string
}

func DeleteBucket(input DeleteBucketInput) error {
	client, err := getClient()
	if err != nil {
		return fmt.Errorf("error getting S3 Client %w", err)
	}

	if utils.ChaosExists("aws") {
		return fmt.Errorf("error deleting S3 bucket -- AWS is totally down")
	}
	_, err = client.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: aws.String(input.Bucket),
	})
	if err != nil {
		return fmt.Errorf("error deleting S3 Bucket %w", err)
	}

	return nil
}

func PutObject(bucket string, key string, body []byte) error {
	if utils.ChaosExists("aws") {
		return fmt.Errorf("error with S3.Put -- AWS is totally down")
	}
	client, err := getClient()
	if err != nil {
		return fmt.Errorf("error getting S3 Client %w", err)
	}
	_, err = client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(body),
	})
	if err != nil {
		return err
	}
	return nil
}

func GetObject(bucket string, key string) ([]byte, error) {
	client, err := getClient()
	if err != nil {
		return nil, fmt.Errorf("error getting S3 Client %w", err)
	}
	resp, err := client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("error from s3Client.GetS3Object: %w", err)
	}
	defer resp.Body.Close()

	body := new(bytes.Buffer)
	_, err = body.ReadFrom(resp.Body)
	return body.Bytes(), err
}

type DeleteObjectInput struct {
	Bucket string
	Key    string
}

func DeleteObject(input DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	s3Client, _ := getClient()

	if utils.ChaosExists("aws") {
		return nil, fmt.Errorf("error deleting S3 bucket -- AWS is totally down")
	}
	_, _ = s3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(input.Bucket),
		Key:    aws.String(input.Key),
	})

	return nil, nil
}
