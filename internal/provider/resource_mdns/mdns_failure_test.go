package resource_mdns

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestMDNSConfigurationAndImport(t *testing.T) {
	ctx := context.Background()
	r := New().(*mdnsResource)
	var metadata resource.MetadataResponse
	r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "pfsense"}, &metadata)
	if metadata.TypeName != "pfsense_mdns" {
		t.Fatalf("unexpected resource type %q", metadata.TypeName)
	}
	client, err := api.NewClientWithResponses("http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name string
		data any
		err  bool
	}{
		{name: "not yet configured"},
		{name: "invalid type", data: "invalid", err: true},
		{name: "nil client", data: (*api.ClientWithResponses)(nil), err: true},
		{name: "client", data: client},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var resp resource.ConfigureResponse
			r.Configure(ctx, resource.ConfigureRequest{ProviderData: tt.data}, &resp)
			if resp.Diagnostics.HasError() != tt.err {
				t.Fatalf("unexpected configure diagnostics: %v", resp.Diagnostics)
			}
		})
	}
	s := testSchema(t, r)
	state := tfsdk.State{Schema: s}
	if diags := state.Set(ctx, &mdnsModel{
		IncludeInterfaces: types.ListNull(types.StringType), ExcludeInterfaces: types.ListNull(types.StringType),
		Publish: types.ObjectNull(publishTypes()),
	}); diags.HasError() {
		t.Fatal(diags)
	}
	response := resource.ImportStateResponse{State: state}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "mdns"}, &response)
	var id types.String
	response.Diagnostics.Append(response.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if response.Diagnostics.HasError() || id.ValueString() != "mdns" {
		t.Fatalf("import ID %v, diagnostics %v", id, response.Diagnostics)
	}
}

func TestMDNSInvalidTerraformInputs(t *testing.T) {
	ctx := context.Background()
	r := New().(*mdnsResource)
	s := testSchema(t, r)
	invalid := tftypes.NewValue(tftypes.String, "invalid")
	plan := tfsdk.Plan{Schema: s, Raw: invalid}
	state := tfsdk.State{Schema: s, Raw: invalid}
	create := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &create)
	read := resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, &read)
	update := resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, &update)
	for operation, diagnostics := range map[string]bool{
		"create": create.Diagnostics.HasError(), "read": read.Diagnostics.HasError(),
		"update": update.Diagnostics.HasError(),
	} {
		if !diagnostics {
			t.Errorf("%s accepted an invalid Terraform value", operation)
		}
	}
}

func TestMDNSLifecycleFailuresPreserveState(t *testing.T) {
	tests := []struct {
		name       string
		operation  string
		getStatus  int
		getBody    string
		postBody   string
		verifyFail bool
		wantError  bool
	}{
		{name: "create rejects read failure", operation: "create", getStatus: 503, wantError: true},
		{name: "update rejects read failure", operation: "update", getStatus: 503, wantError: true},
		{name: "read retains state on error", operation: "read", getStatus: 503, wantError: true},
		{name: "read refreshes state", operation: "read", getStatus: 200, getBody: `{"enable":false}`},
		{name: "delete missing is successful", operation: "delete", getStatus: 404},
		{name: "delete rejects read failure", operation: "delete", getStatus: 503, wantError: true},
		{
			name: "delete rejects malformed response", operation: "delete",
			getStatus: 200, getBody: `{"enable":true}`, postBody: `{`, wantError: true,
		},
		{
			name: "delete verification error", operation: "delete", getStatus: 200, getBody: `{"enable":true}`,
			postBody: `{"enable":false}`, verifyFail: true, wantError: true,
		},
		{
			name: "update rejects malformed response", operation: "update", getStatus: 200,
			getBody: `{"enable":true}`, postBody: `{`, wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gets, posts := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if req.Method == http.MethodGet {
					gets++
					if tt.verifyFail && gets > 1 {
						w.WriteHeader(http.StatusServiceUnavailable)
						return
					}
					w.WriteHeader(tt.getStatus)
					_, _ = w.Write([]byte(tt.getBody))
					return
				}
				posts++
				_, _ = w.Write([]byte(tt.postBody))
			}))
			defer server.Close()
			client, err := api.NewClientWithResponses(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			r := &mdnsResource{client: client}
			s := testSchema(t, r)
			model := mdnsModel{ID: types.StringValue("mdns"), Enable: types.BoolValue(true),
				IncludeInterfaces: types.ListNull(types.StringType), ExcludeInterfaces: types.ListNull(types.StringType),
				Publish: types.ObjectNull(publishTypes())}
			state := tfsdk.State{Schema: s}
			if diags := state.Set(context.Background(), model); diags.HasError() {
				t.Fatal(diags)
			}
			var gotError bool
			switch tt.operation {
			case "create":
				resp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
				r.Create(context.Background(), resource.CreateRequest{Plan: testPlan(t, s, model)}, &resp)
				gotError = resp.Diagnostics.HasError()
			case "read":
				resp := resource.ReadResponse{State: state}
				r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
				gotError = resp.Diagnostics.HasError()
				var refreshed mdnsModel
				if diags := resp.State.Get(context.Background(), &refreshed); diags.HasError() {
					t.Fatal(diags)
				}
				if refreshed.Enable.ValueBool() != tt.wantError {
					t.Fatalf("read returned unexpected state: %#v", refreshed)
				}
			case "update":
				resp := resource.UpdateResponse{State: state}
				r.Update(context.Background(), resource.UpdateRequest{Plan: testPlan(t, s, model), State: state}, &resp)
				gotError = resp.Diagnostics.HasError()
				if gotError && !resp.State.Raw.Equal(state.Raw) {
					t.Fatal("failed update did not preserve state")
				}
			case "delete":
				resp := resource.DeleteResponse{State: state}
				r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
				gotError = resp.Diagnostics.HasError()
			}
			if gotError != tt.wantError {
				t.Fatalf("error = %t, want %t", gotError, tt.wantError)
			}
			if tt.getStatus != http.StatusOK && posts != 0 {
				t.Fatal("attempted to write after a failed or missing read")
			}
		})
	}
}

func TestMDNSInvalidPublishingConversion(t *testing.T) {
	ctx := context.Background()
	invalid := types.ObjectValueMust(map[string]attr.Type{"enable": types.StringType},
		map[string]attr.Value{"enable": types.StringValue("not a boolean")})
	data := mdnsModel{Publish: invalid}
	config := &api.MDNSConfig{}
	if diags := data.merge(ctx, config); !diags.HasError() || config.Publish != nil {
		t.Fatalf("invalid publishing conversion was accepted: %#v, %v", config, diags)
	}
}

func TestMDNSWriteRejectsInvalidPublishingBeforePosting(t *testing.T) {
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			posts++
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"enable":true}`))
	}))
	defer server.Close()
	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	data := mdnsModel{Publish: types.ObjectValueMust(
		map[string]attr.Type{"enable": types.StringType},
		map[string]attr.Value{"enable": types.StringValue("invalid")},
	)}
	if _, err := (&mdnsResource{client: client}).write(context.Background(), &data); err == nil || posts != 0 {
		t.Fatalf("invalid publishing error = %v; POST count = %d", err, posts)
	}
}

func TestMDNSGetReportsTransportFailure(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	server.Close()
	if _, _, err := (&mdnsResource{client: client}).get(context.Background()); err == nil {
		t.Fatal("expected transport error")
	}
}
