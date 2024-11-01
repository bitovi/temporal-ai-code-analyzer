package workflows

import (
	"fmt"
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
	err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, defaultActivityOptions),
		git.CloneRepository,
		input.Repository,
	).Get(ctx, &result)

	if err != nil {
		fmt.Printf("error cloning repository %s", err)
		return AnalyzeOutput{}, err
	}

	return AnalyzeOutput{}, nil
}
