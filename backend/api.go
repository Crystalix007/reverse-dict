package backend

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type API struct {
	address   url.URL
	swamaAPI  *SwamaAPI
	sqliteVec *SQLiteVec
}

// NewAPI creates a new API instance with the provided Swama API and SQLite
// vector database.
func NewAPI(swamaAPI *SwamaAPI, sqliteVec *SQLiteVec, address url.URL) *API {
	return &API{
		address:   address,
		swamaAPI:  swamaAPI,
		sqliteVec: sqliteVec,
	}
}

// Serve returns an HTTP handler that serves the API.
func (a *API) Serve() http.Handler {
	router := chi.NewMux()

	// Strip the API prefix.
	router.Use(middleware.StripPrefix("/api"))

	api := humachi.New(
		router,
		huma.DefaultConfig("Reverse Dictionary API", "0.0.1"),
	)

	api.OpenAPI().Servers = []*huma.Server{
		{
			Description: "Current environment",
			URL:         a.address.String(),
		},
	}

	// Register endpoints.
	huma.Get(api, "/search", a.Search)

	return router
}

// Response structure for search results.
type SearchResponse struct {
	Body SearchResponseBody
}

type SearchResponseBody struct {
	Results []SimilarDefinition `json:"results"`
}

// Search queries the DB for words with definitions that are semantically
// similar to the provided query.
func (a *API) Search(
	ctx context.Context,
	input *struct {
		Query string `query:"query" json:"query" description:"The phrase to search for"`
		Limit int    `query:"limit" json:"limit" description:"The maximum number of results to return" default:"10"`
	},
) (*SearchResponse, error) {
	queryEmbedding, err := a.swamaAPI.EmbedQuery(ctx, input.Query)
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

	results, err := a.sqliteVec.RelatedWords(
		ctx,
		ModelQwen3Embedding8B4B_DWQ,
		primaryEmbedding,
		input.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"searching in SQLiteVec: %w",
			err,
		)
	}

	if len(results) == 0 {
		return nil, huma.Error404NotFound("no matching definitions found")
	}

	return &SearchResponse{
		Body: SearchResponseBody{
			Results: results,
		},
	}, nil
}
