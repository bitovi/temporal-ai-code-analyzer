package llm

import (
	"fmt"
	"os"
	"strings"

	"bitovi.com/code-analyzer/src/activities/s3"
	"bitovi.com/code-analyzer/src/utils/http"
)

var OpenAPIKey string = os.Getenv("OPENAI_API_KEY")

type GetEmbeddingDataInput struct {
	Bucket string
	Key    string
}

type GetEmbeddingDataOutput struct {
	Key       string
	Embedding []float32
}

func GetEmbeddingData(input GetEmbeddingDataInput) (GetEmbeddingDataOutput, error) {
	body, err := s3.GetObject(
		input.Bucket,
		input.Key,
	)
	if err != nil {
		return GetEmbeddingDataOutput{}, fmt.Errorf("error fetching %s from S3 bucket: %w", input.Key, err)
	}

	result, err := FetchEmbedding(string(body))
	if err != nil {
		if strings.Contains(err.Error(), "maximum context length") {
			return GetEmbeddingDataOutput{}, nil
		}
		return GetEmbeddingDataOutput{}, fmt.Errorf("error getting embeddings data for %s: %w", input.Key, err)
	}

	return GetEmbeddingDataOutput{
		Key:       input.Key,
		Embedding: result,
	}, nil
}

type FetchEmbeddingsApiRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

type EmbeddingResponse struct {
	Data []struct {
		Embedding []float32
	}
}

func FetchEmbedding(text string) ([]float32, error) {
	url := "https://api.openai.com/v1/embeddings"

	data := &FetchEmbeddingsApiRequest{
		Input: text,
		Model: "text-embedding-3-small",
	}

	var result EmbeddingResponse
	result, err := http.PostRequest(url, data, result, OpenAPIKey)
	if err != nil {
		return []float32{}, err
	}

	return result.Data[0].Embedding, nil
}

type ChatCompletion struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message Message `json:"message"`
}

type Message struct {
	Content string `json:"content"`
}
type InvokeApiRequest struct {
	Model    string             `json:"model"`
	Messages []InvokeApiMessage `json:"messages"`
}

type InvokeApiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func FetchCompletion(input [][]string) (ChatCompletion, error) {
	url := "https://api.openai.com/v1/chat/completions"

	messages := make([]InvokeApiMessage, len(input))
	for i, p := range input {
		messages[i] = InvokeApiMessage{
			Role:    p[0],
			Content: p[1],
		}
	}

	data := InvokeApiRequest{
		Model:    "gpt-3.5-turbo",
		Messages: messages,
	}

	var result ChatCompletion
	result, err := http.PostRequest(url, data, result, OpenAPIKey)
	if err != nil {
		return ChatCompletion{}, err
	}

	return result, nil
}

type InvokePromptInput struct {
	Query            string
	RelatedDocuments []string
}

func InvokePrompt(input InvokePromptInput) (string, error) {
	prompt := [][]string{
		{"system", "You are a friendly, helpful software assistant. Your goal is to help users understand the code within a Git repository."},
		{"system", "You should respond in short paragraphs, using Markdown formatting for any blocks of code, separated with two newlines to keep your responses easily readable."},
		{"system", "Whenever possible, use code examples derived from the documentation provided."},
		{"system", "Here are the files from the Git repository that are relevant to the user's question: " + strings.Join(input.RelatedDocuments, "\n\n")},
		{"user", input.Query},
	}

	invokeResponse, _ := FetchCompletion(prompt)
	return invokeResponse.Choices[0].Message.Content, nil
}
