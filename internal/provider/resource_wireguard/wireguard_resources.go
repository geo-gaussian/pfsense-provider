package resource_wireguard

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ resource.Resource = &settingsResource{}
var _ resource.ResourceWithConfigure = &settingsResource{}
var _ resource.ResourceWithImportState = &settingsResource{}
var _ resource.Resource = &tunnelResource{}
var _ resource.ResourceWithConfigure = &tunnelResource{}
var _ resource.ResourceWithImportState = &tunnelResource{}
var _ resource.Resource = &peerResource{}
var _ resource.ResourceWithConfigure = &peerResource{}
var _ resource.ResourceWithImportState = &peerResource{}

const settingsID = "wireguard"

type resourceBase struct {
	client *api.ClientWithResponses
}

func (r *resourceBase) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	r.client, ok = req.ProviderData.(*api.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid provider configuration",
			"The WireGuard resource received an unexpected API client.",
		)
	}
}

func NewSettings() resource.Resource {
	return &settingsResource{}
}

func NewTunnel() resource.Resource {
	return &tunnelResource{}
}

func NewPeer() resource.Resource {
	return &peerResource{}
}

type settingsResource struct{ resourceBase }

type settingsModel struct {
	ID                   types.String `tfsdk:"id"`
	Enable               types.Bool   `tfsdk:"enable"`
	KeepConf             types.Bool   `tfsdk:"keep_conf"`
	ResolveInterval      types.String `tfsdk:"resolve_interval"`
	ResolveIntervalTrack types.Bool   `tfsdk:"resolve_interval_track"`
	InterfaceGroup       types.String `tfsdk:"interface_group"`
	HideSecrets          types.Bool   `tfsdk:"hide_secrets"`
	HidePeers            types.Bool   `tfsdk:"hide_peers"`
}

func (r *settingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wireguard_settings"
}

func (r *settingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the global WireGuard settings. Removing this resource from configuration only " +
			"relinquishes Terraform management; the API has no operation to delete global settings.",
		Attributes: map[string]schema.Attribute{
			"id":                     schema.StringAttribute{Computed: true},
			"enable":                 schema.BoolAttribute{Optional: true, Computed: true},
			"keep_conf":              schema.BoolAttribute{Optional: true, Computed: true},
			"resolve_interval":       schema.StringAttribute{Optional: true, Computed: true},
			"resolve_interval_track": schema.BoolAttribute{Optional: true, Computed: true},
			"interface_group": schema.StringAttribute{
				Optional: true, Computed: true,
				Validators: []validator.String{stringvalidator.OneOf("all", "unassigned", "none")},
			},
			"hide_secrets": schema.BoolAttribute{Optional: true, Computed: true},
			"hide_peers":   schema.BoolAttribute{Optional: true, Computed: true},
		},
	}
}

func (r *settingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data settingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	config := data.toAPI()
	result, err := r.client.SetWireGuardSettingsWithResponse(ctx, config)
	if err != nil {
		resp.Diagnostics.AddError("Unable to set WireGuard settings", err.Error())
		return
	}
	if err := validateWriteResult(result.StatusCode(), result.JSON200, "set WireGuard settings"); err != nil {
		resp.Diagnostics.AddError("Unable to set WireGuard settings", err.Error())
		return
	}
	data.ID = types.StringValue(settingsID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readSettingsIntoState(ctx, &data, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}
}

func (r *settingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data settingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	config, missing, err := r.getSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read WireGuard settings", err.Error())
		return
	}
	if missing {
		resp.State.RemoveResource(ctx)
		return
	}
	data.fromAPI(config)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *settingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data settingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.client.SetWireGuardSettingsWithResponse(ctx, data.toAPI())
	if err != nil {
		resp.Diagnostics.AddError("Unable to update WireGuard settings", err.Error())
		return
	}
	if err := validateWriteResult(result.StatusCode(), result.JSON200, "update WireGuard settings"); err != nil {
		resp.Diagnostics.AddError("Unable to update WireGuard settings", err.Error())
		return
	}
	resp.State = req.State
	r.readSettingsIntoState(ctx, &data, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}
}

func (r *settingsResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
	return
}

