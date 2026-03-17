package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
)

// Format represents an output format.
type Format int

const (
	FormatTable Format = iota
	FormatJSON
	FormatCSV
	FormatNDJSON
)

// PrinterConfig configures a Printer.
type PrinterConfig struct {
	Format    Format
	IsTTY     bool
	NoColor   bool
	Quiet     bool
	Fields    string // comma-separated field list
	Writer    io.Writer
	ErrWriter io.Writer
}

// Printer handles all output formatting.
type Printer struct {
	config PrinterConfig
	fields []string
}

// NewPrinter creates a new Printer with the given configuration.
func NewPrinter(cfg PrinterConfig) *Printer {
	if cfg.Writer == nil {
		cfg.Writer = os.Stdout
	}
	if cfg.ErrWriter == nil {
		cfg.ErrWriter = os.Stderr
	}
	if cfg.NoColor {
		color.NoColor = true
	}

	var fields []string
	if cfg.Fields != "" {
		for _, f := range strings.Split(cfg.Fields, ",") {
			f = strings.TrimSpace(f)
			if f != "" {
				fields = append(fields, f)
			}
		}
	}

	return &Printer{
		config: cfg,
		fields: fields,
	}
}

// PrintResult outputs data in the configured format.
// data should be a slice of maps or a single map.
func (p *Printer) PrintResult(data any) error {
	switch p.config.Format {
	case FormatJSON:
		return p.printJSON(data)
	case FormatNDJSON:
		return p.printNDJSON(data)
	case FormatCSV:
		return p.printCSV(data)
	case FormatTable:
		return p.printTable(data)
	default:
		return p.printJSON(data)
	}
}

// PrintError outputs a structured error to stderr.
func (p *Printer) PrintError(err AppError) {
	if p.shouldStructuredErrors() {
		errObj := map[string]string{
			"code":     err.Code,
			"message":  err.Message,
			"guidance": err.Guidance,
		}
		enc := json.NewEncoder(p.config.ErrWriter)
		enc.SetIndent("", "  ")
		_ = enc.Encode(errObj)
	} else {
		errColor := color.New(color.FgRed, color.Bold)
		_, _ = fmt.Fprintf(p.config.ErrWriter, "%s %s\n", errColor.Sprint("Error:"), err.Message)
		if err.Guidance != "" {
			hintColor := color.New(color.FgYellow)
			_, _ = fmt.Fprintf(p.config.ErrWriter, "%s %s\n", hintColor.Sprint("Hint:"), err.Guidance)
		}
	}
}

// Status prints a status message to stderr (never pollutes stdout).
func (p *Printer) Status(msg string) {
	if !p.config.Quiet {
		_, _ = fmt.Fprintln(p.config.ErrWriter, msg)
	}
}

// Success prints a success message to stderr.
func (p *Printer) Success(msg string) {
	if !p.config.Quiet {
		successColor := color.New(color.FgGreen)
		_, _ = fmt.Fprintf(p.config.ErrWriter, "%s %s\n", successColor.Sprint("✓"), msg)
	}
}

func (p *Printer) shouldStructuredErrors() bool {
	return !p.config.IsTTY || p.config.Format == FormatJSON || p.config.Format == FormatNDJSON
}

func (p *Printer) printJSON(data any) error {
	filtered := p.applyFieldMask(data)
	enc := json.NewEncoder(p.config.Writer)
	enc.SetIndent("", "  ")
	return enc.Encode(filtered)
}

func (p *Printer) printNDJSON(data any) error {
	items, ok := toSlice(data)
	if !ok {
		// Single item — just print as one JSON line
		filtered := p.applyFieldMask(data)
		return json.NewEncoder(p.config.Writer).Encode(filtered)
	}

	enc := json.NewEncoder(p.config.Writer)
	for _, item := range items {
		filtered := p.applyFieldMask(item)
		if err := enc.Encode(filtered); err != nil {
			return err
		}
	}
	return nil
}

