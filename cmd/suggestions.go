package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// EnableSuggestions configures Cobra's built-in "did you mean?" suggestions
// and sets a Levenshtein distance threshold.
func EnableSuggestions() {
	rootCmd.SuggestionsMinimumDistance = 2
}

func init() {
	rootCmd.SetFlagErrorFunc(flagErrorFunc)
}

func flagErrorFunc(cmd *cobra.Command, err error) error {
	msg := err.Error()

	// Check if it's an unknown flag error and try to suggest
	if strings.Contains(msg, "unknown flag") || strings.Contains(msg, "unknown shorthand flag") {
		flagName := extractFlagName(msg)
		if flagName != "" {
			suggestions := suggestFlags(cmd, flagName)
			if len(suggestions) > 0 {
				return fmt.Errorf("%s\n\nDid you mean one of these?\n  %s\n\nRun '%s --help' for usage",
					msg, strings.Join(suggestions, "\n  "), cmd.CommandPath())
			}
		}
	}

	return fmt.Errorf("%s\n\nRun '%s --help' for usage", msg, cmd.CommandPath())
}

func extractFlagName(errMsg string) string {
	// Handle "unknown flag: --foo"
	if idx := strings.Index(errMsg, "unknown flag: --"); idx >= 0 {
		return strings.TrimSpace(errMsg[idx+len("unknown flag: --"):])
	}
	return ""
}

func suggestFlags(cmd *cobra.Command, flagName string) []string {
	seen := make(map[string]bool)
	var suggestions []string

	visitor := func(name string) {
		if !seen[name] && (levenshtein(flagName, name) <= 3 || strings.Contains(name, flagName)) {
			seen[name] = true
			suggestions = append(suggestions, "--"+name)
		}
	}

	cmd.Flags().VisitAll(func(f *pflag.Flag) { visitor(f.Name) })
	cmd.InheritedFlags().VisitAll(func(f *pflag.Flag) { visitor(f.Name) })

	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}
	return suggestions
}

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	d := make([][]int, la+1)
	for i := range d {
		d[i] = make([]int, lb+1)
		d[i][0] = i
	}
	for j := 0; j <= lb; j++ {
		d[0][j] = j
	}

	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			d[i][j] = minInt(d[i-1][j]+1, d[i][j-1]+1, d[i-1][j-1]+cost)
		}
	}
	return d[la][lb]
}

func minInt(vals ...int) int {
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}