func (r *settingsResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (m *settingsModel) toAPI() api.WGConfig {
	return api.WGConfig{
		Enable:               boolPointer(m.Enable),
		KeepConf:             boolPointer(m.KeepConf),
		ResolveInterval:      stringPointer(m.ResolveInterval),
		ResolveIntervalTrack: boolPointer(m.ResolveIntervalTrack),
		InterfaceGroup:       stringPointer(m.InterfaceGroup),
		HideSecrets:          boolPointer(m.HideSecrets),
		HidePeers:            boolPointer(m.HidePeers),
	}
}

func (m *settingsModel) fromAPI(value *api.WGConfig) {
	m.ID = types.StringValue(settingsID)
	m.Enable = boolValue(value.Enable)
	m.KeepConf = boolValue(value.KeepConf)
	m.ResolveInterval = stringValue(value.ResolveInterval)
	m.ResolveIntervalTrack = boolValue(value.ResolveIntervalTrack)
	m.InterfaceGroup = stringValue(value.InterfaceGroup)
	m.HideSecrets = boolValue(value.HideSecrets)
	m.HidePeers = boolValue(value.HidePeers)
}

func (r *settingsResource) getSettings(ctx context.Context) (*api.WGConfig, bool, error) {
	result, err := r.client.GetWireGuardWithResponse(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("get WireGuard configuration: %w", err)
	}
	if result.StatusCode() == http.StatusNotFound {
		return nil, true, nil
	}
	if result.StatusCode() != http.StatusOK || result.JSON200 == nil {
		return nil, false, fmt.Errorf("get WireGuard configuration: HTTP %d or missing JSON response", result.StatusCode())
	}
	if result.JSON200.Config == nil || len(*result.JSON200.Config) == 0 {
		return nil, false, fmt.Errorf("get WireGuard configuration: API omitted global settings")
	}
	if len(*result.JSON200.Config) != 1 {
		return nil, false, fmt.Errorf(
			"get WireGuard configuration: expected one settings object, got %d",
			len(*result.JSON200.Config),
		)
	}
	return &(*result.JSON200.Config)[0], false, nil
}

func (r *settingsResource) readSettingsIntoState(ctx context.Context, data *settingsModel, diags *diag.Diagnostics) {
	config, missing, err := r.getSettings(ctx)
	if err != nil || missing {
		if err == nil {
			err = fmt.Errorf("API did not return global WireGuard settings after the write")
		}
		diags.AddError("Unable to verify WireGuard settings", err.Error())
		return
	}
	data.fromAPI(config)
}

func (r *resourceBase) getAll(ctx context.Context) (*api.WireGuardSettings, bool, error) {
	result, err := r.client.GetWireGuardWithResponse(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("get WireGuard configuration: %w", err)
	}
	if result.StatusCode() == http.StatusNotFound {
		return nil, true, nil
	}
	if result.StatusCode() != http.StatusOK || result.JSON200 == nil {
		return nil, false, fmt.Errorf("get WireGuard configuration: HTTP %d or missing JSON response", result.StatusCode())
	}
	return result.JSON200, false, nil
}

func addressListSchema(_ context.Context) schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Optional: true,
		Computed: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"address": schema.StringAttribute{
					Required:   true,
					Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
				},
				"descr": schema.StringAttribute{Optional: true, Computed: true},
			},
		},
	}
}

func addressListType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"address": types.StringType,
		"descr":   types.StringType,
	}}
}

func addressListValue(ctx context.Context, source *[]api.WGIPAddress) (basetypes.ListValue, error) {
	return addressListValueWithFactory(ctx, source, types.ListValueFrom)
}

func addressListValueWithFactory(
	ctx context.Context,
	source *[]api.WGIPAddress,
	newList func(context.Context, attr.Type, any) (basetypes.ListValue, diag.Diagnostics),
) (basetypes.ListValue, error) {
	if source == nil {
		return types.ListNull(addressListType()), nil
	}
	addresses := make([]ipAddressModel, 0, len(*source))
	for _, address := range *source {
		if address.Address == "" {
			return types.ListNull(addressListType()), fmt.Errorf("API returned a WireGuard IP address without a value")
		}
		descr := types.StringNull()
		if address.Descr != nil {
			descr = types.StringValue(*address.Descr)
		}
		addresses = append(addresses, ipAddressModel{Address: types.StringValue(address.Address), Descr: descr})
	}
	result, diags := newList(ctx, addressListType(), addresses)
	if diags.HasError() {
		return types.ListNull(
				addressListType(),
			), fmt.Errorf(
				"convert WireGuard IP addresses: %s",
				diags.Errors()[0].Detail(),
			)
	}
	return result, nil
}

func stringPointer(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := value.ValueString()
	return &result
}

func boolPointer(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := value.ValueBool()
	return &result
}

func stringValue(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func boolValue(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*value)
}

func intValue(value *int) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*value))
}

func validateWriteResult(status int, result *api.PfsenseResult, action string) error {
	if status != http.StatusOK || result == nil || strings.TrimSpace(result.Msg) == "" {
		return fmt.Errorf("%s: HTTP %d or invalid API result", action, status)
	}
	return nil
}
