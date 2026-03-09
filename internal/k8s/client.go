package k8s

import (
	"context"
	"fmt"
	"log/slog"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

// ResourceFetcher abstracts Kubernetes resource retrieval (enables mocking in tests).
type ResourceFetcher interface {
	GetResource(ctx context.Context, gvr schema.GroupVersionResource, name, namespace string) (*unstructured.Unstructured, error)
	ListResources(ctx context.Context, gvr schema.GroupVersionResource, namespace string) (*unstructured.UnstructuredList, error)
}

// Client wraps a dynamic Kubernetes client with a request-scoped cache.
type Client struct {
	dynamic   dynamic.Interface
	discovery discovery.DiscoveryInterface
	cache     map[string]*unstructured.Unstructured
	mapper    meta.RESTMapper
}

// NewClient builds a Client from a kubeconfig path (empty = default location / in-cluster).
func NewClient(kubeconfig string) (*Client, error) {
	slog.Debug("initializing Kubernetes client", "kubeconfig", kubeconfig)

	var cfg *rest.Config
	var err error

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfig != "" {
		slog.Debug("using explicit kubeconfig path", "path", kubeconfig)
		loadingRules.ExplicitPath = kubeconfig
	} else {
		slog.Debug("no explicit kubeconfig; using default loading rules (KUBECONFIG env / ~/.kube/config)")
	}

	cfg, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		slog.Debug("kubeconfig load failed, falling back to in-cluster config", "err", err)
		cfg, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("build kubeconfig: %w", err)
		}
		slog.Debug("using in-cluster config", "host", cfg.Host)
	} else {
		slog.Debug("kubeconfig loaded successfully", "host", cfg.Host)
	}

	dynClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create dynamic client: %w", err)
	}
	slog.Debug("dynamic client created")

	discClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create discovery client: %w", err)
	}
	slog.Debug("discovery client created")

	slog.Debug("fetching API group resources from cluster (this calls the discovery API)")
	gr, err := restmapper.GetAPIGroupResources(discClient)
	if err != nil {
		return nil, fmt.Errorf("get API group resources: %w", err)
	}
	slog.Debug("API group resources fetched", "group_count", len(gr))

	mapper := restmapper.NewDiscoveryRESTMapper(gr)
	slog.Debug("REST mapper built from discovery data")

	return &Client{
		dynamic:   dynClient,
		discovery: discClient,
		cache:     make(map[string]*unstructured.Unstructured),
		mapper:    mapper,
	}, nil
}

// GetResource retrieves a single resource, using a request-scoped in-memory cache.
func (c *Client) GetResource(ctx context.Context, gvr schema.GroupVersionResource, name, namespace string) (*unstructured.Unstructured, error) {
	key := fmt.Sprintf("%s/%s/%s/%s/%s", gvr.Group, gvr.Version, gvr.Resource, namespace, name)
	slog.Debug("GetResource called", "gvr", gvr.String(), "name", name, "namespace", namespace, "cache_key", key)

	if cached, ok := c.cache[key]; ok {
		slog.Debug("cache hit - skipping API call", "key", key)
		return cached, nil
	}
	slog.Debug("cache miss - issuing GET to Kubernetes API", "key", key)

	var ri dynamic.ResourceInterface
	if namespace != "" {
		slog.Debug("using namespaced resource interface", "namespace", namespace)
		ri = c.dynamic.Resource(gvr).Namespace(namespace)
	} else {
		slog.Debug("using cluster-scoped resource interface")
		ri = c.dynamic.Resource(gvr)
	}

	slog.Debug("sending GET request to API server", "resource", gvr.Resource, "name", name)
	obj, err := ri.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		slog.Debug("GET request failed", "err", err)
		return nil, fmt.Errorf("get %s/%s in %q: %w", gvr.Resource, name, namespace, err)
	}

	slog.Debug("GET request successful, storing in cache",
		"resource_version", obj.GetResourceVersion(),
		"uid", obj.GetUID(),
	)
	c.cache[key] = obj
	return obj, nil
}

// ListResources retrieves all resources of a given GVR in a namespace.
func (c *Client) ListResources(ctx context.Context, gvr schema.GroupVersionResource, namespace string) (*unstructured.UnstructuredList, error) {
	slog.Debug("ListResources called", "gvr", gvr.String(), "namespace", namespace)

	var ri dynamic.ResourceInterface
	if namespace != "" {
		slog.Debug("using namespaced resource interface for list", "namespace", namespace)
		ri = c.dynamic.Resource(gvr).Namespace(namespace)
	} else {
		slog.Debug("listing across all namespaces (cluster-scoped or -A flag)")
		ri = c.dynamic.Resource(gvr)
	}

	slog.Debug("sending LIST request to API server", "resource", gvr.Resource)
	list, err := ri.List(ctx, metav1.ListOptions{})
	if err != nil {
		slog.Debug("LIST request failed", "err", err)
		return nil, fmt.Errorf("list %s in %q: %w", gvr.Resource, namespace, err)
	}

	slog.Debug("LIST request successful", "item_count", len(list.Items))
	return list, nil
}

// Mapper exposes the REST mapper for GVR resolution.
func (c *Client) Mapper() meta.RESTMapper {
	return c.mapper
}
