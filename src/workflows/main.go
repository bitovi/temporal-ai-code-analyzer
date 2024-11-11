package workflows

import (
	"fmt"
	"time"

	"bitovi.com/code-analyzer/src/activities/db"
	"bitovi.com/code-analyzer/src/activities/git"
	"bitovi.com/code-analyzer/src/activities/llm"
	"bitovi.com/code-analyzer/src/activities/s3"
	"bitovi.com/code-analyzer/src/utils"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var defaultActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 1 * time.Minute,
}

type AnalyzeInput struct {
	Repository string
	Query      string
}
type AnalyzeOutput struct {
	Response string
}

func AnalyzeCode(ctx workflow.Context, input AnalyzeInput) (AnalyzeOutput, error) {
	bucketName := utils.CleanRepository(input.Repository)
	cctx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID: "process-documents-" + utils.CleanRepository(input.Repository),
	})
	var processDocumentsResult ProcessDocumentsOutput
	err := workflow.ExecuteChildWorkflow(cctx, ProcessDocuments, ProcessDocumentsInput{
		Repository: input.Repository,
		BucketName: bucketName,
	}).Get(cctx, &processDocumentsResult)
	if err != nil {
		return AnalyzeOutput{}, fmt.Errorf("failed to process documents for %s: %w", input.Repository, err)
	}

	cctx = workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID: "invoke-prompt-" + utils.CleanRepository(input.Repository),
	})
	var invokePromptResult InvokePromptOutput
	err = workflow.ExecuteChildWorkflow(cctx, InvokePrompt, InvokePromptInput{
		Repository: input.Repository,
		BucketName: bucketName,
		Query:      input.Query,
	}).Get(cctx, &invokePromptResult)
	if err != nil {
		return AnalyzeOutput{}, fmt.Errorf("failed to invoke prompt %s for %s: %w", input.Query, input.Repository, err)
	}

	return AnalyzeOutput{
		Response: invokePromptResult.Response,
	}, nil
}

type ProcessDocumentsInput struct {
	Repository string
	BucketName string
}
type ProcessDocumentsOutput struct {
	Records int
}

func ProcessDocuments(ctx workflow.Context, input ProcessDocumentsInput) (ProcessDocumentsOutput, error) {
	workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, defaultActivityOptions),
		s3.CreateBucket,
		s3.CreateBucketInput{
			Bucket: input.BucketName,
		},
	).Get(ctx, nil)

	var archiveResult git.ArchiveRepositoryOutput
	workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, defaultActivityOptions),
		git.ArchiveRepository,
		git.ArchiveRepositoryInput{
			Repository: input.Repository,
			Bucket:     input.BucketName,
		},
	).Get(ctx, &archiveResult)

	embeddingsFutures := make([]workflow.Future, len(archiveResult.Keys))
	for i, key := range archiveResult.Keys {
		f := workflow.ExecuteActivity(
			workflow.WithRetryPolicy(
				workflow.WithActivityOptions(ctx, defaultActivityOptions),
				temporal.RetryPolicy{
					InitialInterval: time.Second * 8,
					MaximumAttempts: 5,
				},
			),
			llm.GetEmbeddingData,
			llm.GetEmbeddingDataInput{
				Bucket: input.BucketName,
				Key:    key,
			},
		)
		embeddingsFutures[i] = f
	}

	var embeddings []llm.GetEmbeddingDataOutput
	for _, f := range embeddingsFutures {
		var embeddingResult llm.GetEmbeddingDataOutput
		f.Get(ctx, &embeddingResult)
		if len(embeddingResult.Embedding) > 0 {
			embeddings = append(embeddings, embeddingResult)
		}
	}

	insertFutures := make([]workflow.Future, len(embeddings))
	for i, e := range embeddings {
		f := workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, defaultActivityOptions),
			db.InsertEmbedding,
			db.InsertEmbeddingInput{
				Bucket: input.BucketName,
				EmbeddingRecord: db.EmbeddingRecord{
					Repository: input.Repository,
					Key:        e.Key,
					Embedding:  e.Embedding,
				},
			},
		)
		insertFutures[i] = f
	}
	for _, f := range insertFutures {
		f.Get(ctx, nil)
	}
	records := len(embeddings)

	deleteObjectFutures := make([]workflow.Future, len(archiveResult.Keys))
	for i, key := range archiveResult.Keys {
		f := workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, defaultActivityOptions),
			s3.DeleteObject,
			s3.DeleteObjectInput{
				Bucket: input.BucketName,
				Key:    key,
			},
		)
		deleteObjectFutures[i] = f
	}
	for _, f := range deleteObjectFutures {
		f.Get(ctx, nil)
	}

	workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, defaultActivityOptions),
		s3.DeleteBucket,
		s3.DeleteBucketInput{
			Bucket: input.BucketName,
		},
	).Get(ctx, nil)

	return ProcessDocumentsOutput{
		Records: records,
	}, nil
}

type InvokePromptInput struct {
	Repository string
	BucketName string
	Query      string
}
type InvokePromptOutput struct {
	Response string
}

func InvokePrompt(ctx workflow.Context, input InvokePromptInput) (InvokePromptOutput, error) {
	var relatedDocuments db.GetRelatedDocumentsOutput
	workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, defaultActivityOptions),
		db.GetRelatedDocuments,
		db.GetRelatedDocumentsInput{
			Repository: input.Repository,
			Query:      input.Query,
		},
	).Get(ctx, &relatedDocuments)

	var response string
	workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, defaultActivityOptions),
		llm.InvokePrompt,
		llm.InvokePromptInput{
			Query:            input.Query,
			RelatedDocuments: relatedDocuments.Contents,
		},
	).Get(ctx, &response)

	return InvokePromptOutput{
		Response: response,
	}, nil
}
