package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/provider/firewall_resources"
	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/provider/resource_firewall_alias"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestListRejectsIncompleteResponses(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    any
		wantErr string
	}{
		{name: "HTTP failure", status: http.StatusForbidden, body: map[string]any{}, wantErr: "HTTP 403"},
		{name: "missing list", status: http.StatusOK, body: map[string]any{}, wantErr: "invalid"},
		{
			name:    "missing identifier",
			status:  http.StatusOK,
			body:    map[string]any{"rules": []any{map[string]any{}}},
			wantErr: "identifier",
		},
		{
			name:    "duplicate identifiers",
			status:  http.StatusOK,
			body:    map[string]any{"rules": []any{map[string]any{"id": "same"}, map[string]any{"id": "same"}}},
			wantErr: "duplicate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("unexpected method %s", r.Method)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				if err := json.NewEncoder(w).Encode(tt.body); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			client, err := api.NewClientWithResponses(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			l := configurationList{client: client}
			_, err = l.identifiers(context.Background(), "firewall_nat_npt")
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestRuleDiscoveryAcrossInterfaces(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var body any
		switch r.URL.Path {
		case "/firewall/rules/interface":
			body = map[string]any{"interfaces": []string{"lan", "wan"}}
		case "/firewall/rules/interface/lan":
			body = map[string]any{"entries": []any{
				map[string]any{"rule": map[string]any{"id": "first"}},
				map[string]any{"separator": map[string]any{"text": "section"}},
				map[string]any{"rule": map[string]any{"readonly": true}},
			}}
		case "/firewall/rules/interface/wan":
			body = map[string]any{"entries": []any{map[string]any{"rule": map[string]any{"id": "second"}}}}
		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
			return
		}
		if err := json.NewEncoder(w).Encode(body); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	l := configurationList{client: client}
	ids, skipped, err := l.ruleIdentifiers(context.Background())
	if err != nil || skipped != 1 || !reflect.DeepEqual(ids, []string{"lan/first", "wan/second"}) {
		t.Fatalf("identifiers = %v, skipped = %d, error = %v", ids, skipped, err)
	}
}

func TestListRejectsInconsistentRuleIdentity(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var body any
		switch r.URL.Path {
		case "/firewall/rules/interface":
			body = map[string]any{"interfaces": []string{"lan"}}
		case "/firewall/rules/interface/lan":
			body = map[string]any{
				"entries": []any{map[string]any{"rule": map[string]any{"id": "rule-1", "interface": "wan"}}},
			}
		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
			return
		}
		if err := json.NewEncoder(w).Encode(body); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	l := &configurationList{managed: identified(firewall_resources.NewRule, "interface", "id")().(*identifiedResource)}
	var configured resource.ConfigureResponse
	l.Configure(ctx, resource.ConfigureRequest{ProviderData: client}, &configured)
	if configured.Diagnostics.HasError() {
		t.Fatal(configured.Diagnostics)
	}
	var schemaResp resource.SchemaResponse
	l.managed.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	var identityResp resource.IdentitySchemaResponse
	l.managed.IdentitySchema(ctx, resource.IdentitySchemaRequest{}, &identityResp)
	req := list.ListRequest{
		IncludeResource:        true,
		ResourceSchema:         schemaResp.Schema,
		ResourceIdentitySchema: identityResp.IdentitySchema,
	}
	var stream list.ListResultsStream
	l.List(ctx, req, &stream)
	var found bool
	for result := range stream.Results {
		for _, diagnostic := range result.Diagnostics.Errors() {
			if diagnostic.Summary() == "Inconsistent resource identity" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("listing accepted a rule with a mismatched interface")
	}
}

func TestImportIdentityAndLegacyID(t *testing.T) {
	ctx := context.Background()
	client, err := api.NewClientWithResponses("https://example.test")
	if err != nil {
		t.Fatal(err)
	}
	r := identified(resource_firewall_alias.New, "name")().(*identifiedResource)
	var configured resource.ConfigureResponse
	r.Configure(ctx, resource.ConfigureRequest{ProviderData: client}, &configured)
	if configured.Diagnostics.HasError() {
		t.Fatal(configured.Diagnostics)
	}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	var identityResp resource.IdentitySchemaResponse
	r.IdentitySchema(ctx, resource.IdentitySchemaRequest{}, &identityResp)
	makeIdentity := func(controller string) *tfsdk.ResourceIdentity {
		identity := &tfsdk.ResourceIdentity{Schema: identityResp.IdentitySchema}
		diags := identity.Set(ctx, resourceIdentity{
			Controller: types.StringValue(controller), ID: types.StringValue("example"),
		})
		if diags.HasError() {
			t.Fatal(diags)
		}
		return identity
	}
	newState := func() tfsdk.State {
		return tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
		}
	}
	for _, tt := range []struct {
		name     string
		id       string
		identity *tfsdk.ResourceIdentity
		wantErr  bool
	}{
		{name: "legacy ID", id: "example"},
		{name: "identity", identity: makeIdentity(r.controller)},
		{name: "different controller", identity: makeIdentity("https://other.example.test"), wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resp := resource.ImportStateResponse{State: newState()}
			r.ImportState(ctx, resource.ImportStateRequest{ID: tt.id, Identity: tt.identity}, &resp)
			if resp.Diagnostics.HasError() != tt.wantErr {
				t.Fatalf("diagnostics: %v", resp.Diagnostics)
			}
			if !tt.wantErr {
				var value resource_firewall_alias.FirewallAliasModel
				if diags := resp.State.Get(ctx, &value); diags.HasError() || value.Name.ValueString() != "example" {
					t.Fatalf("imported state = %#v, diagnostics = %v", value, diags)
				}
			}
		})
	}
}
