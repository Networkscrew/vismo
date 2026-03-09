package source

import (
	"log/slog"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Analyze runs all detectors against obj and returns the combined result.
func Analyze(obj *unstructured.Unstructured) *AnalysisResult {
	slog.Debug("starting source analysis",
		"resource", obj.GetKind(),
		"name", obj.GetName(),
		"namespace", obj.GetNamespace(),
	)

	result := &AnalysisResult{}
	detectors := []struct {
		name string
		fn   func(*unstructured.Unstructured) *SourceInfo
	}{
		{"Helm", DetectHelm},
		{"ArgoCD", DetectArgoCD},
		{"Kustomize", DetectKustomize},
		{"Flux", DetectFlux},
		{"kubectl", DetectKubectl},
	}

	for _, d := range detectors {
		slog.Debug("running detector", "detector", d.name)
		si := d.fn(obj)
		if si != nil {
			slog.Debug("detector matched", "detector", d.name, "source_name", si.Name, "version", si.Version)
			result.Sources = append(result.Sources, *si)
		} else {
			slog.Debug("detector did not match", "detector", d.name)
		}
	}

	slog.Debug("source analysis complete", "sources_found", len(result.Sources))
	return result
}

// DetectHelm checks for Helm management markers.
func DetectHelm(obj *unstructured.Unstructured) *SourceInfo {
	labels := obj.GetLabels()
	annotations := obj.GetAnnotations()

	slog.Debug("DetectHelm: checking managed-by label", "value", labels["app.kubernetes.io/managed-by"])
	if labels["app.kubernetes.io/managed-by"] != "Helm" {
		slog.Debug("DetectHelm: not Helm-managed, skipping")
		return nil
	}

	si := &SourceInfo{
		Type:    SourceHelm,
		Details: make(map[string]string),
	}

	if v, ok := annotations["meta.helm.sh/release-name"]; ok {
		slog.Debug("DetectHelm: found release-name annotation", "release", v)
		si.Name = v
		si.Details["release-name"] = v
	}
	if v, ok := annotations["meta.helm.sh/release-namespace"]; ok {
		slog.Debug("DetectHelm: found release-namespace annotation", "ns", v)
		si.Details["release-namespace"] = v
	}
	if v, ok := annotations["helm.sh/chart"]; ok {
		slog.Debug("DetectHelm: found chart annotation", "chart", v)
		si.Version = v
		si.Details["chart"] = v
	}

	slog.Debug("DetectHelm: Helm source detected", "release", si.Name, "chart", si.Version)
	return si
}

// DetectArgoCD checks for ArgoCD management markers.
func DetectArgoCD(obj *unstructured.Unstructured) *SourceInfo {
	labels := obj.GetLabels()
	annotations := obj.GetAnnotations()

	appName, hasLabel := labels["argocd.argoproj.io/app-name"]
	trackingID, hasTracking := annotations["argocd.argoproj.io/tracking-id"]

	slog.Debug("DetectArgoCD: checking markers",
		"has_app_label", hasLabel,
		"app_name", appName,
		"has_tracking_annotation", hasTracking,
		"tracking_id", trackingID,
	)

	if !hasLabel && !hasTracking {
		slog.Debug("DetectArgoCD: no ArgoCD markers found")
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
	if hasTracking {
		si.Details["tracking-id"] = trackingID
	}

	slog.Debug("DetectArgoCD: ArgoCD source detected", "app", appName)
	return si
}

// DetectKustomize checks for Kustomize management markers.
func DetectKustomize(obj *unstructured.Unstructured) *SourceInfo {
	labels := obj.GetLabels()
	annotations := obj.GetAnnotations()

	managedBy := labels["app.kubernetes.io/managed-by"]
	slog.Debug("DetectKustomize: checking managed-by label", "value", managedBy)

	hasFluxKustomize := false
	for k := range annotations {
		if len(k) > 30 && k[:31] == "kustomize.toolkit.fluxcd.io/" {
			slog.Debug("DetectKustomize: found Flux Kustomize annotation", "key", k)
			hasFluxKustomize = true
			break
		}
	}

	if managedBy != "kustomize" && !hasFluxKustomize {
		slog.Debug("DetectKustomize: no Kustomize markers found")
		return nil
	}

	si := &SourceInfo{
		Type:    SourceKustomize,
		Details: make(map[string]string),
	}
	for k, v := range annotations {
		if len(k) > 27 && k[:28] == "kustomize.toolkit.fluxcd.io" {
			slog.Debug("DetectKustomize: collecting Flux Kustomize annotation", "key", k, "value", v)
			si.Details[k] = v
		}
	}

	slog.Debug("DetectKustomize: Kustomize source detected", "details_count", len(si.Details))
	return si
}

// DetectFlux checks for Flux CD management markers.
func DetectFlux(obj *unstructured.Unstructured) *SourceInfo {
	labels := obj.GetLabels()

	kustomizeName := labels["kustomize.toolkit.fluxcd.io/name"]
	helmName := labels["helm.toolkit.fluxcd.io/name"]

	slog.Debug("DetectFlux: checking Flux labels",
		"kustomize_name", kustomizeName,
		"helm_name", helmName,
	)

	if kustomizeName == "" && helmName == "" {
		slog.Debug("DetectFlux: no Flux labels found")
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
			slog.Debug("DetectFlux: found kustomize namespace label", "ns", ns)
			si.Details["kustomize-namespace"] = ns
		}
	}
	if helmName != "" {
		si.Name = helmName
		si.Details["helm-name"] = helmName
		if ns := labels["helm.toolkit.fluxcd.io/namespace"]; ns != "" {
			slog.Debug("DetectFlux: found helm namespace label", "ns", ns)
			si.Details["helm-namespace"] = ns
		}
	}

	slog.Debug("DetectFlux: Flux source detected", "name", si.Name)
	return si
}

// DetectKubectl checks for kubectl last-applied-configuration annotation.
func DetectKubectl(obj *unstructured.Unstructured) *SourceInfo {
	annotations := obj.GetAnnotations()
	_, ok := annotations["kubectl.kubernetes.io/last-applied-configuration"]
	slog.Debug("DetectKubectl: checking last-applied-configuration annotation", "present", ok)

	if !ok {
		slog.Debug("DetectKubectl: annotation not found")
		return nil
	}

	slog.Debug("DetectKubectl: kubectl source detected via last-applied-configuration")
	return &SourceInfo{
		Type:    SourceKubectl,
		Details: map[string]string{"source": "last-applied-configuration annotation"},
	}
}
