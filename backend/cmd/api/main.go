package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/openai/openai-go/v2"
	slogchi "github.com/samber/slog-chi"
	"github.com/spf13/cobra"

	"github.com/Crystalix007/reverse-dict/backend"
)

type args struct {
	host         string
	modelNames   []string
	swamaAddress string
	quiet        bool
}

func main() {
	var args args

	cmd := cobra.Command{
		Use:   "api",
		Short: "Start the API server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd.Context(), &args)
		},
	}

	cmd.Flags().StringVarP(&args.host, "listen", "l", "localhost:8080", "Address to bind the server to")
	cmd.Flags().BoolVarP(&args.quiet, "quiet", "q", false, "Suppress debug log output")
	cmd.Flags().StringSliceVar(&args.modelNames, "model", nil, "Models to use for query embeddings")
	cmd.Flags().StringVar(&args.swamaAddress, "swama-address", "http://localhost:28100", "Address of the Swama API server")

	if err := cmd.Execute(); err != nil {
		slog.Error("Error running server", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(ctx context.Context, args *args) error {
	// If quiet mode is not enabled, set the log level to debug.
	if !args.quiet {
		slog.SetDefault(
			slog.New(
				slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
					Level: slog.LevelDebug,
				}),
			),
		)
	}

	var models []backend.Model

	for _, model := range args.modelNames {
		model, err := backend.ModelFromString(model)
		if err != nil {
			return fmt.Errorf("parsing model %q: %w", model, err)
		}

		models = append(models, model)
	}

	// Default to all models if unspecified.
	if len(models) == 0 {
		models = backend.Models
	}

	embedders, err := getEmbedders(args, models)
	if err != nil {
		return fmt.Errorf("getting embedders: %w", err)
	}

	sqlite, err := backend.NewSQLiteVec(
		ctx,
		"words.db",
	)
	if err != nil {
		return fmt.Errorf("creating SQLiteVec: %w", err)
	}

	listenAddress := url.URL{
		Scheme: "http",
		Host:   args.host,
	}

	apiAddress := listenAddress
	apiAddress.Path = "/api"

	api := backend.NewAPI(embedders, sqlite, apiAddress)

	// Create global mux.
	router := chi.NewMux()

	http.Handle("/", router)

	router.Use(
		slogchi.NewWithConfig(
			slog.Default(),
			slogchi.Config{
				DefaultLevel:  slog.LevelDebug,
				WithTraceID:   true,
				WithUserAgent: true,
				WithSpanID:    true,
			},
		),
	)

	// Add CORS to the API endpoints.
	router.Group(func(router chi.Router) {
		router.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{listenAddress.String()},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Content-Type", "Authorization"},
			ExposedHeaders:   []string{"Content-Length"},
			AllowCredentials: true,
		}))

		router.Handle("/api/*", api.Serve())
		router.Get("/api", http.RedirectHandler("/api/docs", http.StatusMovedPermanently).ServeHTTP)
	})

	slog.InfoContext(ctx, "Starting server", slog.String("address", listenAddress.String()))

	if err := http.ListenAndServe(args.host, nil); err != nil {
		return fmt.Errorf("starting server: %w", err)
	}

	return nil
}

func getEmbedders(
	args *args,
	models []backend.Model,
) (backend.Embedders, error) {
	embedders := make(backend.Embedders)

	for _, model := range models {
		switch model {
		case backend.ModelQwen3Embedding8B4B_DWQ:
			swamaURL, err := url.Parse(args.swamaAddress)
			if err != nil {
				return nil, fmt.Errorf("parsing Swama address: %w", err)
			}

			swamaAPI, err := backend.NewSwamaAPI(
				*swamaURL,
			)
			if err != nil {
				return nil, fmt.Errorf("creating SwamaAPI: %w", err)
			}

			embedders[model] = backend.NewSwamaQueryEmbedder(swamaAPI)
		case backend.ModelOpenAITextEmbedding3Large:
			embedder := backend.NewOpenAIEmbedder(openai.EmbeddingModelTextEmbedding3Large)

			embedders[model] = embedder
		}
	}

	return embedders, nil
}
