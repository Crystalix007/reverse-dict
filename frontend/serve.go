package frontend

import (
	"net/http"
	"net/url"
	"path"

	"github.com/Crystalix007/reverse-dict/frontend/routes"
)

func Serve(backendURL url.URL) http.Handler {
	mux := http.NewServeMux()

	frontendHandler := routes.New(backendURL)

	mux.Handle("GET /{$}", serveStaticFile("index.html"))
	mux.Handle("GET /static/", http.FileServerFS(StaticFiles))
	mux.Handle("POST /search", http.HandlerFunc(frontendHandler.SearchResults))
	mux.Handle("GET /", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	return mux
}

func serveStaticFile(filename string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, StaticFiles, path.Join("static", filename))
	})
}
