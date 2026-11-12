package cmd

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/kfriede/nab/internal/api"
)

func TestMutationAttemptsValidation(t *testing.T) {
	for _, value := range []string{"", "0", "2", "3", "-1", "true", "01", " 1", "1 "} {
		if validateMutationAttempts(value) == nil {
			t.Errorf("accepted %q", value)
		}
	}
	if err := validateMutationAttempts("1"); err != nil {
		t.Fatal(err)
	}
}

func TestUnknownMutationErrorClassification(t *testing.T) {
	err := fmt.Errorf("updating fixture: %w", &api.MutationOutcomeUnknownError{Cause: io.ErrUnexpectedEOF})
	result := classifyError(err)
	if result.Code != "MUTATION_OUTCOME_UNKNOWN" || !strings.Contains(result.Guidance, "cannot uniquely prove") {
		t.Fatalf("%+v", result)
	}
	if classifyError(io.EOF).Code == "MUTATION_OUTCOME_UNKNOWN" {
		t.Fatal("read error classified as mutation")
	}
}
