package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/kfriede/nab/internal/config"
	"github.com/kfriede/nab/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

var (
	cfg     *config.Config
	printer *output.Printer

	// Global flags
	flagJSON    bool
	flagCSV     bool
	flagOutput  string
	flagFields  string
	flagQuiet   bool
	flagVerbose bool
	flagDebug   bool
	flagNoColor bool
	flagProfile string
	flagBudget  string
	flagYes     bool
	flagDryRun  bool
)

var rootCmd = &cobra.Command{
	Use:   "nab",
	Short: "CLI for You Need A Budget (YNAB)",
	Long: `nab is a command-line interface for You Need A Budget (YNAB).
Built for both humans and LLM agents.

Get started:
  nab login                        Store your YNAB personal access token
  nab budget list                  List your budgets
  nab account list                 List accounts in a budget
  nab transaction list             List recent transactions
  nab category list                List budget categories

Use --json for machine-readable output, --fields to select specific fields.
Run 'nab schema <resource>.<action>' for command introspection.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	err := rootCmd.Execute()
	if err != nil {
		// Print error to stderr (Cobra's SilenceErrors suppresses its own printing)
		if printer != nil {
			printer.PrintError(output.AppError{
				Code:    output.ErrCodeGeneral,
				Message: err.Error(),
			})
		} else {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
	}
	return err
}

func init() {
	cobra.OnInitialize(initConfig, initPrinter)

	// Output flags
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().BoolVar(&flagCSV, "csv", false, "Output as CSV")
	rootCmd.PersistentFlags().StringVarP(&flagOutput, "output", "o", "", "Output format: table, json, csv, ndjson")
	rootCmd.PersistentFlags().StringVar(&flagFields, "fields", "", "Comma-separated list of fields to include")
	rootCmd.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "Disable colored output")

	// Verbosity flags
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "Enable verbose logging to stderr")
	rootCmd.PersistentFlags().BoolVar(&flagDebug, "debug", false, "Enable debug logging (full request/response bodies)")

	// Connection flags
	rootCmd.PersistentFlags().StringVarP(&flagProfile, "profile", "p", "", "Configuration profile to use")
	rootCmd.PersistentFlags().StringVarP(&flagBudget, "budget", "b", "", "Budget ID or name (supports 'last-used' and 'default')")

	// Safety flags
	rootCmd.PersistentFlags().BoolVarP(&flagYes, "yes", "y", false, "Skip confirmation prompts")
	rootCmd.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "Preview changes without executing")

	// Bind env vars
	viper.SetEnvPrefix("NAB")
	_ = viper.BindEnv("token")
	_ = viper.BindEnv("budget")
	_ = viper.BindEnv("profile")
	_ = viper.BindEnv("output_format")
	_ = viper.BindEnv("debug")
	_ = viper.BindEnv("no_color")

	// Register subcommands
	rootCmd.AddCommand(versionCmd)

	// Enable "did you mean?" suggestions
	EnableSuggestions()
}

func initConfig() {
	var err error
	cfg, err = config.Load(flagProfile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load config: %v\n", err)
		cfg = config.Default()
	}

	// Apply flag overrides
	if flagBudget != "" {
		cfg.Budget = flagBudget
	} else if b := viper.GetString("budget"); b != "" {
		cfg.Budget = b
	}

	if flagDebug || viper.GetBool("debug") {
		cfg.Debug = true
		cfg.Verbose = true
	} else if flagVerbose {
		cfg.Verbose = true
	}

	// Load .nab.env from current directory as a fallback credential source.
	// This supports Claude Cowork where the workspace folder is mounted
	// into a sandboxed VM and env vars can't be passed from the host.
	loadLocalEnvFile()
}

func initPrinter() {
	format := resolveOutputFormat()
	isTTY := term.IsTerminal(int(os.Stdout.Fd()))
	noColor := flagNoColor || viper.GetBool("no_color") || os.Getenv("NO_COLOR") != ""

	printer = output.NewPrinter(output.PrinterConfig{
		Format:    format,
		IsTTY:     isTTY,
		NoColor:   noColor,
		Quiet:     flagQuiet,
		Fields:    flagFields,
		Writer:    os.Stdout,
		ErrWriter: os.Stderr,
	})
}

// resolveOutputFormat determines the output format from flags, env, and TTY detection.
// Precedence: --json/--csv flags > --output flag > NAB_OUTPUT_FORMAT env > TTY detection
func resolveOutputFormat() output.Format {
	if flagJSON {
		return output.FormatJSON
	}
	if flagCSV {
		return output.FormatCSV
	}
	if flagOutput != "" {
		switch flagOutput {
		case "json":
			return output.FormatJSON
		case "csv":
			return output.FormatCSV
		case "ndjson":
			return output.FormatNDJSON
		case "table":
			return output.FormatTable
		default:
			fmt.Fprintf(os.Stderr, "Warning: unknown output format %q, using default\n", flagOutput)
		}
	}
	if envFmt := viper.GetString("output_format"); envFmt != "" {
		switch envFmt {
		case "json":
			return output.FormatJSON
		case "csv":
			return output.FormatCSV
		case "ndjson":
			return output.FormatNDJSON
		case "table":
			return output.FormatTable
		}
	}

	// Auto-detect: non-TTY defaults to JSON (LLM-friendly)
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return output.FormatJSON
	}
	return output.FormatTable
}

// loadLocalEnvFile reads .nab.env from the current directory and applies
// any values as fallbacks (only if not already set via flags/env/config/keyring).
// This supports Claude Cowork where the binary is sideloaded into a workspace
// and the sandboxed VM cannot access host env vars or keyring.
func loadLocalEnvFile() {
	f, err := os.Open(".nab.env")
	if err != nil {
		return // no file — not an error
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Strip optional "export " prefix
		line = strings.TrimPrefix(line, "export ")

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		// Remove surrounding quotes
		if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
			value = value[1 : len(value)-1]
		}

		// Only apply as fallback — never override existing config
		switch key {
		case "NAB_TOKEN":
			if cfg.Token == "" {
				secret, _ := config.GetSecret(cfg.Profile)
				if secret == "" {
					cfg.Token = value
				}
			}
		case "NAB_BUDGET":
			if cfg.Budget == "" || cfg.Budget == "last-used" {
				cfg.Budget = value
			}
		}
	}
}
