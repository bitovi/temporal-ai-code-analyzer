package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"bitovi.com/code-analyzer/src/activities/s3"
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
		Model: "text-embedding-ada-002",
	}

	var result EmbeddingResponse
	result, err := PostRequest(url, data, result, OpenAPIKey)
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
	result, err := PostRequest(url, data, result, OpenAPIKey)
	if err != nil {
		return ChatCompletion{}, err
	}

	return result, nil
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

type InvokePromptInput struct {
	Query                  string
	RelatedDocumentsBucket string
	RelatedDocumentsKeys   []string
}

func InvokePrompt(input InvokePromptInput) (string, error) {
	var relatedDocuments = make([]string, len(input.RelatedDocumentsKeys))
	for i, key := range input.RelatedDocumentsKeys {
		doc, err := s3.GetObject(input.RelatedDocumentsBucket, key)
		if err != nil {
			return "", fmt.Errorf("error getting related document %s from bucket: %w", key, err)
		}
		relatedDocuments[i] = string(doc)
	}

	prompt := [][]string{
		{"system", "You are a friendly, helpful software assistant. Your goal is to help users understand the code within a Git repository."},
		{"system", "You should respond in short paragraphs, using Markdown formatting for any blocks of code, separated with two newlines to keep your responses easily readable."},
		{"system", "Whenever possible, use code examples derived from the documentation provided."},
		// {"system", "Import references must be included where relevant so that the reader can easily figure out how to import the necessary dependencies."},
		// {"system", "Do not use your existing knowledge to determine import references, only use import references as they appear in the relevant documentation."},
		{"system", "Here are the files from the Git repository that are relevant to the user's question: " + strings.Join(relatedDocuments, "\n\n")},
		{"user", input.Query},
	}

	invokeResponse, _ := FetchCompletion(prompt)
	return invokeResponse.Choices[0].Message.Content, nil
}
