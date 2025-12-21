package frontend

import (
	"net/http"
	"net/url"

	"github.com/Crystalix007/reverse-dict/frontend/routes"
)

func Serve(backendURL url.URL) http.Handler {
	mux := http.NewServeMux()

	frontendHandler := routes.New(backendURL)

	mux.Handle("GET /{$}", serveFile("index.html"))
	mux.Handle("GET /static/", serveStatic())
	mux.Handle("POST /search", http.HandlerFunc(frontendHandler.SearchResults))
	mux.Handle("GET /", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	return mux
}
