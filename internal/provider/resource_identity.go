package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type identifiedResource struct {
	resource.Resource
	attributes []string
	controller string
}

type resourceIdentity struct {
	Controller types.String `tfsdk:"controller"`
	ID         types.String `tfsdk:"id"`
}

func identified(factory func() resource.Resource, attributes ...string) func() resource.Resource {
	return func() resource.Resource {
		return &identifiedResource{Resource: factory(), attributes: attributes}
	}
}

func (r *identifiedResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*api.ClientWithResponses)
	if !ok || client == nil {
		resp.Diagnostics.AddError("Invalid provider configuration", "The resource received an unexpected API client.")
		return
	}
	base, ok := client.ClientInterface.(*api.Client)
	if !ok || base == nil {
		resp.Diagnostics.AddError("Invalid provider configuration", "The resource could not determine the controller endpoint.")
		return
	}
	r.controller = strings.TrimSuffix(base.Server, "/")
	r.Resource.(resource.ResourceWithConfigure).Configure(ctx, req, resp)
}

func (r *identifiedResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"controller": identityschema.StringAttribute{RequiredForImport: true},
		"id":         identityschema.StringAttribute{RequiredForImport: true},
	}}
}

func (r *identifiedResource) setIdentity(ctx context.Context, state tfsdk.State, identity *tfsdk.ResourceIdentity, diags *diag.Diagnostics) {
	if identity == nil || diags.HasError() || state.Raw.IsNull() {
		return
	}
	parts := make([]string, 0, len(r.attributes))
	for _, name := range r.attributes {
		var value types.String
		diags.Append(state.GetAttribute(ctx, path.Root(name), &value)...)
		if diags.HasError() {
			return
		}
		if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
			diags.AddError("Missing resource identity", fmt.Sprintf("The resource state does not contain %q.", name))
			return
		}
		parts = append(parts, value.ValueString())
	}
	if r.controller == "" {
		diags.AddError("Missing resource identity", "The controller endpoint is not configured.")
		return
	}
	diags.Append(identity.Set(ctx, resourceIdentity{
		Controller: types.StringValue(r.controller),
		ID:         types.StringValue(strings.Join(parts, "/")),
	})...)
}

func (r *identifiedResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.Resource.Create(ctx, req, resp)
	r.setIdentity(ctx, resp.State, resp.Identity, &resp.Diagnostics)
}

func (r *identifiedResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.Resource.Read(ctx, req, resp)
	r.setIdentity(ctx, resp.State, resp.Identity, &resp.Diagnostics)
}

func (r *identifiedResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.Resource.Update(ctx, req, resp)
	r.setIdentity(ctx, resp.State, resp.Identity, &resp.Diagnostics)
}

func (r *identifiedResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" && req.Identity != nil {
		var identity resourceIdentity
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identity.Controller.ValueString() != r.controller {
			resp.Diagnostics.AddError("Invalid resource identity", "The identity controller does not match the configured controller.")
			return
		}
		req.ID = identity.ID.ValueString()
	}
	r.Resource.(resource.ResourceWithImportState).ImportState(ctx, req, resp)
	r.setIdentity(ctx, resp.State, resp.Identity, &resp.Diagnostics)
}
