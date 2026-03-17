package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestPrinterJSON(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(PrinterConfig{
		Format:    FormatJSON,
		Writer:    &buf,
		ErrWriter: &bytes.Buffer{},
	})

	data := map[string]any{
		"id":   "abc-123",
		"name": "Test Budget",
	}

	if err := p.PrintResult(data); err != nil {
		t.Fatalf("PrintResult failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if result["id"] != "abc-123" {
		t.Errorf("expected id=abc-123, got %v", result["id"])
	}
}

func TestPrinterJSONList(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(PrinterConfig{
		Format:    FormatJSON,
		Writer:    &buf,
		ErrWriter: &bytes.Buffer{},
	})

	data := []any{
		map[string]any{"id": "1", "name": "Budget 1"},
		map[string]any{"id": "2", "name": "Budget 2"},
	}

	if err := p.PrintResult(data); err != nil {
		t.Fatalf("PrintResult failed: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Output is not valid JSON array: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
}

func TestPrinterNDJSON(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(PrinterConfig{
		Format:    FormatNDJSON,
		Writer:    &buf,
		ErrWriter: &bytes.Buffer{},
	})

	data := []any{
		map[string]any{"id": "1"},
		map[string]any{"id": "2"},
		map[string]any{"id": "3"},
	}

	if err := p.PrintResult(data); err != nil {
		t.Fatalf("PrintResult failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 NDJSON lines, got %d", len(lines))
	}

	for i, line := range lines {
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("line %d is not valid JSON: %v", i, err)
		}
	}
}

func TestPrinterFieldMask(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(PrinterConfig{
		Format:    FormatJSON,
		Fields:    "id,name",
		Writer:    &buf,
		ErrWriter: &bytes.Buffer{},
	})

	data := map[string]any{
		"id":      "abc-123",
		"name":    "Test",
		"balance": 150000,
		"note":    "some note",
	}

	if err := p.PrintResult(data); err != nil {
		t.Fatalf("PrintResult failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if _, ok := result["id"]; !ok {
		t.Error("expected field 'id' to be present")
	}
	if _, ok := result["name"]; !ok {
		t.Error("expected field 'name' to be present")
	}
	if _, ok := result["balance"]; ok {
		t.Error("expected field 'balance' to be filtered out")
	}
	if _, ok := result["note"]; ok {
		t.Error("expected field 'note' to be filtered out")
	}
}

func TestPrinterFieldMaskList(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(PrinterConfig{
		Format:    FormatJSON,
		Fields:    "id",
		Writer:    &buf,
		ErrWriter: &bytes.Buffer{},
	})

	data := []any{
		map[string]any{"id": "1", "name": "A", "extra": "x"},
		map[string]any{"id": "2", "name": "B", "extra": "y"},
	}

	if err := p.PrintResult(data); err != nil {
		t.Fatalf("PrintResult failed: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Output is not valid JSON array: %v", err)
	}

	for i, item := range result {
		if len(item) != 1 {
			t.Errorf("item %d: expected 1 field, got %d", i, len(item))
		}
		if _, ok := item["id"]; !ok {
			t.Errorf("item %d: expected field 'id'", i)
		}
	}
}

func TestPrinterCSV(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(PrinterConfig{
		Format:    FormatCSV,
		Fields:    "id,name",
		Writer:    &buf,
		ErrWriter: &bytes.Buffer{},
	})

	data := []any{
		map[string]any{"id": "1", "name": "Budget 1"},
		map[string]any{"id": "2", "name": "Budget 2"},
	}

	if err := p.PrintResult(data); err != nil {
		t.Fatalf("PrintResult failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 { // header + 2 data rows
		t.Errorf("expected 3 CSV lines, got %d: %q", len(lines), buf.String())
	}

	if lines[0] != "id,name" {
		t.Errorf("expected header 'id,name', got %q", lines[0])
	}
}

func TestPrinterTable(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(PrinterConfig{
		Format:    FormatTable,
		NoColor:   true,
		Fields:    "id,name",
		Writer:    &buf,
		ErrWriter: &bytes.Buffer{},
	})

	data := []any{
		map[string]any{"id": "1", "name": "Budget 1"},
		map[string]any{"id": "2", "name": "Budget 2"},
	}

	if err := p.PrintResult(data); err != nil {
		t.Fatalf("PrintResult failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "ID") {
		t.Error("expected table header to contain 'ID'")
	}
	if !strings.Contains(output, "Budget 1") {
		t.Error("expected table to contain 'Budget 1'")
	}
}

func TestPrinterErrorStructured(t *testing.T) {
	var errBuf bytes.Buffer
	p := NewPrinter(PrinterConfig{
		Format:    FormatJSON,
		IsTTY:     false,
		Writer:    &bytes.Buffer{},
		ErrWriter: &errBuf,
	})

	p.PrintError(AppError{
		Code:     ErrCodeAuth,
		Message:  "Token expired",
		Guidance: "Run `nab login` to re-authenticate.",
	})

	var result map[string]string
	if err := json.Unmarshal(errBuf.Bytes(), &result); err != nil {
		t.Fatalf("Error output is not valid JSON: %v\nOutput: %s", err, errBuf.String())
	}

	if result["code"] != ErrCodeAuth {
		t.Errorf("expected code=%s, got %s", ErrCodeAuth, result["code"])
	}
	if result["guidance"] == "" {
		t.Error("expected guidance to be non-empty")
	}
}

func TestAppErrorExitCodes(t *testing.T) {
	tests := []struct {
		code     string
		expected int
	}{
		{ErrCodeGeneral, ExitGeneral},
		{ErrCodeAuth, ExitAuth},
		{ErrCodeNotFound, ExitNotFound},
		{ErrCodeConflict, ExitConflict},
		{ErrCodeValidation, ExitConflict},
	}

	for _, tt := range tests {
		e := AppError{Code: tt.code}
		if got := e.ExitCode(); got != tt.expected {
			t.Errorf("code=%s: expected exit %d, got %d", tt.code, tt.expected, got)
		}
	}
}

func TestEmptyListOutput(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(PrinterConfig{
		Format:    FormatTable,
		NoColor:   true,
		Writer:    &buf,
		ErrWriter: &bytes.Buffer{},
	})

	data := []any{}
	if err := p.PrintResult(data); err != nil {
		t.Fatalf("PrintResult failed: %v", err)
	}

	if !strings.Contains(buf.String(), "No results") {
		t.Error("expected 'No results' for empty list")
	}
}
