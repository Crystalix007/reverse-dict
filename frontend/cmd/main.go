package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"

	"github.com/spf13/cobra"

	"github.com/Crystalix007/reverse-dict/frontend"
)

type args struct {
	listenAddress string
	serverAddress string
}

func main() {
	var args args

	cmd := cobra.Command{
		Use:   "frontend",
		Short: "Start the frontend server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd.Context(), args)
		},
	}

	cmd.Flags().StringVarP(&args.listenAddress, "listen", "l", "localhost:3000", "Address to bind the server to")
	cmd.Flags().StringVarP(&args.serverAddress, "server", "s", "http://localhost:8080/api/", "Address of the backend server")

	if err := cmd.Execute(); err != nil {
		slog.Error("error starting server", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(ctx context.Context, args args) error {
	slog.InfoContext(ctx, "starting frontend server", slog.Any("address", args.listenAddress))

	serverURL, err := url.Parse(args.serverAddress)
	if err != nil {
		return fmt.Errorf("parsing server address: %w", err)
	}

	return http.ListenAndServe(args.listenAddress, frontend.Serve(*serverURL))
}
