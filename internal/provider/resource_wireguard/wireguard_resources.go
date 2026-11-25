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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
		resp.Diagnostics.AddError("Invalid provider configuration", "The WireGuard resource received an unexpected API client.")
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
		Description: "Manages the global WireGuard settings. Removing this resource from configuration only relinquishes Terraform management; the API has no operation to delete global settings.",
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
}

func (r *settingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
		return nil, false, fmt.Errorf("get WireGuard configuration: expected one settings object, got %d", len(*result.JSON200.Config))
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

type tunnelResource struct{ resourceBase }

type ipAddressModel struct {
	Address types.String `tfsdk:"address"`
	Descr   types.String `tfsdk:"descr"`
}

type tunnelModel struct {
	Name       types.String `tfsdk:"name"`
	Descr      types.String `tfsdk:"descr"`
	Enabled    types.Bool   `tfsdk:"enabled"`
	ListenPort types.Int64  `tfsdk:"listenport"`
	PrivateKey types.String `tfsdk:"privatekey"`
	PublicKey  types.String `tfsdk:"publickey"`
	MTU        types.String `tfsdk:"mtu"`
	Addresses  types.List   `tfsdk:"addresses"`
}

func (r *tunnelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wireguard_tunnel"
}

func (r *tunnelResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	name := schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}}
	resp.Schema = schema.Schema{
		Description: "Manages a WireGuard tunnel.",
		Attributes: map[string]schema.Attribute{
			"name":       name,
			"descr":      schema.StringAttribute{Optional: true, Computed: true},
			"enabled":    schema.BoolAttribute{Optional: true, Computed: true},
			"listenport": schema.Int64Attribute{Optional: true, Computed: true},
			"privatekey": schema.StringAttribute{Optional: true, Computed: true, Sensitive: true},
			"publickey":  schema.StringAttribute{Required: true},
			"mtu":        schema.StringAttribute{Optional: true, Computed: true},
			"addresses":  addressListSchema(ctx),
		},
	}
}

