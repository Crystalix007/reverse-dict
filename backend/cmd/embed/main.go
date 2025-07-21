package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/Crystalix007/reverse-dict/backend"
	"github.com/spf13/cobra"
)

var (
	ErrNoPhraseProvided = errors.New("no phrase provided")
	ErrNoEmbeddingFound = errors.New("no embedding found for the provided phrase")
)

func main() {
	cmd := &cobra.Command{
		Use:   "compare",
		Short: "Compare two phrases and ingest the result",
		RunE: func(cmd *cobra.Command, args []string) error {
			return embed(cmd.Context(), args)
		},
	}

	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}

func embed(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return ErrNoPhraseProvided
	}

	phrase := args[0]

	swama, err := backend.NewSwamaAPI(
		url.URL{
			Scheme: "http",
			Host:   "localhost:28100",
		},
	)
	if err != nil {
		return fmt.Errorf("creating swama api: %w", err)
	}

	embeddings, err := swama.Embed(ctx, phrase)
	if err != nil {
		return fmt.Errorf("embedding phrase: %w", err)
	}

	if len(embeddings) == 0 {
		return ErrNoEmbeddingFound
	}

	embeddingJSON, err := json.Marshal(embeddings[0])
	if err != nil {
		return fmt.Errorf("marshalling embedding: %w", err)
	}

	fmt.Printf("%s\n", embeddingJSON)

	return nil
}
