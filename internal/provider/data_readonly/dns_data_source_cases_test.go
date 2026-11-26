package data_readonly

import (
	"net/http"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var dnsDataSourceCases = []readDataSourceCase{
	{
		name: "DNS lookup", dataSource: NewDNSLookup, path: "/diag/dns", query: "host=example.test",
		configured: map[string]tftypes.Value{"host": tftypes.NewValue(tftypes.String, "example.test")},
		status:     http.StatusOK, body: `{}`, wantJSON: `{}`, sensitive: true,
	},
	{
		name: "DNS lookup missing host", dataSource: NewDNSLookup, path: "/diag/dns",
		configured: map[string]tftypes.Value{"host": tftypes.NewValue(tftypes.String, nil)},
		status:     http.StatusOK, body: `{}`, sensitive: true, wantError: true,
	},
	{
		name: "mDNS browse service", dataSource: NewMDNSBrowse, path: "/services/mdns/browse",
		query:      "service=_http._tcp",
		configured: map[string]tftypes.Value{"service": tftypes.NewValue(tftypes.String, "_http._tcp")},
		status:     http.StatusOK, body: `{"records":[]}`, wantJSON: `{"records":[]}`, sensitive: true,
	},
	{
		name: "mDNS browse all", dataSource: NewMDNSBrowse, path: "/services/mdns/browse",
		configured: map[string]tftypes.Value{"service": tftypes.NewValue(tftypes.String, nil)},
		status:     http.StatusOK, body: `{"records":[]}`, wantJSON: `{"records":[]}`, sensitive: true,
	},
	{
		name: "mDNS browse unknown", dataSource: NewMDNSBrowse, path: "/services/mdns/browse",
		configured: map[string]tftypes.Value{"service": tftypes.NewValue(tftypes.String, tftypes.UnknownValue)},
		status:     http.StatusOK, body: `{}`, sensitive: true, wantError: true,
	},
	{
		name: "DNS resolver", dataSource: NewDNSResolver, path: "/services/dnsresolver",
		status: http.StatusOK, body: `{"config":{"enable":true}}`, wantJSON: `{"config":{"enable":true}}`,
		sensitive: true,
	},
	{
		name: "CoreDNS", dataSource: NewCoreDNS, path: "/services/coredns",
		status: http.StatusOK, body: `{"enabled":true}`, wantJSON: `{"enabled":true}`, sensitive: true,
	},
	{
		name: "CoreDNS easy setup", dataSource: NewCoreDNSEasySetup, path: "/services/coredns/easysetup",
		status: http.StatusOK, body: `{"can_easy_setup":true}`, wantJSON: `{"can_easy_setup":true}`,
	},
	{
		name: "Dynamic DNS", dataSource: NewDynamicDNS, path: "/services/dyndns",
		status: http.StatusOK, body: `{"dyndnses":[]}`, wantJSON: `{"dyndnses":[]}`, sensitive: true,
	},
	{
		name: "CheckIP services", dataSource: NewDynamicDNSCheckIP, path: "/services/dyndns/checkip",
		status: http.StatusOK, body: `{"checkipservice":[]}`, wantJSON: `{"checkipservice":[]}`, sensitive: true,
	},
	{
		name: "RFC DNS updates", dataSource: NewDynamicDNSRFC, path: "/services/dyndns/rfc",
		status: http.StatusOK, body: `{"dnsupdate":[]}`, wantJSON: `{"dnsupdate":[]}`, sensitive: true,
	},
	{
		name: "mDNS statistics", dataSource: NewMDNSStats, path: "/services/mdns/stats",
		status: http.StatusOK, body: `{"enabled":true}`, wantJSON: `{"enabled":true}`, sensitive: true,
	},
	{
		name: "HAProxy resolvers", dataSource: NewHAProxyResolvers, path: "/ports/haproxy/resolvers",
		status: http.StatusOK, body: `{"resolvers":[]}`, wantJSON: `{"resolvers":[]}`, sensitive: true,
	},
	{
		name: "HAProxy resolver statistics", dataSource: NewHAProxyResolverStats, path: "/ports/haproxy/resolverstats",
		status: http.StatusOK, body: `{"lines":[]}`, wantJSON: `{"lines":[]}`, sensitive: true,
	},
	{
		name: "DNS resolver HTTP error", dataSource: NewDNSResolver, path: "/services/dnsresolver",
		status: http.StatusBadRequest, body: `{"message":"failure"}`, sensitive: true, wantError: true,
	},
	{
		name: "Dynamic DNS malformed response", dataSource: NewDynamicDNS, path: "/services/dyndns",
		status: http.StatusOK, body: `{`, sensitive: true, wantError: true,
	},
}
