package main

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"github.com/Crystalix007/reverse-dict/backend"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use: "search-phrase",
		RunE: func(cmd *cobra.Command, args []string) error {
			return Run(cmd.Context(), args)
		},
	}

	rootCmd.Args = cobra.ExactArgs(1)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func Run(ctx context.Context, args []string) error {
	db, err := backend.NewSQLiteVec(ctx, "words.db")
	if err != nil {
		return fmt.Errorf("creating sqlite database: %w", err)
	}

	swama, err := backend.NewSwamaAPI(
		url.URL{
			Scheme: "http",
			Host:   "localhost:28100",
		},
	)
	if err != nil {
		return fmt.Errorf("creating swama api: %w", err)
	}

	embeddings, err := swama.EmbedQuery(ctx, args[0])
	if err != nil {
		return fmt.Errorf("embedding phrase: %w", err)
	}

	embedding := backend.NewEmbeddingFromFloat64(embeddings[0])

	relatedWords, err := db.RelatedWords(ctx, backend.ModelQwen3Embedding8B4B_DWQ, embedding, 10)
	if err != nil {
		return fmt.Errorf("getting related words: %w", err)
	}

	for _, word := range relatedWords {
		fmt.Printf("%s: %s (%.2f)\n", word.Word.Word, word.Word.Definition, word.Distance)
		fmt.Printf("\t-> %s\n", word.Phrase)
	}

	return nil
}
