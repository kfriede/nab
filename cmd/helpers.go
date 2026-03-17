package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kfriede/nab/internal/api"
	"github.com/kfriede/nab/internal/config"
	"github.com/kfriede/nab/internal/output"
)

// newAPIClient creates an API client from the current configuration.
func newAPIClient() (*api.Client, error) {
	token := cfg.Token
	if token == "" {
		// Try keyring (non-fatal if keyring unavailable)
		secret, err := config.GetSecret(cfg.Profile)
		if err == nil {
			token = secret
		}
	}

	if token == "" {
		printer.PrintError(output.NewAuthError("No YNAB personal access token configured"))
		return nil, fmt.Errorf("no token configured")
	}

	return api.NewClient(api.ClientConfig{
		Token:     token,
		Verbose:   cfg.Verbose,
		Debug:     cfg.Debug,
		ErrWriter: os.Stderr,
	}), nil
}

// requireBudget returns the budget ID, resolving names like "last-used" and "default".
func requireBudget() (string, error) {
	budget := cfg.Budget
	if budget == "" {
		printer.PrintError(output.NewError(
			output.ErrCodeConfig,
			"No budget specified",
			"Use --budget flag or set NAB_BUDGET, or run `nab config set budget <id>`.",
		))
		return "", fmt.Errorf("no budget specified")
	}

	// "last-used" and "default" are special YNAB API values passed directly
	if budget == "last-used" || budget == "default" {
		return budget, nil
	}

	// If it looks like a UUID already, use it directly
	if isUUID(budget) {
		return budget, nil
	}

	// Otherwise resolve the name to a UUID
	client, err := newAPIClient()
	if err != nil {
		return "", err
	}

	var result struct {
		Budgets []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"budgets"`
	}
	if err := client.GetJSON("/budgets", &result); err != nil {
		return "", fmt.Errorf("resolving budget %q: %w", budget, err)
	}

	for _, b := range result.Budgets {
		if equalFold(b.Name, budget) {
			return b.ID, nil
		}
	}

	return "", fmt.Errorf("budget %q not found; run `nab budget list` to see available budgets", budget)
}

// confirmAction asks the user to confirm a destructive action.
// Returns true if confirmed or --yes was passed.
func confirmAction(action string) bool {
	if flagYes {
		return true
	}

	fmt.Fprintf(os.Stderr, "Are you sure you want to %s? (y/N): ", action)

	var response string
	_, _ = fmt.Scanln(&response)
	return response == "y" || response == "yes" || response == "Y"
}

// parseJSONInput parses the --json-input flag value into a map.
func parseJSONInput(jsonStr string) (map[string]any, error) {
	var result map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON input: %w", err)
	}
	return result, nil
}

// printAPIResult handles the common pattern of printing API results
// with proper error handling and exit codes.
func printAPIResult(data any) error {
	return printer.PrintResult(data)
}

func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range len(a) {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
