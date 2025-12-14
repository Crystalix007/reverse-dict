// Package routes defines the per-route templates and construction logic.
package routes

import "net/url"

//go:generate go tool templ generate

type Handler struct {
	backendURL url.URL
}

func New(backendURL url.URL) *Handler {
	return &Handler{
		backendURL: backendURL,
	}
}
