package resource_firewall_alias

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type mockResponse struct {
	status int
	body   string
}

func TestLifecycleFailures(t *testing.T) {
	tests := []struct {
		name        string
		operation   string
		responses   map[string]mockResponse
		cancel      bool
		wantSummary string
	}{
		{"create transport failure", "create", nil, true, "Unable to create alias"},
		{"create rejected", "create", map[string]mockResponse{"PUT /aliases": {500, `{}`}}, false, "Unable to create alias"},
		{"create invalid result", "create", map[string]mockResponse{"PUT /aliases": {200, `{}`}}, false, "Unable to create alias"},
		{"create reload failure", "create", map[string]mockResponse{"POST /system/filterreload/reload": {500, `{}`}}, false, "Alias created but firewall reload failed"},
		{"create missing on verification", "create", map[string]mockResponse{"GET /aliases/example": {404, `{}`}}, false, "Unable to verify created alias"},
		{"create truncated on verification", "create", map[string]mockResponse{"GET /aliases/example": {200, `{"name":"example","truncated":true}`}}, false, "Unable to read created alias"},
		{"read transport failure", "read", nil, true, "Unable to read alias"},
		{"read rejected", "read", map[string]mockResponse{"GET /aliases/example": {500, `{}`}}, false, "Unable to read alias"},
		{"read truncated", "read", map[string]mockResponse{"GET /aliases/example": {200, `{"name":"example","truncated":true}`}}, false, "Unable to read alias"},
		{"read success", "read", nil, false, ""},
		{"update transport failure", "update", nil, true, "Unable to update alias"},
		{"update rejected", "update", map[string]mockResponse{"POST /aliases/example": {500, `{}`}}, false, "Unable to update alias"},
		{"update invalid result", "update", map[string]mockResponse{"POST /aliases/example": {200, `{}`}}, false, "Unable to update alias"},
		{"update reload failure", "update", map[string]mockResponse{"POST /system/filterreload/reload": {500, `{}`}}, false, "Alias updated but firewall reload failed"},
		{"update missing on verification", "update", map[string]mockResponse{"GET /aliases/example": {404, `{}`}}, false, "Unable to verify updated alias"},
		{"update truncated on verification", "update", map[string]mockResponse{"GET /aliases/example": {200, `{"name":"example","truncated":true}`}}, false, "Unable to read updated alias"},
		{"delete transport failure", "delete", nil, true, "Unable to delete alias"},
		{"delete rejected", "delete", map[string]mockResponse{"DELETE /aliases/example": {500, `{}`}}, false, "Unable to delete alias"},
		{"delete invalid result", "delete", map[string]mockResponse{"DELETE /aliases/example": {200, `{}`}}, false, "Unable to delete alias"},
		{"delete reload failure", "delete", map[string]mockResponse{"POST /system/filterreload/reload": {500, `{}`}}, false, "Alias deleted but firewall reload failed"},
		{"delete verification error", "delete", map[string]mockResponse{"GET /aliases/example": {500, `{}`}}, false, "Unable to verify deleted alias"},
		{"delete still present", "delete", map[string]mockResponse{"GET /aliases/example": {200, `{"name":"example"}`}}, false, "Unable to verify deleted alias"},
		{"delete bad request and still present", "delete", map[string]mockResponse{"DELETE /aliases/example": {400, `{}`}, "GET /aliases/example": {200, `{"name":"example"}`}}, false, "Unable to delete alias"},
		{"delete bad request and absent", "delete", map[string]mockResponse{"DELETE /aliases/example": {400, `{}`}}, false, ""},
		{"delete already absent", "delete", map[string]mockResponse{"DELETE /aliases/example": {404, `{}`}}, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				key := req.Method + " " + req.URL.Path
				result, ok := tt.responses[key]
				if !ok {
					switch key {
					case "PUT /aliases", "POST /aliases/example", "DELETE /aliases/example":
						result = mockResponse{200, `{"msg":"ok"}`}
					case "POST /system/filterreload/reload":
						result = mockResponse{200, `{"result":"started"}`}
					case "GET /aliases/example":
						if tt.operation == "delete" {
							result = mockResponse{404, `{}`}
						} else {
							result = mockResponse{200, `{"name":"example"}`}
						}
					default:
						t.Errorf("unexpected request %s", key)
						result = mockResponse{500, `{}`}
					}
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(result.status)
				_, _ = fmt.Fprint(w, result.body)
			}))
			defer server.Close()
			client, err := api.NewClientWithResponses(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			r := &aliasResource{client: client}
			ctx := context.Background()
			var schemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			s := schemaResp.Schema
			model := FirewallAliasModel{
				Name: types.StringValue("example"), Type: types.StringNull(),
				Address: types.StringNull(), Descr: types.StringNull(), Detail: types.StringNull(),
				Tags: types.StringNull(), Updatefreq: types.StringNull(),
				Targets: types.ListNull(TargetsType{ObjectType: types.ObjectType{AttrTypes: TargetsValue{}.AttributeTypes(ctx)}}),
			}
			plan := tfsdk.Plan{Schema: s}
			state := tfsdk.State{Schema: s}
			if diags := plan.Set(ctx, &model); diags.HasError() {
				t.Fatal(diags)
			}
			if diags := state.Set(ctx, &model); diags.HasError() {
				t.Fatal(diags)
			}
			if tt.cancel {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			var summaries []string
			switch tt.operation {
			case "create":
				resp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
				r.Create(ctx, resource.CreateRequest{Plan: plan}, &resp)
				for _, d := range resp.Diagnostics.Errors() {
					summaries = append(summaries, d.Summary())
				}
				if tt.name == "create reload failure" {
					var recovered FirewallAliasModel
					if diags := resp.State.Get(ctx, &recovered); diags.HasError() || recovered.Name.ValueString() != "example" {
						t.Fatalf("recovery state lost: %v, %#v", diags, recovered)
					}
				}
			case "read":
				resp := resource.ReadResponse{State: state}
				r.Read(ctx, resource.ReadRequest{State: state}, &resp)
				for _, d := range resp.Diagnostics.Errors() {
					summaries = append(summaries, d.Summary())
				}
			case "update":
				resp := resource.UpdateResponse{State: state}
				r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, &resp)
				for _, d := range resp.Diagnostics.Errors() {
					summaries = append(summaries, d.Summary())
				}
				if tt.name == "update reload failure" && !resp.State.Raw.Equal(state.Raw) {
					t.Fatal("prior state was not retained")
				}
			case "delete":
				resp := resource.DeleteResponse{State: state}
				r.Delete(ctx, resource.DeleteRequest{State: state}, &resp)
				for _, d := range resp.Diagnostics.Errors() {
					summaries = append(summaries, d.Summary())
				}
			default:
				t.Fatalf("unexpected operation %q", tt.operation)
			}
			if tt.wantSummary == "" && len(summaries) != 0 {
				t.Fatalf("unexpected errors: %v", summaries)
			}
			if tt.wantSummary != "" && (len(summaries) == 0 || summaries[0] != tt.wantSummary) {
				t.Fatalf("errors = %v, want %q", summaries, tt.wantSummary)
			}
		})
	}
}

