package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/Crystalix007/reverse-dict/backend"
	"github.com/davidscholberg/go-urbandict"
	"github.com/spf13/cobra"
)

type Flags struct {
	count     uint
	rateLimit time.Duration
}

func main() {
	var flags Flags

	rootCmd := &cobra.Command{
		Use: "add-words",
		RunE: func(cmd *cobra.Command, args []string) error {
			return Run(cmd.Context(), flags)
		},
	}

	rootCmd.Flags().UintVarP(
		&flags.count,
		"count",
		"c",
		1,
		"number of words to add",
	)

	rootCmd.Flags().DurationVarP(
		&flags.rateLimit,
		"rate-limit",
		"r",
		1*time.Second,
		"rate limit for adding words",
	)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func Run(ctx context.Context, flags Flags) error {
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

	rateLimit := time.After(0)

	for range flags.count {
		<-rateLimit

		rateLimit = time.After(flags.rateLimit)

		randWord, err := urbandict.Random()
		if err != nil {
			return fmt.Errorf("getting random word: %w", err)
		}

		splitDef := backend.SplitDefinition(randWord.Definition)

		embeddingsF64, err := swama.Embed(ctx, splitDef...)
		if err != nil {
			return fmt.Errorf("embedding word: %w", err)
		}

		subdefinitions := make([]backend.SubDefinition, len(embeddingsF64))

		for i, embedding := range embeddingsF64 {
			subdefinitions[i] = backend.SubDefinition{
				Phrase: splitDef[i],
				Vector: backend.NewEmbeddingFromFloat64(embedding),
			}
		}

		definition := backend.Definition{
			Word:       randWord.Word,
			Definition: randWord.Definition,
			Example:    randWord.Example,
			Embeddings: subdefinitions,
		}

		if _, err := db.AddWord(ctx, definition); err != nil {
			return fmt.Errorf("adding word: %w", err)
		}
	}

	return nil
}
