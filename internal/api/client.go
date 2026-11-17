package api

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTimeout = 30 * time.Second
	maxRetries     = 3
	retryBaseDelay = 1 * time.Second
	userAgent      = "nab/0.2.6"

	// YNAB API base URL
	BaseURL = "https://api.ynab.com/v1"
)

// Client is the YNAB API client.
type Client struct {
	httpClient        *http.Client
	mutationTransport *http.Transport
	baseURL           string
	token             string
	verbose           bool
	debug             bool
	errWriter         io.Writer
	sleep             func(time.Duration) // injectable for deterministic read-backoff tests
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
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTimeout
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = BaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	if cfg.ErrWriter == nil {
		cfg.ErrWriter = io.Discard
	}
	// Fresh HTTP/1 connections prevent hidden net/http stale-connection and HTTP/2
	// retries. Application-level attempts alone are not a wire-attempt ceiling.
	mutationTransport := http.DefaultTransport.(*http.Transport).Clone()
	mutationTransport.DisableKeepAlives = true
	mutationTransport.ForceAttemptHTTP2 = false
	mutationTransport.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
	return &Client{
		mutationTransport: mutationTransport,
		sleep:             time.Sleep,
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

// MutationOutcomeUnknownError means a write may have committed. Never retry it
// blindly: reconcile the exact target with GET before a fresh preview/confirmation.
// Cause is retained for errors.Is/As, but not printed (it may contain a private URL).
type MutationOutcomeUnknownError struct{ Cause error }

func (e *MutationOutcomeUnknownError) Error() string {
	return "MUTATION_OUTCOME_UNKNOWN: mutation may have committed; reconcile with GET before a fresh dry-run and confirmation; do not blindly retry"
}
func (e *MutationOutcomeUnknownError) Unwrap() error { return e.Cause }

func isRead(method string) bool { return method == http.MethodGet || method == http.MethodHead }

// Do retries reads only. All other methods have one application attempt, no
// redirects and no replayable body (including net/http's transport retry path).
func (c *Client) Do(method, path string, body io.Reader) (*http.Response, error) {
	read := isRead(method)
	attempts := 1
	if read {
		attempts = maxRetries
	}
	var lastErr error
	for attempt := range attempts {
		req, err := http.NewRequest(method, c.baseURL+path, body)
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
		client := *c.httpClient
		if !read {
			if client.Transport == nil {
				client.Transport = c.mutationTransport
			}
			// A non-nil, non-rewindable body also prevents replay of empty DELETE
			// requests on stale keepalive connections. Never add idempotency headers.
			if req.Body == nil || req.Body == http.NoBody {
				req.Body = io.NopCloser(strings.NewReader(""))
			}
			req.GetBody = nil
			client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
		}
		if c.verbose {
			_, _ = fmt.Fprintf(c.errWriter, "[%s] %s\n", method, path)
		}
		resp, err := client.Do(req)
		if err != nil {
			if !read {
				return nil, &MutationOutcomeUnknownError{Cause: err}
			}
			lastErr = err
			if attempt < attempts-1 {
				c.sleep(retryDelay(attempt))
			}
			continue
		}
		if !read {
			if resp.StatusCode >= 300 && resp.StatusCode < 400 || resp.StatusCode >= 500 {
				_ = resp.Body.Close()
				return nil, &MutationOutcomeUnknownError{Cause: fmt.Errorf("HTTP %d", resp.StatusCode)}
			}
			return resp, nil // including 429: return the rejection, never replay
		}
		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d from %s", resp.StatusCode, method)
			if attempt < attempts-1 {
				delay := retryDelay(attempt)
				if resp.StatusCode == 429 {
					delay = retryAfter(resp.Header.Get("Retry-After"), time.Now(), delay)
				}
				c.sleep(delay)
			}
			continue
		}
		return resp, nil
	}
	return nil, fmt.Errorf("request failed after %d attempts: %w", attempts, lastErr)
}

// Invalid/negative Retry-After falls back to bounded exponential read backoff.
func retryAfter(value string, now time.Time, fallback time.Duration) time.Duration {
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil && seconds >= 0 {
		// Cap server-directed sleep at 30 seconds; each HTTP attempt also has a timeout.
		if seconds > 30 {
			return 30 * time.Second
		}
		return time.Duration(seconds) * time.Second
	}
	if date, err := http.ParseTime(value); err == nil {
		if date.Before(now) {
			return 0
		}
		if date.Sub(now) > 30*time.Second {
			return 30 * time.Second
		}
		return date.Sub(now)
	}
	return fallback
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

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return &MutationOutcomeUnknownError{Cause: err}
	}
	if resp.StatusCode >= 400 {
		return mutationRejection(data)
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
		return nil, &MutationOutcomeUnknownError{Cause: err}
	}

	if c.debug {
		_, _ = fmt.Fprintf(c.errWriter, "  response body: %s\n", truncate(string(respData), 2000))
	}

	if resp.StatusCode >= 400 {
		return nil, mutationRejection(respData)
	}

	var wrapper dataWrapper
	if err := json.Unmarshal(respData, &wrapper); err != nil {
		return nil, &MutationOutcomeUnknownError{Cause: err}
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(wrapper.Data, &object); err != nil {
		return nil, &MutationOutcomeUnknownError{Cause: err}
	}
	if object == nil {
		return nil, &MutationOutcomeUnknownError{Cause: fmt.Errorf("missing mutation data envelope")}
	}
	return wrapper.Data, nil
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

// Only a recognized API rejection is definite; malformed proxy/error pages
// cannot establish whether the upstream write committed.
func mutationRejection(data []byte) error {
	var wrapper errorWrapper
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return &MutationOutcomeUnknownError{Cause: err}
	}
	if wrapper.Error.Name == "" {
		return &MutationOutcomeUnknownError{Cause: fmt.Errorf("unrecognized rejection envelope")}
	}
	return &wrapper.Error
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
