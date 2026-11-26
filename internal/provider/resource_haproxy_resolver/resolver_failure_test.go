package resource_haproxy_resolver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestResolverConfigurationAndImport(t *testing.T) {
	ctx := context.Background()
	r := New().(*resolverResource)
	var metadata resource.MetadataResponse
	r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "pfsense"}, &metadata)
	if metadata.TypeName != "pfsense_haproxy_resolver" {
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
	s := resolverSchema(t, r)
	state := tfsdk.State{Schema: s}
	if diags := state.Set(ctx, &resolverModel{}); diags.HasError() {
		t.Fatal(diags)
	}
	response := resource.ImportStateResponse{State: state}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "resolver-1"}, &response)
	var id types.String
	response.Diagnostics.Append(response.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if response.Diagnostics.HasError() || id.ValueString() != "resolver-1" {
		t.Fatalf("import ID %v, diagnostics %v", id, response.Diagnostics)
	}
}

func TestResolverInvalidTerraformInputs(t *testing.T) {
	ctx := context.Background()
	r := New().(*resolverResource)
	s := resolverSchema(t, r)
	invalid := tftypes.NewValue(tftypes.String, "invalid")
	plan := tfsdk.Plan{Schema: s, Raw: invalid}
	state := tfsdk.State{Schema: s, Raw: invalid}
	create := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &create)
	read := resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, &read)
	update := resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, &update)
	deleted := resource.DeleteResponse{State: state}
	r.Delete(ctx, resource.DeleteRequest{State: state}, &deleted)
	for operation, diagnostics := range map[string]bool{
		"create": create.Diagnostics.HasError(), "read": read.Diagnostics.HasError(),
		"update": update.Diagnostics.HasError(), "delete": deleted.Diagnostics.HasError(),
	} {
		if !diagnostics {
			t.Errorf("%s accepted an invalid Terraform value", operation)
		}
	}
}

func TestResolverVerificationFailuresPreserveState(t *testing.T) {
	tests := []struct {
		name       string
		operation  string
		listStatus int
		listBody   string
		wantAbsent bool
		wantError  bool
	}{
		{name: "create missing", operation: "create", listStatus: 200, listBody: `{"resolvers":[]}`, wantError: true},
		{name: "create verification failure", operation: "create", listStatus: 503, wantError: true},
		{name: "update missing", operation: "update", listStatus: 200, listBody: `{"resolvers":[]}`, wantError: true},
		{name: "update verification failure", operation: "update", listStatus: 503, wantError: true},
		{name: "read failure", operation: "read", listStatus: 503, wantError: true},
		{name: "read HTTP 404", operation: "read", listStatus: 404, wantAbsent: true},
		{
			name: "delete still present", operation: "delete", listStatus: 200,
			listBody: `{"resolvers":[{"id":"x","name":"example"}]}`, wantError: true,
		},
		{name: "delete verification failure", operation: "delete", listStatus: 503, wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applies := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case req.Method == http.MethodGet:
					w.WriteHeader(tt.listStatus)
					_, _ = w.Write([]byte(tt.listBody))
				case req.URL.Path == "/ports/haproxy/apply":
					applies++
					_, _ = w.Write([]byte(`{"result":"success"}`))
				case req.Method == http.MethodDelete:
					_, _ = w.Write([]byte(`{"result":"success"}`))
				default:
					_, _ = w.Write([]byte(`{"id":"x","name":"example"}`))
				}
			}))
			defer server.Close()
			client, err := api.NewClientWithResponses(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			r := &resolverResource{client: client}
			s := resolverSchema(t, r)
			model := resolverModel{ID: types.StringValue("x"), Name: types.StringValue("example"),
				Server: types.StringNull(), Port: types.StringNull()}
			state := tfsdk.State{Schema: s}
			if diags := state.Set(context.Background(), model); diags.HasError() {
				t.Fatal(diags)
			}
			var gotError bool
			switch tt.operation {
			case "create":
				model.ID = types.StringUnknown()
				model.Server, model.Port = types.StringUnknown(), types.StringUnknown()
				resp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
				r.Create(context.Background(), resource.CreateRequest{Plan: resolverPlan(t, s, model)}, &resp)
				gotError = resp.Diagnostics.HasError()
				var retained resolverModel
				if diags := resp.State.Get(context.Background(), &retained); diags.HasError() {
					t.Fatal(diags)
				}
				if retained.ID.ValueString() != "x" || !retained.Server.IsNull() || !retained.Port.IsNull() {
					t.Fatalf("created ID was not retained: %#v", retained)
				}
			case "update":
				resp := resource.UpdateResponse{State: state}
				r.Update(context.Background(), resource.UpdateRequest{Plan: resolverPlan(t, s, model), State: state}, &resp)
				gotError = resp.Diagnostics.HasError()
				if gotError && !resp.State.Raw.Equal(state.Raw) {
					t.Fatal("failed update did not preserve state")
				}
			case "read":
				resp := resource.ReadResponse{State: state}
				r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
				gotError = resp.Diagnostics.HasError()
				if resp.State.Raw.IsNull() != tt.wantAbsent {
					t.Fatalf("state absent = %t, want %t", resp.State.Raw.IsNull(), tt.wantAbsent)
				}
			case "delete":
				resp := resource.DeleteResponse{State: state}
				r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
				gotError = resp.Diagnostics.HasError()
			}
			if gotError != tt.wantError {
				t.Fatalf("error = %t, want %t", gotError, tt.wantError)
			}
			if tt.operation != "read" && applies != 1 {
				t.Fatalf("apply count = %d, want 1", applies)
			}
		})
	}
}

