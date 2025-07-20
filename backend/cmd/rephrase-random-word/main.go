package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/Crystalix007/reverse-dict/backend"
)

func main() {
	ctx := context.Background()

	if err := rephraseRandomWord(ctx); err != nil {
		slog.ErrorContext(
			ctx,
			"rephrasing random word",
			slog.Any("error", err),
		)
	}
}

func rephraseRandomWord(ctx context.Context) error {
	vec, err := backend.NewSQLiteVec(ctx, "words.db")
	if err != nil {
		return fmt.Errorf("creating SQLiteVec: %w", err)
	}

	defer vec.Close()

	def, err := vec.GetRandomDefinition(ctx)
	if err != nil {
		return fmt.Errorf("getting random definition: %w", err)
	}

	fmt.Printf(
		"Random Definition:\nWord: %s\nDefinition: %s\nExample: %s\n",
		def.Word,
		def.Definition,
		def.Example,
	)

	swama, err := backend.NewSwamaAPI(url.URL{
		Scheme: "http",
		Host:   "127.0.0.1:28100",
	})
	if err != nil {
		return fmt.Errorf("creating SwamaAPI: %w", err)
	}

	rephrased, err := swama.RephraseDefinition(ctx, *def)
	if err != nil {
		return fmt.Errorf("rephrasing definition: %w", err)
	}

	fmt.Printf("Rephrased:\n")

	for i, sentence := range rephrased {
		fmt.Printf("Def. %d: %s\n", i+1, sentence)
	}

	return nil
}
