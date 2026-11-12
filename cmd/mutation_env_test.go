package cmd

import (
	"github.com/spf13/cobra"
	"os"
	"testing"
)

// Exercise the actual pre-run closure with parsed inherited flags, without
// executing config/printer initializers or any credential consumer.
func TestReviewCLIEnvHook(t *testing.T) {
	old := flagMutationAttempts
	defer func() { flagMutationAttempts = old }()
	for _, tt := range []struct {
		name, env string
		args      []string
		wantErr   bool
	}{
		{"default", "unset", nil, false},
		{"env-one", "1", nil, false},
		{"env-empty", "", nil, true},
		{"env-zero", "0", nil, true},
		{"env-two", "2", nil, true},
		{"env-nonnumeric", "abc", nil, true},
		{"flag-two", "1", []string{"--mutation-attempts=2"}, true},
		{"flag-empty", "1", []string{"--mutation-attempts="}, true},
		{"conflicting-env-rejected", "2", []string{"--mutation-attempts=1"}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("NAB_MUTATION_ATTEMPTS", tt.env)
			if tt.env == "unset" {
				os.Unsetenv("NAB_MUTATION_ATTEMPTS")
			}
			parent := &cobra.Command{Use: "fixture"}
			child := &cobra.Command{Use: "write"}
			parent.AddCommand(child)
			parent.PersistentFlags().StringVar(&flagMutationAttempts, "mutation-attempts", "1", "")
			if err := child.ParseFlags(tt.args); err != nil {
				t.Fatal(err)
			}
			err := rootCmd.PersistentPreRunE(child, nil)
			t.Logf("env=%q args=%v inherited_changed=%v err=%v", tt.env, tt.args, child.Flags().Changed("mutation-attempts"), err)
			if (err != nil) != tt.wantErr {
				t.Fatalf("got %v", err)
			}
		})
	}
}
