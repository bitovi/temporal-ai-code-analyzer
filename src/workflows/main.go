package workflows

import (
	"os"
	"time"

	"bitovi.com/code-analyzer/src/activities/git"
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
	var result git.CloneRepositoryOutput
	workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, defaultActivityOptions),
		git.CloneRepository,
		git.CloneRepositoryInput{
			Repository:       input.Repository,
			StorageDirectory: os.Getenv("STORAGE_DIRECTORY"),
		},
	).Get(ctx, &result)

	return AnalyzeOutput{}, nil
}
