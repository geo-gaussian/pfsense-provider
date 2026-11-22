package data_readonly

import (
	"context"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func NewDNSForwarder() datasource.DataSource {
	return newSensitiveDataSource("dns_forwarder", "Reads DNS Forwarder settings. The JSON result is sensitive because custom options may contain arbitrary configuration.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetDNSForwarderConfigWithResponse(ctx)
		},
	)
}

func NewIGMPProxies() datasource.DataSource {
	return newDataSource("igmp_proxies", "Reads configured IGMP proxy settings.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetIGMPProxiesWithResponse(ctx)
		},
	)
}

func NewMDNS() datasource.DataSource {
	return newDataSource("mdns", "Reads multicast DNS service settings.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetMDNSConfigWithResponse(ctx)
		},
	)
}

func NewDHCPv4Relay() datasource.DataSource {
	return newDataSource("dhcpv4_relay", "Reads DHCPv4 relay settings.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetDhcpV4RelayConfigWithResponse(ctx)
		},
	)
}

func NewDHCPv6Relay() datasource.DataSource {
	return newDataSource("dhcpv6_relay", "Reads DHCPv6 relay settings.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetDhcpV6RelayConfigWithResponse(ctx)
		},
	)
}

func NewRouterAdvertisements() datasource.DataSource {
	return newDataSource("router_advertisements", "Reads IPv6 router advertisement settings.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetRouterAdvSettingsWithResponse(ctx)
		},
	)
}
