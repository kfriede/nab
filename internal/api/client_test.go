package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("expected Bearer token header")
		}
		if r.Header.Get("User-Agent") != userAgent {
			t.Errorf("expected User-Agent=%s, got %s", userAgent, r.Header.Get("User-Agent"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"budgets": []map[string]any{
					{"id": "budget-1", "name": "My Budget"},
				},
			},
		})
	}))
	defer server.Close()

	c := NewClient(ClientConfig{
		Token:     "test-token",
		BaseURL:   server.URL,
		ErrWriter: discardWriter{},
	})

	var result map[string]any
	if err := c.GetJSON("", &result); err != nil {
		t.Fatalf("GetJSON failed: %v", err)
	}

	budgets, ok := result["budgets"].([]any)
	if !ok {
		t.Fatalf("expected budgets array, got %T", result["budgets"])
	}
	if len(budgets) != 1 {
		t.Errorf("expected 1 budget, got %d", len(budgets))
	}
}

func TestClientRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]string{"status": "ok"},
		})
	}))
	defer server.Close()

	c := NewClient(ClientConfig{
		Token:     "test-token",
		BaseURL:   server.URL,
		ErrWriter: discardWriter{},
	})

	var result map[string]string
	if err := c.GetJSON("", &result); err != nil {
		t.Fatalf("GetJSON failed after retries: %v", err)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestClientAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"id":     "404",
				"name":   "not_found",
				"detail": "Budget not found",
			},
		})
	}))
	defer server.Close()

	c := NewClient(ClientConfig{
		BaseURL:   server.URL,
		ErrWriter: discardWriter{},
	})

	_, err := c.Get("")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Name != "not_found" {
		t.Errorf("expected name 'not_found', got %q", apiErr.Name)
	}
}

func TestUnwrapData(t *testing.T) {
	input := `{"data":{"budgets":[{"id":"1","name":"Test"}]}}`
	result, err := unwrapData([]byte(input))
	if err != nil {
		t.Fatalf("unwrapData failed: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result, &data); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if _, ok := data["budgets"]; !ok {
		t.Error("expected 'budgets' key in unwrapped data")
	}
}

func TestMilliunitsConversion(t *testing.T) {
	tests := []struct {
		milliunits int64
		expected   string
	}{
		{1000, "$1.00"},
		{-1500, "-$1.50"},
		{0, "$0.00"},
		{1234560, "$1234.56"},
	}

	for _, tt := range tests {
		got := FormatMilliunits(tt.milliunits)
		if got != tt.expected {
			t.Errorf("FormatMilliunits(%d) = %q, want %q", tt.milliunits, got, tt.expected)
		}
	}
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }
