package resource_haproxy_resolver

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type resolverResource struct {
	client *api.ClientWithResponses
}

type resolverModel struct {
	ID     types.String `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Server types.String `tfsdk:"server"`
	Port   types.String `tfsdk:"port"`
}

func New() resource.Resource { return &resolverResource{} }

func (r *resolverResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_haproxy_resolver"
}

func (r *resolverResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an HAProxy DNS resolver and applies the HAProxy configuration after changes.",
		Attributes: map[string]schema.Attribute{
			"id":     schema.StringAttribute{Computed: true},
			"name":   schema.StringAttribute{Required: true},
			"server": schema.StringAttribute{Optional: true, Computed: true},
			"port":   schema.StringAttribute{Optional: true, Computed: true},
		},
	}
}

func (r *resolverResource) Configure(
	_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*api.ClientWithResponses)
	if !ok || client == nil {
		resp.Diagnostics.AddError(
			"Invalid provider configuration", "The HAProxy resolver received an invalid API client.",
		)
		return
	}
	r.client = client
}

func (m resolverModel) body() api.HaproxyResolver {
	return api.HaproxyResolver{Name: stringPointer(m.Name), Server: stringPointer(m.Server), Port: stringPointer(m.Port)}
}

func stringPointer(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := value.ValueString()
	return &result
}

func stringValue(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func (m *resolverModel) fromAPI(value api.HaproxyResolver) {
	m.ID = stringValue(value.Id)
	m.Name = stringValue(value.Name)
	m.Server = stringValue(value.Server)
	m.Port = stringValue(value.Port)
}

func (r *resolverResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resolverModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.client.CreateHaproxyResolverWithResponse(ctx, data.body())
	if err != nil {
		resp.Diagnostics.AddError("Unable to create HAProxy resolver", err.Error())
		return
	}
	if result.StatusCode() != http.StatusOK || result.JSON200 == nil || result.JSON200.Id == nil ||
		strings.TrimSpace(*result.JSON200.Id) == "" {
		resp.Diagnostics.AddError("Unable to create HAProxy resolver",
			fmt.Sprintf("The API returned HTTP %d or omitted the created resolver ID.", result.StatusCode()))
		return
	}
	data.ID = types.StringValue(*result.JSON200.Id)
	if data.Server.IsUnknown() {
		data.Server = types.StringNull()
	}
	if data.Port.IsUnknown() {
		data.Port = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx); err != nil {
		resp.Diagnostics.AddError("HAProxy resolver created but apply failed", err.Error())
		return
	}
	value, missing, err := r.get(ctx, data.ID.ValueString())
	if err != nil || missing {
		resp.Diagnostics.AddError("Unable to verify HAProxy resolver", readError(err))
		return
	}
	data.fromAPI(*value)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resolverResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resolverModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	value, missing, err := r.get(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read HAProxy resolver", err.Error())
		return
	}
	if missing {
		resp.State.RemoveResource(ctx)
		return
	}
	data.fromAPI(*value)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resolverResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, prior resolverModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.client.UpdateHaproxyResolverWithResponse(ctx, prior.ID.ValueString(), data.body())
	if err != nil {
		resp.Diagnostics.AddError("Unable to update HAProxy resolver", err.Error())
		return
	}
	if result.StatusCode() != http.StatusOK || result.JSON200 == nil {
		resp.Diagnostics.AddError("Unable to update HAProxy resolver",
			fmt.Sprintf("The API returned HTTP %d or invalid JSON.", result.StatusCode()))
		return
	}
	resp.State = req.State
	if err := r.apply(ctx); err != nil {
		resp.Diagnostics.AddError("HAProxy resolver updated but apply failed", err.Error())
		return
	}
	value, missing, err := r.get(ctx, prior.ID.ValueString())
	if err != nil || missing {
		resp.Diagnostics.AddError("Unable to verify HAProxy resolver", readError(err))
		return
	}
	data.fromAPI(*value)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resolverResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resolverModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.client.DeleteHaproxyResolverWithResponse(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete HAProxy resolver", err.Error())
		return
	}
	if result.StatusCode() != http.StatusNotFound &&
		(result.StatusCode() != http.StatusOK || result.JSON200 == nil || result.JSON200.Result == nil ||
			strings.TrimSpace(*result.JSON200.Result) == "") {
		resp.Diagnostics.AddError("Unable to delete HAProxy resolver",
			fmt.Sprintf("The API returned HTTP %d or invalid JSON.", result.StatusCode()))
		return
	}
	if result.StatusCode() == http.StatusOK && failedResult(*result.JSON200.Result) {
		resp.Diagnostics.AddError("Unable to delete HAProxy resolver", "The API reported deletion failure.")
		return
	}
	if err := r.apply(ctx); err != nil {
		resp.Diagnostics.AddError("HAProxy resolver deleted but apply failed", err.Error())
		return
	}
	_, missing, err := r.get(ctx, data.ID.ValueString())
	if err != nil || !missing {
		resp.Diagnostics.AddError("Unable to verify deleted HAProxy resolver", readError(err))
	}
}

func (r *resolverResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func readError(err error) string {
	if err != nil {
		return err.Error()
	}
	return "The API did not return the expected resolver state."
}

func (r *resolverResource) get(ctx context.Context, id string) (*api.HaproxyResolver, bool, error) {
	result, err := r.client.GetHaproxyResolversWithResponse(ctx)
	if err != nil {
		return nil, false, err
	}
	if result.StatusCode() == http.StatusNotFound {
		return nil, true, nil
	}
	if result.StatusCode() != http.StatusOK || result.JSON200 == nil || result.JSON200.Resolvers == nil {
		return nil, false, fmt.Errorf("API returned HTTP %d or omitted the resolver list", result.StatusCode())
	}
	var found *api.HaproxyResolver
	for _, entry := range *result.JSON200.Resolvers {
		if entry.Id == nil || strings.TrimSpace(*entry.Id) == "" {
			return nil, false, fmt.Errorf("API returned a resolver without an ID")
		}
		if *entry.Id == id {
			if found != nil {
				return nil, false, fmt.Errorf("API returned duplicate resolver ID %q", id)
			}
			item := entry
			found = &item
		}
	}
	if found == nil {
		return nil, true, nil
	}
	if found.Name == nil || strings.TrimSpace(*found.Name) == "" {
		return nil, false, fmt.Errorf("API omitted the resolver name")
	}
	return found, false, nil
}

func (r *resolverResource) apply(ctx context.Context) error {
	result, err := r.client.ApplyHaproxyConfigWithResponse(ctx)
	if err != nil {
		return err
	}
	if result.StatusCode() != http.StatusOK || result.JSON200 == nil || result.JSON200.Result == nil ||
		strings.TrimSpace(*result.JSON200.Result) == "" {
		return fmt.Errorf("API returned HTTP %d or an invalid apply response", result.StatusCode())
	}
	if result.JSON200.Errors != nil && strings.TrimSpace(*result.JSON200.Errors) != "" {
		return fmt.Errorf("HAProxy validation failed: %s", *result.JSON200.Errors)
	}
	if failedResult(*result.JSON200.Result) {
		return fmt.Errorf("HAProxy apply reported %q", *result.JSON200.Result)
	}
	return nil
}

func failedResult(value string) bool {
	status := strings.ToLower(strings.TrimSpace(value))
	return status == "error" || status == "failed" || status == "failure"
}
