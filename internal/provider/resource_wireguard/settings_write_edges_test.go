package resource_wireguard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSettingsStateWriteAndTransportFailures(t *testing.T) {
	ctx := context.Background()
	s := testResourceSchema(t, NewSettings())
	plan := testPlan(t, s, &settingsModel{Enable: types.BoolValue(true)})
	t.Run("create state write failure", func(t *testing.T) {
		client := testAPIClient(t, func(w http.ResponseWriter, req *http.Request) {
			if req.Method != http.MethodPost || req.URL.Path != "/vpn/wg/settings" {
				t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
				w.WriteHeader(http.StatusNotFound)
				return
			}
			writeJSON(t, w, api.PfsenseResult{Msg: "ok"})
		})
		r := &settingsResource{resourceBase{client: client}}
		resp := resource.CreateResponse{State: tfsdk.State{Schema: schema.Schema{
			Attributes: map[string]schema.Attribute{
				"id":     schema.StringAttribute{Computed: true},
				"enable": schema.StringAttribute{Computed: true},
			},
		}}}
		r.Create(ctx, resource.CreateRequest{Plan: plan}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected incompatible state schema to fail")
		}
	})

	t.Run("write transport failure", func(t *testing.T) {
		server := httptest.NewServer(http.NotFoundHandler())
		client, err := api.NewClientWithResponses(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		server.Close()
		r := &settingsResource{resourceBase{client: client}}
		resp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
		r.Create(ctx, resource.CreateRequest{Plan: plan}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected a transport failure")
		}
	})

	for _, status := range []int{http.StatusNotFound, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status)+" settings verification", func(t *testing.T) {
			r := &settingsResource{resourceBase{client: testAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			})}}
			var diags diag.Diagnostics
			r.readSettingsIntoState(ctx, &settingsModel{}, &diags)
			if !diags.HasError() {
				t.Fatal("expected settings verification failure")
			}
		})
	}
}
