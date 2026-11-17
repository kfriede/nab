package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func mutateForTest(c *Client, method string) error {
	switch method {
	case "POST":
		_, err := c.Post("/target", map[string]string{"memo": "fixture"})
		return err
	case "PUT":
		_, err := c.Put("/target", map[string]string{"memo": "fixture"})
		return err
	case "PATCH":
		_, err := c.Patch("/target", map[string]string{"memo": "fixture"})
		return err
	default:
		return c.Delete("/target")
	}
}

func TestMutationSingleAttempt(t *testing.T) {
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		for _, status := range []int{200, 400, 401, 404, 409, 429, 500, 502, 503, 504, 301, 302, 303, 307, 308} {
			t.Run(fmt.Sprintf("%s/%d", method, status), func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Method != method {
						t.Errorf("method changed: %s", r.Method)
					}
					w.Header().Set("Location", "/redirected")
					w.Header().Set("Retry-After", "0")
					w.WriteHeader(status)
					if status >= 400 {
						_, _ = io.WriteString(w, `{"error":{"name":"fixture_rejection"}}`)
					} else {
						_, _ = io.WriteString(w, `{"data":{"id":"fixture"}}`)
					}
				}))
				defer server.Close()
				c := NewClient(ClientConfig{BaseURL: server.URL})
				c.sleep = func(time.Duration) { t.Fatal("mutation slept/retried") }
				err := mutateForTest(c, method)
				var unknown *MutationOutcomeUnknownError
				wantUnknown := status >= 500 || status >= 300 && status < 400
				if errors.As(err, &unknown) != wantUnknown {
					t.Fatalf("unknown=%v, err=%v", wantUnknown, err)
				}
				if status == 200 && err != nil {
					t.Fatal(err)
				}
				if status >= 400 && err == nil {
					t.Fatal("rejection accepted")
				}
				if calls.Load() != 1 {
					t.Fatalf("%d attempts", calls.Load())
				}
			})
		}
	}
}

func TestMutationLostResponseAndTruncatedBody(t *testing.T) {
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		for _, mode := range []string{"lost", "truncated", "malformed", "null", "missing", "array", "boolean", "number", "string"} {
			if method == "DELETE" && mode != "lost" && mode != "truncated" {
				continue
			} // DELETE permits empty success
			t.Run(method+"/"+mode, func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if mode == "lost" || mode == "truncated" {
						conn, buf, err := w.(http.Hijacker).Hijack()
						if err != nil {
							t.Error(err)
							return
						}
						defer func() { _ = conn.Close() }()
						if mode == "truncated" {
							_, _ = buf.WriteString("HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\n{\"data\":")
							_ = buf.Flush()
						}
						return
					}
					switch mode {
					case "array":
						_, _ = io.WriteString(w, `{"data":[]}`)
					case "boolean":
						_, _ = io.WriteString(w, `{"data":true}`)
					case "number":
						_, _ = io.WriteString(w, `{"data":42}`)
					case "string":
						_, _ = io.WriteString(w, `{"data":"unexpected"}`)
					case "malformed":
						_, _ = io.WriteString(w, `{"data":`)
					case "null":
						_, _ = io.WriteString(w, `{"data":null}`)
					case "missing":
						_, _ = io.WriteString(w, `{}`)
					}
				}))
				defer server.Close()
				c := NewClient(ClientConfig{BaseURL: server.URL})
				err := mutateForTest(c, method)
				var unknown *MutationOutcomeUnknownError
				if !errors.As(err, &unknown) {
					t.Fatalf("expected typed unknown: %T %v", err, err)
				}
				if calls.Load() != 1 {
					t.Fatalf("%d attempts", calls.Load())
				}
			})
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestMutationTransportReplayGuards(t *testing.T) {
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		for _, body := range []io.Reader{nil, strings.NewReader(""), strings.NewReader(`{"fixture":true}`)} {
			c := NewClient(ClientConfig{})
			var calls atomic.Int32
			sentinel := errors.New("private-url-must-not-leak")
			c.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Body == nil || r.Body == http.NoBody || r.GetBody != nil {
					t.Error("replayable mutation body")
				}
				if r.Header.Get("Idempotency-Key") != "" {
					t.Error("idempotency header")
				}
				return nil, sentinel
			})
			_, err := c.Do(method, "/fixture", body)
			var unknown *MutationOutcomeUnknownError
			if !errors.As(err, &unknown) || !errors.Is(err, sentinel) {
				t.Fatalf("typed cause lost: %v", err)
			}
			if strings.Contains(err.Error(), "private-url") {
				t.Fatal("cause leaked")
			}
			if calls.Load() != 1 {
				t.Fatalf("%d attempts", calls.Load())
			}
		}
	}
}

func TestReadRetryAfter(t *testing.T) {
	for _, status := range []int{429, 503} {
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if calls.Load() < 3 {
				w.Header().Set("Retry-After", "7")
				w.WriteHeader(status)
				return
			}
			_, _ = io.WriteString(w, `{"data":{"ok":true}}`)
		}))
		c := NewClient(ClientConfig{BaseURL: server.URL})
		var delays []time.Duration
		c.sleep = func(d time.Duration) { delays = append(delays, d) }
		_, err := c.Get("/fixture")
		server.Close()
		if err != nil || calls.Load() != 3 || len(delays) != 2 {
			t.Fatalf("calls=%d delays=%v err=%v", calls.Load(), delays, err)
		}
		want := []time.Duration{time.Second, 2 * time.Second}
		if status == 429 {
			want = []time.Duration{7 * time.Second, 7 * time.Second}
		}
		for i := range want {
			if delays[i] != want[i] {
				t.Fatalf("delays=%v want=%v", delays, want)
			}
		}
	}
}

func TestRetryAfterBounds(t *testing.T) {
	now := time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		value string
		want  time.Duration
	}{
		{"0", 0}, {"7", 7 * time.Second}, {"999999999", 30 * time.Second}, {"-1", time.Second}, {"invalid", time.Second},
		{now.Add(8 * time.Second).Format(http.TimeFormat), 8 * time.Second},
		{now.Add(time.Hour).Format(http.TimeFormat), 30 * time.Second},
		{now.Add(-time.Hour).Format(http.TimeFormat), 0},
	} {
		if got := retryAfter(tt.value, now, time.Second); got != tt.want {
			t.Errorf("%q got %v want %v", tt.value, got, tt.want)
		}
	}
}

func TestReadTransportRetryExhaustion(t *testing.T) {
	c := NewClient(ClientConfig{})
	var calls atomic.Int32
	sleeps := 0
	c.httpClient.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, io.EOF })
	c.sleep = func(time.Duration) { sleeps++ }
	_, err := c.Get("/fixture")
	var unknown *MutationOutcomeUnknownError
	if err == nil || errors.As(err, &unknown) || calls.Load() != 3 || sleeps != 2 {
		t.Fatalf("calls=%d sleeps=%d err=%v", calls.Load(), sleeps, err)
	}
}
