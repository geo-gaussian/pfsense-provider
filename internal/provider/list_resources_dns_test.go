package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
)

func TestMDNSDiscoveryRejectsInvalidResponse(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status int
		body   string
	}{
		{name: "HTTP failure", status: http.StatusServiceUnavailable},
		{name: "null response", status: http.StatusOK, body: `null`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet || req.URL.Path != "/services/mdns" {
					t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			client, err := api.NewClientWithResponses(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			identifiers, err := (&configurationList{client: client}).identifiers(context.Background(), "mdns")
			if err == nil || identifiers != nil {
				t.Fatalf("invalid response returned identifiers %v, error %v", identifiers, err)
			}
		})
	}
}
