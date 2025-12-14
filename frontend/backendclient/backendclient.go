package backendclient

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path"
)

// Get performs a GET request to the backend API.
func Get[T any](
	ctx context.Context,
	baseURL url.URL,
	overlayURL url.URL,
) (*T, error) {
	client := &http.Client{}

	joinedURL := url.URL{
		Scheme:   baseURL.Scheme,
		Host:     baseURL.Host,
		Path:     path.Join(baseURL.Path, overlayURL.Path),
		RawQuery: overlayURL.RawQuery,
		Fragment: overlayURL.Fragment,
	}

	var statusCode int

	defer func() {
		slog.InfoContext(ctx, "backendclient: performing GET request", slog.String("url", joinedURL.String()), slog.Int("status_code", statusCode))
	}()

	req, err := http.NewRequestWithContext(ctx, "GET", joinedURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("backendclient: creating GET request: %w", err)
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("backendclient: performing GET request: %w", err)
	}

	statusCode = res.StatusCode

	defer res.Body.Close()

	var result T
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("backendclient: decoding GET response: %w", err)
	}

	return &result, nil
}
