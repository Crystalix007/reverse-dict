package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"

	"github.com/Crystalix007/reverse-dict/backend"
)

func main() {
	if err := run(); err != nil {
		slog.Error("Error running server", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	sqlite, err := backend.NewSQLiteVec(
		ctx,
		"words.db",
	)
	if err != nil {
		return fmt.Errorf("creating SQLiteVec: %w", err)
	}

	swamaAPI, err := backend.NewSwamaAPI(
		url.URL{
			Scheme: "http",
			Host:   "localhost:28100",
		},
	)
	if err != nil {
		return fmt.Errorf("creating SwamaAPI: %w", err)
	}

	a := backend.NewAPI(swamaAPI, sqlite)

	http.Handle("/", a.Serve())

	fmt.Println("Starting server on :8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		return fmt.Errorf("starting server: %w", err)
	}

	return nil
}
