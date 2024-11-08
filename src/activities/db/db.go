package db

import (
	"context"
	"fmt"
	"os"

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

type InsertEmbeddingInput struct {
	Repository string
	Key        string
	Embedding  []float32
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
