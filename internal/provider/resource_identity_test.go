package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/provider/resource_firewall_alias"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type identityLifecycleResource struct {
	resource.Resource
	fail bool
}

func (r *identityLifecycleResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	if r.fail {
		resp.Diagnostics.AddError("Create failed", "The remote object was not created.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), "created")...)
}

func (r *identityLifecycleResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	if r.fail {
		resp.Diagnostics.AddError("Update failed", "The remote object was not updated.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), "updated")...)
}

func TestIdentityLifecycle(t *testing.T) {
	ctx := context.Background()
	stateSchema := schema.Schema{Attributes: map[string]schema.Attribute{"name": schema.StringAttribute{Required: true}}}
	identitySchema := identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"controller": identityschema.StringAttribute{RequiredForImport: true},
		"id":         identityschema.StringAttribute{RequiredForImport: true},
	}}
	for _, fail := range []bool{false, true} {
		r := &identifiedResource{
			Resource: &identityLifecycleResource{
				fail: fail,
			}, attributes: []string{"name"}, controller: "https://example.test",
		}
		state := func() tfsdk.State {
			return tfsdk.State{Schema: stateSchema, Raw: tftypes.NewValue(stateSchema.Type().TerraformType(ctx), nil)}
		}
		identity := func() *tfsdk.ResourceIdentity {
			return &tfsdk.ResourceIdentity{
				Schema: identitySchema,
				Raw:    tftypes.NewValue(identitySchema.Type().TerraformType(ctx), nil),
			}
		}
		created := resource.CreateResponse{State: state(), Identity: identity()}
		r.Create(ctx, resource.CreateRequest{}, &created)
		updated := resource.UpdateResponse{State: state(), Identity: identity()}
		r.Update(ctx, resource.UpdateRequest{}, &updated)
		for _, result := range []struct {
			id       string
			state    tfsdk.State
			identity *tfsdk.ResourceIdentity
			diags    diag.Diagnostics
		}{
			{id: "created", state: created.State, identity: created.Identity, diags: created.Diagnostics},
			{id: "updated", state: updated.State, identity: updated.Identity, diags: updated.Diagnostics},
		} {
			if fail {
				if !result.diags.HasError() || !result.identity.Raw.IsNull() {
					t.Fatalf("failed lifecycle changed identity: %#v", result)
				}
				continue
			}
			var value resourceIdentity
			if result.diags.HasError() {
				t.Fatal(result.diags)
			}
			if diags := result.identity.Get(ctx, &value); diags.HasError() {
				t.Fatal(diags)
			}
			var name types.String
			if diags := result.state.GetAttribute(ctx, path.Root("name"), &name); diags.HasError() {
				t.Fatal(diags)
			}
			if value.ID.ValueString() != result.id || name.ValueString() != result.id ||
				value.Controller.ValueString() != r.controller {
				t.Fatalf("identity does not match lifecycle state: %#v", value)
			}
		}
	}
}

func TestIdentityConfigurationFailures(t *testing.T) {
	for _, tt := range []struct {
		name string
		data any
		want string
	}{
		{name: "not configured"},
		{name: "unexpected data", data: "invalid", want: "unexpected API client"},
		{name: "nil client", data: (*api.ClientWithResponses)(nil), want: "unexpected API client"},
		{name: "missing API endpoint", data: &api.ClientWithResponses{}, want: "controller endpoint"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := identified(resource_firewall_alias.New, "name")().(*identifiedResource)
			var resp resource.ConfigureResponse
			r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: tt.data}, &resp)
			if tt.want == "" {
				if resp.Diagnostics.HasError() || r.controller != "" {
					t.Fatalf("unexpected configuration: %v", resp.Diagnostics)
				}
			} else if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), tt.want) {
				t.Fatalf("diagnostics = %v, want %q", resp.Diagnostics, tt.want)
			}
		})
	}
}

func TestIdentityRejectsInvalidState(t *testing.T) {
	ctx := context.Background()
	stateSchema := schema.Schema{Attributes: map[string]schema.Attribute{"name": schema.StringAttribute{Required: true}}}
	identitySchema := identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"controller": identityschema.StringAttribute{RequiredForImport: true},
		"id":         identityschema.StringAttribute{RequiredForImport: true},
	}}
	for _, tt := range []struct {
		name       string
		attribute  string
		value      any
		controller string
		want       string
	}{
		{
			name: "missing attribute", attribute: "absent", value: "example",
			controller: "https://example.test", want: "State Read Error",
		},
		{name: "null name", attribute: "name", controller: "https://example.test", want: "Missing resource identity"},
		{
			name: "unknown name", attribute: "name", value: tftypes.UnknownValue,
			controller: "https://example.test", want: "Missing resource identity",
		},
		{
			name: "empty name", attribute: "name", value: "",
			controller: "https://example.test", want: "Missing resource identity",
		},
		{name: "missing controller", attribute: "name", value: "example", want: "controller endpoint"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			state := tfsdk.State{
				Schema: stateSchema,
				Raw: tftypes.NewValue(stateSchema.Type().TerraformType(ctx), map[string]tftypes.Value{
					"name": tftypes.NewValue(tftypes.String, tt.value),
				}),
			}
			identity := &tfsdk.ResourceIdentity{
				Schema: identitySchema,
				Raw:    tftypes.NewValue(identitySchema.Type().TerraformType(ctx), nil),
			}
			r := &identifiedResource{attributes: []string{tt.attribute}, controller: tt.controller}
			var diags diag.Diagnostics
			r.setIdentity(ctx, state, identity, &diags)
			if !diags.HasError() || !strings.Contains(diags.Errors()[0].Summary()+diags.Errors()[0].Detail(), tt.want) ||
				!identity.Raw.IsNull() {
				t.Fatalf("invalid state did not preserve identity: %v", diags)
			}
		})
	}
}

func TestIdentityImportRejectsMalformedIdentity(t *testing.T) {
	ctx := context.Background()
	r := identified(resource_firewall_alias.New, "name")().(*identifiedResource)
	invalidSchema := identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"id": identityschema.StringAttribute{RequiredForImport: true},
	}}
	identity := &tfsdk.ResourceIdentity{
		Schema: invalidSchema,
		Raw: tftypes.NewValue(invalidSchema.Type().TerraformType(ctx), map[string]tftypes.Value{
			"id": tftypes.NewValue(tftypes.String, "example"),
		}),
	}
	var resp resource.ImportStateResponse
	r.ImportState(ctx, resource.ImportStateRequest{Identity: identity}, &resp)
	if !resp.Diagnostics.HasError() || resp.State.Raw.Type() != nil {
		t.Fatalf("malformed identity reached state import: %v", resp.Diagnostics)
	}
}
