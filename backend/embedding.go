package backend

import (
	"context"
	"fmt"
	"time"

	"github.com/openai/openai-go/v2"
	"golang.org/x/time/rate"
)

// Embedder represents a service that can create embeddings for phrases.
type Embedder interface {
	Embed(ctx context.Context, phrases ...string) ([]Embedding, error)
}

// Embedders represents a collection of embedding services.
type Embedders map[Model]Embedder

// Embed runs an embedding request against all configured embedding services.
//
// Returns a map between the backend and its embeddings.
func (e *Embedders) Embed(ctx context.Context, phrases ...string) (map[Model][]Embedding, error) {
	embeddings := make(map[Model][]Embedding, len(*e))

	for model, embedder := range *e {
		embedding, err := embedder.Embed(ctx, phrases...)
		if err != nil {
			return nil, err
		}

		embeddings[model] = embedding
	}

	return embeddings, nil
}

// swamaEmbedder is an implementation of the [Embedder] interface for the Swama
// API.
type swamaEmbedder struct {
	api *SwamaAPI
}

var _ Embedder = &swamaEmbedder{}

func NewSwamaEmbedder(api *SwamaAPI) Embedder {
	return &swamaEmbedder{
		api: api,
	}
}

// Embed returns the embeddings for the given phrases from the Swama API.
func (s *swamaEmbedder) Embed(ctx context.Context, phrases ...string) ([]Embedding, error) {
	// Call the Swama API to get embeddings for the phrases
	swamaEmbeddings, err := s.api.Embed(ctx, phrases...)
	if err != nil {
		return nil, err
	}

	// Convert the embeddings from [][]float64 to []Embedding (which is []float32)
	embeddings := make([]Embedding, 0, len(swamaEmbeddings))

	for _, vec := range swamaEmbeddings {
		embeddings = append(embeddings, NewEmbeddingFromFloat64(vec))
	}

	return embeddings, nil
}

// openaiEmbedder is an implementation of the [Embedder] interface for the
// OpenAI API.
type openaiEmbedder struct {
	api       openai.Client
	model     openai.EmbeddingModel
	ratelimit rate.Limiter
}

var _ Embedder = &openaiEmbedder{}

func NewOpenAIEmbedder(model openai.EmbeddingModel) Embedder {
	return &openaiEmbedder{
		api:       openai.NewClient(),
		model:     model,
		ratelimit: *rate.NewLimiter(rate.Every(500*time.Millisecond), 5),
	}
}

// Embed returns the embeddings for the given phrases from the OpenAI API.
func (o *openaiEmbedder) Embed(ctx context.Context, phrases ...string) ([]Embedding, error) {
	if err := o.ratelimit.Wait(ctx); err != nil {
		return nil, fmt.Errorf("waiting for rate limit: %w", err)
	}

	// Call the OpenAI API to get embeddings for the phrases.
	openaiEmbeddings, err := o.api.Embeddings.New(
		ctx,
		openai.EmbeddingNewParams{
			Model: o.model,
			Input: openai.EmbeddingNewParamsInputUnion{
				OfArrayOfStrings: phrases,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("requesting OpenAI embeddings: %w", err)
	}

	embeddings := make([]Embedding, 0, len(openaiEmbeddings.Data))

	for _, data := range openaiEmbeddings.Data {
		embeddings = append(embeddings, NewEmbeddingFromFloat64(data.Embedding))
	}

	return embeddings, nil
}
