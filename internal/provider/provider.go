package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ provider.Provider = &pfsenseProvider{}

type pfsenseProvider struct{}

func New() provider.Provider {
	return &pfsenseProvider{}
}

func (p *pfsenseProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "pfsense_nexus"
	resp.Version = "0.1.0"
}

func (p *pfsenseProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provider for managing pfSense Nexus resources.",
	}
}

func (p *pfsenseProvider) Configure(context.Context, provider.ConfigureRequest, *provider.ConfigureResponse) {
}

func (p *pfsenseProvider) Resources(context.Context) []func() resource.Resource {
	return nil
}

func (p *pfsenseProvider) DataSources(context.Context) []func() datasource.DataSource {
	return nil
}
