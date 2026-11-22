package resource_firewall_alias

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestRequestRejectsInvalidTargets(t *testing.T) {
	targets, diags := types.ListValue(types.StringType, []attr.Value{types.StringValue("invalid")})
	if diags.HasError() {
		t.Fatal(diags)
	}
	model := FirewallAliasModel{Name: types.StringValue("example"), Targets: targets}
	if _, err := model.request(context.Background()); err == nil {
		t.Fatal("accepted targets with an invalid element type")
	}
}

func TestGetRejectsNonJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = fmt.Fprint(w, "not JSON")
	}))
	defer server.Close()
	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, missing, err := (&aliasResource{client: client}).get(context.Background(), "example")
	if err == nil || missing {
		t.Fatalf("non-JSON response accepted: missing=%v, error=%v", missing, err)
	}
}

func TestReloadTransportFailure(t *testing.T) {
	client, err := api.NewClientWithResponses("https://example.test")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := (&aliasResource{client: client}).reload(ctx); err == nil {
		t.Fatal("reload transport error was not reported")
	}
}

func TestGetListTransportFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/aliases" {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			_ = conn.Close()
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, `{}`)
	}))
	defer server.Close()
	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, missing, err := (&aliasResource{client: client}).get(context.Background(), "example")
	if err == nil || missing {
		t.Fatalf("list failure was not reported: missing=%v, error=%v", missing, err)
	}
}
