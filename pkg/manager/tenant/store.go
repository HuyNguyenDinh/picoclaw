package tenant

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

const createTableSQL = `
CREATE TABLE IF NOT EXISTS tenants (
    id           TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    namespace    TEXT NOT NULL UNIQUE,
    config_json  JSONB NOT NULL,
    resources    JSONB NOT NULL,
    status       TEXT NOT NULL DEFAULT 'provisioning',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`

// Store provides PostgreSQL persistence for tenants.
type Store struct {
	db *sql.DB
}

// NewStore creates a new Store and initializes the schema.
func NewStore(db *sql.DB) (*Store, error) {
	if _, err := db.Exec(createTableSQL); err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}
	return &Store{db: db}, nil
}

// Create inserts a new tenant record.
func (s *Store) Create(t *Tenant) error {
	resourcesJSON, err := json.Marshal(t.Resources)
	if err != nil {
		return fmt.Errorf("marshal resources: %w", err)
	}
	_, err = s.db.Exec(`
		INSERT INTO tenants (id, display_name, namespace, config_json, resources, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		t.ID, t.DisplayName, t.Namespace, t.ConfigJSON, resourcesJSON, t.Status, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert tenant: %w", err)
	}
	return nil
}

// Get retrieves a tenant by ID.
func (s *Store) Get(id string) (*Tenant, error) {
	t := &Tenant{}
	var resourcesJSON []byte
	err := s.db.QueryRow(`
		SELECT id, display_name, namespace, config_json, resources, status, created_at, updated_at
		FROM tenants WHERE id = $1`, id,
	).Scan(&t.ID, &t.DisplayName, &t.Namespace, &t.ConfigJSON, &resourcesJSON, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get tenant: %w", err)
	}
	if err := json.Unmarshal(resourcesJSON, &t.Resources); err != nil {
		return nil, fmt.Errorf("unmarshal resources: %w", err)
	}
	return t, nil
}

// List returns all tenants.
func (s *Store) List() ([]Tenant, error) {
	rows, err := s.db.Query(`
		SELECT id, display_name, namespace, config_json, resources, status, created_at, updated_at
		FROM tenants ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []Tenant
	for rows.Next() {
		var t Tenant
		var resourcesJSON []byte
		if err := rows.Scan(&t.ID, &t.DisplayName, &t.Namespace, &t.ConfigJSON, &resourcesJSON, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan tenant: %w", err)
		}
		if err := json.Unmarshal(resourcesJSON, &t.Resources); err != nil {
			return nil, fmt.Errorf("unmarshal resources: %w", err)
		}
		tenants = append(tenants, t)
	}
	return tenants, rows.Err()
}

// Update modifies an existing tenant's config, resources, and status.
func (s *Store) Update(t *Tenant) error {
	resourcesJSON, err := json.Marshal(t.Resources)
	if err != nil {
		return fmt.Errorf("marshal resources: %w", err)
	}
	t.UpdatedAt = time.Now()
	res, err := s.db.Exec(`
		UPDATE tenants SET display_name=$1, config_json=$2, resources=$3, status=$4, updated_at=$5
		WHERE id=$6`,
		t.DisplayName, t.ConfigJSON, resourcesJSON, t.Status, t.UpdatedAt, t.ID,
	)
	if err != nil {
		return fmt.Errorf("update tenant: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("tenant %q not found", t.ID)
	}
	return nil
}

// Delete removes a tenant record by ID.
func (s *Store) Delete(id string) error {
	res, err := s.db.Exec(`DELETE FROM tenants WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete tenant: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("tenant %q not found", id)
	}
	return nil
}
