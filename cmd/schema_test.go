package cmd

import (
	"sort"
	"testing"
)

func TestBuildSchemaRegistry(t *testing.T) {
	registry := buildSchemaRegistry()

	// Verify expected number of entries
	if got := len(registry); got != 33 {
		t.Errorf("expected 33 entries, got %d", got)
	}

	// Spot-check specific entries exist
	spotChecks := []struct {
		key      string
		resource string
		action   string
		mutating bool
	}{
		{"budget.list", "budget", "list", false},
		{"transaction.create", "transaction", "create", true},
		{"transaction.delete", "transaction", "delete", true},
		{"account.list", "account", "list", false},
		{"category.get", "category", "get", false},
		{"month.get", "month", "get", false},
	}

	for _, sc := range spotChecks {
		t.Run(sc.key, func(t *testing.T) {
			entry, ok := registry[sc.key]
			if !ok {
				t.Fatalf("expected key %q to exist in registry", sc.key)
			}
			if entry.Resource != sc.resource {
				t.Errorf("resource = %q, want %q", entry.Resource, sc.resource)
			}
			if entry.Action != sc.action {
				t.Errorf("action = %q, want %q", entry.Action, sc.action)
			}
			if entry.Mutating != sc.mutating {
				t.Errorf("mutating = %v, want %v", entry.Mutating, sc.mutating)
			}
		})
	}

	// Verify all entries have non-empty descriptions and examples
	for key, entry := range registry {
		if entry.Description == "" {
			t.Errorf("entry %q has empty description", key)
		}
		if entry.Example == "" {
			t.Errorf("entry %q has empty example", key)
		}
	}
}

func TestFindSimilar(t *testing.T) {
	registry := buildSchemaRegistry()

	tests := []struct {
		name      string
		input     string
		wantAny   []string // at least one of these should appear
		wantEmpty bool
	}{
		{
			name:    "search for transaction",
			input:   "transaction",
			wantAny: []string{"transaction.list", "transaction.get", "transaction.create"},
		},
		{
			name:    "search for budget",
			input:   "budget",
			wantAny: []string{"budget.list", "budget.get"},
		},
		{
			name:    "search for partial key",
			input:   "account.list",
			wantAny: []string{"account.list"},
		},
		{
			name:      "search for nonexistent",
			input:     "zzzznotfound",
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findSimilar(tt.input, registry)
			if tt.wantEmpty {
				if len(got) != 0 {
					t.Errorf("expected empty results, got %v", got)
				}
				return
			}
			if len(got) == 0 {
				t.Fatalf("expected results for %q, got none", tt.input)
			}

			// Check that at least one expected result appears
			gotSet := make(map[string]bool)
			for _, s := range got {
				gotSet[s] = true
			}
			found := false
			for _, want := range tt.wantAny {
				if gotSet[want] {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("findSimilar(%q) = %v, expected at least one of %v", tt.input, got, tt.wantAny)
			}
		})
	}
}

func TestMapSlice(t *testing.T) {
	input := []map[string]string{
		{"key": "a", "value": "1"},
		{"key": "b", "value": "2"},
	}

	got := mapSlice(input)

	if len(got) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(got))
	}

	for i, m := range got {
		for k, v := range input[i] {
			gotVal, ok := m[k]
			if !ok {
				t.Errorf("element %d: missing key %q", i, k)
				continue
			}
			if gotVal != v {
				t.Errorf("element %d: key %q = %v, want %v", i, k, gotVal, v)
			}
		}
	}

	// Test empty slice
	empty := mapSlice(nil)
	if len(empty) != 0 {
		t.Errorf("expected empty result for nil input, got %d elements", len(empty))
	}
}

func TestToAnySlice(t *testing.T) {
	input := []map[string]any{
		{"id": "1", "name": "first"},
		{"id": "2", "name": "second"},
	}

	got := toAnySlice(input)

	if len(got) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(got))
	}

	for i, item := range got {
		m, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("element %d: expected map[string]any, got %T", i, item)
		}
		if m["id"] != input[i]["id"] {
			t.Errorf("element %d: id = %v, want %v", i, m["id"], input[i]["id"])
		}
	}

	// Test empty slice
	empty := toAnySlice(nil)
	if len(empty) != 0 {
		t.Errorf("expected empty result for nil input, got %d elements", len(empty))
	}

	// Verify result is usable as []any (the purpose of toAnySlice)
	_ = sort.SliceIsSorted(got, func(i, j int) bool {
		return got[i].(map[string]any)["id"].(string) < got[j].(map[string]any)["id"].(string)
	})
}
