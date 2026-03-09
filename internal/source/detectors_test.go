package source_test

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Veinar/vismo/internal/source"
)

func makeObj(labels, annotations map[string]string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetLabels(labels)
	obj.SetAnnotations(annotations)
	return obj
}

func TestDetectHelm(t *testing.T) {
	tests := []struct {
		name        string
		labels      map[string]string
		annotations map[string]string
		wantNil     bool
		wantName    string
		wantVersion string
	}{
		{
			name:        "helm managed",
			labels:      map[string]string{"app.kubernetes.io/managed-by": "Helm"},
			annotations: map[string]string{"meta.helm.sh/release-name": "my-app", "helm.sh/chart": "my-app-1.2.3"},
			wantNil:     false,
			wantName:    "my-app",
			wantVersion: "my-app-1.2.3",
		},
		{
			name:    "not helm managed",
			labels:  map[string]string{"app.kubernetes.io/managed-by": "kustomize"},
			wantNil: true,
		},
		{
			name:    "no labels",
			labels:  map[string]string{},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := makeObj(tt.labels, tt.annotations)
			si := source.DetectHelm(obj)
			if tt.wantNil {
				if si != nil {
					t.Errorf("expected nil, got %+v", si)
				}
				return
			}
			if si == nil {
				t.Fatal("expected SourceInfo, got nil")
			}
			if si.Type != source.SourceHelm {
				t.Errorf("type = %q, want %q", si.Type, source.SourceHelm)
			}
			if si.Name != tt.wantName {
				t.Errorf("name = %q, want %q", si.Name, tt.wantName)
			}
			if si.Version != tt.wantVersion {
				t.Errorf("version = %q, want %q", si.Version, tt.wantVersion)
			}
		})
	}
}

func TestDetectArgoCD(t *testing.T) {
	tests := []struct {
		name        string
		labels      map[string]string
		annotations map[string]string
		wantNil     bool
		wantName    string
	}{
		{
			name:     "argocd label",
			labels:   map[string]string{"argocd.argoproj.io/app-name": "my-argocd-app"},
			wantNil:  false,
			wantName: "my-argocd-app",
		},
		{
			name:        "argocd tracking annotation only",
			labels:      map[string]string{},
			annotations: map[string]string{"argocd.argoproj.io/tracking-id": "my-app:apps/Deployment:ns/name"},
			wantNil:     false,
		},
		{
			name:    "not argocd",
			labels:  map[string]string{"app": "my-app"},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := makeObj(tt.labels, tt.annotations)
			si := source.DetectArgoCD(obj)
			if tt.wantNil {
				if si != nil {
					t.Errorf("expected nil, got %+v", si)
				}
				return
			}
			if si == nil {
				t.Fatal("expected SourceInfo, got nil")
			}
			if si.Type != source.SourceArgoCD {
				t.Errorf("type = %q, want %q", si.Type, source.SourceArgoCD)
			}
			if tt.wantName != "" && si.Name != tt.wantName {
				t.Errorf("name = %q, want %q", si.Name, tt.wantName)
			}
		})
	}
}

func TestDetectFlux(t *testing.T) {
	tests := []struct {
		name    string
		labels  map[string]string
		wantNil bool
	}{
		{
			name:    "flux kustomize",
			labels:  map[string]string{"kustomize.toolkit.fluxcd.io/name": "infra"},
			wantNil: false,
		},
		{
			name:    "flux helm",
			labels:  map[string]string{"helm.toolkit.fluxcd.io/name": "prometheus"},
			wantNil: false,
		},
		{
			name:    "not flux",
			labels:  map[string]string{"app": "foo"},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := makeObj(tt.labels, nil)
			si := source.DetectFlux(obj)
			if tt.wantNil && si != nil {
				t.Errorf("expected nil, got %+v", si)
			}
			if !tt.wantNil && si == nil {
				t.Error("expected SourceInfo, got nil")
			}
		})
	}
}

func TestDetectKubectl(t *testing.T) {
	t.Run("with last-applied", func(t *testing.T) {
		obj := makeObj(nil, map[string]string{
			"kubectl.kubernetes.io/last-applied-configuration": `{"apiVersion":"apps/v1"}`,
		})
		si := source.DetectKubectl(obj)
		if si == nil {
			t.Fatal("expected SourceInfo, got nil")
		}
		if si.Type != source.SourceKubectl {
			t.Errorf("type = %q, want %q", si.Type, source.SourceKubectl)
		}
	})

	t.Run("without annotation", func(t *testing.T) {
		obj := makeObj(nil, nil)
		if si := source.DetectKubectl(obj); si != nil {
			t.Errorf("expected nil, got %+v", si)
		}
	})
}

func TestAnalyze(t *testing.T) {
	obj := makeObj(
		map[string]string{
			"app.kubernetes.io/managed-by": "Helm",
			"argocd.argoproj.io/app-name":  "my-app",
		},
		map[string]string{
			"meta.helm.sh/release-name": "my-app",
			"helm.sh/chart":             "my-app-1.0.0",
		},
	)
	result := source.Analyze(obj)
	if len(result.Sources) != 2 {
		t.Errorf("expected 2 sources, got %d: %+v", len(result.Sources), result.Sources)
	}
}
