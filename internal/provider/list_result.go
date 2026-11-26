package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func (l *configurationList) listResult(ctx context.Context, req list.ListRequest, id string) list.ListResult {
	result := req.NewListResult(ctx)
	result.DisplayName = id
	result.Diagnostics.Append(result.Identity.Set(ctx, resourceIdentity{
		Controller: types.StringValue(l.managed.controller),
		ID:         types.StringValue(id),
	})...)
	if req.IncludeResource && !result.Diagnostics.HasError() {
		l.readListedResource(ctx, req, id, &result)
	}
	return result
}

func (l *configurationList) readListedResource(
	ctx context.Context, req list.ListRequest, id string, result *list.ListResult,
) {
	imported := resource.ImportStateResponse{State: tfsdk.State{
		Schema: req.ResourceSchema,
		Raw:    tftypes.NewValue(req.ResourceSchema.Type().TerraformType(ctx), nil),
	}}
	l.managed.ImportState(ctx, resource.ImportStateRequest{ID: id}, &imported)
	result.Diagnostics.Append(imported.Diagnostics...)
	if !result.Diagnostics.HasError() {
		read := resource.ReadResponse{State: imported.State}
		l.managed.Read(ctx, resource.ReadRequest{State: imported.State}, &read)
		result.Diagnostics.Append(read.Diagnostics...)
		if !result.Diagnostics.HasError() {
			if read.State.Raw.IsNull() {
				result.Diagnostics.AddError("Unable to read discovered resource", "The resource disappeared during discovery.")
			} else {
				l.setListedResource(ctx, req, id, read.State, result)
			}
		}
	}
}

func (l *configurationList) setListedResource(
	ctx context.Context, req list.ListRequest, id string, state tfsdk.State, result *list.ListResult,
) {
	identity := &tfsdk.ResourceIdentity{
		Schema: req.ResourceIdentitySchema,
		Raw:    tftypes.NewValue(req.ResourceIdentitySchema.Type().TerraformType(ctx), nil),
	}
	l.managed.setIdentity(ctx, state, identity, &result.Diagnostics)
	var actual resourceIdentity
	if !result.Diagnostics.HasError() {
		result.Diagnostics.Append(identity.Get(ctx, &actual)...)
	}
	if !result.Diagnostics.HasError() && actual.ID.ValueString() != id {
		result.Diagnostics.AddError(
			"Inconsistent resource identity",
			"The resource identity returned by the API does not match the discovered identity.",
		)
	}
	if !result.Diagnostics.HasError() {
		result.Resource.Raw = state.Raw
	}
}
