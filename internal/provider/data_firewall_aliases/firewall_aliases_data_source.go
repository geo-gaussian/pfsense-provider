package data_firewall_aliases

import (
	"context"
	"fmt"
	"net/http"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &aliasesDataSource{}
var _ datasource.DataSourceWithConfigure = &aliasesDataSource{}

type aliasesDataSource struct {
	client *api.ClientWithResponses
}

type aliasesModel struct {
	Aliases []aliasModel `tfsdk:"aliases"`
}

type aliasModel struct {
	Name      types.String `tfsdk:"name"`
	Type      types.String `tfsdk:"type"`
	Descr     types.String `tfsdk:"descr"`
	Truncated types.Bool   `tfsdk:"truncated"`
}

func New() datasource.DataSource { return &aliasesDataSource{} }

func (d *aliasesDataSource) Metadata(
	_ context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_firewall_aliases"
}

func (d *aliasesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists user-defined firewall alias metadata on the local firewall.",
		Attributes: map[string]schema.Attribute{
			"aliases": schema.ListNestedAttribute{
				Computed:    true,
				Description: "User-defined alias metadata, or null when the API omits the aliases field.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"name":      schema.StringAttribute{Computed: true, Description: "Alias name."},
					"type":      schema.StringAttribute{Computed: true, Description: "Alias type."},
					"descr":     schema.StringAttribute{Computed: true, Description: "Alias description."},
					"truncated": schema.BoolAttribute{Computed: true, Description: "Whether the API truncated alias values."},
				}},
			},
		},
	}
}

func (d *aliasesDataSource) Configure(
	_ context.Context,
	req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	d.client, ok = req.ProviderData.(*api.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid provider configuration",
			"The firewall aliases data source received an unexpected client.",
		)
	}
}

func (d *aliasesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	result, err := d.client.FirewallGetAliasesWithResponse(ctx, nil)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read firewall aliases", err.Error())
		return
	}
	if result.StatusCode() != http.StatusOK || result.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Unable to read firewall aliases",
			fmt.Sprintf("API returned HTTP %d or an invalid JSON response.", result.StatusCode()),
		)
		return
	}
	if result.JSON200.Aliases == nil {
		resp.Diagnostics.AddWarning(
			"Alias list omitted",
			"The API response did not include an aliases list. The data source cannot determine whether any aliases exist.",
		)
		resp.Diagnostics.Append(resp.State.Set(ctx, &aliasesModel{})...)
		return
	}

	data := aliasesModel{Aliases: make([]aliasModel, 0, len(*result.JSON200.Aliases))}
	for _, alias := range *result.JSON200.Aliases {
		if alias.Name == "" {
			resp.Diagnostics.AddError("Unable to read firewall aliases", "The API returned an alias without a name.")
			return
		}
		kind := types.StringNull()
		if alias.Type != nil {
			kind = types.StringValue(string(*alias.Type))
		}
		data.Aliases = append(data.Aliases, aliasModel{
			Name: types.StringValue(alias.Name), Type: kind, Descr: types.StringPointerValue(alias.Descr),
			Truncated: types.BoolPointerValue(alias.Truncated),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
