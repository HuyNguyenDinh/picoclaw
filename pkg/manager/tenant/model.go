package tenant

import (
	"encoding/json"
	"time"
)

// Tenant represents a tenant record in the database.
type Tenant struct {
	ID          string          `json:"id"`
	DisplayName string          `json:"display_name"`
	Namespace   string          `json:"namespace"`
	ConfigJSON  json.RawMessage `json:"config_json"`
	Resources   Resources       `json:"resources"`
	Status      string          `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// Resources defines resource limits for a tenant's workloads.
type Resources struct {
	AgentCPU      string `json:"agent_cpu"`
	AgentMemory   string `json:"agent_memory"`
	GatewayCPU    string `json:"gateway_cpu"`
	GatewayMemory string `json:"gateway_memory"`
	WorkspaceSize string `json:"workspace_size"`
}

// TenantVars holds template variables for rendering K8s manifests.
type TenantVars struct {
	TenantID      string
	Namespace     string
	ConfigJSON    string
	Image         string
	AgentReplicas int
	AgentCPU      string
	AgentMemory   string
	GatewayCPU    string
	GatewayMemory string
	WorkspaceSize string
	Labels        map[string]string
}

// DefaultResources returns sensible defaults for tenant resources.
func DefaultResources() Resources {
	return Resources{
		AgentCPU:      "500m",
		AgentMemory:   "1Gi",
		GatewayCPU:    "250m",
		GatewayMemory: "512Mi",
		WorkspaceSize: "500Mi",
	}
}

// ToVars converts a Tenant into TenantVars for template rendering.
func (t *Tenant) ToVars(image string) TenantVars {
	return TenantVars{
		TenantID:      t.ID,
		Namespace:     t.Namespace,
		ConfigJSON:    string(t.ConfigJSON),
		Image:         image,
		AgentReplicas: 1,
		AgentCPU:      t.Resources.AgentCPU,
		AgentMemory:   t.Resources.AgentMemory,
		GatewayCPU:    t.Resources.GatewayCPU,
		GatewayMemory: t.Resources.GatewayMemory,
		WorkspaceSize: t.Resources.WorkspaceSize,
		Labels: map[string]string{
			"app.kubernetes.io/managed-by": "picoclaw-manager",
			"picoclaw.io/tenant":           t.ID,
		},
	}
}
