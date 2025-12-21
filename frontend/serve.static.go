//go:build !debug

package frontend

import (
	"embed"
	"net/http"
	"path"
)

//go:embed static/*
var StaticFiles embed.FS

func serveStatic() http.Handler {
	return http.FileServerFS(StaticFiles)
}

func serveFile(filename string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, StaticFiles, path.Join("static", filename))
	})
}
