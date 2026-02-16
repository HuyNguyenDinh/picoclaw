package api

import "time"

// CreateTenantRequest is the JSON body for POST /api/v1/tenants.
type CreateTenantRequest struct {
	TenantID    string                 `json:"tenant_id"`
	DisplayName string                 `json:"display_name"`
	Providers   map[string]interface{} `json:"providers"`
	Agents      map[string]interface{} `json:"agents"`
	Channels    map[string]interface{} `json:"channels"`
	Resources   *ResourcesRequest      `json:"resources,omitempty"`
}

// UpdateTenantRequest is the JSON body for PUT /api/v1/tenants/:id.
type UpdateTenantRequest struct {
	DisplayName string                 `json:"display_name,omitempty"`
	Providers   map[string]interface{} `json:"providers,omitempty"`
	Agents      map[string]interface{} `json:"agents,omitempty"`
	Channels    map[string]interface{} `json:"channels,omitempty"`
	Resources   *ResourcesRequest      `json:"resources,omitempty"`
}

// ResourcesRequest mirrors tenant.Resources for API input.
type ResourcesRequest struct {
	AgentCPU      string `json:"agent_cpu,omitempty"`
	AgentMemory   string `json:"agent_memory,omitempty"`
	GatewayCPU    string `json:"gateway_cpu,omitempty"`
	GatewayMemory string `json:"gateway_memory,omitempty"`
	WorkspaceSize string `json:"workspace_size,omitempty"`
}

// TenantResponse is the JSON response for tenant operations.
type TenantResponse struct {
	ID          string            `json:"id"`
	DisplayName string            `json:"display_name"`
	Namespace   string            `json:"namespace"`
	Status      string            `json:"status"`
	Resources   ResourcesResponse `json:"resources"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// ResourcesResponse mirrors tenant.Resources for API output.
type ResourcesResponse struct {
	AgentCPU      string `json:"agent_cpu"`
	AgentMemory   string `json:"agent_memory"`
	GatewayCPU    string `json:"gateway_cpu"`
	GatewayMemory string `json:"gateway_memory"`
	WorkspaceSize string `json:"workspace_size"`
}

// ErrorResponse is a standard error payload.
type ErrorResponse struct {
	Error string `json:"error"`
}
