package managedfields

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ParseOwners inspects obj.metadata.managedFields and returns ownership trace
// for the dot-separated fieldPath (e.g. "spec.replicas").
func ParseOwners(obj *unstructured.Unstructured, fieldPath string) (*FieldTrace, error) {
	slog.Debug("ParseOwners called",
		"resource", obj.GetKind(),
		"name", obj.GetName(),
		"namespace", obj.GetNamespace(),
		"field_path", fieldPath,
	)

	trace := &FieldTrace{FieldPath: fieldPath}

	// Resolve the actual field value from the object
	slog.Debug("resolving current field value from object", "path", fieldPath)
	val, found, err := nestedField(obj.Object, strings.Split(fieldPath, ".")...)
	if err != nil {
		return nil, fmt.Errorf("read field %q: %w", fieldPath, err)
	}
	if found {
		slog.Debug("field value resolved", "path", fieldPath, "value", val)
		trace.FinalValue = val
	} else {
		slog.Debug("field not present in object", "path", fieldPath)
	}

	mf := obj.GetManagedFields()
	slog.Debug("inspecting managedFields", "entry_count", len(mf))

	if len(mf) == 0 {
		slog.Info("managedFields is empty - ownership trace unavailable",
			"resource", obj.GetName(),
			"hint", "cluster may have been created with --server-side-apply disabled",
		)
		trace.Note = "managedFields is empty - cluster may have been created with --server-side-apply disabled"
		return trace, nil
	}

	for i, entry := range mf {
		slog.Debug("managedFields entry",
			"index", i,
			"manager", entry.Manager,
			"operation", entry.Operation,
			"api_version", entry.APIVersion,
			"time", entry.Time,
			"has_fieldsV1", entry.FieldsV1 != nil,
		)
	}

	// Collect managers that own the field
	slog.Debug("scanning managedFields entries for field ownership", "field", fieldPath)
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
		slog.Info("field not claimed by any manager in managedFields",
			"field", fieldPath,
			"resource", obj.GetName(),
		)
		trace.Note = fmt.Sprintf("field %q not found in any managedFields entry", fieldPath)
	} else {
		slog.Debug("ownership resolution complete",
			"field", fieldPath,
			"owners_found", len(owners),
			"final_owner", owners[0].Manager,
			"final_operation", owners[0].Operation,
		)
	}

	return trace, nil
}

// ownersForField iterates managedFields entries and returns those that claim ownership of fieldPath.
func ownersForField(mf []metav1.ManagedFieldsEntry, fieldPath string) ([]FieldOwner, error) {
	var owners []FieldOwner
	parts := strings.Split(fieldPath, ".")

	for _, entry := range mf {
		if entry.FieldsV1 == nil {
			slog.Debug("skipping managedFields entry - FieldsV1 is nil", "manager", entry.Manager)
			continue
		}

		raw := entry.FieldsV1.Raw
		slog.Debug("unmarshalling FieldsV1", "manager", entry.Manager, "raw_bytes", len(raw))

		var fieldsMap map[string]interface{}
		if err := json.Unmarshal(raw, &fieldsMap); err != nil {
			return nil, fmt.Errorf("unmarshal fieldsV1 for manager %q: %w", entry.Manager, err)
		}

		slog.Debug("navigating FieldsV1 for field path",
			"manager", entry.Manager,
			"path", fieldPath,
			"parts", parts,
		)

		if fieldExistsInFieldsV1(fieldsMap, parts) {
			var t time.Time
			if entry.Time != nil {
				t = entry.Time.Time
			}
			slog.Debug("manager claims ownership of field",
				"manager", entry.Manager,
				"operation", entry.Operation,
				"time", t,
				"api_version", entry.APIVersion,
			)
			owners = append(owners, FieldOwner{
				Manager:    entry.Manager,
				Operation:  string(entry.Operation),
				Time:       t,
				APIVersion: entry.APIVersion,
			})
		} else {
			slog.Debug("manager does not claim field", "manager", entry.Manager, "field", fieldPath)
		}
	}

	slog.Debug("ownersForField complete", "field", fieldPath, "owners_count", len(owners))
	return owners, nil
}

// fieldExistsInFieldsV1 navigates a fieldsV1 map using a dot-split path.
// fieldsV1 keys use prefix "f:" for fields, so "spec" → "f:spec".
func fieldExistsInFieldsV1(m map[string]interface{}, parts []string) bool {
	if len(parts) == 0 {
		return true
	}
	key := "f:" + parts[0]
	slog.Debug("fieldExistsInFieldsV1: looking for key", "key", key, "remaining_parts", parts[1:])

	val, ok := m[key]
	if !ok {
		slog.Debug("fieldExistsInFieldsV1: key not found", "key", key)
		return false
	}
	if len(parts) == 1 {
		slog.Debug("fieldExistsInFieldsV1: leaf key found", "key", key)
		return true
	}
	sub, ok := val.(map[string]interface{})
	if !ok {
		slog.Debug("fieldExistsInFieldsV1: value is not a nested map, cannot descend", "key", key)
		return false
	}
	return fieldExistsInFieldsV1(sub, parts[1:])
}

// nestedField is a thin wrapper around unstructured.NestedFieldNoCopy that accepts variadic path.
func nestedField(obj map[string]interface{}, fields ...string) (interface{}, bool, error) {
	slog.Debug("nestedField: reading value", "path", strings.Join(fields, "."))
	val, found, err := unstructured.NestedFieldNoCopy(obj, fields...)
	if err != nil {
		slog.Debug("nestedField: error reading value", "path", strings.Join(fields, "."), "err", err)
	}
	return val, found, err
}
