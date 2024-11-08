package workflows

import (
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
}
type AnalyzeOutput struct {
	Records int
}

func CodeAnalyzer(ctx workflow.Context, input AnalyzeInput) (AnalyzeOutput, error) {
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
				Repository: input.Repository,
				Key:        e.Key,
				Embedding:  e.Embedding,
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

	return AnalyzeOutput{
		Records: records,
	}, nil
}
