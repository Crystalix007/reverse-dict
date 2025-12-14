package routes

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/Crystalix007/reverse-dict/backend"
	"github.com/Crystalix007/reverse-dict/frontend/backendclient"
)

// SearchResults renders the search results page.
func (h *Handler) SearchResults(w http.ResponseWriter, r *http.Request) {
	words := make(map[backend.Model][]backend.Word)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	searchInput := r.Form.Get("query")

	if searchInput != "" {
		searchResults, err := backendclient.Get[backend.SearchResponseBody](
			r.Context(),
			h.backendURL,
			url.URL{
				Path: "search",
				RawQuery: url.Values{
					"query": []string{searchInput},
				}.Encode(),
			},
		)
		if err != nil {
			slog.ErrorContext(r.Context(), "Failed to get search results", slog.String("query", searchInput), slog.Any("error", err))
			http.Error(w, "Failed to get search results", http.StatusInternalServerError)
			return
		}

		for model, results := range searchResults.Results {
			words[model] = make([]backend.Word, 0, len(results))

			for _, result := range results {
				words[model] = append(words[model], result.Word)
			}
		}
	}

	component := searchResults(words)
	component.Render(r.Context(), w)
}
