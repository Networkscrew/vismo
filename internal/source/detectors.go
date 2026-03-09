package source

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Analyze runs all detectors against obj and returns the combined result.
func Analyze(obj *unstructured.Unstructured) *AnalysisResult {
	result := &AnalysisResult{}
	detectors := []func(*unstructured.Unstructured) *SourceInfo{
		DetectHelm,
		DetectArgoCD,
		DetectKustomize,
		DetectFlux,
		DetectKubectl,
	}
	for _, d := range detectors {
		if si := d(obj); si != nil {
			result.Sources = append(result.Sources, *si)
		}
	}
	return result
}

// DetectHelm checks for Helm management markers.
func DetectHelm(obj *unstructured.Unstructured) *SourceInfo {
	labels := obj.GetLabels()
	annotations := obj.GetAnnotations()

	if labels["app.kubernetes.io/managed-by"] != "Helm" {
		return nil
	}

	si := &SourceInfo{
		Type:    SourceHelm,
		Details: make(map[string]string),
	}

	if v, ok := annotations["meta.helm.sh/release-name"]; ok {
		si.Name = v
		si.Details["release-name"] = v
	}
	if v, ok := annotations["meta.helm.sh/release-namespace"]; ok {
		si.Details["release-namespace"] = v
	}
	if v, ok := annotations["helm.sh/chart"]; ok {
		si.Version = v
		si.Details["chart"] = v
	}

	return si
}

// DetectArgoCD checks for ArgoCD management markers.
func DetectArgoCD(obj *unstructured.Unstructured) *SourceInfo {
	labels := obj.GetLabels()
	annotations := obj.GetAnnotations()

	appName, hasLabel := labels["argocd.argoproj.io/app-name"]
	_, hasTracking := annotations["argocd.argoproj.io/tracking-id"]

	if !hasLabel && !hasTracking {
		return nil
	}

	si := &SourceInfo{
		Type:    SourceArgoCD,
		Name:    appName,
		Details: make(map[string]string),
	}
	if hasLabel {
		si.Details["app-name"] = appName
	}
	if v, ok := annotations["argocd.argoproj.io/tracking-id"]; ok {
		si.Details["tracking-id"] = v
	}
	return si
}

// DetectKustomize checks for Kustomize management markers.
func DetectKustomize(obj *unstructured.Unstructured) *SourceInfo {
	labels := obj.GetLabels()
	annotations := obj.GetAnnotations()

	managedBy := labels["app.kubernetes.io/managed-by"]
	hasFluxKustomize := false
	for k := range annotations {
		if len(k) > 30 && k[:31] == "kustomize.toolkit.fluxcd.io/" {
			hasFluxKustomize = true
			break
		}
	}

	if managedBy != "kustomize" && !hasFluxKustomize {
		return nil
	}

	si := &SourceInfo{
		Type:    SourceKustomize,
		Details: make(map[string]string),
	}
	for k, v := range annotations {
		if len(k) > 27 && k[:28] == "kustomize.toolkit.fluxcd.io" {
			si.Details[k] = v
		}
	}
	return si
}

// DetectFlux checks for Flux CD management markers.
func DetectFlux(obj *unstructured.Unstructured) *SourceInfo {
	labels := obj.GetLabels()

	kustomizeName := labels["kustomize.toolkit.fluxcd.io/name"]
	helmName := labels["helm.toolkit.fluxcd.io/name"]

	if kustomizeName == "" && helmName == "" {
		return nil
	}

	si := &SourceInfo{
		Type:    SourceFlux,
		Details: make(map[string]string),
	}
	if kustomizeName != "" {
		si.Name = kustomizeName
		si.Details["kustomize-name"] = kustomizeName
		if ns := labels["kustomize.toolkit.fluxcd.io/namespace"]; ns != "" {
			si.Details["kustomize-namespace"] = ns
		}
	}
	if helmName != "" {
		si.Name = helmName
		si.Details["helm-name"] = helmName
		if ns := labels["helm.toolkit.fluxcd.io/namespace"]; ns != "" {
			si.Details["helm-namespace"] = ns
		}
	}
	return si
}

// DetectKubectl checks for kubectl last-applied-configuration annotation.
func DetectKubectl(obj *unstructured.Unstructured) *SourceInfo {
	annotations := obj.GetAnnotations()
	if _, ok := annotations["kubectl.kubernetes.io/last-applied-configuration"]; !ok {
		return nil
	}
	return &SourceInfo{
		Type:    SourceKubectl,
		Details: map[string]string{"source": "last-applied-configuration annotation"},
	}
}
