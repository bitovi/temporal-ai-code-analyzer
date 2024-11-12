package db

import (
	"context"
	"fmt"
	"os"

	"bitovi.com/code-analyzer/src/activities/s3"
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
	Content    string
	Embedding  []float32
}
type InsertEmbeddingInput struct {
	EmbeddingRecord
	Bucket string
}

func InsertEmbedding(ctx context.Context, input InsertEmbeddingInput) error {
	conn, err := getConnection(ctx)
	if err != nil {
		return err
	}

	content, err := s3.GetObject(input.Bucket, input.Key)
	if err != nil {
		return fmt.Errorf("error getting related document %s from bucket: %w", input.Key, err)
	}

	_, err = conn.Exec(
		ctx,
		"INSERT INTO documents (repository, key, content, embedding) VALUES ($1, $2, $3, $4)",
		input.Repository,
		input.Key,
		content,
		pgvector.NewVector(input.Embedding),
	)
	return err
}

type GetEmbeddingCountInput struct {
	Repository string
}

func GetEmbeddingCount(ctx context.Context, input GetEmbeddingCountInput) (int, error) {
	conn, err := getConnection(ctx)
	if err != nil {
		return 0, err
	}

	var count int
	query := "SELECT COUNT(*) FROM documents WHERE repository=$1"
	err = conn.QueryRow(ctx, query, input.Repository).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("error fetching document count: %w", err)
	}

	return count, nil
}

type GetRelatedDocumentsInput struct {
	Repository string
	Embedding  []float32
	Limit      int
}
type GetRelatedDocumentsOutput struct {
	Records []EmbeddingRecord
}

func GetRelatedDocuments(ctx context.Context, input GetRelatedDocumentsInput) (GetRelatedDocumentsOutput, error) {
	conn, err := getConnection(ctx)
	if err != nil {
		return GetRelatedDocumentsOutput{}, err
	}

	query := "SELECT key, content FROM documents WHERE repository=$1 ORDER BY embedding <=> $2 LIMIT $3"
	rows, err := conn.Query(ctx, query, input.Repository, pgvector.NewVector(input.Embedding), input.Limit)
	if err != nil {
		return GetRelatedDocumentsOutput{}, fmt.Errorf("error fetching related documents: %w", err)
	}
	defer rows.Close()

	var relatedRecords []EmbeddingRecord
	for rows.Next() {
		var doc EmbeddingRecord
		err = rows.Scan(&doc.Key, &doc.Content)
		if err != nil {
			return GetRelatedDocumentsOutput{}, err
		}
		relatedRecords = append(relatedRecords, doc)
	}

	if rows.Err() != nil {
		return GetRelatedDocumentsOutput{}, fmt.Errorf("error fetching related documents: %w", err)
	}

	return GetRelatedDocumentsOutput{
		Records: relatedRecords,
	}, nil
}