func (p *Printer) printCSV(data any) error {
	items, ok := toSlice(data)
	if !ok {
		items = []any{data}
	}
	if len(items) == 0 {
		return nil
	}

	w := csv.NewWriter(p.config.Writer)
	defer w.Flush()

	// Determine columns from first item
	firstMap, ok := items[0].(map[string]any)
	if !ok {
		return fmt.Errorf("CSV output requires map data")
	}

	columns := p.csvColumns(firstMap)
	if err := w.Write(columns); err != nil {
		return err
	}

	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		row := make([]string, len(columns))
		for i, col := range columns {
			row[i] = fmt.Sprintf("%v", m[col])
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}

func (p *Printer) printTable(data any) error {
	items, ok := toSlice(data)
	if !ok {
		items = []any{data}
	}
	if len(items) == 0 {
		_, _ = fmt.Fprintln(p.config.Writer, "No results.")
		return nil
	}

	firstMap, ok := items[0].(map[string]any)
	if !ok {
		// Fall back to JSON for non-map data
		return p.printJSON(data)
	}

	columns := p.csvColumns(firstMap)
	if len(columns) == 0 {
		return nil
	}

	// Calculate column widths
	widths := make([]int, len(columns))
	for i, col := range columns {
		widths[i] = len(col)
	}
	rows := make([][]string, len(items))
	for i, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		row := make([]string, len(columns))
		for j, col := range columns {
			val := fmt.Sprintf("%v", m[col])
			row[j] = val
			if len(val) > widths[j] {
				widths[j] = len(val)
			}
		}
		rows[i] = row
	}

	// Cap column widths at 50 chars
	for i := range widths {
		if widths[i] > 50 {
			widths[i] = 50
		}
	}

	// Print header
	headerColor := color.New(color.Bold)
	for i, col := range columns {
		if i > 0 {
			_, _ = fmt.Fprint(p.config.Writer, "  ")
		}
		_, _ = headerColor.Fprintf(p.config.Writer, "%-*s", widths[i], strings.ToUpper(col))
	}
	_, _ = fmt.Fprintln(p.config.Writer)

	// Print separator
	for i, w := range widths {
		if i > 0 {
			_, _ = fmt.Fprint(p.config.Writer, "  ")
		}
		_, _ = fmt.Fprint(p.config.Writer, strings.Repeat("─", w))
	}
	_, _ = fmt.Fprintln(p.config.Writer)

	// Print rows
	for _, row := range rows {
		if row == nil {
			continue
		}
		for i, val := range row {
			if i > 0 {
				_, _ = fmt.Fprint(p.config.Writer, "  ")
			}
			// Truncate if too long
			if len(val) > widths[i] {
				val = val[:widths[i]-1] + "…"
			}
			_, _ = fmt.Fprintf(p.config.Writer, "%-*s", widths[i], val)
		}
		_, _ = fmt.Fprintln(p.config.Writer)
	}

	return nil
}

// applyFieldMask filters data to only include the requested fields.
func (p *Printer) applyFieldMask(data any) any {
	if len(p.fields) == 0 {
		return data
	}

	switch v := data.(type) {
	case map[string]any:
		filtered := make(map[string]any, len(p.fields))
		for _, f := range p.fields {
			if val, ok := v[f]; ok {
				filtered[f] = val
			}
		}
		return filtered
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = p.applyFieldMask(item)
		}
		return result
	default:
		return data
	}
}

func (p *Printer) csvColumns(m map[string]any) []string {
	if len(p.fields) > 0 {
		return p.fields
	}
	columns := make([]string, 0, len(m))
	for k := range m {
		columns = append(columns, k)
	}
	// Sort for deterministic output
	sortStrings(columns)
	return columns
}

func toSlice(data any) ([]any, bool) {
	switch v := data.(type) {
	case []any:
		return v, true
	case []map[string]any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result, true
	default:
		return nil, false
	}
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
