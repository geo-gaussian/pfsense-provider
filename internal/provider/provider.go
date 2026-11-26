package provider

import (
	"context"
	"strings"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/client"
	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/provider/data_firewall_aliases"
	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/provider/data_readonly"
	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/provider/ephemeral_wireguard"
	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/provider/firewall_resources"
	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/provider/resource_firewall_alias"
	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/provider/resource_wireguard"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &pfsenseProvider{}
var _ provider.ProviderWithEphemeralResources = &pfsenseProvider{}
var _ provider.ProviderWithListResources = &pfsenseProvider{}

type pfsenseProvider struct {
	version string
}

type providerModel struct {
	URL      types.String `tfsdk:"url"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	Insecure types.Bool   `tfsdk:"insecure"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &pfsenseProvider{version: version}
	}
}

func (p *pfsenseProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "pfsense"
	resp.Version = p.version
}

func (p *pfsenseProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provider for managing the local pfSense Nexus firewall.",
		Attributes: map[string]schema.Attribute{
			"url":      schema.StringAttribute{Required: true, Description: "HTTPS origin of the Nexus controller."},
			"username": schema.StringAttribute{Required: true},
			"password": schema.StringAttribute{Required: true, Sensitive: true},
			"insecure": schema.BoolAttribute{
				Optional:    true,
				Description: "Disable TLS certificate verification. Use only in a test environment.",
			},
		},
	}
}

func (p *pfsenseProvider) Configure(
	ctx context.Context,
	req provider.ConfigureRequest,
	resp *provider.ConfigureResponse,
) {
	p.configure(ctx, req, resp, api.NewClientWithResponses)
}

func (p *pfsenseProvider) configure(
	ctx context.Context,
	req provider.ConfigureRequest,
	resp *provider.ConfigureResponse,
	newAPI func(string, ...api.ClientOption) (*api.ClientWithResponses, error),
) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if config.isUnknown() {
		resp.Diagnostics.AddError(
			"Unknown provider configuration",
			"The controller URL, credentials, and TLS settings must be known before configuring the provider.",
		)
		return
	}
	nexus, err := client.New(
		config.URL.ValueString(),
		config.Username.ValueString(),
		config.Password.ValueString(),
		config.Insecure.ValueBool(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Invalid provider configuration", err.Error())
		return
	}
	apiClient, err := newAPI(
		strings.TrimRight(config.URL.ValueString(), "/")+"/api/device/pfsense/localhost/api",
		api.WithHTTPClient(nexus),
	)
	if err != nil {
		resp.Diagnostics.AddError("Unable to configure API client", err.Error())
		return
	}
	setProviderData(resp, apiClient)
}

func setProviderData(resp *provider.ConfigureResponse, apiClient *api.ClientWithResponses) {
	resp.ResourceData = apiClient
	resp.DataSourceData = apiClient
	resp.EphemeralResourceData = apiClient
	resp.ListResourceData = apiClient
}

func (config providerModel) isUnknown() bool {
	return config.URL.IsUnknown() || config.Username.IsUnknown() || config.Password.IsUnknown() ||
		config.Insecure.IsUnknown()
}

func (p *pfsenseProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		identified(resource_firewall_alias.New, "name"),
		identified(firewall_resources.NewNATNpt, "id"),
		identified(firewall_resources.NewNATOneToOne, "id"),
		identified(firewall_resources.NewNATPortForward, "id"),
		identified(firewall_resources.NewRule, "interface", "id"),
		identified(firewall_resources.NewSchedule, "id"),
		identified(firewall_resources.NewALTQ, "name"),
		identified(firewall_resources.NewLimiter, "name"),
		identified(firewall_resources.NewPFlowExporter, "id"),
		identified(firewall_resources.NewVirtualIP, "id"),
		identified(resource_wireguard.NewSettings, "id"),
		identified(resource_wireguard.NewTunnel, "name"),
		identified(resource_wireguard.NewPeer, "publickey"),
	}
}

func (p *pfsenseProvider) ListResources(ctx context.Context) []func() list.ListResource {
	factories := p.Resources(ctx)
	result := make([]func() list.ListResource, 0, len(factories))
	for _, factory := range factories {
		result = append(result, func() list.ListResource {
			return &configurationList{managed: factory().(*identifiedResource)}
		})
	}
	return result
}

func (p *pfsenseProvider) DataSources(context.Context) []func() datasource.DataSource {
	sources := []func() datasource.DataSource{
		data_firewall_aliases.New,
		data_readonly.NewFirewallNATNpt,
		data_readonly.NewFirewallNATOneToOne,
		data_readonly.NewFirewallNATOutbound,
		data_readonly.NewFirewallNATPortForward,
		data_readonly.NewFirewallInterfaces,
		data_readonly.NewFirewallRules,
		data_readonly.NewFirewallSchedules,
		data_readonly.NewFirewallTrafficShaper,
		data_readonly.NewFirewallALTQ,
		data_readonly.NewFirewallLimiters,
		data_readonly.NewFirewallPFlow,
		data_readonly.NewFirewallVirtualIPs,
		data_readonly.NewInterfaces,
		data_readonly.NewInterfaceAssignments,
		data_readonly.NewHostInterfaces,
		data_readonly.NewInterfaceDescriptors,
		data_readonly.NewBridgeInterfaces,
		data_readonly.NewGIFInterfaces,
		data_readonly.NewGREInterfaces,
		data_readonly.NewInterfaceGroups,
		data_readonly.NewLAGGInterfaces,
		data_readonly.NewQinQInterfaces,
		data_readonly.NewVLANInterfaces,
		data_readonly.NewVXLANInterfaces,
		data_readonly.NewGateways,
		data_readonly.NewStaticRoutes,
	}
	return append(sources, serviceDataSources()...)
}

func serviceDataSources() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		data_readonly.NewDNSForwarder,
		data_readonly.NewIGMPProxies,
		data_readonly.NewMDNS,
		data_readonly.NewDHCPv4Relay,
		data_readonly.NewDHCPv6Relay,
		data_readonly.NewRouterAdvertisements,
		data_readonly.NewIPSecWidget,
		data_readonly.NewOpenVPNStatus,
		data_readonly.NewWireGuardConfig,
		data_readonly.NewWireGuardTunnelConf,
		data_readonly.NewWireGuardStatus,
		data_readonly.NewDHCPLeases,
		data_readonly.NewDNSResolverStatus,
		data_readonly.NewNDPTable,
		data_readonly.NewServicesStatus,
		data_readonly.NewUPnPConfig,
		data_readonly.NewUPnPStatus,
		data_readonly.NewServiceWatchdog,
		data_readonly.NewSystemSummary,
		data_readonly.NewCertificateOptions,
		data_readonly.NewDirtySubsystems,
		data_readonly.NewFirmwareInfo,
		data_readonly.NewGatewayPriorities,
		data_readonly.NewGatewayGroups,
		data_readonly.NewGatewaysStatus,
		data_readonly.NewInterfacesStatus,
		data_readonly.NewMonitoringDatasets,
		data_readonly.NewQueueStats,
		data_readonly.NewRoutesDirtyState,
		data_readonly.NewSystemStatus,
		data_readonly.NewInterfaceStats,
		data_readonly.NewInterfaceWidgets,
		data_readonly.NewInterfaceStatisticsWidget,
	}
}

func (p *pfsenseProvider) EphemeralResources(context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{ephemeral_wireguard.New}
}
