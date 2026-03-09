package managedfields

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ParseOwners inspects obj.metadata.managedFields and returns ownership trace
// for the dot-separated fieldPath (e.g. "spec.replicas").
func ParseOwners(obj *unstructured.Unstructured, fieldPath string) (*FieldTrace, error) {
	trace := &FieldTrace{FieldPath: fieldPath}

	// Resolve the actual field value from the object
	val, found, err := nestedField(obj.Object, strings.Split(fieldPath, ".")...)
	if err != nil {
		return nil, fmt.Errorf("read field %q: %w", fieldPath, err)
	}
	if found {
		trace.FinalValue = val
	}

	mf := obj.GetManagedFields()
	if len(mf) == 0 {
		trace.Note = "managedFields is empty — cluster may have been created with --server-side-apply disabled"
		return trace, nil
	}

	// Collect managers that own the field
	owners, err := ownersForField(mf, fieldPath)
	if err != nil {
		return nil, err
	}

	// Sort newest-first
	sort.Slice(owners, func(i, j int) bool {
		return owners[i].Time.After(owners[j].Time)
	})
	trace.Owners = owners

	if len(owners) == 0 {
		trace.Note = fmt.Sprintf("field %q not found in any managedFields entry", fieldPath)
	}
	return trace, nil
}

// ownersForField iterates managedFields entries and returns those that claim ownership of fieldPath.
func ownersForField(mf []metav1.ManagedFieldsEntry, fieldPath string) ([]FieldOwner, error) {
	var owners []FieldOwner
	for _, entry := range mf {
		if entry.FieldsV1 == nil {
			continue
		}
		raw := entry.FieldsV1.Raw
		var fieldsMap map[string]interface{}
		if err := json.Unmarshal(raw, &fieldsMap); err != nil {
			return nil, fmt.Errorf("unmarshal fieldsV1 for manager %q: %w", entry.Manager, err)
		}

		if fieldExistsInFieldsV1(fieldsMap, strings.Split(fieldPath, ".")) {
			var t time.Time
			if entry.Time != nil {
				t = entry.Time.Time
			}
			owners = append(owners, FieldOwner{
				Manager:    entry.Manager,
				Operation:  string(entry.Operation),
				Time:       t,
				APIVersion: entry.APIVersion,
			})
		}
	}
	return owners, nil
}

// fieldExistsInFieldsV1 navigates a fieldsV1 map using a dot-split path.
// fieldsV1 keys use prefix "f:" for fields, so "spec" → "f:spec".
func fieldExistsInFieldsV1(m map[string]interface{}, parts []string) bool {
	if len(parts) == 0 {
		return true
	}
	key := "f:" + parts[0]
	val, ok := m[key]
	if !ok {
		return false
	}
	if len(parts) == 1 {
		return true
	}
	sub, ok := val.(map[string]interface{})
	if !ok {
		return false
	}
	return fieldExistsInFieldsV1(sub, parts[1:])
}

// nestedField is a thin wrapper around unstructured.NestedFieldNoCopy that accepts variadic path.
func nestedField(obj map[string]interface{}, fields ...string) (interface{}, bool, error) {
	val, found, err := unstructured.NestedFieldNoCopy(obj, fields...)
	return val, found, err
}
