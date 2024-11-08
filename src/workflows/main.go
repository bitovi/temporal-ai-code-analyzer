package workflows

import (
	"fmt"
	"time"

	"bitovi.com/code-analyzer/src/activities/db"
	"bitovi.com/code-analyzer/src/activities/git"
	"bitovi.com/code-analyzer/src/activities/llm"
	"bitovi.com/code-analyzer/src/activities/s3"
	"bitovi.com/code-analyzer/src/utils"
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
	options := workflow.ChildWorkflowOptions{
		WorkflowID: "process-documents-" + utils.CleanRepository(input.Repository),
	}
	cctx := workflow.WithChildOptions(ctx, options)

	var processDocumentsResult ProcessDocumentsOutput
	err := workflow.ExecuteChildWorkflow(cctx, ProcessDocuments, ProcessDocumentsInput{
		Repository: input.Repository,
	}).Get(cctx, &processDocumentsResult)
	if err != nil {
		return AnalyzeOutput{}, fmt.Errorf("failed to process documents for %s: %w", input.Repository, err)
	}

	var invokePromptResult ProcessDocumentsOutput
	err = workflow.ExecuteChildWorkflow(cctx, InvokePrompt, InvokePromptInput(
		input,
	)).Get(cctx, &invokePromptResult)
	if err != nil {
		return AnalyzeOutput{}, fmt.Errorf("failed to invoke prompt %s for %s: %w", input.Query, input.Repository, err)
	}

	return AnalyzeOutput{
		Response: "",
	}, nil
}

type ProcessDocumentsInput struct {
	Repository string
}
type ProcessDocumentsOutput struct {
	Records int
}

func ProcessDocuments(ctx workflow.Context, input ProcessDocumentsInput) (ProcessDocumentsOutput, error) {
	bucketName := utils.CleanRepository(input.Repository)

	workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, defaultActivityOptions),
		s3.CreateBucket,
		s3.CreateBucketInput{
			Bucket: bucketName,
		},
	).Get(ctx, nil)

	var archiveResult git.ArchiveRepositoryOutput
	workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, defaultActivityOptions),
		git.ArchiveRepository,
		git.ArchiveRepositoryInput{
			Repository: input.Repository,
			Bucket:     bucketName,
		},
	).Get(ctx, &archiveResult)

	embeddingsFutures := make([]workflow.Future, len(archiveResult.Keys))
	for i, key := range archiveResult.Keys {
		f := workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, defaultActivityOptions),
			llm.GetEmbeddingData,
			llm.GetEmbeddingDataInput{
				Bucket: bucketName,
				Key:    key,
			},
		)
		embeddingsFutures[i] = f
	}

	var embeddings []llm.GetEmbeddingDataOutput
	for _, f := range embeddingsFutures {
		var embeddingResult llm.GetEmbeddingDataOutput
		f.Get(ctx, &embeddingResult)
		embeddings = append(embeddings, embeddingResult)
	}

	insertFutures := make([]workflow.Future, len(embeddings))
	for i, e := range embeddings {
		f := workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, defaultActivityOptions),
			db.InsertEmbedding,
			db.InsertEmbeddingInput{
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
				Bucket: bucketName,
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
			Bucket: bucketName,
		},
	).Get(ctx, nil)

	return ProcessDocumentsOutput{
		Records: records,
	}, nil
}

type InvokePromptInput struct {
	Repository string
	Query      string
}
type InvokePromptOutput struct {
	// Response string
	RelatedDocuments []string
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

	return InvokePromptOutput{
		// Response: "",
		RelatedDocuments: relatedDocuments.Keys,
	}, nil
}
