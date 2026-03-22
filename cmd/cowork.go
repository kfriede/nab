package cmd

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kfriede/nab/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(coworkCmd)
	coworkCmd.AddCommand(coworkSetupCmd)

	coworkSetupCmd.Flags().StringP("dir", "d", ".", "Directory to install the nab binary into")
}

var coworkCmd = &cobra.Command{
	Use:   "cowork",
	Short: "Set up nab for Claude Cowork",
	Long: `Set up nab for use inside Claude Cowork's sandboxed Linux VM.

Cowork runs in an isolated VM where host-installed binaries are not available.
This command downloads the correct Linux binary and helps you configure
environment variables for Cowork.

Examples:
  nab cowork setup                  Download binary + show env config
  nab cowork setup --dir ~/Cowork   Install to a specific folder`,
}

var coworkSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Download nab binary and configure for Cowork",
	Long: `Download the correct nab binary for Claude Cowork's Linux VM and
display the environment variables needed for Cowork configuration.

This command:
  1. Downloads a static Linux binary from GitHub Releases
  2. Places it in the target directory (default: current directory)
  3. Shows your current NAB_TOKEN and NAB_BUDGET values to configure in Cowork

Run this on your host machine, then open the target directory in Cowork
using "Work in a folder". Claude will use ./nab from the workspace.`,
	Args: cobra.NoArgs,
	RunE: runCoworkSetup,
}

func runCoworkSetup(cmd *cobra.Command, _ []string) error {
	dir, _ := cmd.Flags().GetString("dir")

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolving directory: %w", err)
	}

	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", absDir, err)
	}

	printer.Status("Setting up nab for Claude Cowork...")
	fmt.Fprintln(os.Stderr)

	// Step 1: Download binary
	if err := downloadCoworkBinary(absDir); err != nil {
		return err
	}

	// Step 2: Show environment configuration
	fmt.Fprintln(os.Stderr)
	showCoworkEnvConfig()

	// Step 3: Show next steps
	fmt.Fprintln(os.Stderr)
	showCoworkNextSteps(absDir)

	return nil
}

func downloadCoworkBinary(dir string) error {
	// Cowork VM is always Linux; detect host arch to match
	goarch := runtime.GOARCH
	if goarch != "amd64" && goarch != "arm64" {
		return fmt.Errorf("unsupported architecture %q; Cowork requires amd64 or arm64", goarch)
	}

	version := Version
	if version == "dev" || version == "" {
		printer.Status("Detecting latest release version...")
		v, err := fetchLatestVersion()
		if err != nil {
			return fmt.Errorf("could not determine version: %w\nSpecify a version by building with ldflags or download manually from GitHub", err)
		}
		version = v
	}

	versionNum := strings.TrimPrefix(version, "v")
	archiveName := fmt.Sprintf("nab_%s_linux_%s.tar.gz", versionNum, goarch)
	downloadURL := fmt.Sprintf("https://github.com/kfriede/nab/releases/download/v%s/%s", versionNum, archiveName)

	destPath := filepath.Join(dir, "nab")

	// Check if binary already exists
	if info, err := os.Stat(destPath); err == nil && !info.IsDir() {
		printer.Status(fmt.Sprintf("Binary already exists at %s — replacing", destPath))
	}

	printer.Status(fmt.Sprintf("Downloading nab v%s for linux/%s...", versionNum, goarch))
	printer.Status(fmt.Sprintf("URL: %s", downloadURL))

	resp, err := http.Get(downloadURL) //nolint:gosec // URL is constructed from known constants
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d — check that v%s exists at github.com/kfriede/nab/releases", resp.StatusCode, versionNum)
	}

	// Extract the nab binary from the tar.gz archive
	if err := extractBinaryFromTarGz(resp.Body, destPath); err != nil {
		return fmt.Errorf("extracting binary: %w", err)
	}

	if err := os.Chmod(destPath, 0o755); err != nil {
		return fmt.Errorf("setting permissions: %w", err)
	}

	printer.Success(fmt.Sprintf("Installed nab to %s", destPath))
	return nil
}

func extractBinaryFromTarGz(r io.Reader, destPath string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("binary 'nab' not found in archive")
		}
		if err != nil {
			return fmt.Errorf("reading archive: %w", err)
		}

		if filepath.Base(header.Name) == "nab" && header.Typeflag == tar.TypeReg {
			out, err := os.Create(destPath)
			if err != nil {
				return fmt.Errorf("creating file: %w", err)
			}
			defer out.Close()

			if _, err := io.Copy(out, tr); err != nil { //nolint:gosec // archive from known source
				return fmt.Errorf("writing file: %w", err)
			}
			return nil
		}
	}
}

func fetchLatestVersion() (string, error) {
	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get("https://github.com/kfriede/nab/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", fmt.Errorf("no redirect from /releases/latest")
	}

	// Location is like: https://github.com/kfriede/nab/releases/tag/v0.1.0
	parts := strings.Split(loc, "/")
	tag := parts[len(parts)-1]
	if !strings.HasPrefix(tag, "v") {
		return "", fmt.Errorf("unexpected tag format: %s", tag)
	}
	return tag, nil
}

func showCoworkEnvConfig() {
	// Read current token
	token := cfg.Token
	if token == "" {
		secret, err := config.GetSecret(cfg.Profile)
		if err == nil && secret != "" {
			token = secret
		}
	}

	budget := cfg.Budget

	printer.Status("── Cowork Environment Variables ──")
	fmt.Fprintln(os.Stderr)

	if token != "" {
		// Show full token so the user can copy it for Cowork settings
		fmt.Fprintf(os.Stderr, "  NAB_TOKEN=%s\n", token)
	} else {
		fmt.Fprintln(os.Stderr, "  NAB_TOKEN=<not configured — run `nab login` first>")
	}

	if budget != "" {
		fmt.Fprintf(os.Stderr, "  NAB_BUDGET=%s\n", budget)
	} else {
		fmt.Fprintf(os.Stderr, "  NAB_BUDGET=last-used\n")
	}

	fmt.Fprintln(os.Stderr)
	printer.Status("Add these in Claude Desktop → Settings → Environment Variables")
	printer.Status("or set them in your Cowork workspace settings.")
}

func showCoworkNextSteps(dir string) {
	printer.Status("── Next Steps ──")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "  1. Open Claude Desktop → Cowork tab\n")
	fmt.Fprintf(os.Stderr, "  2. Click \"Work in a folder\" → select: %s\n", dir)
	fmt.Fprintf(os.Stderr, "  3. Set the environment variables shown above\n")
	fmt.Fprintf(os.Stderr, "  4. Ask Claude: \"Run ./nab version to verify setup\"\n")
	fmt.Fprintln(os.Stderr)
	printer.Status("Claude will use ./nab from the workspace for all YNAB commands.")
}
