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