func TestResolverTransportFailures(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	server.Close()
	r := &resolverResource{client: client}
	ctx := context.Background()
	s := resolverSchema(t, r)
	model := resolverModel{ID: types.StringValue("x"), Name: types.StringValue("example")}
	state := tfsdk.State{Schema: s}
	if diags := state.Set(ctx, model); diags.HasError() {
		t.Fatal(diags)
	}
	create := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r.Create(ctx, resource.CreateRequest{Plan: resolverPlan(t, s, model)}, &create)
	update := resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: resolverPlan(t, s, model), State: state}, &update)
	deleted := resource.DeleteResponse{State: state}
	r.Delete(ctx, resource.DeleteRequest{State: state}, &deleted)
	read := resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, &read)
	if !create.Diagnostics.HasError() || !update.Diagnostics.HasError() ||
		!deleted.Diagnostics.HasError() || !read.Diagnostics.HasError() {
		t.Fatal("transport failure was not reported by every lifecycle operation")
	}
	if err := r.apply(ctx); err == nil {
		t.Fatal("apply transport failure was not reported")
	}
}

func TestResolverCreateStopsWhenStateCannotBePersisted(t *testing.T) {
	applies := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if req.URL.Path == "/ports/haproxy/apply" {
			applies++
		}
		_, _ = w.Write([]byte(`{"id":"x","name":"example"}`))
	}))
	defer server.Close()
	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	r := &resolverResource{client: client}
	validSchema := resolverSchema(t, r)
	invalidStateSchema := resolverSchema(t, r)
	invalidStateSchema.Attributes["id"] = schema.BoolAttribute{Computed: true}
	model := resolverModel{ID: types.StringUnknown(), Name: types.StringValue("example"),
		Server: types.StringNull(), Port: types.StringNull()}
	resp := resource.CreateResponse{State: tfsdk.State{Schema: invalidStateSchema}}
	r.Create(context.Background(), resource.CreateRequest{Plan: resolverPlan(t, validSchema, model)}, &resp)
	if !resp.Diagnostics.HasError() || applies != 0 {
		t.Fatalf("state conversion diagnostics %v, apply count %d", resp.Diagnostics, applies)
	}
}
