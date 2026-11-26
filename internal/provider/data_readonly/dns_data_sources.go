package data_readonly

import (
	"context"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func NewDNSResolver() datasource.DataSource {
	return newSensitiveDataSource("dns_resolver", "Reads DNS Resolver settings, including custom options.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetDNSResolverConfigWithResponse(ctx)
		})
}

func NewCoreDNS() datasource.DataSource {
	return newSensitiveDataSource("coredns", "Reads CoreDNS configuration.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetCorednsConfigurationWithResponse(ctx)
		})
}

func NewCoreDNSEasySetup() datasource.DataSource {
	return newDataSource("coredns_easy_setup", "Reads whether CoreDNS easy setup is required.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.CorednsGetEasySetupWithResponse(ctx)
		})
}

func NewDynamicDNS() datasource.DataSource {
	return newSensitiveDataSource("dynamic_dns", "Reads Dynamic DNS configurations, including credentials.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetDynDNSConfigWithResponse(ctx)
		})
}

func NewDynamicDNSCheckIP() datasource.DataSource {
	return newSensitiveDataSource("dynamic_dns_check_ip", "Reads CheckIP services, including credentials.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetCheckIPServicesWithResponse(ctx)
		})
}

func NewDynamicDNSRFC() datasource.DataSource {
	return newSensitiveDataSource("dynamic_dns_rfc", "Reads RFC DNS update configurations, including key data.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetRFCConfigWithResponse(ctx)
		})
}

func NewMDNSStats() datasource.DataSource {
	return newSensitiveDataSource("mdns_stats", "Reads multicast DNS service statistics.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetMDNSStateWithResponse(ctx)
		})
}

func NewHAProxyResolvers() datasource.DataSource {
	return newSensitiveDataSource("haproxy_resolvers", "Reads HAProxy DNS resolvers.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetHaproxyResolversWithResponse(ctx)
		})
}

func NewHAProxyResolverStats() datasource.DataSource {
	return newSensitiveDataSource("haproxy_resolver_stats", "Reads HAProxy DNS resolver statistics.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetHaproxyResolverStatsWithResponse(ctx)
		})
}
