package api

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestReviewFreshHTTP1MutationConnections(t *testing.T) {
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			var calls, connections atomic.Int32
			s := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.ProtoMajor != 1 {
					t.Errorf("mutation used %s", r.Proto)
				}
				if !r.Close {
					t.Error("keepalive enabled")
				}
				_, _ = io.WriteString(w, `{"data":{"id":"fixture"}}`)
			}))
			s.EnableHTTP2 = true
			s.Config.ConnState = func(_ net.Conn, state http.ConnState) {
				if state == http.StateNew {
					connections.Add(1)
				}
			}
			s.StartTLS()
			defer s.Close()
			c := NewClient(ClientConfig{BaseURL: s.URL})
			c.mutationTransport.TLSClientConfig = s.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
			for i := 0; i < 2; i++ {
				if err := mutateForTest(c, method); err != nil {
					t.Fatal(err)
				}
			}
			t.Logf("requests=%d fresh_connections=%d", calls.Load(), connections.Load())
			if calls.Load() != 2 || connections.Load() != 2 {
				t.Fatal("connection replay/reuse")
			}
		})
	}
}
