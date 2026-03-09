package output_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/Veinar/vismo/internal/managedfields"
	"github.com/Veinar/vismo/internal/output"
	"github.com/Veinar/vismo/internal/source"
)

func newFormatter() (*output.TextFormatter, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	return output.NewTextFormatter(buf), buf
}

func TestRenderResource(t *testing.T) {
	f, buf := newFormatter()
	f.RenderResource([]byte("apiVersion: apps/v1\n"))
	if !strings.Contains(buf.String(), "apiVersion: apps/v1") {
		t.Errorf("resource YAML not in output: %s", buf.String())
	}
}

func TestRenderSources_NoSources(t *testing.T) {
	f, buf := newFormatter()
	f.RenderSources(&source.AnalysisResult{})
	if !strings.Contains(buf.String(), "no sources detected") {
		t.Errorf("expected no-sources message, got: %s", buf.String())
	}
}

func TestRenderSources_WithSources(t *testing.T) {
	f, buf := newFormatter()
	f.RenderSources(&source.AnalysisResult{
		Sources: []source.SourceInfo{
			{Type: source.SourceHelm, Name: "my-release", Version: "chart-1.0.0"},
		},
	})
	out := buf.String()
	if !strings.Contains(out, "Helm") {
		t.Errorf("expected Helm in output: %s", out)
	}
	if !strings.Contains(out, "my-release") {
		t.Errorf("expected release name in output: %s", out)
	}
}

func TestRenderFieldTrace(t *testing.T) {
	f, buf := newFormatter()
	ts, _ := time.Parse(time.RFC3339, "2023-10-27T10:00:05Z")
	f.RenderFieldTrace(&managedfields.FieldTrace{
		FieldPath:  "spec.replicas",
		FinalValue: int64(5),
		Owners: []managedfields.FieldOwner{
			{Manager: "kubectl-scale", Operation: "Update", Time: ts},
			{Manager: "helm", Operation: "Update"},
		},
	})
	out := buf.String()
	if !strings.Contains(out, "spec.replicas") {
		t.Errorf("expected field path: %s", out)
	}
	if !strings.Contains(out, "kubectl-scale") {
		t.Errorf("expected manager name: %s", out)
	}
	if !strings.Contains(out, "final owner") {
		t.Errorf("expected final owner note: %s", out)
	}
}

func TestRenderFieldTrace_WithNote(t *testing.T) {
	f, buf := newFormatter()
	f.RenderFieldTrace(&managedfields.FieldTrace{
		FieldPath: "spec.replicas",
		Note:      "managedFields is empty",
	})
	if !strings.Contains(buf.String(), "managedFields is empty") {
		t.Errorf("expected note in output: %s", buf.String())
	}
}
