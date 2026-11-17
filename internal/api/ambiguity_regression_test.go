package api

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestMutationTimeoutAndUnrecognizedRejection(t *testing.T) {
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		for _, mode := range []string{"timeout", "malformed_400", "missing_429"} {
			t.Run(method+"/"+mode, func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					switch mode {
					case "timeout":
						time.Sleep(50 * time.Millisecond)
						_, _ = io.WriteString(w, `{"data":{"id":"fixture"}}`)
					case "malformed_400":
						w.WriteHeader(400)
						_, _ = io.WriteString(w, `<html>proxy failure</html>`)
					case "missing_429":
						w.WriteHeader(429)
						_, _ = io.WriteString(w, `{}`)
					}
				}))
				defer server.Close()
				timeout := time.Second
				if mode == "timeout" {
					timeout = 10 * time.Millisecond
				}
				c := NewClient(ClientConfig{BaseURL: server.URL, Timeout: timeout})
				err := mutateForTest(c, method)
				var unknown *MutationOutcomeUnknownError
				if !errors.As(err, &unknown) || calls.Load() != 1 {
					t.Fatalf("calls=%d err=%v", calls.Load(), err)
				}
			})
		}
	}
}
