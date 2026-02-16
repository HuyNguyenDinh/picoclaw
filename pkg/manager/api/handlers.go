package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sipeed/picoclaw/pkg/manager/tenant"
)

// Handlers implements the tenant management REST API.
type Handlers struct {
	svc *tenant.Service
}

// NewHandlers creates Handlers with a tenant service.
func NewHandlers(svc *tenant.Service) *Handlers {
	return &Handlers{svc: svc}
}

// CreateTenant handles POST /api/v1/tenants.
func (h *Handlers) CreateTenant(w http.ResponseWriter, r *http.Request) {
	var req CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid json: " + err.Error()})
		return
	}

	var resources *tenant.Resources
	if req.Resources != nil {
		resources = &tenant.Resources{
			AgentCPU:      req.Resources.AgentCPU,
			AgentMemory:   req.Resources.AgentMemory,
			GatewayCPU:    req.Resources.GatewayCPU,
			GatewayMemory: req.Resources.GatewayMemory,
			WorkspaceSize: req.Resources.WorkspaceSize,
		}
	}

	t, err := h.svc.Create(r.Context(), tenant.CreateRequest{
		TenantID:    req.TenantID,
		DisplayName: req.DisplayName,
		Providers:   req.Providers,
		Agents:      req.Agents,
		Channels:    req.Channels,
		Resources:   resources,
	})
	if err != nil {
		log.Printf("create tenant error: %v", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, toTenantResponse(t))
}

// ListTenants handles GET /api/v1/tenants.
func (h *Handlers) ListTenants(w http.ResponseWriter, r *http.Request) {
	tenants, err := h.svc.List(r.Context())
	if err != nil {
		log.Printf("list tenants error: %v", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	resp := make([]TenantResponse, len(tenants))
	for i := range tenants {
		resp[i] = toTenantResponse(&tenants[i])
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetTenant handles GET /api/v1/tenants/:id.
func (h *Handlers) GetTenant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	t, err := h.svc.Get(r.Context(), id)
	if err != nil {
		log.Printf("get tenant error: %v", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	if t == nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "tenant not found"})
		return
	}
	writeJSON(w, http.StatusOK, toTenantResponse(t))
}

// UpdateTenant handles PUT /api/v1/tenants/:id.
func (h *Handlers) UpdateTenant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req UpdateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid json: " + err.Error()})
		return
	}

	var resources *tenant.Resources
	if req.Resources != nil {
		resources = &tenant.Resources{
			AgentCPU:      req.Resources.AgentCPU,
			AgentMemory:   req.Resources.AgentMemory,
			GatewayCPU:    req.Resources.GatewayCPU,
			GatewayMemory: req.Resources.GatewayMemory,
			WorkspaceSize: req.Resources.WorkspaceSize,
		}
	}

	t, err := h.svc.Update(r.Context(), id, tenant.UpdateRequest{
		DisplayName: req.DisplayName,
		Providers:   req.Providers,
		Agents:      req.Agents,
		Channels:    req.Channels,
		Resources:   resources,
	})
	if err != nil {
		log.Printf("update tenant error: %v", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, toTenantResponse(t))
}

// DeleteTenant handles DELETE /api/v1/tenants/:id.
func (h *Handlers) DeleteTenant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		log.Printf("delete tenant error: %v", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RestartTenant handles POST /api/v1/tenants/:id/restart.
func (h *Handlers) RestartTenant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Restart(r.Context(), id); err != nil {
		log.Printf("restart tenant error: %v", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "restarting"})
}

// HealthCheck handles GET /health.
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func toTenantResponse(t *tenant.Tenant) TenantResponse {
	return TenantResponse{
		ID:          t.ID,
		DisplayName: t.DisplayName,
		Namespace:   t.Namespace,
		Status:      t.Status,
		Resources: ResourcesResponse{
			AgentCPU:      t.Resources.AgentCPU,
			AgentMemory:   t.Resources.AgentMemory,
			GatewayCPU:    t.Resources.GatewayCPU,
			GatewayMemory: t.Resources.GatewayMemory,
			WorkspaceSize: t.Resources.WorkspaceSize,
		},
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
