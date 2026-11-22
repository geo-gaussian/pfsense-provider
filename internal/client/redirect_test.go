package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestAuthenticationDoesNotFollowRedirects(t *testing.T) {
	var redirected atomic.Bool
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/login" {
			http.Redirect(w, r, "/redirected", http.StatusFound)
			return
		}
		redirected.Store(true)
	}))
	defer server.Close()
	c, err := New(server.URL, "test-user", strings.Repeat("x", 12), true)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.ensureSession(context.Background()); err == nil {
		t.Fatal("authentication unexpectedly followed a redirect")
	}
	if redirected.Load() {
		t.Fatal("authentication sent a request to the redirect target")
	}
}