func TestGetRejectsUnexpectedResponses(t *testing.T) {
	tests := []struct {
		name    string
		alias   mockResponse
		list    mockResponse
		missing bool
		wantErr bool
	}{
		{"not found", mockResponse{404, `{}`}, mockResponse{}, true, false},
		{"no JSON", mockResponse{200, `null`}, mockResponse{}, false, true},
		{"different name", mockResponse{200, `{"name":"other"}`}, mockResponse{}, false, true},
		{"failed list", mockResponse{400, `{}`}, mockResponse{500, `{}`}, false, true},
		{"missing list", mockResponse{400, `{}`}, mockResponse{200, `{}`}, false, true},
		{"absent from list", mockResponse{400, `{}`}, mockResponse{200, `{"aliases":[]}`}, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				result := tt.alias
				if req.URL.Path == "/aliases" {
					result = tt.list
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(result.status)
				_, _ = fmt.Fprint(w, result.body)
			}))
			defer server.Close()
			client, err := api.NewClientWithResponses(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			_, missing, err := (&aliasResource{client: client}).get(context.Background(), "example")
			if missing != tt.missing || (err != nil) != tt.wantErr {
				t.Fatalf("missing = %v, error = %v", missing, err)
			}
		})
	}
	if _, missing, err := (&aliasResource{}).get(context.Background(), "not valid"); err == nil || missing {
		t.Fatal("invalid alias name was accepted")
	}
}
