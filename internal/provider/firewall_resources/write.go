package firewall_resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

func (r *resourceModel) finishWrite(
	ctx context.Context,
	state *tfsdk.State,
	diagnostics *diag.Diagnostics,
	selector, plan map[string]any,
	match bool,
	action string,
) {
	item, err := r.readItem(ctx, selector, match)
	if err != nil {
		diagnostics.AddError("Unable to verify "+action+" firewall configuration", err.Error())
		return
	}
	if diags := r.setState(ctx, state, item, plan); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}
	if err := r.reload(ctx); err != nil {
		diagnostics.AddError("Firewall configuration "+action+" but reload failed", err.Error())
		return
	}
	item, err = r.readItem(ctx, item, false)
	if err != nil {
		diagnostics.AddError("Unable to read "+action+" firewall configuration", err.Error())
		return
	}
	diagnostics.Append(r.setState(ctx, state, item, plan)...)
}
