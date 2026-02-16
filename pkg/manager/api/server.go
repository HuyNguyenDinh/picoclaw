package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sipeed/picoclaw/pkg/manager/tenant"
)

// NewServer creates and configures the HTTP router.
func NewServer(svc *tenant.Service, apiKey string) http.Handler {
	h := NewHandlers(svc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// Public endpoints.
	r.Get("/health", h.HealthCheck)

	// Protected API endpoints.
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(APIKeyAuth(apiKey))

		r.Post("/tenants", h.CreateTenant)
		r.Get("/tenants", h.ListTenants)
		r.Get("/tenants/{id}", h.GetTenant)
		r.Put("/tenants/{id}", h.UpdateTenant)
		r.Delete("/tenants/{id}", h.DeleteTenant)
		r.Post("/tenants/{id}/restart", h.RestartTenant)
	})

	return r
}
