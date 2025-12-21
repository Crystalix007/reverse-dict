//go:build debug

package frontend

import (
	"net/http"
	"os"
	"path"
)

func serveStatic() http.Handler {
	staticDir, err := os.OpenRoot("static")
	if err != nil {
		panic(err)
	}

	staticFS := staticDir.FS()

	return http.StripPrefix("/static/", http.FileServerFS(staticFS))
}

func serveFile(filename string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Join("static", filename))
	})
}
