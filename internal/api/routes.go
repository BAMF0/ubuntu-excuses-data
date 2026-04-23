package api

import (
	"net/http"

	"github.com/BAMF0/ubuntu-excuses-data/internal/domain"
)

// RegisterRoutes wires all API handlers onto the given ServeMux.
func RegisterRoutes(mux *http.ServeMux, e *domain.Excuses, teams domain.TeamMappings) {
	h := NewHandler(e, teams)

	mux.HandleFunc("GET /meta", h.GetMeta)
	mux.HandleFunc("GET /blocked", h.ListBlocked)
	mux.HandleFunc("GET /sources", h.ListSources)
	mux.HandleFunc("GET /sources/{name}", h.GetSource)
	mux.HandleFunc("GET /sources/{name}/autopkgtest", h.GetSourceAutopkgtest)
}
