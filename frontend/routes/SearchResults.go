package routes

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/Crystalix007/reverse-dict/backend"
	"github.com/Crystalix007/reverse-dict/frontend/backendclient"
)

// SearchResults renders the search results page.
func SearchResults(w http.ResponseWriter, r *http.Request) {
	var words []backend.Word

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	searchInput := r.Form.Get("query")

	if searchInput != "" {
		searchResults, err := backendclient.Get[backend.SearchResponseBody](
			r.Context(),
			url.URL{
				Path:     "search",
				RawQuery: "query=" + searchInput,
			},
		)
		if err != nil {
			slog.ErrorContext(r.Context(), "Failed to get search results", slog.String("query", searchInput), slog.Any("error", err))
			http.Error(w, "Failed to get search results", http.StatusInternalServerError)
			return
		}

		words = make([]backend.Word, 0, len(searchResults.Results))

		for _, result := range searchResults.Results {
			words = append(words, result.Word)
		}
	}

	component := searchResults(words)
	component.Render(r.Context(), w)
}
