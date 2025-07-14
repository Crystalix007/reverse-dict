package backend

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// SwamaResponse is the response from the swama embedding API.
type SwamaResponse struct {
	Model string              `json:"model"`
	Usage SwamaResponseUsage  `json:"usage"`
	Data  []SwamaResponseData `json:"data"`
}

// SwamaRequest is the request to the swama embedding API.
type SwamaRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// ResponseUsage is the usage from the swama embedding API.
type SwamaResponseUsage struct {
	TotalTokens  int `json:"total_tokens"`
	PromptTokens int `json:"prompt_tokens"`
}

// ResponseData is the data from the swama embedding API.
type SwamaResponseData struct {
	Embedding SwamaEmbedding `json:"embedding"`
}

// SwamaEmbedding is the embedding from the swama embedding API.
type SwamaEmbedding []float64

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

func (s *SwamaAPI) Embed(texts ...string) ([]SwamaEmbedding, error) {
	req := SwamaRequest{
		Model: "mlx-community/Qwen3-8B-4bit",
		Input: texts,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", s.endpoint.String(), bytes.NewBuffer(reqBody))
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

	var response SwamaResponse
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
