package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/kfriede/nab/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(configCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and manage configuration",
	Long: `View and manage nab configuration.

Examples:
  nab config show                  Show current configuration
  nab config path                  Show config file path
  nab config set budget <id>       Set default budget`,
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configSetCmd)
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		display := map[string]any{
			"budget":  cfg.Budget,
			"profile": cfg.Profile,
			"verbose": cfg.Verbose,
			"debug":   cfg.Debug,
		}

		// Check if token is set (don't reveal it)
		if cfg.Token != "" {
			display["token"] = "****" + cfg.Token[max(0, len(cfg.Token)-4):]
		} else {
			secret, _ := config.GetSecret(cfg.Profile)
			if secret != "" {
				display["token"] = "(stored in keyring)"
			} else {
				display["token"] = "(not set)"
			}
		}

		return printer.PrintResult(display)
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show config directory path",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _ = fmt.Fprintln(os.Stdout, config.Dir())
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value.

Supported keys: budget

Examples:
  nab config set budget last-used
  nab config set budget <budget-uuid>`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := strings.ToLower(args[0])
		value := args[1]

		switch key {
		case "budget":
			cfg.Budget = value
		default:
			return fmt.Errorf("unknown config key: %s (supported: budget)", key)
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		printer.Success(fmt.Sprintf("Set %s = %s", key, value))
		return nil
	},
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
