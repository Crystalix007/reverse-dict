package backend

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

type API struct {
	swamaAPI  *SwamaAPI
	sqliteVec *SQLiteVec
}

// NewAPI creates a new API instance with the provided Swama API and SQLite
// vector database.
func NewAPI(swamaAPI *SwamaAPI, sqliteVec *SQLiteVec) *API {
	return &API{
		swamaAPI:  swamaAPI,
		sqliteVec: sqliteVec,
	}
}

// Serve returns an HTTP handler that serves the API.
func (a *API) Serve() http.Handler {
	router := chi.NewMux()

	api := humachi.New(
		router,
		huma.Config{},
	)

	huma.Get(api, "/search", a.Search)

	return router
}

// Response structure for search results.
type SearchResponse struct {
	Results []SimilarDefinition `json:"results"`
}

// Search queries the DB for words with definitions that are semantically
// similar to the provided query.
func (a *API) Search(
	ctx context.Context,
	input *struct {
		Query string `json:"query" description:"The phrase to search for"`
		Limit int    `json:"limit" description:"The maximum number of results to return" default:"10"`
	},
) (*SearchResponse, error) {
	queryEmbedding, err := a.swamaAPI.Embed(ctx, input.Query)
	if err != nil {
		return nil, fmt.Errorf(
			"embedding query: %w",
			err,
		)
	}

	if len(queryEmbedding) == 0 {
		return nil, fmt.Errorf("no embeddings returned for query: %s", input.Query)
	}

	primaryEmbedding := NewEmbeddingFromFloat64(queryEmbedding[0])

	results, err := a.sqliteVec.RelatedWords(ctx, primaryEmbedding, input.Limit)
	if err != nil {
		return nil, fmt.Errorf(
			"searching in SQLiteVec: %w",
			err,
		)
	}

	return &SearchResponse{
		Results: results,
	}, nil
}
