package data_readonly

import (
	"context"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func NewFirewallNATNpt() datasource.DataSource {
	return newDataSource("firewall_nat_npt", "Reads configured IPv6 Network Prefix Translation rules.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			result, err := client.FirewallGetNATNptWithResponse(ctx)
			return result, err
		},
	)
}

func NewFirewallNATOneToOne() datasource.DataSource {
	return newDataSource("firewall_nat_one_to_one", "Reads configured one-to-one NAT rules.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			result, err := client.FirewallGetNAT1To1RulesWithResponse(ctx)
			return result, err
		},
	)
}

func NewFirewallNATOutbound() datasource.DataSource {
	return newDataSource("firewall_nat_outbound", "Reads configured outbound NAT rules.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			result, err := client.FirewallGetNATOutWithResponse(ctx)
			return result, err
		},
	)
}

func NewFirewallNATPortForward() datasource.DataSource {
	return newDataSource("firewall_nat_port_forward", "Reads configured firewall port-forward NAT rules.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			result, err := client.FirewallGetNATWithResponse(ctx)
			return result, err
		},
	)
}

func NewFirewallInterfaces() datasource.DataSource {
	return newDataSource("firewall_interfaces", "Reads the interfaces available for firewall rules.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			result, err := client.GetFWInterfacesWithResponse(ctx)
			return result, err
		},
	)
}

func NewFirewallRules() datasource.DataSource {
	return newDataSource("firewall_rules", "Reads firewall rules and related lookup values for an interface.",
		map[string]schema.Attribute{
			"interface": schema.StringAttribute{
				Required:    true,
				Description: "Firewall interface whose rules are read.",
			},
		},
		func(ctx context.Context, client *api.ClientWithResponses, arguments map[string]string) (apiResponse, error) {
			result, err := client.GetFirewallRulesWithResponse(ctx, arguments["interface"])
			return result, err
		},
	)
}

func NewFirewallSchedules() datasource.DataSource {
	return newDataSource("firewall_schedules", "Reads configured firewall schedules.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			result, err := client.FirewallGetSchedulesWithResponse(ctx)
			return result, err
		},
	)
}

func NewFirewallTrafficShaper() datasource.DataSource {
	return newDataSource("firewall_traffic_shaper", "Reads configured firewall traffic shapers.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.FirewallGetTrafficShaperWithResponse(ctx)
		},
	)
}

func NewFirewallALTQ() datasource.DataSource {
	return newDataSource("firewall_altq", "Reads configured ALTQ queues.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.FirewallGetALTQWithResponse(ctx)
		},
	)
}

func NewFirewallLimiters() datasource.DataSource {
	return newDataSource("firewall_limiters", "Reads configured firewall limiters.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.FirewallGetLimiterWithResponse(ctx)
		},
	)
}

func NewInterfaces() datasource.DataSource {
	return newDataSource("interfaces", "Reads basic information about configured network interfaces.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			result, err := client.GetInterfacesWithResponse(ctx)
			return result, err
		},
	)
}

func NewInterfaceAssignments() datasource.DataSource {
	return newDataSource("interface_assignments", "Reads configured interface assignments.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetInterfaceAssignmentsWithResponse(ctx)
		},
	)
}

func NewHostInterfaces() datasource.DataSource {
	return newDataSource("host_interfaces", "Reads the host network interfaces available to pfSense.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetHostInterfacesWithResponse(ctx)
		},
	)
}

func NewInterfaceDescriptors() datasource.DataSource {
	return newDataSource("interface_descriptors", "Reads interface descriptions grouped by interface type.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetInterfaceDescriptorsWithResponse(ctx)
		},
	)
}

func NewBridgeInterfaces() datasource.DataSource {
	return newDataSource("bridge_interfaces", "Reads configured bridge interfaces.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetBridgeInterfacesWithResponse(ctx)
		},
	)
}

func NewGIFInterfaces() datasource.DataSource {
	return newDataSource("gif_interfaces", "Reads configured GIF tunnel interfaces.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetGIFsWithResponse(ctx)
		},
	)
}

func NewGREInterfaces() datasource.DataSource {
	return newDataSource("gre_interfaces", "Reads configured GRE tunnel interfaces.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetGREsWithResponse(ctx)
		},
	)
}

func NewInterfaceGroups() datasource.DataSource {
	return newDataSource("interface_groups", "Reads configured interface groups.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetInterfaceGroupsWithResponse(ctx)
		},
	)
}

func NewLAGGInterfaces() datasource.DataSource {
	return newDataSource("lagg_interfaces", "Reads configured link aggregation interfaces.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetLAGGsWithResponse(ctx)
		},
	)
}

func NewQinQInterfaces() datasource.DataSource {
	return newDataSource("qinq_interfaces", "Reads configured QinQ interfaces.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetQinQsWithResponse(ctx)
		},
	)
}

func NewVLANInterfaces() datasource.DataSource {
	return newDataSource("vlan_interfaces", "Reads configured VLAN interfaces.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetVLANsWithResponse(ctx)
		},
	)
}

func NewVXLANInterfaces() datasource.DataSource {
	return newDataSource("vxlan_interfaces", "Reads configured VXLAN interfaces.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetVXLANsWithResponse(ctx)
		},
	)
}

func NewGateways() datasource.DataSource {
	return newDataSource("gateways", "Reads configured system gateways.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			result, err := client.GetGatewaysWithResponse(ctx)
			return result, err
		},
	)
}

func NewStaticRoutes() datasource.DataSource {
	return newDataSource("static_routes", "Reads configured static routes.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetStaticRoutesWithResponse(ctx)
		},
	)
}
