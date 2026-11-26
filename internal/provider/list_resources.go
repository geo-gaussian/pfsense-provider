package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type configurationList struct {
	managed *identifiedResource
	client  *api.ClientWithResponses
}

type listResponse interface {
	StatusCode() int
	GetBody() []byte
}

func (l *configurationList) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	l.managed.Metadata(ctx, req, resp)
}

func (l *configurationList) ListResourceConfigSchema(_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{}
}

func (l *configurationList) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	l.managed.Configure(ctx, req, resp)
	if resp.Diagnostics.HasError() || req.ProviderData == nil {
		return
	}
	l.client = req.ProviderData.(*api.ClientWithResponses)
}

func (l *configurationList) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	var metadata resource.MetadataResponse
	l.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "pfsense"}, &metadata)
	if l.client == nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unable to list configuration", "The API client is not configured."),
		})
		return
	}
	name := strings.TrimPrefix(metadata.TypeName, "pfsense_")
	var ids []string
	var skipped int
	var err error
	if name == "firewall_rule" {
		ids, skipped, err = l.ruleIdentifiers(ctx)
	} else {
		ids, err = l.identifiers(ctx, name)
	}
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unable to list "+metadata.TypeName, err.Error()),
		})
		return
	}
	stream.Results = func(push func(list.ListResult) bool) {
		if skipped > 0 {
			warning := list.ListResult{}
			warning.Diagnostics.AddWarning("Read-only firewall rules omitted",
				fmt.Sprintf("The API marked %d firewall rules as read-only; they cannot be managed by this resource.", skipped))
			if !push(warning) {
				return
			}
		}
		for _, id := range ids {
			result := req.NewListResult(ctx)
			result.DisplayName = id
			result.Diagnostics.Append(result.Identity.Set(ctx, resourceIdentity{
				Controller: types.StringValue(l.managed.controller),
				ID:         types.StringValue(id),
			})...)
			if req.IncludeResource && !result.Diagnostics.HasError() {
				imported := resource.ImportStateResponse{State: tfsdk.State{
					Schema: req.ResourceSchema,
					Raw:    tftypes.NewValue(req.ResourceSchema.Type().TerraformType(ctx), nil),
				}}
				l.managed.ImportState(ctx, resource.ImportStateRequest{ID: id}, &imported)
				result.Diagnostics.Append(imported.Diagnostics...)
				if !result.Diagnostics.HasError() {
					read := resource.ReadResponse{State: imported.State}
					l.managed.Read(ctx, resource.ReadRequest{State: imported.State}, &read)
					result.Diagnostics.Append(read.Diagnostics...)
					if !result.Diagnostics.HasError() {
						if read.State.Raw.IsNull() {
							result.Diagnostics.AddError("Unable to read discovered resource", "The resource disappeared during discovery.")
						} else {
							identity := &tfsdk.ResourceIdentity{
								Schema: req.ResourceIdentitySchema,
								Raw:    tftypes.NewValue(req.ResourceIdentitySchema.Type().TerraformType(ctx), nil),
							}
							l.managed.setIdentity(ctx, read.State, identity, &result.Diagnostics)
							var actual resourceIdentity
							if !result.Diagnostics.HasError() {
								result.Diagnostics.Append(identity.Get(ctx, &actual)...)
							}
							if !result.Diagnostics.HasError() && actual.ID.ValueString() != id {
								result.Diagnostics.AddError("Inconsistent resource identity", "The resource identity returned by the API does not match the discovered identity.")
							}
							if !result.Diagnostics.HasError() {
								result.Resource.Raw = read.State.Raw
							}
						}
					}
				}
			}
			if !push(result) || result.Diagnostics.HasError() {
				return
			}
		}
	}
}

func (l *configurationList) identifiers(ctx context.Context, name string) ([]string, error) {
	var response listResponse
	var err error
	var path []string
	var itemPath, key string
	switch name {
	case "firewall_alias":
		response, err = l.client.FirewallGetAliasesWithResponse(ctx, nil)
		path, key = []string{"aliases"}, "name"
	case "firewall_nat_npt":
		response, err = l.client.FirewallGetNATNptWithResponse(ctx)
		path, key = []string{"rules"}, "id"
	case "firewall_nat_one_to_one":
		response, err = l.client.FirewallGetNAT1To1RulesWithResponse(ctx)
		path, key = []string{"rules"}, "id"
	case "firewall_nat_port_forward":
		response, err = l.client.FirewallGetNATWithResponse(ctx)
		path, itemPath, key = []string{"rules"}, "rule", "id"
	case "firewall_schedule":
		response, err = l.client.FirewallGetSchedulesWithResponse(ctx)
		path, key = []string{"schedules"}, "id"
	case "firewall_altq":
		response, err = l.client.FirewallGetALTQWithResponse(ctx)
		path, key = []string{"altq"}, "name"
	case "firewall_limiter":
		response, err = l.client.FirewallGetLimiterWithResponse(ctx)
		path, key = []string{"limiter"}, "name"
	case "firewall_pflow_exporter":
		response, err = l.client.FirewallGetPFlowWithResponse(ctx)
		path, key = []string{"exporters"}, "id"
	case "firewall_virtual_ip":
		response, err = l.client.FirewallGetVirtualIPsWithResponse(ctx)
		path, key = []string{"virtualips"}, "id"
	case "wireguard_settings", "wireguard_tunnel", "wireguard_peer":
		response, err = l.client.GetWireGuardWithResponse(ctx)
		switch name {
		case "wireguard_settings":
			path = []string{"config"}
		case "wireguard_tunnel":
			path, key = []string{"tunnels", "item"}, "name"
		case "wireguard_peer":
			path, key = []string{"peers", "item"}, "publickey"
		}
	default:
		return nil, fmt.Errorf("unsupported list resource %q", name)
	}
	items, err := listItems(response, err, path, itemPath)
	if err != nil {
		return nil, err
	}
	if name == "wireguard_settings" {
		if len(items) != 1 {
			return nil, fmt.Errorf("expected one WireGuard settings object, got %d", len(items))
		}
		return []string{"wireguard"}, nil
	}
	return itemIdentifiers(items, key, "")
}

