package tenant

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	k8sclient "github.com/sipeed/picoclaw/pkg/manager/k8s"
	"github.com/sipeed/picoclaw/pkg/manager/templates"
)

var dnsNameRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9\-]{0,61}[a-z0-9])?$`)

// Service orchestrates tenant lifecycle operations.
type Service struct {
	store    *Store
	renderer *templates.Renderer
	applier  *k8sclient.Applier
	image    string
}

// NewService creates a tenant service.
func NewService(store *Store, renderer *templates.Renderer, applier *k8sclient.Applier, image string) *Service {
	return &Service{
		store:    store,
		renderer: renderer,
		applier:  applier,
		image:    image,
	}
}

// CreateRequest holds the parameters for creating a tenant.
type CreateRequest struct {
	TenantID    string                 `json:"tenant_id"`
	DisplayName string                 `json:"display_name"`
	Providers   map[string]interface{} `json:"providers"`
	Agents      map[string]interface{} `json:"agents"`
	Channels    map[string]interface{} `json:"channels"`
	Resources   *Resources             `json:"resources,omitempty"`
}

// UpdateRequest holds the parameters for updating a tenant.
type UpdateRequest struct {
	DisplayName string                 `json:"display_name,omitempty"`
	Providers   map[string]interface{} `json:"providers,omitempty"`
	Agents      map[string]interface{} `json:"agents,omitempty"`
	Channels    map[string]interface{} `json:"channels,omitempty"`
	Resources   *Resources             `json:"resources,omitempty"`
}

// Create provisions a new tenant: validate → render → apply → store.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Tenant, error) {
	if err := validateTenantID(req.TenantID); err != nil {
		return nil, err
	}
	if req.DisplayName == "" {
		return nil, fmt.Errorf("display_name is required")
	}

	existing, err := s.store.Get(req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("check existing: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("tenant %q already exists", req.TenantID)
	}

	configJSON, err := buildConfigJSON(req.Providers, req.Agents, req.Channels)
	if err != nil {
		return nil, fmt.Errorf("build config: %w", err)
	}

	resources := DefaultResources()
	if req.Resources != nil {
		resources = mergeResources(resources, *req.Resources)
	}

	now := time.Now()
	t := &Tenant{
		ID:          req.TenantID,
		DisplayName: req.DisplayName,
		Namespace:   "picoclaw-tenant-" + req.TenantID,
		ConfigJSON:  configJSON,
		Resources:   resources,
		Status:      "provisioning",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	vars := t.ToVars(s.image)
	manifests, err := s.renderer.RenderAll(vars)
	if err != nil {
		return nil, fmt.Errorf("render templates: %w", err)
	}

	if err := s.applier.Apply(ctx, manifests); err != nil {
		return nil, fmt.Errorf("apply manifests: %w", err)
	}

	if err := s.store.Create(t); err != nil {
		// Best-effort cleanup: delete the namespace if DB insert fails.
		_ = s.applier.DeleteNamespace(ctx, t.Namespace)
		return nil, fmt.Errorf("store tenant: %w", err)
	}

	t.Status = "active"
	_ = s.store.Update(t)

	return t, nil
}

// Get retrieves a tenant with optional K8s status enrichment.
func (s *Service) Get(ctx context.Context, id string) (*Tenant, error) {
	t, err := s.store.Get(id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, nil
	}
	return t, nil
}

// List returns all tenants.
func (s *Service) List(ctx context.Context) ([]Tenant, error) {
	return s.store.List()
}

// Update modifies a tenant's config and re-applies manifests.
func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (*Tenant, error) {
	t, err := s.store.Get(id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, fmt.Errorf("tenant %q not found", id)
	}

	if req.DisplayName != "" {
		t.DisplayName = req.DisplayName
	}

	if req.Providers != nil || req.Agents != nil || req.Channels != nil {
		// Rebuild config from existing + updates.
		var existing map[string]interface{}
		if err := json.Unmarshal(t.ConfigJSON, &existing); err != nil {
			return nil, fmt.Errorf("unmarshal existing config: %w", err)
		}
		if req.Providers != nil {
			existing["providers"] = req.Providers
		}
		if req.Agents != nil {
			existing["agents"] = req.Agents
		}
		if req.Channels != nil {
			existing["channels"] = req.Channels
		}
		configJSON, err := json.MarshalIndent(existing, "    ", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal config: %w", err)
		}
		t.ConfigJSON = configJSON
	}

	if req.Resources != nil {
		t.Resources = mergeResources(t.Resources, *req.Resources)
	}

	vars := t.ToVars(s.image)
	manifests, err := s.renderer.RenderAll(vars)
	if err != nil {
		return nil, fmt.Errorf("render templates: %w", err)
	}

	if err := s.applier.Apply(ctx, manifests); err != nil {
		return nil, fmt.Errorf("apply manifests: %w", err)
	}

	if err := s.store.Update(t); err != nil {
		return nil, fmt.Errorf("store update: %w", err)
	}

	return t, nil
}

// Delete removes a tenant and its namespace.
func (s *Service) Delete(ctx context.Context, id string) error {
	t, err := s.store.Get(id)
	if err != nil {
		return err
	}
	if t == nil {
		return fmt.Errorf("tenant %q not found", id)
	}

	if err := s.applier.DeleteNamespace(ctx, t.Namespace); err != nil {
		return fmt.Errorf("delete namespace: %w", err)
	}

	return s.store.Delete(id)
}

// Restart triggers a rollout restart for both agent and gateway deployments.
func (s *Service) Restart(ctx context.Context, id string) error {
	t, err := s.store.Get(id)
	if err != nil {
		return err
	}
	if t == nil {
		return fmt.Errorf("tenant %q not found", id)
	}

	if err := s.applier.RestartDeployment(ctx, t.Namespace, "picoclaw-agent"); err != nil {
		return fmt.Errorf("restart agent: %w", err)
	}
	if err := s.applier.RestartDeployment(ctx, t.Namespace, "picoclaw-gateway"); err != nil {
		return fmt.Errorf("restart gateway: %w", err)
	}
	return nil
}

func validateTenantID(id string) error {
	if id == "" {
		return fmt.Errorf("tenant_id is required")
	}
	if len(id) > 53 { // 63 - len("picoclaw-tenant-")
		return fmt.Errorf("tenant_id too long (max 53 characters)")
	}
	if !dnsNameRegex.MatchString(id) {
		return fmt.Errorf("tenant_id must be a valid DNS subdomain (lowercase alphanumeric and hyphens)")
	}
	return nil
}

func buildConfigJSON(providers, agents, channels map[string]interface{}) (json.RawMessage, error) {
	config := map[string]interface{}{
		"gateway": map[string]interface{}{
			"host": "0.0.0.0",
			"port": 18790,
		},
	}
	if providers != nil {
		config["providers"] = providers
	}
	if agents != nil {
		config["agents"] = agents
	}
	if channels != nil {
		config["channels"] = channels
	}
	data, err := json.MarshalIndent(config, "    ", "  ")
	if err != nil {
		return nil, err
	}
	return data, nil
}

func mergeResources(base, override Resources) Resources {
	if override.AgentCPU != "" {
		base.AgentCPU = override.AgentCPU
	}
	if override.AgentMemory != "" {
		base.AgentMemory = override.AgentMemory
	}
	if override.GatewayCPU != "" {
		base.GatewayCPU = override.GatewayCPU
	}
	if override.GatewayMemory != "" {
		base.GatewayMemory = override.GatewayMemory
	}
	if override.WorkspaceSize != "" {
		base.WorkspaceSize = override.WorkspaceSize
	}
	return base
}
