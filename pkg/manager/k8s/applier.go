package k8s

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

// Applier applies and deletes K8s manifests using server-side apply.
type Applier struct {
	client *Client
}

// NewApplier creates a new Applier.
func NewApplier(client *Client) *Applier {
	return &Applier{client: client}
}

// Apply parses multi-document YAML and applies each resource via server-side apply.
func (a *Applier) Apply(ctx context.Context, manifests []byte) error {
	objects, err := parseYAMLDocuments(manifests)
	if err != nil {
		return fmt.Errorf("parse manifests: %w", err)
	}

	mapper, err := a.buildMapper()
	if err != nil {
		return fmt.Errorf("build REST mapper: %w", err)
	}

	for _, obj := range objects {
		gvk := obj.GroupVersionKind()
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return fmt.Errorf("find REST mapping for %s: %w", gvk, err)
		}

		var dr dynamic.ResourceInterface
		if mapping.Scope.Name() == "namespace" {
			dr = a.client.DynamicClient.Resource(mapping.Resource).Namespace(obj.GetNamespace())
		} else {
			dr = a.client.DynamicClient.Resource(mapping.Resource)
		}

		data, err := obj.MarshalJSON()
		if err != nil {
			return fmt.Errorf("marshal %s/%s: %w", obj.GetKind(), obj.GetName(), err)
		}

		_, err = dr.Patch(ctx, obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
			FieldManager: "picoclaw-manager",
		})
		if err != nil {
			return fmt.Errorf("apply %s/%s: %w", obj.GetKind(), obj.GetName(), err)
		}
	}
	return nil
}

// DeleteNamespace deletes a namespace and all its resources.
func (a *Applier) DeleteNamespace(ctx context.Context, namespace string) error {
	err := a.client.Clientset.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("delete namespace %s: %w", namespace, err)
	}
	return nil
}

// RestartDeployment triggers a rollout restart by patching the deployment annotation.
func (a *Applier) RestartDeployment(ctx context.Context, namespace, name string) error {
	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`,
		metav1.Now().Format("2006-01-02T15:04:05Z07:00"))

	_, err := a.client.Clientset.AppsV1().Deployments(namespace).Patch(
		ctx, name, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("restart deployment %s/%s: %w", namespace, name, err)
	}
	return nil
}

// GetDeploymentStatus returns the ready/total replicas for a deployment.
func (a *Applier) GetDeploymentStatus(ctx context.Context, namespace, name string) (ready, total int32, err error) {
	dep, err := a.client.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return 0, 0, fmt.Errorf("get deployment %s/%s: %w", namespace, name, err)
	}
	return dep.Status.ReadyReplicas, dep.Status.Replicas, nil
}

func (a *Applier) buildMapper() (*restmapper.DeferredDiscoveryRESTMapper, error) {
	dc, err := discovery.NewDiscoveryClientForConfig(a.client.Config)
	if err != nil {
		return nil, err
	}
	return restmapper.NewDeferredDiscoveryRESTMapper(
		&cachedDiscovery{DiscoveryInterface: dc},
	), nil
}

func parseYAMLDocuments(data []byte) ([]*unstructured.Unstructured, error) {
	var objects []*unstructured.Unstructured
	reader := yamlutil.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))

	for {
		doc, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		doc = bytes.TrimSpace(doc)
		if len(doc) == 0 || strings.TrimSpace(string(doc)) == "---" {
			continue
		}

		jsonData, err := yamlutil.ToJSON(doc)
		if err != nil {
			return nil, fmt.Errorf("convert YAML to JSON: %w", err)
		}

		obj := &unstructured.Unstructured{}
		if err := obj.UnmarshalJSON(jsonData); err != nil {
			return nil, fmt.Errorf("unmarshal JSON: %w", err)
		}

		if obj.GetKind() == "" {
			continue
		}
		objects = append(objects, obj)
	}
	return objects, nil
}

// cachedDiscovery wraps a DiscoveryInterface to satisfy the CachedDiscoveryInterface.
type cachedDiscovery struct {
	discovery.DiscoveryInterface
}

func (c *cachedDiscovery) Fresh() bool                         { return true }
func (c *cachedDiscovery) Invalidate()                         {}
func (c *cachedDiscovery) ServerGroups() (*metav1.APIGroupList, error) {
	return c.DiscoveryInterface.ServerGroups()
}
func (c *cachedDiscovery) ServerResourcesForGroupVersion(groupVersion string) (*metav1.APIResourceList, error) {
	return c.DiscoveryInterface.ServerResourcesForGroupVersion(groupVersion)
}
func (c *cachedDiscovery) ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
	return c.DiscoveryInterface.ServerGroupsAndResources()
}
func (c *cachedDiscovery) ServerPreferredResources() ([]*metav1.APIResourceList, error) {
	return c.DiscoveryInterface.ServerPreferredResources()
}
func (c *cachedDiscovery) ServerPreferredNamespacedResources() ([]*metav1.APIResourceList, error) {
	return c.DiscoveryInterface.ServerPreferredNamespacedResources()
}

