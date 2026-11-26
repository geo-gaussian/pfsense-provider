package resource_mdns

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type mdnsResource struct{ client *api.ClientWithResponses }

type mdnsModel struct {
	ID                types.String `tfsdk:"id"`
	Enable            types.Bool   `tfsdk:"enable"`
	CarpVIPInterface  types.String `tfsdk:"carp_vip_intf"`
	IncludeInterfaces types.List   `tfsdk:"include_interfaces"`
	ExcludeInterfaces types.List   `tfsdk:"exclude_interfaces"`
	DisableIPv4       types.Bool   `tfsdk:"disable_ipv4"`
	DisableIPv6       types.Bool   `tfsdk:"disable_ipv6"`
	Reflection        types.Bool   `tfsdk:"reflection"`
	Publish           types.Object `tfsdk:"publish"`
}

type publishModel struct {
	Enable         types.Bool   `tfsdk:"enable"`
	Addresses      types.Bool   `tfsdk:"addresses"`
	HostInfo       types.Bool   `tfsdk:"hostinfo"`
	Workstation    types.Bool   `tfsdk:"workstation"`
	Domain         types.Bool   `tfsdk:"domain"`
	CustomHostname types.String `tfsdk:"custom_hostname"`
	CustomDomain   types.String `tfsdk:"custom_domain"`
	AAAAOverIPv4   types.Bool   `tfsdk:"aaaa_over_ipv4"`
	AOverIPv6      types.Bool   `tfsdk:"a_over_ipv6"`
}

func New() resource.Resource { return &mdnsResource{} }

func (r *mdnsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mdns"
}

func (r *mdnsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	boolAttr := schema.BoolAttribute{Optional: true, Computed: true}
	stringAttr := schema.StringAttribute{Optional: true, Computed: true}
	resp.Schema = schema.Schema{
		Description: "Manages multicast DNS settings. Destroy disables the service and retains other settings.",
		Attributes: map[string]schema.Attribute{
			"id":                 schema.StringAttribute{Computed: true},
			"enable":             boolAttr,
			"carp_vip_intf":      stringAttr,
			"include_interfaces": schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType},
			"exclude_interfaces": schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType},
			"disable_ipv4":       boolAttr,
			"disable_ipv6":       boolAttr,
			"reflection":         boolAttr,
			"publish": schema.SingleNestedAttribute{
				Optional: true, Computed: true,
				Attributes: map[string]schema.Attribute{
					"enable": boolAttr, "addresses": boolAttr, "hostinfo": boolAttr, "workstation": boolAttr,
					"domain": boolAttr, "custom_hostname": stringAttr, "custom_domain": stringAttr,
					"aaaa_over_ipv4": boolAttr, "a_over_ipv6": boolAttr,
				},
			},
		},
	}
}

func (r *mdnsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*api.ClientWithResponses)
	if !ok || client == nil {
		resp.Diagnostics.AddError("Invalid provider configuration", "The mDNS resource received an invalid API client.")
		return
	}
	r.client = client
}

func (r *mdnsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data mdnsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	config, err := r.write(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Unable to set mDNS settings", err.Error())
		return
	}
	resp.Diagnostics.Append(data.fromAPI(ctx, config)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}
}

func (r *mdnsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data mdnsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	config, missing, err := r.get(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read mDNS settings", err.Error())
		return
	}
	if missing {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(data.fromAPI(ctx, config)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}
}

func (r *mdnsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data mdnsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.State = req.State
	config, err := r.write(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update mDNS settings", err.Error())
		return
	}
	resp.Diagnostics.Append(data.fromAPI(ctx, config)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}
}

func (r *mdnsResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	config, missing, err := r.get(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to disable mDNS", err.Error())
		return
	}
	if missing {
		return
	}
	disabled := false
	config.Enable = &disabled
	config.AvailInterfaces, config.CarpVips = nil, nil
	result, err := r.client.SetMDNSConfigWithResponse(ctx, *config)
	if err != nil {
		resp.Diagnostics.AddError("Unable to disable mDNS", err.Error())
		return
	}
	if result.StatusCode() != http.StatusOK || result.JSON200 == nil || jsonNull(result.Body) ||
		result.JSON200.Enable == nil ||
		*result.JSON200.Enable {
		resp.Diagnostics.AddError("Unable to disable mDNS",
			fmt.Sprintf("API returned HTTP %d or did not confirm disabling.", result.StatusCode()))
		return
	}
	config, missing, err = r.get(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to verify disabled mDNS", err.Error())
		return
	}
	if !missing && (config.Enable == nil || *config.Enable) {
		resp.Diagnostics.AddError("Unable to verify disabled mDNS", "The API did not confirm that mDNS is disabled.")
	}
}

func (r *mdnsResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	if req.ID != "mdns" {
		resp.Diagnostics.AddError("Invalid mDNS identifier", "The singleton import ID must be mdns.")
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *mdnsResource) get(ctx context.Context) (*api.MDNSConfig, bool, error) {
	result, err := r.client.GetMDNSConfigWithResponse(ctx)
	if err != nil {
		return nil, false, err
	}
	if result.StatusCode() == http.StatusNotFound {
		return nil, true, nil
	}
	if result.StatusCode() != http.StatusOK || result.JSON200 == nil || jsonNull(result.Body) {
		return nil, false, fmt.Errorf("API returned HTTP %d or an invalid mDNS response", result.StatusCode())
	}
	return result.JSON200, false, nil
}

func (r *mdnsResource) write(ctx context.Context, data *mdnsModel) (*api.MDNSConfig, error) {
	config, missing, err := r.get(ctx)
	if err != nil {
		return nil, err
	}
	if missing {
		return nil, fmt.Errorf("API did not return existing mDNS settings")
	}
	diags := data.merge(ctx, config)
	if diags.HasError() {
		return nil, fmt.Errorf("invalid mDNS settings: %v", diags)
	}
	config.AvailInterfaces, config.CarpVips = nil, nil
	result, err := r.client.SetMDNSConfigWithResponse(ctx, *config)
	if err != nil {
		return nil, err
	}
	if result.StatusCode() != http.StatusOK || result.JSON200 == nil || jsonNull(result.Body) {
		return nil, fmt.Errorf("API returned HTTP %d or an invalid mDNS response", result.StatusCode())
	}
	return result.JSON200, nil
}

func jsonNull(body []byte) bool {
	return bytes.Equal(bytes.TrimSpace(body), []byte("null"))
}

func boolPtr(v types.Bool, current *bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return current
	}
	value := v.ValueBool()
	return &value
}

func stringPtr(v types.String, current *string) *string {
	if v.IsNull() || v.IsUnknown() {
		return current
	}
	value := v.ValueString()
	return &value
}

func boolValue(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*value)
}

