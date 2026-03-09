package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// shortNames maps kubectl short names and common aliases to their plural resource names.
var shortNames = map[string]string{
	"deploy":         "deployments",
	"deployment":     "deployments",
	"sts":            "statefulsets",
	"statefulset":    "statefulsets",
	"svc":            "services",
	"service":        "services",
	"po":             "pods",
	"pod":            "pods",
	"cm":             "configmaps",
	"configmap":      "configmaps",
	"secret":         "secrets",
	"ing":            "ingresses",
	"ingress":        "ingresses",
	"ds":             "daemonsets",
	"daemonset":      "daemonsets",
	"rs":             "replicasets",
	"replicaset":     "replicasets",
	"cj":             "cronjobs",
	"cronjob":        "cronjobs",
	"job":            "jobs",
	"pvc":            "persistentvolumeclaims",
	"pv":             "persistentvolumes",
	"ns":             "namespaces",
	"namespace":      "namespaces",
	"sa":             "serviceaccounts",
	"serviceaccount": "serviceaccounts",
	"node":           "nodes",
	"no":             "nodes",
}

// ParseResourceArg splits "deploy/api" → ("deploy", "api") or "deploy" → ("deploy", "").
func ParseResourceArg(arg string) (resourceType, name string) {
	slog.Debug("parsing resource argument", "arg", arg)
	parts := strings.SplitN(arg, "/", 2)
	if len(parts) == 2 {
		slog.Debug("slash syntax detected", "resource_type", parts[0], "name", parts[1])
		return parts[0], parts[1]
	}
	slog.Debug("no slash - resource type only (list mode)", "resource_type", arg)
	return arg, ""
}

// ResolveGVR maps a resource type string (short name, plural, singular) to a GVR.
// Uses the discovery REST mapper first; falls back to the built-in short name table.
func ResolveGVR(ctx context.Context, mapper meta.RESTMapper, resourceType string) (schema.GroupVersionResource, error) {
	rt := strings.ToLower(resourceType)
	slog.Debug("resolving GVR", "input", resourceType, "normalized", rt)

	// Attempt 1: pass the string directly to the discovery mapper
	slog.Debug("attempt 1: direct discovery mapper lookup", "resource", rt)
	gvr, err := tryMapper(mapper, rt)
	if err == nil {
		slog.Debug("direct lookup succeeded", "gvr", gvr.String())
		return gvr, nil
	}
	slog.Debug("direct lookup failed", "err", err)

	// Attempt 2: expand via built-in short name / alias table
	if plural, ok := shortNames[rt]; ok {
		slog.Debug("attempt 2: short name table expansion", "short", rt, "plural", plural)
		gvr, err2 := tryMapper(mapper, plural)
		if err2 == nil {
			slog.Debug("short name expansion succeeded", "gvr", gvr.String())
			return gvr, nil
		}
		slog.Debug("short name expansion failed", "err", err2)
	} else {
		slog.Debug("attempt 2: no entry in short name table for", "resource", rt)
	}

	// Attempt 3: rt might itself be the plural value already in the table
	slog.Debug("attempt 3: checking if input matches a known plural directly")
	for short, plural := range shortNames {
		if plural == rt {
			slog.Debug("found as plural match via table reverse scan", "short", short, "plural", plural)
			gvr, err2 := tryMapper(mapper, plural)
			if err2 == nil {
				slog.Debug("reverse plural lookup succeeded", "gvr", gvr.String())
				return gvr, nil
			}
		}
	}

	slog.Debug("all GVR resolution attempts exhausted", "resource", rt)
	return schema.GroupVersionResource{}, fmt.Errorf("resolve GVR for %q: %w", resourceType, err)
}

// tryMapper attempts to resolve a resource string using the REST mapper.
func tryMapper(mapper meta.RESTMapper, resource string) (schema.GroupVersionResource, error) {
	slog.Debug("calling REST mapper ResourceFor", "resource", resource)
	gvr, err := mapper.ResourceFor(schema.GroupVersionResource{Resource: resource})
	if err != nil {
		slog.Debug("REST mapper returned error", "resource", resource, "err", err)
	}
	return gvr, err
}
