package backend

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type API struct {
	address   url.URL
	embedder  Embedders
	sqliteVec *SQLiteVec
}

// NewAPI creates a new API instance with the provided [Embedder] backend and
// SQLite vector database.
func NewAPI(embedders Embedders, sqliteVec *SQLiteVec, address url.URL) *API {
	return &API{
		address:   address,
		embedder:  embedders,
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
	RegisterLogged(
		api,
		huma.Operation{
			Method: http.MethodGet,
			Path:   "/search",
		},
		a.Search,
	)

	return router
}

// Response structure for search results.
type SearchResponse struct {
	Body SearchResponseBody
}

type SearchResponseBody struct {
	Results map[Model][]SimilarDefinition `json:"results"`
}

func RegisterLogged[I, O any](
	api huma.API,
	op huma.Operation,
	handler func(ctx context.Context, input *I) (*O, error),
) {
	huma.Register(
		api,
		op,
		func(ctx context.Context, i *I) (*O, error) {
			output, err := handler(ctx, i)
			if err != nil {
				slog.ErrorContext(
					ctx,
					"request failed",
					slog.String("error", err.Error()),
				)

				return output, err
			}

			return output, nil
		},
	)
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
	queryEmbeddings, err := a.embedder.Embed(ctx, input.Query)
	if err != nil {
		return nil, fmt.Errorf(
			"embedding query: %w",
			err,
		)
	}

	results := make(map[Model][]SimilarDefinition)

	for model, embeddings := range queryEmbeddings {
		if len(embeddings) == 0 {
			return nil, fmt.Errorf("no embeddings returned for query: %s", input.Query)
		}

		modelResults, err := a.sqliteVec.RelatedWords(
			ctx,
			model,
			embeddings[0],
			input.Limit,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"searching in SQLiteVec: %w",
				err,
			)
		}

		if len(modelResults) == 0 {
			return nil, huma.Error404NotFound("no matching definitions found")
		}

		results[model] = modelResults
	}

	return &SearchResponse{
		Body: SearchResponseBody{
			Results: results,
		},
	}, nil
}
