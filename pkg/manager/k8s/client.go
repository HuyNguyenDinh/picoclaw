package k8s

import (
	"fmt"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps Kubernetes client-go for the manager.
type Client struct {
	Clientset     kubernetes.Interface
	DynamicClient dynamic.Interface
	Config        *rest.Config
}

// NewClient creates a K8s client using in-cluster config, falling back to kubeconfig.
func NewClient(kubeconfig string) (*Client, error) {
	cfg, err := buildConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("build k8s config: %w", err)
	}

	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create clientset: %w", err)
	}

	dc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create dynamic client: %w", err)
	}

	return &Client{
		Clientset:     cs,
		DynamicClient: dc,
		Config:        cfg,
	}, nil
}

func buildConfig(kubeconfig string) (*rest.Config, error) {
	// Try in-cluster first.
	cfg, err := rest.InClusterConfig()
	if err == nil {
		return cfg, nil
	}

	// Fall back to kubeconfig file.
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	// Try default kubeconfig location.
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		rules, &clientcmd.ConfigOverrides{},
	).ClientConfig()
}
