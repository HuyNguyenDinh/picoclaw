package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
	"github.com/sipeed/picoclaw/pkg/manager/api"
	k8sclient "github.com/sipeed/picoclaw/pkg/manager/k8s"
	"github.com/sipeed/picoclaw/pkg/manager/templates"
	"github.com/sipeed/picoclaw/pkg/manager/tenant"
)

func main() {
	dbURL := envOrDefault("DATABASE_URL", "postgres://postgres:test@localhost:5432/picoclaw?sslmode=disable")
	apiKey := envOrDefault("API_KEY", "test-api-key")
	listenAddr := envOrDefault("LISTEN_ADDR", ":8080")
	image := envOrDefault("PICOCLAW_IMAGE", "huy2408/picoclaw:latest")
	templateDir := envOrDefault("TEMPLATE_DIR", "k8s/base")
	kubeconfig := os.Getenv("KUBECONFIG")

	// Database.
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("ping database: %v", err)
	}
	log.Println("connected to database")

	store, err := tenant.NewStore(db)
	if err != nil {
		log.Fatalf("init tenant store: %v", err)
	}

	// Template renderer.
	renderer, err := templates.NewRendererFromDir(templateDir)
	if err != nil {
		log.Fatalf("init template renderer: %v", err)
	}

	// K8s client.
	k8sClient, err := k8sclient.NewClient(kubeconfig)
	if err != nil {
		log.Fatalf("init k8s client: %v", err)
	}
	applier := k8sclient.NewApplier(k8sClient)

	// Tenant service.
	svc := tenant.NewService(store, renderer, applier, image)

	// HTTP server.
	handler := api.NewServer(svc, apiKey)

	log.Printf("picoclaw-manager listening on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
	fmt.Println()
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
