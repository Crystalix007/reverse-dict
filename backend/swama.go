package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const Model = "mlx-community/Qwen3-8B-4bit"

// SwamaEmbeddingResponse is the response from the swama embedding API.
type SwamaEmbeddingResponse struct {
	Model string                       `json:"model"`
	Usage SwamaResponseUsage           `json:"usage"`
	Data  []SwamaEmbeddingResponseData `json:"data"`
}

// SwamaEmbeddingRequest is the request to the swama embedding API.
type SwamaEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// ResponseUsage is the usage from the swama embedding API.
type SwamaResponseUsage struct {
	TotalTokens  int `json:"total_tokens"`
	PromptTokens int `json:"prompt_tokens"`
}

// ResponseData is the data from the swama embedding API.
type SwamaEmbeddingResponseData struct {
	Embedding SwamaEmbedding `json:"embedding"`
}

// SwamaEmbedding is the embedding from the swama embedding API.
type SwamaEmbedding []float64

// SwamaCompletionRequest is the request to the swama completion API.
type SwamaCompletionRequest struct {
	Model       string         `json:"model"`
	Messages    []SwamaMessage `json:"messages"`
	Temperature float64        `json:"temperature"`
	MaxTokens   int            `json:"max_tokens"`
}

// SwamaMessage is a message in the swama completion API.
type SwamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// SwamaCompletionsResponse
type SwamaCompletionsResponse struct {
	Choices []SwamaChoice      `json:"choices"`
	Created int64              `json:"created"`
	Object  string             `json:"object"`
	Model   string             `json:"model"`
	Usage   SwamaResponseUsage `json:"usage"`
	Id      string             `json:"id"`
}

// SwamaChoice represents a response to choose from.
type SwamaChoice struct {
	Message      SwamaMessage `json:"message"`
	Index        int          `json:"index"`
	FinishReason string       `json:"finish_reason"`
}

type SwamaAPI struct {
	endpoint url.URL
	client   *http.Client
}

func NewSwamaAPI(endpoint url.URL) (*SwamaAPI, error) {
	return &SwamaAPI{
		endpoint: endpoint,
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}, nil
}

func (s *SwamaAPI) Embed(
	ctx context.Context,
	texts ...string,
) ([]SwamaEmbedding, error) {
	req := SwamaEmbeddingRequest{
		Model: Model,
		Input: texts,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	embedEndpoint := s.endpoint
	embedEndpoint.Path = "/v1/embeddings"

	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		embedEndpoint.String(),
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to embed text: %s", resp.Status)
	}

	var response SwamaEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if len(response.Data) == 0 {
		return nil, errors.New("no data returned from swama embedding API")
	}

	embeddings := make([]SwamaEmbedding, len(response.Data))

	for i, data := range response.Data {
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}

// Complete will generate a completion for the given prompt using the swama API.
func (s *SwamaAPI) Complete(ctx context.Context, prompt string, data string) (string, error) {
	req := SwamaCompletionRequest{
		Model: Model,
		Messages: []SwamaMessage{
			{
				Role:    "system",
				Content: prompt,
			},
			{
				Role:    "user",
				Content: data,
			},
		},
		Temperature: 0.7,
		MaxTokens:   2048,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	endpoint := s.endpoint
	endpoint.Path = "/v1/chat/completions"

	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		endpoint.String(),
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return "", err
	}

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to complete prompt: %s", resp.Status)
	}

	var response SwamaCompletionsResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	if len(response.Choices) == 0 {
		return "", errors.New("no choices returned from swama completion API")
	}

	return response.Choices[0].Message.Content, nil
}

var pruneThinkingRegex = regexp.MustCompile(`(?is)<think>.*?</think>`)

// PruneThinking will prune the thinking tags from the given completion text.
func PruneThinking(text string) string {
	removeThinking := pruneThinkingRegex.ReplaceAllString(text, "")

	return strings.TrimSpace(removeThinking)
}
