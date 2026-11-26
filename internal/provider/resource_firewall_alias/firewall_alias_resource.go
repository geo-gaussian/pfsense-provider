package resource_firewall_alias

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ resource.Resource = &aliasResource{}
var _ resource.ResourceWithConfigure = &aliasResource{}
var _ resource.ResourceWithImportState = &aliasResource{}

type aliasResource struct {
	client *api.ClientWithResponses
}

var aliasName = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

func New() resource.Resource { return &aliasResource{} }

func (r *aliasResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_alias"
}

func (r *aliasResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = FirewallAliasResourceSchema(ctx)
	name := resp.Schema.Attributes["name"].(schema.StringAttribute)
	name.PlanModifiers = []planmodifier.String{stringplanmodifier.RequiresReplace()}
	name.Validators = append(
		name.Validators,
		stringvalidator.RegexMatches(aliasName, "must contain only letters, digits, and underscores"),
	)
	resp.Schema.Attributes["name"] = name
}

func (r *aliasResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	r.client, ok = req.ProviderData.(*api.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid provider configuration",
			"The firewall alias resource received an unexpected client.",
		)
	}
}

func stringPointer(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := value.ValueString()
	return &v
}

func (m *FirewallAliasModel) request(ctx context.Context) (api.FWAliasReq, error) {
	if !aliasName.MatchString(m.Name.ValueString()) {
		return api.FWAliasReq{}, fmt.Errorf("invalid alias name")
	}
	result := api.FWAliasReq{
		Name: m.Name.ValueString(), Address: stringPointer(m.Address),
		Descr: stringPointer(m.Descr), Detail: stringPointer(m.Detail),
		Tags: stringPointer(m.Tags), Updatefreq: stringPointer(m.Updatefreq),
	}
	if v := stringPointer(m.Type); v != nil {
		t := api.FWAliasReqType(*v)
		result.Type = &t
	}
	if !m.Targets.IsNull() && !m.Targets.IsUnknown() {
		var values []TargetsValue
		diags := m.Targets.ElementsAs(ctx, &values, false)
		if diags.HasError() {
			return result, fmt.Errorf("invalid alias targets: %s", diags.Errors()[0].Detail())
		}
		targets := make([]api.FWTarget, 0, len(values))
		for _, v := range values {
			targets = append(
				targets,
				api.FWTarget{
					Name:       v.Name.ValueString(),
					Descr:      stringPointer(v.Descr),
					Updatefreq: stringPointer(v.Updatefreq),
				},
			)
		}
		result.Targets = &targets
	}
	return result, nil
}

func stringValue(v *string) types.String {
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(*v)
}

func (m *FirewallAliasModel) recoveryState(ctx context.Context) FirewallAliasModel {
	targetType := TargetsType{ObjectType: types.ObjectType{AttrTypes: TargetsValue{}.AttributeTypes(ctx)}}
	return FirewallAliasModel{
		Name: m.Name, Address: types.StringNull(), Descr: types.StringNull(),
		Detail: types.StringNull(), Tags: types.StringNull(), Type: types.StringNull(),
		Updatefreq: types.StringNull(), Targets: types.ListNull(targetType),
	}
}

func (m *FirewallAliasModel) fromAPI(ctx context.Context, value *api.FWAlias) error {
	return m.fromAPIWithFactories(ctx, value, NewTargetsValue, types.ListValue)
}

func (m *FirewallAliasModel) fromAPIWithFactories(
	ctx context.Context,
	value *api.FWAlias,
	newTarget func(map[string]attr.Type, map[string]attr.Value) (TargetsValue, diag.Diagnostics),
	newList func(attr.Type, []attr.Value) (basetypes.ListValue, diag.Diagnostics),
) error {
	if value.Truncated != nil && *value.Truncated {
		return fmt.Errorf("API returned a truncated alias; refusing to store incomplete state")
	}
	if value.Name == "" {
		return fmt.Errorf("API returned an alias without a name")
	}
	m.Name = types.StringValue(value.Name)
	m.Address = stringValue(value.Address)
	m.Descr = stringValue(value.Descr)
	m.Detail = stringValue(value.Detail)
	m.Tags = stringValue(value.Tags)
	m.Updatefreq = stringValue(value.Updatefreq)
	m.Type = types.StringNull()
	if value.Type != nil {
		m.Type = types.StringValue(string(*value.Type))
	}
	return m.setTargets(ctx, value.Targets, newTarget, newList)
}

