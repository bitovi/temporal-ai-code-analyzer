package db

import (
	"context"
	"fmt"
	"os"

	"bitovi.com/code-analyzer/src/activities/llm"
	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"
)

var DatabaseURL string = os.Getenv("DATABASE_CONNECTION_STRING")

func getConnection(ctx context.Context) (*pgx.Conn, error) {
	conn, err := pgx.Connect(ctx, DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}
	return conn, nil
}

type EmbeddingRecord struct {
	Repository string
	Key        string
	Embedding  []float32
}
type InsertEmbeddingInput struct {
	EmbeddingRecord
}

func InsertEmbedding(ctx context.Context, input InsertEmbeddingInput) error {
	conn, err := getConnection(ctx)
	if err != nil {
		return err
	}

	_, err = conn.Exec(
		ctx,
		"INSERT INTO documents (repository, key, embedding) VALUES ($1, $2, $3)",
		input.Repository,
		input.Key,
		pgvector.NewVector(input.Embedding),
	)
	return err
}

type GetRelatedDocumentsInput struct {
	Repository string
	Query      string
	Limit      int
}
type GetRelatedDocumentsOutput struct {
	Keys []string
}

func GetRelatedDocuments(ctx context.Context, input GetRelatedDocumentsInput) (GetRelatedDocumentsOutput, error) {
	embeddingForQuery, err := llm.FetchEmbedding(input.Query)
	if err != nil {
		return GetRelatedDocumentsOutput{}, fmt.Errorf("error getting embeddings data for query %s: %w", input.Query, err)
	}

	conn, err := getConnection(ctx)
	if err != nil {
		return GetRelatedDocumentsOutput{}, err
	}

	limit := input.Limit
	if limit == 0 {
		limit = 5
	}
	query := "SELECT key FROM documents WHERE repository=$1  ORDER BY embedding <=> $2 LIMIT $3"
	rows, err := conn.Query(ctx, query, input.Repository, pgvector.NewVector(embeddingForQuery), limit)
	if err != nil {
		return GetRelatedDocumentsOutput{}, fmt.Errorf("error fetching related documents: %w", err)
	}

	defer rows.Close()

	var keys []string
	for rows.Next() {
		var doc EmbeddingRecord
		err = rows.Scan(&doc.Key)
		if err != nil {
			return GetRelatedDocumentsOutput{}, err
		}
		keys = append(keys, doc.Key)
	}

	if rows.Err() != nil {
		return GetRelatedDocumentsOutput{}, fmt.Errorf("error fetching related documents: %w", err)
	}

	return GetRelatedDocumentsOutput{
		Keys: keys,
	}, nil
}
