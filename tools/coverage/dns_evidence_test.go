package main

import (
	"os"
	"testing"
)

func TestDNSOperationEvidence(t *testing.T) {
	t.Chdir("../..")
	spec, err := os.ReadFile("docs/nexus.openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	ops, err := inventory(spec, implemented)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateEvidence(implemented); err != nil {
		t.Fatal(err)
	}
	byID := make(map[string]operation, len(ops))
	for _, op := range ops {
		byID[op.ID] = op
	}
	for _, tt := range []struct {
		id     string
		method string
		path   string
	}{
		{"ApplyHaproxyConfig", "POST", "/ports/haproxy/apply"},
		{"BrowseMDNS", "GET", "/services/mdns/browse"},
		{"CorednsGetEasySetup", "GET", "/services/coredns/easysetup"},
		{"CreateHaproxyResolver", "POST", "/ports/haproxy/resolvers"},
		{"DeleteHaproxyResolver", "DELETE", "/ports/haproxy/resolvers/{id}"},
		{"DiagLookupHostDNS", "GET", "/diag/dns"},
		{"GetCheckIPServices", "GET", "/services/dyndns/checkip"},
		{"GetCorednsConfiguration", "GET", "/services/coredns"},
		{"GetDNSResolverConfig", "GET", "/services/dnsresolver"},
		{"GetDynDNSConfig", "GET", "/services/dyndns"},
		{"GetHaproxyResolverStats", "GET", "/ports/haproxy/resolverstats"},
		{"GetHaproxyResolvers", "GET", "/ports/haproxy/resolvers"},
		{"GetMDNSState", "GET", "/services/mdns/stats"},
		{"GetRFCConfig", "GET", "/services/dyndns/rfc"},
		{"SetMDNSConfig", "POST", "/services/mdns"},
		{"UpdateHaproxyResolver", "POST", "/ports/haproxy/resolvers/{id}"},
	} {
		op, ok := byID[tt.id]
		if !ok || !op.Supported || op.Method != tt.method || op.Path != tt.path ||
			op.Evidence.Test == "" || op.Evidence.Live != "" {
			t.Errorf("%s has unexpected evidence: %#v", tt.id, op)
		}
	}
	for _, id := range []string{
		"AddCheckIPService", "AddDynDnsConfig", "AddRFCConfig", "CorednsEasySetup", "DiagDNSCreateAlias",
		"SetCorednsConfiguration", "SetDNSForwarderConfig", "SetDNSResolverConfig",
	} {
		op, ok := byID[id]
		if !ok || op.Supported {
			t.Errorf("%s should be present but unsupported: %#v", id, op)
		}
	}
}
