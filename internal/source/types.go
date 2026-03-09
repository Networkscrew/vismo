package source

// SourceType identifies the tool that manages a resource.
type SourceType string

const (
	SourceHelm      SourceType = "Helm"
	SourceArgoCD    SourceType = "ArgoCD"
	SourceKustomize SourceType = "Kustomize"
	SourceFlux      SourceType = "Flux"
	SourceKubectl   SourceType = "kubectl"
	SourceUnknown   SourceType = "Unknown"
)

// SourceInfo describes a single detected managing tool.
type SourceInfo struct {
	Type    SourceType
	Name    string
	Version string
	Details map[string]string
}

// AnalysisResult holds all detected sources for a resource.
type AnalysisResult struct {
	Sources []SourceInfo
}