func (r *aliasResource) get(ctx context.Context, name string) (*api.FWAlias, bool, error) {
	if !aliasName.MatchString(name) {
		return nil, false, fmt.Errorf("invalid alias name")
	}
	response, err := r.client.FirewallGetAliasWithResponse(ctx, name)
	if err != nil {
		return nil, false, fmt.Errorf("read alias: %w", err)
	}
	switch response.StatusCode() {
	case http.StatusNotFound:
		return nil, true, nil
	case http.StatusOK:
		if response.JSON200 == nil {
			return nil, false, fmt.Errorf("read alias: missing JSON response")
		}
		if response.JSON200.Name != name {
			return nil, false, fmt.Errorf("read alias: API returned a different alias name")
		}
		return response.JSON200, false, nil
	case http.StatusBadRequest:
		// Some Nexus versions return 400 when an alias is absent.
		list, err := r.client.FirewallGetAliasesWithResponse(ctx, nil)
		if err != nil {
			return nil, false, fmt.Errorf("verify missing alias: %w", err)
		}
		if list.StatusCode() != http.StatusOK || list.JSON200 == nil || list.JSON200.Aliases == nil {
			return nil, false, fmt.Errorf("verify missing alias: HTTP %d or missing alias list", list.StatusCode())
		}
		for _, alias := range *list.JSON200.Aliases {
			if strings.EqualFold(alias.Name, name) {
				return nil, false, fmt.Errorf("read alias: HTTP 400, but alias still exists in the list")
			}
		}
		return nil, true, nil
	default:
		return nil, false, fmt.Errorf("read alias: HTTP %d", response.StatusCode())
	}
}

func (r *aliasResource) reload(ctx context.Context) error {
	result, err := r.client.DoFilterReloadWithResponse(ctx)
	if err != nil {
		return fmt.Errorf("initiate firewall reload: %w", err)
	}
	if result.StatusCode() != http.StatusOK || result.JSON200 == nil {
		return fmt.Errorf("initiate firewall reload: HTTP %d or missing JSON response", result.StatusCode())
	}
	return nil
}

func (r *aliasResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FirewallAliasModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, err := data.request(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Invalid alias configuration", err.Error())
		return
	}
	result, err := r.client.FirewallAddAliasWithResponse(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create alias", err.Error())
		return
	}
	if result.StatusCode() != http.StatusOK || result.JSON200 == nil || result.JSON200.Msg == "" {
		resp.Diagnostics.AddError(
			"Unable to create alias",
			fmt.Sprintf("API returned HTTP %d or an invalid result.", result.StatusCode()),
		)
		return
	}
	recovery := data.recoveryState(ctx)
	resp.Diagnostics.Append(resp.State.Set(ctx, &recovery)...)
	r.finishWrite(ctx, body.Name, &data, &resp.State, &resp.Diagnostics, "created")
}

func (r *aliasResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FirewallAliasModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	value, missing, err := r.get(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read alias", err.Error())
		return
	}
	if missing {
		resp.State.RemoveResource(ctx)
		return
	}
	if err := data.fromAPI(ctx, value); err != nil {
		resp.Diagnostics.AddError("Unable to read alias", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *aliasResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FirewallAliasModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, err := data.request(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Invalid alias configuration", err.Error())
		return
	}
	update := updateRequest(body)
	result, err := r.client.FirewallUpdateAliasWithResponse(ctx, body.Name, update)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update alias", err.Error())
		return
	}
	if result.StatusCode() != http.StatusOK || result.JSON200 == nil || result.JSON200.Msg == "" {
		resp.Diagnostics.AddError(
			"Unable to update alias",
			fmt.Sprintf("API returned HTTP %d or an invalid result.", result.StatusCode()),
		)
		return
	}
	resp.State = req.State
	r.finishWrite(ctx, body.Name, &data, &resp.State, &resp.Diagnostics, "updated")
}

func (r *aliasResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FirewallAliasModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !aliasName.MatchString(data.Name.ValueString()) {
		resp.Diagnostics.AddError("Unable to delete alias", "The alias name in state is invalid.")
		return
	}
	result, err := r.client.FirewallDeleteAliasWithResponse(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete alias", err.Error())
		return
	}
	if err := r.validateDelete(ctx, data.Name.ValueString(), result); err != nil {
		resp.Diagnostics.AddError("Unable to delete alias", err.Error())
		return
	}
	if err := r.reload(ctx); err != nil {
		resp.Diagnostics.AddError("Alias deleted but firewall reload failed", err.Error())
		return
	}
	_, missing, err := r.get(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to verify deleted alias", err.Error())
		return
	}
	if !missing {
		resp.Diagnostics.AddError("Unable to verify deleted alias", "The alias is still present after the delete request.")
	}
}

func (r *aliasResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
