package cmd

import (
	"testing"
)

func TestIsUUID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid lowercase", "550e8400-e29b-41d4-a716-446655440000", true},
		{"valid uppercase", "550E8400-E29B-41D4-A716-446655440000", true},
		{"valid mixed case", "550e8400-E29B-41d4-a716-446655440000", true},
		{"all zeros", "00000000-0000-0000-0000-000000000000", true},
		{"all f's", "ffffffff-ffff-ffff-ffff-ffffffffffff", true},
		{"empty string", "", false},
		{"too short", "550e8400-e29b-41d4-a716", false},
		{"too long", "550e8400-e29b-41d4-a716-4466554400001", false},
		{"missing first dash", "550e8400e29b-41d4-a716-446655440000", false},
		{"missing second dash", "550e8400-e29b41d4-a716-446655440000", false},
		{"missing third dash", "550e8400-e29b-41d4a716-446655440000", false},
		{"missing fourth dash", "550e8400-e29b-41d4-a716446655440000", false},
		{"invalid char g", "550e8400-e29b-41d4-a716-44665544000g", false},
		{"invalid char z", "z50e8400-e29b-41d4-a716-446655440000", false},
		{"spaces", "550e8400 e29b 41d4 a716 446655440000", false},
		{"36 chars no dashes", "550e8400xe29bx41d4xa716x446655440000", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUUID(tt.input)
			if got != tt.want {
				t.Errorf("isUUID(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestEqualFold(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want bool
	}{
		{"same case", "hello", "hello", true},
		{"different case", "Hello", "hello", true},
		{"all upper vs lower", "ABC", "abc", true},
		{"all lower vs upper", "abc", "ABC", true},
		{"mixed case match", "HeLLo", "hEllO", true},
		{"both empty", "", "", true},
		{"mismatch same length", "hello", "world", false},
		{"different lengths", "hello", "hi", false},
		{"one empty", "hello", "", false},
		{"numbers match", "test123", "TEST123", true},
		{"numbers mismatch", "test123", "test456", false},
		{"single char match", "A", "a", true},
		{"single char mismatch", "A", "b", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalFold(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("equalFold(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestParseJSONInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, result map[string]any)
	}{
		{
			name:  "valid simple object",
			input: `{"name":"test","amount":1000}`,
			check: func(t *testing.T, result map[string]any) {
				if result["name"] != "test" {
					t.Errorf("expected name=test, got %v", result["name"])
				}
				if result["amount"] != float64(1000) {
					t.Errorf("expected amount=1000, got %v", result["amount"])
				}
			},
		},
		{
			name:  "nested object",
			input: `{"transaction":{"date":"2024-01-15","amount":-50000}}`,
			check: func(t *testing.T, result map[string]any) {
				nested, ok := result["transaction"].(map[string]any)
				if !ok {
					t.Fatalf("expected nested map, got %T", result["transaction"])
				}
				if nested["date"] != "2024-01-15" {
					t.Errorf("expected date=2024-01-15, got %v", nested["date"])
				}
			},
		},
		{
			name:  "array in values",
			input: `{"tags":["food","grocery"],"amount":-5000}`,
			check: func(t *testing.T, result map[string]any) {
				tags, ok := result["tags"].([]any)
				if !ok {
					t.Fatalf("expected tags to be array, got %T", result["tags"])
				}
				if len(tags) != 2 {
					t.Errorf("expected 2 tags, got %d", len(tags))
				}
			},
		},
		{
			name:    "invalid JSON",
			input:   `{not valid json}`,
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   ``,
			wantErr: true,
		},
		{
			name:    "json array instead of object",
			input:   `[1, 2, 3]`,
			wantErr: true,
		},
		{
			name:    "plain string",
			input:   `"hello"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseJSONInput(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

func TestConfirmActionWithYes(t *testing.T) {
	saved := flagYes
	defer func() { flagYes = saved }()

	flagYes = true
	if !confirmAction("delete this transaction") {
		t.Error("confirmAction should return true when flagYes is set")
	}
}
