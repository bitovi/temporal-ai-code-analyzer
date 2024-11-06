package workflows

import (
	"os"
	"time"

	"bitovi.com/code-analyzer/src/activities/git"
	"bitovi.com/code-analyzer/src/activities/s3"
	"bitovi.com/code-analyzer/src/utils"
	"go.temporal.io/sdk/workflow"
)

var defaultActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 1 * time.Minute,
}

type AnalyzeInput struct {
	Repository string
}
type AnalyzeOutput struct {
}

func CodeAnalyzer(ctx workflow.Context, input AnalyzeInput) (AnalyzeOutput, error) {
	awsConfigInput := s3.AWSConfigInput{
		Endpoint: os.Getenv("AWS_CONFIG_ENDPOINT"),
		Region:   os.Getenv("AWS_CONFIG_REGION"),
		Id:       os.Getenv("AWS_CONFIG_CREDENTIALS_ID"),
		Secret:   os.Getenv("AWS_CONFIG_CREDENTIALS_KEY"),
		Token:    os.Getenv("AWS_CONFIG_CREDENTIALS_TOKEN"),
	}

	workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, defaultActivityOptions),
		s3.CreateBucket,
		s3.CreateBucketInput{
			AWSConfigInput: awsConfigInput,
			Bucket:         utils.CleanRepository(input.Repository),
		},
	).Get(ctx, nil)

	var result git.CloneRepositoryOutput
	workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, defaultActivityOptions),
		git.CloneRepository,
		input.Repository,
	).Get(ctx, &result)

	workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, defaultActivityOptions),
		s3.DeleteBucket,
		s3.DeleteBucketInput{
			AWSConfigInput: awsConfigInput,
			Bucket:         utils.CleanRepository(input.Repository),
		},
	).Get(ctx, nil)

	return AnalyzeOutput{}, nil
}
