package workflows

import (
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
	var embeddingsCount int
	workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, defaultActivityOptions),
		db.GetEmbeddingCount,
		git.ArchiveRepositoryInput{
			Repository: input.Repository,
		},
	).Get(ctx, &embeddingsCount)

	fetchEmbeddingCtx := workflow.WithRetryPolicy(
		workflow.WithActivityOptions(ctx, defaultActivityOptions),
		temporal.RetryPolicy{
			InitialInterval: time.Second * 8,
			MaximumAttempts: 5,
		},
	)

	if embeddingsCount == 0 {
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
				fetchEmbeddingCtx,
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
					Bucket: bucketName,
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
	}

	var embeddingsForQuery []float32
	workflow.ExecuteActivity(
		fetchEmbeddingCtx,
		llm.FetchEmbedding,
		input.Query,
	).Get(ctx, &embeddingsForQuery)

	var relatedDocuments db.GetRelatedDocumentsOutput
	workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, defaultActivityOptions),
		db.GetRelatedDocuments,
		db.GetRelatedDocumentsInput{
			Repository: input.Repository,
			Embedding:  embeddingsForQuery,
			Limit:      5,
		},
	).Get(ctx, &relatedDocuments)

	var relatedContent = make([]string, len(relatedDocuments.Records))
	for i, record := range relatedDocuments.Records {
		relatedContent[i] = record.Content
	}

	var response string
	workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, defaultActivityOptions),
		llm.InvokePrompt,
		llm.InvokePromptInput{
			Query:          input.Query,
			RelatedContent: relatedContent,
		},
	).Get(ctx, &response)

	return AnalyzeOutput{
		Response: response,
	}, nil
}
