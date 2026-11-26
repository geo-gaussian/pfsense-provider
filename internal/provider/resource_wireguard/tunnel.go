package resource_wireguard

import (
	"context"
	"fmt"
	"net/http"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
