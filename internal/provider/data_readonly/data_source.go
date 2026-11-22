package data_readonly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type apiResponse interface {
	StatusCode() int
	GetBody() []byte
}

type readFunc func(context.Context, *api.ClientWithResponses, map[string]string) (apiResponse, error)

type dataSource struct {
	client          *api.ClientWithResponses
	typeName        string
	description     string
	jsonDescription string
	sensitiveJSON   bool
	arguments       map[string]schema.Attribute
	read            readFunc
}

var _ datasource.DataSource = &dataSource{}
var _ datasource.DataSourceWithConfigure = &dataSource{}

func newDataSource(typeName, description string, arguments map[string]schema.Attribute, read readFunc) datasource.DataSource {
	return newDataSourceWithJSONDescription(
		typeName,
		description,
		arguments,
		"The complete API response as JSON. Use jsondecode() to access its fields.",
		false,
		read,
	)
}

func newSensitiveDataSource(typeName, description string, arguments map[string]schema.Attribute, read readFunc) datasource.DataSource {
	return newDataSourceWithJSONDescription(
		typeName,
		description,
		arguments,
		"The complete API response as sensitive JSON. Use jsondecode() to access its fields. Protect Terraform state and saved plans.",
		true,
		read,
	)
}

func newDataSourceWithJSONDescription(
	typeName, description string,
	arguments map[string]schema.Attribute,
	jsonDescription string,
	sensitiveJSON bool,
	read readFunc,
) datasource.DataSource {
	return &dataSource{
		typeName:        typeName,
		description:     description,
		jsonDescription: jsonDescription,
		sensitiveJSON:   sensitiveJSON,
		arguments:       arguments,
		read:            read,
	}
}

func (d *dataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + d.typeName
}

func (d *dataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := make(map[string]schema.Attribute, len(d.arguments)+1)
	for name, attribute := range d.arguments {
		attributes[name] = attribute
	}
	attributes["json"] = schema.StringAttribute{
		Computed:    true,
		Description: d.jsonDescription,
		Sensitive:   d.sensitiveJSON,
	}
	resp.Schema = schema.Schema{
		Description: d.description,
		Attributes:  attributes,
	}
}

func (d *dataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*api.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError("Invalid provider configuration", "The read-only data source received an unexpected API client.")
		return
	}
	d.client = client
}

func (d *dataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Unable to read data source", "The API client is not configured.")
		return
	}

	var configured types.Object
	resp.Diagnostics.Append(req.Config.Get(ctx, &configured)...)
	if resp.Diagnostics.HasError() {
		return
	}

	configuredAttributes := configured.Attributes()
	arguments := make(map[string]string, len(d.arguments))
	state := make(map[string]attr.Value, len(d.arguments)+1)
	for name := range d.arguments {
		value, ok := configuredAttributes[name].(types.String)
		if !ok || value.IsNull() || value.IsUnknown() {
			resp.Diagnostics.AddError("Invalid data source argument", fmt.Sprintf("Argument %q must be a known string value.", name))
			return
		}
		arguments[name] = value.ValueString()
		state[name] = value
	}

	result, err := d.read(ctx, d.client, arguments)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read data source", err.Error())
		return
	}
	if result == nil || result.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Unable to read data source",
			fmt.Sprintf("The API returned HTTP %d or an invalid JSON response.", responseStatusCode(result)),
		)
		return
	}

	body := result.GetBody()
	if !json.Valid(body) || bytes.Equal(bytes.TrimSpace(body), []byte("null")) {
		resp.Diagnostics.AddError("Unable to read data source", "The API returned an empty JSON response.")
		return
	}

	state["json"] = types.StringValue(string(body))
	setReadState(ctx, state, resp)
}

func setReadState(ctx context.Context, state map[string]attr.Value, resp *datasource.ReadResponse) {
	stateTypes := make(map[string]attr.Type, len(state))
	for name := range state {
		stateTypes[name] = types.StringType
	}
	stateValue, diags := types.ObjectValue(stateTypes, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &stateValue)...)
}

func responseStatusCode(response apiResponse) int {
	if response == nil {
		return 0
	}
	return response.StatusCode()
}