func (r *tunnelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data tunnelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if data.PrivateKey.IsNull() || data.PrivateKey.IsUnknown() || data.PrivateKey.ValueString() == "" {
		resp.Diagnostics.AddError("Invalid WireGuard tunnel configuration", "privatekey must be set when creating a tunnel.")
		return
	}
	body, diags := data.toAPI(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.client.SetWireGuardTunnelWithResponse(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create WireGuard tunnel", err.Error())
		return
	}
	if err := validateWriteResult(result.StatusCode(), result.JSON200, "create WireGuard tunnel"); err != nil {
		resp.Diagnostics.AddError("Unable to create WireGuard tunnel", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readTunnelIntoState(ctx, data.Name.ValueString(), &data, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}
}

func (r *tunnelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data tunnelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	value, missing, err := r.getTunnel(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read WireGuard tunnel", err.Error())
		return
	}
	if missing {
		resp.State.RemoveResource(ctx)
		return
	}
	if err := data.fromAPI(ctx, value); err != nil {
		resp.Diagnostics.AddError("Unable to read WireGuard tunnel", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *tunnelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data tunnelModel
	var prior tunnelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if (data.PrivateKey.IsNull() || data.PrivateKey.IsUnknown()) &&
		!prior.PrivateKey.IsNull() && !prior.PrivateKey.IsUnknown() {
		data.PrivateKey = prior.PrivateKey
	}
	if data.PrivateKey.IsNull() || data.PrivateKey.IsUnknown() || data.PrivateKey.ValueString() == "" {
		resp.Diagnostics.AddError("Invalid WireGuard tunnel configuration", "privatekey must be set before updating this tunnel because the API requires it on writes.")
		return
	}
	body, diags := data.toAPI(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.client.UpdateWireGuardTunnelWithResponse(ctx, prior.Name.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update WireGuard tunnel", err.Error())
		return
	}
	if err := validateWriteResult(result.StatusCode(), result.JSON200, "update WireGuard tunnel"); err != nil {
		resp.Diagnostics.AddError("Unable to update WireGuard tunnel", err.Error())
		return
	}
	resp.State = req.State
	r.readTunnelIntoState(ctx, prior.Name.ValueString(), &data, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}
}

func (r *tunnelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data tunnelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.client.DeleteWireGuardTunnelWithResponse(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete WireGuard tunnel", err.Error())
		return
	}
	if result.StatusCode() == http.StatusBadRequest {
		_, missing, err := r.getTunnel(ctx, data.Name.ValueString())
		if err != nil || !missing {
			if err == nil {
				err = fmt.Errorf("tunnel still exists after HTTP 400")
			}
			resp.Diagnostics.AddError("Unable to delete WireGuard tunnel", err.Error())
		}
		return
	}
	if result.StatusCode() != http.StatusNotFound {
		if err := validateWriteResult(result.StatusCode(), result.JSON200, "delete WireGuard tunnel"); err != nil {
			resp.Diagnostics.AddError("Unable to delete WireGuard tunnel", err.Error())
			return
		}
	}
	_, missing, err := r.getTunnel(ctx, data.Name.ValueString())
	if err != nil || !missing {
		if err == nil {
			err = fmt.Errorf("tunnel is still present after the delete request")
		}
		resp.Diagnostics.AddError("Unable to verify deleted WireGuard tunnel", err.Error())
	}
}

func (r *tunnelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func (m *tunnelModel) toAPI(ctx context.Context) (api.WGTunnel, diag.Diagnostics) {
	var diags diag.Diagnostics
	body := api.WGTunnel{
		Name:       m.Name.ValueString(),
		Descr:      stringPointer(m.Descr),
		Enabled:    boolPointer(m.Enabled),
		Privatekey: m.PrivateKey.ValueString(),
		Publickey:  m.PublicKey.ValueString(),
		Mtu:        stringPointer(m.MTU),
	}
	if !m.ListenPort.IsNull() && !m.ListenPort.IsUnknown() {
		port := int(m.ListenPort.ValueInt64())
		body.Listenport = &port
	}
	if !m.Addresses.IsNull() && !m.Addresses.IsUnknown() {
		var addresses []ipAddressModel
		diags.Append(m.Addresses.ElementsAs(ctx, &addresses, false)...)
		if diags.HasError() {
			return body, diags
		}
		result := make([]api.WGIPAddress, 0, len(addresses))
		for _, address := range addresses {
			result = append(result, api.WGIPAddress{Address: address.Address.ValueString(), Descr: stringPointer(address.Descr)})
		}
		body.Addresses = &result
	}
	return body, diags
}

func (m *tunnelModel) fromAPI(ctx context.Context, value *api.WGTunnel) error {
	if value.Name == "" || value.Publickey == "" {
		return fmt.Errorf("API returned a tunnel without its name or public key")
	}
	m.Name = types.StringValue(value.Name)
	m.Descr = stringValue(value.Descr)
	m.Enabled = boolValue(value.Enabled)
	m.ListenPort = intValue(value.Listenport)
	if value.Privatekey != "" {
		m.PrivateKey = types.StringValue(value.Privatekey)
	}
	m.PublicKey = types.StringValue(value.Publickey)
	m.MTU = stringValue(value.Mtu)
	addresses, err := addressListValue(ctx, value.Addresses)
	if err != nil {
		return err
	}
	m.Addresses = addresses
	return nil
}

func (r *tunnelResource) getTunnel(ctx context.Context, name string) (*api.WGTunnel, bool, error) {
	settings, missing, err := r.getAll(ctx)
	if err != nil || missing {
		return nil, missing, err
	}
	if settings.Tunnels == nil || settings.Tunnels.Item == nil {
		return nil, false, fmt.Errorf("get WireGuard tunnel %q: API omitted the tunnel list", name)
	}
	for i := range *settings.Tunnels.Item {
		if (*settings.Tunnels.Item)[i].Name == name {
			return &(*settings.Tunnels.Item)[i], false, nil
		}
	}
	return nil, true, nil
}

func (r *tunnelResource) readTunnelIntoState(ctx context.Context, name string, data *tunnelModel, diags *diag.Diagnostics) {
	value, missing, err := r.getTunnel(ctx, name)
	if err != nil || missing {
		if err == nil {
			err = fmt.Errorf("API did not return tunnel %q after the write", name)
		}
		diags.AddError("Unable to verify WireGuard tunnel", err.Error())
		return
	}
	if err := data.fromAPI(ctx, value); err != nil {
		diags.AddError("Unable to read WireGuard tunnel", err.Error())
	}
}

type peerResource struct{ resourceBase }

type peerModel struct {
	PublicKey           types.String `tfsdk:"publickey"`
	Enabled             types.Bool   `tfsdk:"enabled"`
	Tunnel              types.String `tfsdk:"tun"`
	Descr               types.String `tfsdk:"descr"`
	Endpoint            types.String `tfsdk:"endpoint"`
	Port                types.String `tfsdk:"port"`
	PersistentKeepalive types.String `tfsdk:"persistentkeepalive"`
	PresharedKey        types.String `tfsdk:"presharedkey"`
	AllowedIPs          types.List   `tfsdk:"allowedips"`
}

func (r *peerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wireguard_peer"
}

func (r *peerResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	publicKey := schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}}
	resp.Schema = schema.Schema{
		Description: "Manages a WireGuard peer.",
		Attributes: map[string]schema.Attribute{
			"publickey":           publicKey,
			"enabled":             schema.BoolAttribute{Optional: true, Computed: true},
			"tun":                 schema.StringAttribute{Optional: true, Computed: true},
			"descr":               schema.StringAttribute{Optional: true, Computed: true},
			"endpoint":            schema.StringAttribute{Optional: true, Computed: true},
			"port":                schema.StringAttribute{Optional: true, Computed: true},
			"persistentkeepalive": schema.StringAttribute{Optional: true, Computed: true},
			"presharedkey":        schema.StringAttribute{Optional: true, Computed: true, Sensitive: true},
			"allowedips":          addressListSchema(ctx),
		},
	}
}

func (r *peerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data peerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, diags := data.toAPI(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.client.SetWireGuardPeerWithResponse(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create WireGuard peer", err.Error())
		return
	}
	if err := validateWriteResult(result.StatusCode(), result.JSON200, "create WireGuard peer"); err != nil {
		resp.Diagnostics.AddError("Unable to create WireGuard peer", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readPeerIntoState(ctx, data.PublicKey.ValueString(), &data, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}
}

func (r *peerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data peerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	value, missing, err := r.getPeer(ctx, data.PublicKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read WireGuard peer", err.Error())
		return
	}
	if missing {
		resp.State.RemoveResource(ctx)
		return
	}
	if err := data.fromAPI(ctx, value); err != nil {
		resp.Diagnostics.AddError("Unable to read WireGuard peer", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *peerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data peerModel
	var prior peerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if (data.PresharedKey.IsNull() || data.PresharedKey.IsUnknown()) &&
		!prior.PresharedKey.IsNull() && !prior.PresharedKey.IsUnknown() {
		data.PresharedKey = prior.PresharedKey
	}
	body, diags := data.toAPI(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.client.UpdateWireGuardPeerWithResponse(ctx, prior.PublicKey.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update WireGuard peer", err.Error())
		return
	}
	if err := validateWriteResult(result.StatusCode(), result.JSON200, "update WireGuard peer"); err != nil {
		resp.Diagnostics.AddError("Unable to update WireGuard peer", err.Error())
		return
	}
	resp.State = req.State
	r.readPeerIntoState(ctx, prior.PublicKey.ValueString(), &data, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}
}

func (r *peerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data peerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	publicKey := data.PublicKey.ValueString()
	result, err := r.client.DeleteWireGuardPeerWithResponse(ctx, publicKey)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete WireGuard peer", err.Error())
		return
	}
	if result.StatusCode() == http.StatusBadRequest {
		_, missing, err := r.getPeer(ctx, publicKey)
		if err != nil || !missing {
			if err == nil {
				err = fmt.Errorf("peer is still present after HTTP 400")
			}
			resp.Diagnostics.AddError("Unable to delete WireGuard peer", err.Error())
		}
		return
	}
	if result.StatusCode() != http.StatusNotFound {
		if err := validateWriteResult(result.StatusCode(), result.JSON200, "delete WireGuard peer"); err != nil {
			resp.Diagnostics.AddError("Unable to delete WireGuard peer", err.Error())
			return
		}
	}
	_, missing, err := r.getPeer(ctx, publicKey)
	if err != nil || !missing {
		if err == nil {
			err = fmt.Errorf("peer %q is still present after the delete request", publicKey)
		}
		resp.Diagnostics.AddError("Unable to verify deleted WireGuard peer", err.Error())
	}
}

func (r *peerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("publickey"), req, resp)
}

func (m *peerModel) toAPI(ctx context.Context) (api.WGPeer, diag.Diagnostics) {
	var diags diag.Diagnostics
	body := api.WGPeer{
		Publickey:           m.PublicKey.ValueString(),
		Enabled:             boolPointer(m.Enabled),
		Tun:                 stringPointer(m.Tunnel),
		Descr:               stringPointer(m.Descr),
		Endpoint:            stringPointer(m.Endpoint),
		Port:                stringPointer(m.Port),
		Persistentkeepalive: stringPointer(m.PersistentKeepalive),
		Presharedkey:        stringPointer(m.PresharedKey),
	}
	if !m.AllowedIPs.IsNull() && !m.AllowedIPs.IsUnknown() {
		var addresses []ipAddressModel
		diags.Append(m.AllowedIPs.ElementsAs(ctx, &addresses, false)...)
		if diags.HasError() {
			return body, diags
		}
		result := make([]api.WGIPAddress, 0, len(addresses))
		for _, address := range addresses {
			result = append(result, api.WGIPAddress{Address: address.Address.ValueString(), Descr: stringPointer(address.Descr)})
		}
		body.Allowedips = &result
	}
	return body, diags
}

func (m *peerModel) fromAPI(ctx context.Context, value *api.WGPeer) error {
	if value.Publickey == "" {
		return fmt.Errorf("API returned a peer without a public key")
	}
	m.PublicKey = types.StringValue(value.Publickey)
	m.Enabled = boolValue(value.Enabled)
	m.Tunnel = stringValue(value.Tun)
	m.Descr = stringValue(value.Descr)
	m.Endpoint = stringValue(value.Endpoint)
	m.Port = stringValue(value.Port)
	m.PersistentKeepalive = stringValue(value.Persistentkeepalive)
	if value.Presharedkey != nil && *value.Presharedkey != "" {
		m.PresharedKey = types.StringValue(*value.Presharedkey)
	}
	addresses, err := addressListValue(ctx, value.Allowedips)
	if err != nil {
		return err
	}
	m.AllowedIPs = addresses
	return nil
}

func (r *peerResource) getPeer(ctx context.Context, publicKey string) (*api.WGPeer, bool, error) {
	settings, missing, err := r.getAll(ctx)
	if err != nil || missing {
		return nil, missing, err
	}
	if settings.Peers == nil || settings.Peers.Item == nil {
		return nil, false, fmt.Errorf("get WireGuard peer %q: API omitted the peer list", publicKey)
	}
	for i := range *settings.Peers.Item {
		if (*settings.Peers.Item)[i].Publickey == publicKey {
			return &(*settings.Peers.Item)[i], false, nil
		}
	}
	return nil, true, nil
}

func (r *peerResource) readPeerIntoState(ctx context.Context, publicKey string, data *peerModel, diags *diag.Diagnostics) {
	value, missing, err := r.getPeer(ctx, publicKey)
	if err != nil || missing {
		if err == nil {
			err = fmt.Errorf("API did not return peer %q after the write", publicKey)
		}
		diags.AddError("Unable to verify WireGuard peer", err.Error())
		return
	}
	if err := data.fromAPI(ctx, value); err != nil {
		diags.AddError("Unable to read WireGuard peer", err.Error())
	}
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
		return types.ListNull(addressListType()), fmt.Errorf("convert WireGuard IP addresses: %s", diags.Errors()[0].Detail())
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
