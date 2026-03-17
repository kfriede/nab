package cmd

import (
	"testing"
)

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{"identical", "hello", "hello", 0},
		{"single substitution", "hello", "hallo", 1},
		{"single insertion", "hello", "helloo", 1},
		{"single deletion", "hello", "helo", 1},
		{"completely different", "abc", "xyz", 3},
		{"both empty", "", "", 0},
		{"first empty", "", "hello", 5},
		{"second empty", "hello", "", 5},
		{"single char same", "a", "a", 0},
		{"single char diff", "a", "b", 1},
		{"transposition", "ab", "ba", 2},
		{"prefix match", "budget", "budge", 1},
		{"similar flags", "json", "jsno", 2},
		{"realistic typo", "buget", "budget", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := levenshtein(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestMinInt(t *testing.T) {
	tests := []struct {
		name string
		vals []int
		want int
	}{
		{"single value", []int{5}, 5},
		{"two values ascending", []int{1, 2}, 1},
		{"two values descending", []int{2, 1}, 1},
		{"three values", []int{3, 1, 2}, 1},
		{"negative values", []int{-1, -5, -3}, -5},
		{"mixed positive negative", []int{10, -2, 5, 0}, -2},
		{"all same", []int{7, 7, 7}, 7},
		{"min at end", []int{5, 4, 3, 2, 1}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := minInt(tt.vals...)
			if got != tt.want {
				t.Errorf("minInt(%v) = %d, want %d", tt.vals, got, tt.want)
			}
		})
	}
}

func TestExtractFlagName(t *testing.T) {
	tests := []struct {
		name   string
		errMsg string
		want   string
	}{
		{"standard unknown flag", "unknown flag: --budget", "budget"},
		{"flag with hyphen", "unknown flag: --json-input", "json-input"},
		{"no match", "some other error", ""},
		{"empty string", "", ""},
		{"partial match no prefix", "flag: --budget", ""},
		{"unknown flag with extra text", "unknown flag: --verbose extra", "verbose extra"},
		{"shorthand flag not matched", "unknown shorthand flag: -x in -x", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFlagName(tt.errMsg)
			if got != tt.want {
				t.Errorf("extractFlagName(%q) = %q, want %q", tt.errMsg, got, tt.want)
			}
		})
	}
}
