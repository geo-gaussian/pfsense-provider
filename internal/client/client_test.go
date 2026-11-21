package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSessionAndSerialization(t *testing.T) {
	var active atomic.Int32
	var maximum atomic.Int32
	var logins atomic.Int32
	user := "test-user"
	password := strings.Repeat("x", 12)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/login" {
			logins.Add(1)
			var body struct{ Username, Password string }
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body.Username != base64.StdEncoding.EncodeToString([]byte(user)) ||
				body.Password != base64.StdEncoding.EncodeToString([]byte(password)) {
				t.Error("credentials were not encoded according to the API schema")
			}
			_, _ = w.Write([]byte(`{"token":"test-token"}`))
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("missing access token")
		}
		n := active.Add(1)
		if n > maximum.Load() {
			maximum.Store(n)
		}
		time.Sleep(5 * time.Millisecond)
		active.Add(-1)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	c, err := New(server.URL, user, password, true)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/api/test", nil)
			if err != nil {
				t.Error(err)
				return
			}
			resp, err := c.Do(req)
			if err != nil {
				t.Error(err)
				return
			}
			resp.Body.Close()
		}()
	}
	wg.Wait()
	if maximum.Load() != 1 || logins.Load() != 1 {
		t.Fatalf("maximum concurrent requests = %d, login count = %d", maximum.Load(), logins.Load())
	}
}

func TestRefreshSession(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/login":
			http.SetCookie(w, &http.Cookie{Name: "refresh", Value: "test", Path: "/api"})
			_, _ = w.Write([]byte(`{"token":"old"}`))
		case "/api/login/refresh":
			if _, err := r.Cookie("refresh"); err != nil {
				t.Error("refresh cookie missing")
			}
			if r.Header.Get("Authorization") != "Bearer old" {
				t.Error("refresh access token missing")
			}
			_, _ = w.Write([]byte(`{"token":"new"}`))
		default:
			if r.Header.Get("Authorization") != "Bearer new" {
				t.Error("refreshed token missing")
			}
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer server.Close()
	c, err := New(server.URL, "test-user", strings.Repeat("x", 12), true)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.ensureSession(context.Background()); err != nil {
		t.Fatal(err)
	}
	c.renewAt = time.Time{}
	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}

func TestRejectInvalidOrigin(t *testing.T) {
	for _, origin := range []string{"http://example.test", "https://example.test/path", "https://user@example.test"} {
		if _, err := New(origin, "test-user", strings.Repeat("x", 12), false); err == nil {
			t.Errorf("accepted invalid origin %q", origin)
		}
	}
}
func TestAuthenticationFailureStopsRequest(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errmsg":"invalid credentials"}`))
	}))
	defer server.Close()
	c, err := New(server.URL, "test-user", strings.Repeat("x", 12), true)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Do(req); err == nil {
		t.Fatal("authentication failure was not reported")
	}
	if requests.Load() != 1 {
		t.Fatalf("sent %d requests after authentication failure", requests.Load())
	}
}
