package merge

import (
	"testing"
)

func TestMerge_AddsNewKeys(t *testing.T) {
	master := map[string]any{"newKey": "value", "another": 42.0}
	local := map[string]any{"existingKey": "local"}

	result := Merge(master, local)

	if result.Merged["newKey"] != "value" {
		t.Errorf("newKey = %v; want %q", result.Merged["newKey"], "value")
	}
	if result.Merged["another"] != 42.0 {
		t.Errorf("another = %v; want 42", result.Merged["another"])
	}
	if len(result.Added) != 2 {
		t.Errorf("Added = %d; want 2", len(result.Added))
	}
	if len(result.Conflicts) != 0 {
		t.Errorf("Conflicts = %d; want 0", len(result.Conflicts))
	}
}

func TestMerge_KeepsLocalOnConflict(t *testing.T) {
	master := map[string]any{"key": "master-value"}
	local := map[string]any{"key": "local-value"}

	result := Merge(master, local)

	if result.Merged["key"] != "local-value" {
		t.Errorf("key = %v; want %q", result.Merged["key"], "local-value")
	}
	if len(result.Added) != 0 {
		t.Errorf("Added = %d; want 0", len(result.Added))
	}
}

func TestMerge_ReportsConflicts(t *testing.T) {
	master := map[string]any{"key": "master-value"}
	local := map[string]any{"key": "local-value"}

	result := Merge(master, local)

	if len(result.Conflicts) != 1 {
		t.Fatalf("Conflicts = %d; want 1", len(result.Conflicts))
	}
	c := result.Conflicts[0]
	if c.Key != "key" {
		t.Errorf("Conflict.Key = %q; want %q", c.Key, "key")
	}
	if c.MasterValue != "master-value" {
		t.Errorf("Conflict.MasterValue = %v; want %q", c.MasterValue, "master-value")
	}
	if c.LocalValue != "local-value" {
		t.Errorf("Conflict.LocalValue = %v; want %q", c.LocalValue, "local-value")
	}
}

func TestMerge_DeepMergesNestedObjects(t *testing.T) {
	master := map[string]any{
		"nested": map[string]any{
			"fromMaster": "yes",
			"shared":     "master",
		},
	}
	local := map[string]any{
		"nested": map[string]any{
			"fromLocal": "yes",
			"shared":    "local",
		},
	}

	result := Merge(master, local)

	nested, ok := result.Merged["nested"].(map[string]any)
	if !ok {
		t.Fatal("nested is not a map")
	}
	if nested["fromMaster"] != "yes" {
		t.Errorf("nested.fromMaster = %v; want %q", nested["fromMaster"], "yes")
	}
	if nested["fromLocal"] != "yes" {
		t.Errorf("nested.fromLocal = %v; want %q", nested["fromLocal"], "yes")
	}
	if nested["shared"] != "local" {
		t.Errorf("nested.shared = %v; want %q (local wins)", nested["shared"], "local")
	}
	if len(result.Conflicts) != 1 {
		t.Errorf("Conflicts = %d; want 1", len(result.Conflicts))
	}
	if result.Conflicts[0].Key != "nested.shared" {
		t.Errorf("Conflict key = %q; want %q", result.Conflicts[0].Key, "nested.shared")
	}
}

func TestMerge_EmptyMaster(t *testing.T) {
	master := map[string]any{}
	local := map[string]any{"key": "value"}

	result := Merge(master, local)

	if result.Merged["key"] != "value" {
		t.Errorf("key = %v; want %q", result.Merged["key"], "value")
	}
	if len(result.Added) != 0 {
		t.Errorf("Added = %d; want 0", len(result.Added))
	}
}

func TestMerge_EmptyLocal(t *testing.T) {
	master := map[string]any{"key": "value"}
	local := map[string]any{}

	result := Merge(master, local)

	if result.Merged["key"] != "value" {
		t.Errorf("key = %v; want %q", result.Merged["key"], "value")
	}
	if len(result.Added) != 1 {
		t.Errorf("Added = %d; want 1", len(result.Added))
	}
}

func TestMerge_MatchingKeysNotConflicts(t *testing.T) {
	master := map[string]any{"key": "same"}
	local := map[string]any{"key": "same"}

	result := Merge(master, local)

	if len(result.Conflicts) != 0 {
		t.Errorf("Conflicts = %d; want 0 (identical values are not conflicts)", len(result.Conflicts))
	}
	if len(result.Matching) != 1 {
		t.Errorf("Matching = %d; want 1", len(result.Matching))
	}
	if result.Matching[0] != "key" {
		t.Errorf("Matching[0] = %q; want %q", result.Matching[0], "key")
	}
}

func TestMerge_MatchingSlicesNotConflicts(t *testing.T) {
	val := []any{"a", "b"}
	master := map[string]any{"key": val}
	local := map[string]any{"key": val}

	result := Merge(master, local)

	if len(result.Conflicts) != 0 {
		t.Errorf("Conflicts = %d; want 0 (identical slices are not conflicts)", len(result.Conflicts))
	}
	if len(result.Matching) != 1 {
		t.Errorf("Matching = %d; want 1", len(result.Matching))
	}
}

func TestMerge_LocalOnlyKeys(t *testing.T) {
	master := map[string]any{"masterKey": "value"}
	local := map[string]any{"masterKey": "value", "localOnly": "mine"}

	result := Merge(master, local)

	if len(result.LocalOnly) != 1 {
		t.Fatalf("LocalOnly = %d; want 1", len(result.LocalOnly))
	}
	if result.LocalOnly[0] != "localOnly" {
		t.Errorf("LocalOnly[0] = %q; want %q", result.LocalOnly[0], "localOnly")
	}
}

func TestMerge_NestedLocalOnlyKeys(t *testing.T) {
	master := map[string]any{
		"nested": map[string]any{"masterOnlyNested": "v"},
	}
	local := map[string]any{
		"nested": map[string]any{"localOnlyNested": "v"},
	}

	result := Merge(master, local)

	// "nested.localOnlyNested" exists only in local — must appear in LocalOnly.
	found := false
	for _, k := range result.LocalOnly {
		if k == "nested.localOnlyNested" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("LocalOnly = %v; want it to contain %q", result.LocalOnly, "nested.localOnlyNested")
	}

	// "nested.masterOnlyNested" is new from master — must appear in Added.
	foundAdded := false
	for _, k := range result.Added {
		if k == "nested.masterOnlyNested" {
			foundAdded = true
			break
		}
	}
	if !foundAdded {
		t.Errorf("Added = %v; want it to contain %q", result.Added, "nested.masterOnlyNested")
	}
}

func TestMerge_ScalarVsObjectConflict(t *testing.T) {
	// master has object, local has scalar at same key — treat as conflict, keep local.
	master := map[string]any{"key": map[string]any{"nested": "val"}}
	local := map[string]any{"key": "scalar"}

	result := Merge(master, local)

	if result.Merged["key"] != "scalar" {
		t.Errorf("key = %v; want %q", result.Merged["key"], "scalar")
	}
	if len(result.Conflicts) != 1 {
		t.Errorf("Conflicts = %d; want 1", len(result.Conflicts))
	}
}