func stringValue(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func (m *mdnsModel) merge(ctx context.Context, config *api.MDNSConfig) diag.Diagnostics {
	var diags diag.Diagnostics
	config.Enable = boolPtr(m.Enable, config.Enable)
	config.CarpVipIntf = stringPtr(m.CarpVIPInterface, config.CarpVipIntf)
	config.DisableIpv4 = boolPtr(m.DisableIPv4, config.DisableIpv4)
	config.DisableIpv6 = boolPtr(m.DisableIPv6, config.DisableIpv6)
	config.Reflection = boolPtr(m.Reflection, config.Reflection)
	for _, field := range []struct {
		value types.List
		dest  **[]string
	}{{m.IncludeInterfaces, &config.IncludeInterfaces}, {m.ExcludeInterfaces, &config.ExcludeInterfaces}} {
		if !field.value.IsNull() && !field.value.IsUnknown() {
			var values []string
			diags.Append(field.value.ElementsAs(ctx, &values, false)...)
			*field.dest = &values
		}
	}
	if !m.Publish.IsNull() && !m.Publish.IsUnknown() {
		var publish publishModel
		diags.Append(m.Publish.As(ctx, &publish, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return diags
		}
		if config.Publish == nil {
			config.Publish = &api.MDNSPublishing{}
		}
		publish.merge(config.Publish)
	}
	return diags
}

func (p publishModel) merge(config *api.MDNSPublishing) {
	config.Enable = boolPtr(p.Enable, config.Enable)
	config.Addresses = boolPtr(p.Addresses, config.Addresses)
	config.Hostinfo = boolPtr(p.HostInfo, config.Hostinfo)
	config.Workstation = boolPtr(p.Workstation, config.Workstation)
	config.Domain = boolPtr(p.Domain, config.Domain)
	config.CustomHostname = stringPtr(p.CustomHostname, config.CustomHostname)
	config.CustomDomain = stringPtr(p.CustomDomain, config.CustomDomain)
	config.AaaaOverIpv4 = boolPtr(p.AAAAOverIPv4, config.AaaaOverIpv4)
	config.AOverIpv6 = boolPtr(p.AOverIPv6, config.AOverIpv6)
}

func (m *mdnsModel) fromAPI(ctx context.Context, config *api.MDNSConfig) diag.Diagnostics {
	var diags diag.Diagnostics
	m.ID = types.StringValue("mdns")
	m.Enable = boolValue(config.Enable)
	m.CarpVIPInterface = stringValue(config.CarpVipIntf)
	m.DisableIPv4 = boolValue(config.DisableIpv4)
	m.DisableIPv6 = boolValue(config.DisableIpv6)
	m.Reflection = boolValue(config.Reflection)
	m.IncludeInterfaces, diags = listValue(ctx, config.IncludeInterfaces)
	excluded, more := listValue(ctx, config.ExcludeInterfaces)
	diags.Append(more...)
	m.ExcludeInterfaces = excluded
	if config.Publish == nil {
		m.Publish = types.ObjectNull(publishTypes())
	} else {
		p := config.Publish
		var more diag.Diagnostics
		m.Publish, more = types.ObjectValueFrom(ctx, publishTypes(), publishModel{
			Enable: boolValue(p.Enable), Addresses: boolValue(p.Addresses), HostInfo: boolValue(p.Hostinfo),
			Workstation: boolValue(p.Workstation), Domain: boolValue(p.Domain),
			CustomHostname: stringValue(p.CustomHostname), CustomDomain: stringValue(p.CustomDomain),
			AAAAOverIPv4: boolValue(p.AaaaOverIpv4), AOverIPv6: boolValue(p.AOverIpv6),
		})
		diags.Append(more...)
	}
	return diags
}

func listValue(ctx context.Context, value *[]string) (types.List, diag.Diagnostics) {
	if value == nil {
		return types.ListNull(types.StringType), nil
	}
	return types.ListValueFrom(ctx, types.StringType, *value)
}

func publishTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enable": types.BoolType, "addresses": types.BoolType, "hostinfo": types.BoolType,
		"workstation": types.BoolType, "domain": types.BoolType, "custom_hostname": types.StringType,
		"custom_domain": types.StringType, "aaaa_over_ipv4": types.BoolType, "a_over_ipv6": types.BoolType,
	}
}
