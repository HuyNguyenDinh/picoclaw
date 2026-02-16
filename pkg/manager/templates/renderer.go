package templates

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// Renderer loads and executes Go text/templates for K8s manifests.
type Renderer struct {
	templates *template.Template
}

// templateNames is the ordered list of templates to render for a tenant.
var templateNames = []string{
	"namespace.yaml.tmpl",
	"configmap.yaml.tmpl",
	"pvc.yaml.tmpl",
	"rbac.yaml.tmpl",
	"agent-deployment.yaml.tmpl",
	"gateway-deployment.yaml.tmpl",
	"gateway-service.yaml.tmpl",
}

// NewRenderer loads all .yaml.tmpl files from the given directory.
func NewRenderer(templateDir string) (*Renderer, error) {
	pattern := filepath.Join(templateDir, "*.yaml.tmpl")
	t, err := template.ParseGlob(pattern)
	if err != nil {
		return nil, fmt.Errorf("parse templates from %s: %w", templateDir, err)
	}
	return &Renderer{templates: t}, nil
}

// NewRendererFromDir loads templates from a directory, verifying it exists.
func NewRendererFromDir(dir string) (*Renderer, error) {
	if _, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("template directory %s: %w", dir, err)
	}
	return NewRenderer(dir)
}

// RenderAll renders all templates with the given data and returns
// the concatenated YAML documents separated by "---".
func (r *Renderer) RenderAll(vars any) ([]byte, error) {
	var buf bytes.Buffer
	for i, name := range templateNames {
		if i > 0 {
			buf.WriteString("---\n")
		}
		if err := r.templates.ExecuteTemplate(&buf, name, vars); err != nil {
			return nil, fmt.Errorf("render template %s: %w", name, err)
		}
	}
	return buf.Bytes(), nil
}

// RenderOne renders a single named template with the given data.
func (r *Renderer) RenderOne(name string, vars any) ([]byte, error) {
	var buf bytes.Buffer
	if err := r.templates.ExecuteTemplate(&buf, name, vars); err != nil {
		return nil, fmt.Errorf("render template %s: %w", name, err)
	}
	return buf.Bytes(), nil
}
