package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"bitovi.com/code-analyzer/src/activities/s3"
)

var OpenAPIKey string = os.Getenv("OPENAI_API_KEY")

type GetEmbeddingDataInput struct {
	Bucket string
	Key    string
}

type GetEmbeddingDataOutput struct {
	Embedding []float32
}

type FetchEmbeddingsApiRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}
type EmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func GetEmbeddingData(input GetEmbeddingDataInput) (GetEmbeddingDataOutput, error) {
	url := "https://api.openai.com/v1/embeddings"

	body, err := s3.GetObject(
		input.Bucket,
		input.Key,
	)
	if err != nil {
		return GetEmbeddingDataOutput{}, fmt.Errorf("error fetching %s from S3 bucket: %w", input.Key, err)
	}

	data := &FetchEmbeddingsApiRequest{
		Input: string(body),
		Model: "text-embedding-ada-002",
	}

	var result EmbeddingResponse
	result, err = PostRequest(url, data, result, OpenAPIKey)
	if err != nil {
		return GetEmbeddingDataOutput{}, fmt.Errorf("error getting embeddings data for %s: %w", input.Key, err)
	}

	return GetEmbeddingDataOutput{
		Embedding: result.Data[0].Embedding,
	}, nil
}

func PostRequest[T any](url string, body any, result T, apiKey string) (T, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return result, err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	if err != nil {
		return result, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return result, fmt.Errorf("bad status code: %d - %s", resp.StatusCode, body)
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return result, err
	}

	return result, nil
}
