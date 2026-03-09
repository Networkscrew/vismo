package k8s

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// shortNames maps kubectl short names and common aliases to their plural resource names.
var shortNames = map[string]string{
	"deploy":      "deployments",
	"deployment":  "deployments",
	"sts":         "statefulsets",
	"statefulset": "statefulsets",
	"svc":         "services",
	"service":     "services",
	"po":          "pods",
	"pod":         "pods",
	"cm":          "configmaps",
	"configmap":   "configmaps",
	"secret":      "secrets",
	"ing":         "ingresses",
	"ingress":     "ingresses",
	"ds":          "daemonsets",
	"daemonset":   "daemonsets",
	"rs":          "replicasets",
	"replicaset":  "replicasets",
	"cj":          "cronjobs",
	"cronjob":     "cronjobs",
	"job":         "jobs",
	"pvc":         "persistentvolumeclaims",
	"pv":          "persistentvolumes",
	"ns":          "namespaces",
	"namespace":   "namespaces",
	"sa":          "serviceaccounts",
	"serviceaccount": "serviceaccounts",
	"node":        "nodes",
	"no":          "nodes",
}

// ParseResourceArg splits "deploy/api" → ("deploy", "api") or "deploy" → ("deploy", "").
// Also accepts slash syntax: "deployments/api".
func ParseResourceArg(arg string) (resourceType, name string) {
	parts := strings.SplitN(arg, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return arg, ""
}

// ResolveGVR maps a resource type string (short name, plural, singular) to a GVR.
// Uses the discovery REST mapper first; falls back to the built-in short name table.
func ResolveGVR(ctx context.Context, mapper meta.RESTMapper, resourceType string) (schema.GroupVersionResource, error) {
	// Normalize: lowercase
	rt := strings.ToLower(resourceType)

	// Try discovery mapper with the resource type as-is (plural preferred)
	gvr, err := tryMapper(mapper, rt)
	if err == nil {
		return gvr, nil
	}

	// Try expanding short names → plural via built-in table
	if plural, ok := shortNames[rt]; ok {
		gvr, err2 := tryMapper(mapper, plural)
		if err2 == nil {
			return gvr, nil
		}
	}

	// Try built-in table reversed: if rt is already plural it might be in table values
	for _, plural := range shortNames {
		if plural == rt {
			gvr, err2 := tryMapper(mapper, plural)
			if err2 == nil {
				return gvr, nil
			}
		}
	}

	return schema.GroupVersionResource{}, fmt.Errorf("resolve GVR for %q: %w", resourceType, err)
}

// tryMapper attempts to resolve a resource string using the REST mapper.
func tryMapper(mapper meta.RESTMapper, resource string) (schema.GroupVersionResource, error) {
	return mapper.ResourceFor(schema.GroupVersionResource{Resource: resource})
}
