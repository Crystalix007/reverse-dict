package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/Crystalix007/reverse-dict/backend"
	"github.com/spf13/cobra"
)

var (
	ErrFromFlagRequired = errors.New("the --from flag is required")
	ErrToFlagRequired   = errors.New("the --to flag is required")
	ErrNoEmbeddingFound = errors.New("no embedding found for the provided phrase")
)

type arguments struct {
	doc   string
	query string
}

func main() {
	var args arguments

	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "Ingest test data into the backend",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return ingestData(cmd.Context(), args)
		},
	}

	cmd.Flags().StringVar(&args.doc, "doc", "", "Document to ingest")
	cmd.Flags().StringVar(&args.query, "query", "", "Query to run against")

	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}

func ingestData(ctx context.Context, args arguments) error {
	if args.doc == "" {
		return ErrFromFlagRequired
	}

	if args.query == "" {
		return ErrToFlagRequired
	}

	db, err := backend.NewSQLiteVec(ctx, "words.db")
	if err != nil {
		return fmt.Errorf("creating sqlite database: %w", err)
	}

	defer db.Close()

	swama, err := backend.NewSwamaAPI(
		url.URL{
			Scheme: "http",
			Host:   "localhost:28100",
		},
	)
	if err != nil {
		return fmt.Errorf("creating swama api: %w", err)
	}

	docEmbeddings, err := swama.Embed(ctx, args.doc)
	if err != nil {
		return fmt.Errorf("embedding 'doc' phrase: %w", err)
	}

	if len(docEmbeddings) == 0 {
		return fmt.Errorf(
			"%w: doc phrase did not work",
			ErrNoEmbeddingFound,
		)
	}

	queryEmbeddings, err := swama.EmbedQuery(ctx, args.query)
	if err != nil {
		return fmt.Errorf("embedding 'query' phrase: %w", err)
	}

	if len(queryEmbeddings) == 0 {
		return fmt.Errorf(
			"%w: query phrase did not work",
			ErrNoEmbeddingFound,
		)
	}

	docEmbedding := backend.NewEmbeddingFromFloat64(docEmbeddings[0])
	queryEmbedding := backend.NewEmbeddingFromFloat64(queryEmbeddings[0])

	distance, err := db.CompareEmbeddings(ctx, docEmbedding, queryEmbedding)
	if err != nil {
		return fmt.Errorf("comparing embeddings: %w", err)
	}

	fmt.Printf("Distance between '%s' and '%s': %f\n", args.doc, args.query, distance)

	return nil
}
