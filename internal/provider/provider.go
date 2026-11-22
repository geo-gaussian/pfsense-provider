package provider

import (
	"context"
	"strings"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/client"
	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/provider/data_firewall_aliases"
	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/provider/resource_firewall_alias"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &pfsenseProvider{}

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
			"insecure": schema.BoolAttribute{Optional: true, Description: "Disable TLS certificate verification. Use only in a test environment."},
		},
	}
}

func (p *pfsenseProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	p.configure(ctx, req, resp, api.NewClientWithResponses)
}

func (p *pfsenseProvider) configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse, newAPI func(string, ...api.ClientOption) (*api.ClientWithResponses, error)) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if config.URL.IsUnknown() || config.Username.IsUnknown() || config.Password.IsUnknown() || config.Insecure.IsUnknown() {
		resp.Diagnostics.AddError("Unknown provider configuration", "The controller URL, credentials, and TLS settings must be known before configuring the provider.")
		return
	}
	nexus, err := client.New(config.URL.ValueString(), config.Username.ValueString(), config.Password.ValueString(), config.Insecure.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Invalid provider configuration", err.Error())
		return
	}
	apiClient, err := newAPI(strings.TrimRight(config.URL.ValueString(), "/")+"/api/device/pfsense/localhost/api", api.WithHTTPClient(nexus))
	if err != nil {
		resp.Diagnostics.AddError("Unable to configure API client", err.Error())
		return
	}
	resp.ResourceData = apiClient
	resp.DataSourceData = apiClient
}

func (p *pfsenseProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{resource_firewall_alias.New}
}

func (p *pfsenseProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{data_firewall_aliases.New}
}
