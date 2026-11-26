package data_readonly

import "net/http"

var dnsDataSourceCases = []readDataSourceCase{
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
