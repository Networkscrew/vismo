package managedfields_test

import (
	"encoding/json"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Veinar/vismo/internal/managedfields"
)

func fieldsV1(t *testing.T, m map[string]interface{}) *metav1.FieldsV1 {
	t.Helper()
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal fieldsV1: %v", err)
	}
	return &metav1.FieldsV1{Raw: raw}
}

func makeTime(s string) *metav1.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return &metav1.Time{Time: t}
}

func TestParseOwners_SingleManager(t *testing.T) {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"replicas": int64(3),
			},
		},
	}
	obj.SetManagedFields([]metav1.ManagedFieldsEntry{
		{
			Manager:    "kubectl-scale",
			Operation:  metav1.ManagedFieldsOperationUpdate,
			APIVersion: "apps/v1",
			Time:       makeTime("2023-10-27T10:00:05Z"),
			FieldsV1: fieldsV1(t, map[string]interface{}{
				"f:spec": map[string]interface{}{
					"f:replicas": map[string]interface{}{},
				},
			}),
		},
	})

	trace, err := managedfields.ParseOwners(obj, "spec.replicas")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(trace.Owners) != 1 {
		t.Fatalf("expected 1 owner, got %d", len(trace.Owners))
	}
	if trace.Owners[0].Manager != "kubectl-scale" {
		t.Errorf("manager = %q, want %q", trace.Owners[0].Manager, "kubectl-scale")
	}
	if trace.FinalValue != int64(3) {
		t.Errorf("final value = %v, want 3", trace.FinalValue)
	}
}

func TestParseOwners_MultipleManagers_NewestFirst(t *testing.T) {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"replicas": int64(5),
			},
		},
	}

	replicasField := map[string]interface{}{
		"f:spec": map[string]interface{}{
			"f:replicas": map[string]interface{}{},
		},
	}

	obj.SetManagedFields([]metav1.ManagedFieldsEntry{
		{
			Manager:    "helm",
			Operation:  metav1.ManagedFieldsOperationUpdate,
			APIVersion: "apps/v1",
			Time:       makeTime("2023-10-27T09:30:12Z"),
			FieldsV1:   fieldsV1(t, replicasField),
		},
		{
			Manager:    "kubectl-scale",
			Operation:  metav1.ManagedFieldsOperationUpdate,
			APIVersion: "apps/v1",
			Time:       makeTime("2023-10-27T10:00:05Z"),
			FieldsV1:   fieldsV1(t, replicasField),
		},
	})

	trace, err := managedfields.ParseOwners(obj, "spec.replicas")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(trace.Owners) != 2 {
		t.Fatalf("expected 2 owners, got %d", len(trace.Owners))
	}
	// Newest first
	if trace.Owners[0].Manager != "kubectl-scale" {
		t.Errorf("first owner = %q, want kubectl-scale", trace.Owners[0].Manager)
	}
	if trace.Owners[1].Manager != "helm" {
		t.Errorf("second owner = %q, want helm", trace.Owners[1].Manager)
	}
}

func TestParseOwners_EmptyManagedFields(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]interface{}{}}
	trace, err := managedfields.ParseOwners(obj, "spec.replicas")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if trace.Note == "" {
		t.Error("expected note about empty managedFields")
	}
}

func TestParseOwners_FieldNotFound(t *testing.T) {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{},
	}
	obj.SetManagedFields([]metav1.ManagedFieldsEntry{
		{
			Manager:   "helm",
			Operation: metav1.ManagedFieldsOperationUpdate,
			Time:      makeTime("2023-10-27T09:00:00Z"),
			FieldsV1: fieldsV1(t, map[string]interface{}{
				"f:spec": map[string]interface{}{
					"f:selector": map[string]interface{}{},
				},
			}),
		},
	})

	trace, err := managedfields.ParseOwners(obj, "spec.replicas")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(trace.Owners) != 0 {
		t.Errorf("expected 0 owners, got %d", len(trace.Owners))
	}
	if trace.Note == "" {
		t.Error("expected note about field not found")
	}
}
