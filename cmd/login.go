package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/kfriede/nab/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func init() {
	rootCmd.AddCommand(loginCmd)
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Store your YNAB personal access token",
	Long: `Store your YNAB personal access token for API access.

Examples:
  nab login                        Interactive login
  nab login --profile family       Save as named profile

You can create a personal access token at:
  https://app.ynab.com/settings/developer

The token is stored in your OS keyring when available,
falling back to the config file with restrictive permissions.`,
	Args: cobra.NoArgs,
	RunE: runLogin,
}

func runLogin(cmd *cobra.Command, args []string) error {
	isTTY := term.IsTerminal(int(os.Stdin.Fd()))
	if !isTTY {
		return fmt.Errorf("login requires an interactive terminal; set NAB_TOKEN environment variable for non-interactive use")
	}

	reader := bufio.NewReader(os.Stdin)

	// Get personal access token
	fmt.Fprint(os.Stderr, "YNAB Personal Access Token: ")
	tokenBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr) // newline after hidden input
	if err != nil {
		return fmt.Errorf("reading token: %w", err)
	}
	token := strings.TrimSpace(string(tokenBytes))
	if token == "" {
		return fmt.Errorf("token is required")
	}

	// Get default budget
	budget := cfg.Budget
	fmt.Fprintf(os.Stderr, "Default budget [%s]: ", budget)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		budget = input
	}

	// Store token in keyring
	profile := flagProfile
	if config.KeyringAvailable() {
		if err := config.StoreSecret(profile, token); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not store token in keyring: %v\n", err)
			fmt.Fprintln(os.Stderr, "Token will be stored in the config file instead.")
			// Fall through to save in config
		} else {
			printer.Status("Token stored in OS keyring")
			token = "" // Don't save in config file
		}
	} else {
		printer.Status("OS keyring not available, storing token in config file")
	}

	// Save config
	newCfg := &config.Config{
		Budget:  budget,
		Profile: profile,
	}
	if token != "" {
		newCfg.Token = token
	}

	if err := config.Save(newCfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	printer.Success("Logged in to YNAB")
	printer.Status("Run `nab budget list` to verify your connection.")
	return nil
}
