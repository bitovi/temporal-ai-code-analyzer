package workflows

import (
	"time"

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
	Embeddings [][]float32
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

	var embeddings [][]float32
	for _, f := range embeddingsFutures {
		var embeddingResult llm.GetEmbeddingDataOutput
		f.Get(ctx, &embeddingResult)
		embeddings = append(embeddings, embeddingResult.Embedding)
	}

	deleteFutures := make([]workflow.Future, len(archiveResult.Keys))
	for i, key := range archiveResult.Keys {
		f := workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, defaultActivityOptions),
			s3.DeleteObject,
			s3.DeleteObjectInput{
				Bucket: bucketName,
				Key:    key,
			},
		)
		deleteFutures[i] = f
	}
	for _, f := range deleteFutures {
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
		Embeddings: embeddings,
	}, nil
}
