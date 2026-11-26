package resource_firewall_alias

import (
	"context"
	"fmt"
	"net/http"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func (m *FirewallAliasModel) setTargets(
	ctx context.Context,
	values *[]api.FWTarget,
	newTarget func(map[string]attr.Type, map[string]attr.Value) (TargetsValue, diag.Diagnostics),
	newList func(attr.Type, []attr.Value) (basetypes.ListValue, diag.Diagnostics),
) error {
	targetType := TargetsType{ObjectType: types.ObjectType{AttrTypes: TargetsValue{}.AttributeTypes(ctx)}}
	m.Targets = types.ListNull(targetType)
	if values != nil {
		targets := make([]attr.Value, 0, len(*values))
		for _, value := range *values {
			target, diags := newTarget(TargetsValue{}.AttributeTypes(ctx), map[string]attr.Value{
				"name": types.StringValue(value.Name), "descr": stringValue(value.Descr),
				"updatefreq": stringValue(value.Updatefreq),
			})
			if diags.HasError() {
				return fmt.Errorf("invalid target returned by API: %s", diags.Errors()[0].Detail())
			}
			targets = append(targets, target)
		}
		list, diags := newList(targetType, targets)
		if diags.HasError() {
			return fmt.Errorf("invalid target list returned by API: %s", diags.Errors()[0].Detail())
		}
		m.Targets = list
	}
	return nil
}

func updateRequest(body api.FWAliasReq) api.FWUpdateAliasReq {
	update := api.FWUpdateAliasReq{
		Name: body.Name, Address: body.Address, Descr: body.Descr, Detail: body.Detail,
		Tags: body.Tags, Targets: body.Targets, Updatefreq: body.Updatefreq,
	}
	if body.Type != nil {
		kind := api.FWUpdateAliasReqType(*body.Type)
		update.Type = &kind
	}
	return update
}

func (r *aliasResource) finishWrite(
	ctx context.Context, name string, data *FirewallAliasModel, state *tfsdk.State, diags *diag.Diagnostics, action string,
) {
	if err := r.reload(ctx); err != nil {
		diags.AddError("Alias "+action+" but firewall reload failed", err.Error())
		return
	}
	value, missing, err := r.get(ctx, name)
	if err != nil || missing {
		diags.AddError(
			"Unable to verify "+action+" alias",
			fmt.Sprintf("The API did not return the %s alias: %v", action, err),
		)
		return
	}
	if err := data.fromAPI(ctx, value); err != nil {
		diags.AddError("Unable to read "+action+" alias", err.Error())
		return
	}
	diags.Append(state.Set(ctx, data)...)
}

func (r *aliasResource) validateDelete(
	ctx context.Context, name string, result *api.FirewallDeleteAliasAPIResponse,
) error {
	if result.StatusCode() == http.StatusBadRequest {
		_, missing, err := r.get(ctx, name)
		if err != nil || !missing {
			return fmt.Errorf("API returned HTTP 400 and absence could not be verified: %v", err)
		}
	} else if result.StatusCode() != http.StatusOK && result.StatusCode() != http.StatusNotFound {
		return fmt.Errorf("API returned HTTP %d.", result.StatusCode())
	}
	if result.StatusCode() == http.StatusOK && (result.JSON200 == nil || result.JSON200.Msg == "") {
		return fmt.Errorf("The API returned an invalid result.")
	}
	return nil
}
