package api

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

const (
	defaultTimeout = 30 * time.Second
	maxRetries     = 3
	retryBaseDelay = 1 * time.Second
	userAgent      = "nab/0.1.0"

	// YNAB API base URL
	BaseURL = "https://api.ynab.com/v1"
)

// Client is the YNAB API client.
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
	verbose    bool
	debug      bool
	errWriter  io.Writer
}

// ClientConfig configures a Client.
type ClientConfig struct {
	Token     string
	BaseURL   string
	Timeout   time.Duration
	Verbose   bool
	Debug     bool
	ErrWriter io.Writer
}

// NewClient creates a new YNAB API client.
func NewClient(cfg ClientConfig) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultTimeout
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = BaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return &Client{
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		baseURL:   baseURL,
		token:     cfg.Token,
		verbose:   cfg.Verbose,
		debug:     cfg.Debug,
		errWriter: cfg.ErrWriter,
	}
}

// APIError represents an error returned by the YNAB API.
type APIError struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

func (e *APIError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s: %s", e.Name, e.Detail)
	}
	return e.Name
}

// dataWrapper is the top-level wrapper for all YNAB API responses.
type dataWrapper struct {
	Data json.RawMessage `json:"data"`
}

// errorWrapper is the top-level wrapper for YNAB error responses.
type errorWrapper struct {
	Error APIError `json:"error"`
}

// Do performs an HTTP request with retry logic.
func (c *Client) Do(method, path string, body io.Reader) (*http.Response, error) {
	url := c.baseURL + path

	var lastErr error
	for attempt := range maxRetries {
		req, err := http.NewRequest(method, url, body)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}

		if c.verbose {
			_, _ = fmt.Fprintf(c.errWriter, "[%s] %s\n", method, url)
		}

		start := time.Now()
		resp, err := c.httpClient.Do(req)
		elapsed := time.Since(start)

		if c.verbose {
			if err != nil {
				_, _ = fmt.Fprintf(c.errWriter, "  error: %v (%.1fs)\n", err, elapsed.Seconds())
			} else {
				_, _ = fmt.Fprintf(c.errWriter, "  %d (%.1fs)\n", resp.StatusCode, elapsed.Seconds())
			}
		}

		if err != nil {
			lastErr = err
			if attempt < maxRetries-1 {
				time.Sleep(retryDelay(attempt))
			}
			continue
		}

		// Retry on 429 and 5xx
		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d from %s %s", resp.StatusCode, method, path)
			if attempt < maxRetries-1 {
				delay := retryDelay(attempt)
				if c.verbose {
					_, _ = fmt.Fprintf(c.errWriter, "  retrying in %v...\n", delay)
				}
				time.Sleep(delay)
			}
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", maxRetries, lastErr)
}

// Get performs a GET request and returns the unwrapped response body.
// YNAB wraps all responses in {"data": {...}} — this unwraps it.
func (c *Client) Get(path string) ([]byte, error) {
	resp, err := c.Do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if c.debug {
		_, _ = fmt.Fprintf(c.errWriter, "  response body: %s\n", truncate(string(data), 2000))
	}

	if resp.StatusCode >= 400 {
		return nil, parseAPIError(data)
	}

	return unwrapData(data)
}

// GetJSON performs a GET request and unmarshals the unwrapped response.
func (c *Client) GetJSON(path string, target any) error {
	data, err := c.Get(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// Post performs a POST request with a JSON body.
func (c *Client) Post(path string, body any) ([]byte, error) {
	return c.mutate("POST", path, body)
}

// Put performs a PUT request with a JSON body.
func (c *Client) Put(path string, body any) ([]byte, error) {
	return c.mutate("PUT", path, body)
}

// Patch performs a PATCH request with a JSON body.
func (c *Client) Patch(path string, body any) ([]byte, error) {
	return c.mutate("PATCH", path, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(path string) error {
	resp, err := c.Do("DELETE", path, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		return parseAPIError(data)
	}

	return nil
}

func (c *Client) mutate(method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		if c.debug {
			_, _ = fmt.Fprintf(c.errWriter, "  request body: %s\n", truncate(string(data), 2000))
		}
		bodyReader = strings.NewReader(string(data))
	}

	resp, err := c.Do(method, path, bodyReader)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if c.debug {
		_, _ = fmt.Fprintf(c.errWriter, "  response body: %s\n", truncate(string(respData), 2000))
	}

	if resp.StatusCode >= 400 {
		return nil, parseAPIError(respData)
	}

	return unwrapData(respData)
}

// unwrapData extracts the "data" field from a YNAB API response.
func unwrapData(raw []byte) ([]byte, error) {
	var wrapper dataWrapper
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		// If it doesn't have a data wrapper, return raw
		return raw, nil
	}
	if wrapper.Data == nil {
		return raw, nil
	}
	return wrapper.Data, nil
}

func parseAPIError(data []byte) *APIError {
	var wrapper errorWrapper
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return &APIError{Name: "unknown_error", Detail: string(data)}
	}
	return &wrapper.Error
}

func retryDelay(attempt int) time.Duration {
	return time.Duration(math.Pow(2, float64(attempt))) * retryBaseDelay
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// MilliunitsToFloat converts YNAB milliunit amounts to float64 dollars.
// YNAB uses milliunits: 1000 milliunits = $1.00
func MilliunitsToFloat(milliunits int64) float64 {
	return float64(milliunits) / 1000.0
}

// FormatMilliunits formats a milliunit amount as a currency string (e.g., "$1,234.56").
func FormatMilliunits(milliunits int64) string {
	dollars := float64(milliunits) / 1000.0
	if dollars < 0 {
		return fmt.Sprintf("-$%.2f", -dollars)
	}
	return fmt.Sprintf("$%.2f", dollars)
}
