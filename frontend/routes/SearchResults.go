package routes

import (
	"net/http"
	"net/url"

	"github.com/Crystalix007/reverse-dict/backend"
	"github.com/Crystalix007/reverse-dict/frontend/backendclient"
)

// SearchResults renders the search results page.
func SearchResults(w http.ResponseWriter, r *http.Request) {
	var definitions []backend.Definition

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	searchInput := r.Form["search-input"]

	if len(searchInput) > 0 {
		searchResults, err := backendclient.Get[backend.SearchResponseBody](
			r.Context(),
			url.URL{
				Path:     "search",
				RawQuery: "query=" + searchInput[0],
			},
		)
		if err != nil {
			http.Error(w, "Failed to get search results", http.StatusInternalServerError)
			return
		}

		definitions = make([]backend.Definition, 0, len(searchResults.Results))

		for _, result := range searchResults.Results {
			definitions = append(definitions, result.Definition)
		}
	}

	component := searchResults(definitions)
	component.Render(r.Context(), w)
}
