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
	var cfg *rest.Config
	var err error

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfig != "" {
		loadingRules.ExplicitPath = kubeconfig
	}
	cfg, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		// Fall back to in-cluster config
		cfg, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("build kubeconfig: %w", err)
		}
	}

	dynClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create dynamic client: %w", err)
	}

	discClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create discovery client: %w", err)
	}

	gr, err := restmapper.GetAPIGroupResources(discClient)
	if err != nil {
		return nil, fmt.Errorf("get API group resources: %w", err)
	}
	mapper := restmapper.NewDiscoveryRESTMapper(gr)

	return &Client{
		dynamic:   dynClient,
		discovery: discClient,
		cache:     make(map[string]*unstructured.Unstructured),
		mapper:    mapper,
	}, nil
}

// GetResource retrieves a single resource, using a request-scoped in-memory cache.
func (c *Client) GetResource(ctx context.Context, gvr schema.GroupVersionResource, name, namespace string) (*unstructured.Unstructured, error) {
	key := fmt.Sprintf("%s/%s/%s/%s", gvr.Group, gvr.Version, gvr.Resource, namespace+"/"+name)
	if cached, ok := c.cache[key]; ok {
		slog.Debug("cache hit", "key", key)
		return cached, nil
	}

	var ri dynamic.ResourceInterface
	if namespace != "" {
		ri = c.dynamic.Resource(gvr).Namespace(namespace)
	} else {
		ri = c.dynamic.Resource(gvr)
	}

	obj, err := ri.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get %s/%s in %q: %w", gvr.Resource, name, namespace, err)
	}

	c.cache[key] = obj
	return obj, nil
}

// ListResources retrieves all resources of a given GVR in a namespace.
func (c *Client) ListResources(ctx context.Context, gvr schema.GroupVersionResource, namespace string) (*unstructured.UnstructuredList, error) {
	var ri dynamic.ResourceInterface
	if namespace != "" {
		ri = c.dynamic.Resource(gvr).Namespace(namespace)
	} else {
		ri = c.dynamic.Resource(gvr)
	}

	list, err := ri.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list %s in %q: %w", gvr.Resource, namespace, err)
	}
	return list, nil
}

// Mapper exposes the REST mapper for GVR resolution.
func (c *Client) Mapper() meta.RESTMapper {
	return c.mapper
}