func (l *configurationList) ruleIdentifiers(ctx context.Context) ([]string, int, error) {
	response, err := l.client.GetFWInterfacesWithResponse(ctx)
	root, err := listObject(response, err)
	if err != nil {
		return nil, 0, err
	}
	interfaces, ok := root["interfaces"].([]any)
	if !ok {
		return nil, 0, fmt.Errorf("API omitted the firewall interface list")
	}
	var ids []string
	var skipped int
	seenInterfaces := make(map[string]bool, len(interfaces))
	for _, value := range interfaces {
		iface, ok := value.(string)
		if !ok || iface == "" || strings.Contains(iface, "/") {
			return nil, 0, fmt.Errorf("API returned an invalid firewall interface identifier")
		}
		if seenInterfaces[iface] {
			return nil, 0, fmt.Errorf("API returned duplicate firewall interface %q", iface)
		}
		seenInterfaces[iface] = true
		rules, err := l.client.GetFirewallRulesWithResponse(ctx, iface)
		items, err := listItems(rules, err, []string{"entries"}, "rule")
		if err != nil {
			return nil, 0, fmt.Errorf("interface %q: %w", iface, err)
		}
		managed := items[:0]
		for _, item := range items {
			if value, present := item["readonly"]; present && value != nil {
				readonly, ok := value.(bool)
				if !ok {
					return nil, 0, fmt.Errorf("interface %q: API returned an invalid read-only flag", iface)
				}
				if readonly {
					skipped++
					continue
				}
			}
			managed = append(managed, item)
		}
		identifiers, err := itemIdentifiers(managed, "id", iface+"/")
		if err != nil {
			return nil, 0, fmt.Errorf("interface %q: %w", iface, err)
		}
		ids = append(ids, identifiers...)
	}
	return ids, skipped, nil
}

func listObject(response listResponse, requestErr error) (map[string]any, error) {
	if requestErr != nil {
		return nil, requestErr
	}
	if response == nil {
		return nil, fmt.Errorf("API returned no response")
	}
	if response.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("API returned HTTP %d", response.StatusCode())
	}
	var object map[string]any
	if err := json.Unmarshal(response.GetBody(), &object); err != nil {
		return nil, fmt.Errorf("decode API response: %w", err)
	}
	if object == nil {
		return nil, fmt.Errorf("API returned an empty response")
	}
	return object, nil
}

func listItems(response listResponse, requestErr error, path []string, itemPath string) ([]map[string]any, error) {
	root, err := listObject(response, requestErr)
	if err != nil {
		return nil, err
	}
	var value any = root
	for _, name := range path {
		object, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("API response field %q is not an object", name)
		}
		value = object[name]
	}
	values, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("API omitted or returned an invalid %q list", strings.Join(path, "."))
	}
	items := make([]map[string]any, 0, len(values))
	for _, value := range values {
		item, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("API returned an invalid item in %q", strings.Join(path, "."))
		}
		if itemPath != "" {
			if item[itemPath] == nil && item["separator"] != nil {
				continue
			}
			item, ok = item[itemPath].(map[string]any)
			if !ok {
				return nil, fmt.Errorf("API returned an invalid %q item", itemPath)
			}
		}
		items = append(items, item)
	}
	return items, nil
}

func itemIdentifiers(items []map[string]any, key, prefix string) ([]string, error) {
	ids := make([]string, 0, len(items))
	seen := make(map[string]bool, len(items))
	for _, item := range items {
		value, ok := item[key].(string)
		if !ok || strings.TrimSpace(value) == "" || (prefix != "" && strings.Contains(value, "/")) {
			return nil, fmt.Errorf("API returned an item without a valid %q identifier", key)
		}
		id := prefix + value
		if seen[id] {
			return nil, fmt.Errorf("API returned duplicate resource identifier %q", id)
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids, nil
}
