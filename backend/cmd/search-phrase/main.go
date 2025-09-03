package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"

	"github.com/Crystalix007/reverse-dict/backend"
	"github.com/openai/openai-go/v2"
	"github.com/spf13/cobra"
)

type flags struct {
	model string
}

func main() {
	var flags flags

	rootCmd := &cobra.Command{
		Use: "search-phrase",
		RunE: func(cmd *cobra.Command, args []string) error {
			return Run(cmd.Context(), args, flags)
		},
	}

	rootCmd.Args = cobra.ExactArgs(1)

	rootCmd.Flags().StringVar(
		&flags.model,
		"model",
		backend.ModelQwen3Embedding8B4B_DWQ.String(),
		"The model to use for embedding",
	)

	if err := rootCmd.Execute(); err != nil {
		slog.Error("Searching phrase failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func Run(ctx context.Context, args []string, flags flags) error {
	model, err := backend.ModelFromString(flags.model)
	if err != nil {
		return fmt.Errorf("parsing model flag: %w", err)
	}

	db, err := backend.NewSQLiteVec(ctx, "words.db")
	if err != nil {
		return fmt.Errorf("creating sqlite database: %w", err)
	}

	var embedder backend.Embedder

	switch model {
	case backend.ModelQwen3Embedding8B4B_DWQ:
		swama, err := backend.NewSwamaAPI(
			url.URL{
				Scheme: "http",
				Host:   "localhost:28100",
			},
		)
		if err != nil {
			return fmt.Errorf("creating swama api: %w", err)
		}

		embedder = backend.NewSwamaQueryEmbedder(swama)
	case backend.ModelOpenAITextEmbedding3Large:
		embedder = backend.NewOpenAIEmbedder(openai.EmbeddingModelTextEmbedding3Large)
	default:
		panic(fmt.Sprintf("model %s not supported yet", model.String()))
	}

	embeddings, err := embedder.Embed(ctx, args[0])
	if err != nil {
		return fmt.Errorf("embedding phrase: %w", err)
	}

	embedding := embeddings[0]

	slog.InfoContext(
		ctx,
		"embedded phrase",
		slog.String("phrase", args[0]),
		slog.Int("embedding_size", len(embedding)),
	)

	relatedWords, err := db.RelatedWords(ctx, model, embedding, 10)
	if err != nil {
		return fmt.Errorf("getting related words: %w", err)
	}

	for _, word := range relatedWords {
		fmt.Printf("%s: %s (%.2f)\n", word.Word.Word, word.Word.Definition, word.Distance)
		fmt.Printf("\t-> %s\n", word.Phrase)
	}

	return nil
}
