package data_readonly

import (
	"context"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func NewDHCPLeases() datasource.DataSource {
	return newSensitiveDataSource("dhcp_leases", "Reads current DHCP leases, including client identifiers and network addresses.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetDHCPLeasesWithResponse(ctx)
		},
	)
}

func NewDNSResolverStatus() datasource.DataSource {
	return newDataSource("dns_resolver_status", "Reads DNS Resolver query and server status statistics.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetDnsResolverStatusWithResponse(ctx)
		},
	)
}

func NewNDPTable() datasource.DataSource {
	return newSensitiveDataSource("ndp_table", "Reads the IPv6 neighbor discovery table, including local device addresses.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetNDPTableWithResponse(ctx)
		},
	)
}

func NewServicesStatus() datasource.DataSource {
	return newDataSource("services_status", "Reads enabled and running status for system services.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetServicesStatusWithResponse(ctx)
		},
	)
}

func NewUPnPConfig() datasource.DataSource {
	return newDataSource("upnp_config", "Reads UPnP and NAT-PMP service settings and access rules.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetUPnPWithResponse(ctx)
		},
	)
}

func NewUPnPStatus() datasource.DataSource {
	return newSensitiveDataSource("upnp_status", "Reads active UPnP mappings, including internal client addresses.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetUPnPStatusWithResponse(ctx)
		},
	)
}

func NewServiceWatchdog() datasource.DataSource {
	return newDataSource("service_watchdog", "Reads the services configured for watchdog monitoring.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetServiceWatchdogServicesWithResponse(ctx)
		},
	)
}
