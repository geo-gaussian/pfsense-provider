package provider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/list"
	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func discoveryResponse(name string) any {
	switch name {
	case "firewall_alias":
		return map[string]any{"aliases": []any{map[string]any{"name": "example"}}}
	case "firewall_nat_npt", "firewall_nat_one_to_one":
		return map[string]any{"rules": []any{map[string]any{"id": "example"}}}
	case "firewall_nat_port_forward":
		return map[string]any{"rules": []any{map[string]any{"rule": map[string]any{"id": "example"}}}}
	case "firewall_rule":
		return map[string]any{"entries": []any{map[string]any{"rule": map[string]any{"id": "example", "interface": "lan"}}}}
	case "firewall_schedule":
		return map[string]any{"schedules": []any{map[string]any{"id": "example", "name": "example"}}}
	case "firewall_altq":
		return map[string]any{
			"altq": []any{
				map[string]any{
					"name":          "example",
					"if_ident":      "lan",
					"scheduler":     "HFSC",
					"bandwidth":     "100",
					"bandwidthtype": "Mb",
				},
			},
		}
	case "firewall_limiter":
		return map[string]any{"limiter": []any{map[string]any{"name": "example"}}}
	case "firewall_pflow_exporter":
		return map[string]any{"exporters": []any{map[string]any{"id": "example"}}}
	case "firewall_virtual_ip":
		return map[string]any{"virtualips": []any{map[string]any{"id": "example"}}}
	case "haproxy_resolver":
		return map[string]any{"resolvers": []any{map[string]any{"id": "example", "name": "example"}}}
	case "mdns":
		return map[string]any{"enable": true}
	default:
		return map[string]any{
			"config":  []any{map[string]any{"enable": true}},
			"tunnels": map[string]any{"item": []any{map[string]any{"name": "example", "publickey": "public-example"}}},
			"peers":   map[string]any{"item": []any{map[string]any{"publickey": "example"}}},
		}
	}
}

func configuredList(t *testing.T, name string, handler http.HandlerFunc) (*configurationList, list.ListRequest) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	p := New("test")().(frameworkprovider.ProviderWithListResources)
	for _, factory := range p.ListResources(ctx) {
		l := factory().(*configurationList)
		var metadata resource.MetadataResponse
		l.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "pfsense"}, &metadata)
		if metadata.TypeName != "pfsense_"+name {
			continue
		}
		var configured resource.ConfigureResponse
		l.Configure(ctx, resource.ConfigureRequest{ProviderData: client}, &configured)
		if configured.Diagnostics.HasError() {
			t.Fatal(configured.Diagnostics)
		}
		var schema resource.SchemaResponse
		l.managed.Schema(ctx, resource.SchemaRequest{}, &schema)
		var identity resource.IdentitySchemaResponse
		l.managed.IdentitySchema(ctx, resource.IdentitySchemaRequest{}, &identity)
		return l, list.ListRequest{ResourceSchema: schema.Schema, ResourceIdentitySchema: identity.IdentitySchema}
	}
	t.Fatalf("list resource %q is not registered", name)
	return nil, list.ListRequest{}
}

func TestListSupportedConfiguration(t *testing.T) {
	for _, name := range []string{
		"firewall_alias", "firewall_nat_npt", "firewall_nat_one_to_one", "firewall_nat_port_forward",
		"firewall_rule", "firewall_schedule", "firewall_altq", "firewall_limiter",
		"firewall_pflow_exporter", "firewall_virtual_ip", "wireguard_settings", "wireguard_tunnel", "wireguard_peer",
		"haproxy_resolver",
		"mdns",
	} {
		t.Run(name, func(t *testing.T) {
			var paths []string
			l, req := configuredList(t, name, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("discovery attempted %s", r.Method)
				}
				paths = append(paths, r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				body := discoveryResponse(name)
				if r.URL.Path == "/aliases/example" {
					body = map[string]any{"name": "example"}
				} else if r.URL.Path == "/firewall/rules/interface" {
					body = map[string]any{"interfaces": []string{"lan"}}
				}
				if err := json.NewEncoder(w).Encode(body); err != nil {
					t.Error(err)
				}
			})
			var schema list.ListResourceSchemaResponse
			l.ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &schema)
			if schema.Diagnostics.HasError() || len(schema.Schema.Attributes) != 0 {
				t.Fatalf("unexpected discovery config schema: %#v", schema)
			}
			for _, include := range []bool{false, true} {
				req.IncludeResource = include
				var stream list.ListResultsStream
				l.List(context.Background(), req, &stream)
				var results []list.ListResult
				for result := range stream.Results {
					results = append(results, result)
				}
				if len(results) != 1 || results[0].Diagnostics.HasError() {
					t.Fatalf("discovery results: %#v", results)
				}
				id := "example"
				if name == "wireguard_settings" {
					id = "wireguard"
				} else if name == "mdns" {
					id = "mdns"
				} else if name == "firewall_rule" {
					id = "lan/example"
				}
				var identity resourceIdentity
				if diags := results[0].Identity.Get(context.Background(), &identity); diags.HasError() {
					t.Fatal(diags)
				}
				if identity.ID.ValueString() != id || identity.Controller.ValueString() != l.managed.controller ||
					results[0].DisplayName != id {
					t.Fatalf("unexpected discovered identity: %#v", identity)
				}
				if include && results[0].Resource.Raw.IsNull() {
					t.Fatal("full discovery omitted resource attributes")
				}
			}
			if name == "firewall_alias" && !reflect.DeepEqual(paths, []string{"/aliases", "/aliases", "/aliases/example"}) {
				t.Fatalf("identity-only discovery fetched alias details: %v", paths)
			}
		})
	}
}

type discoveryBody []byte

func (b discoveryBody) StatusCode() int { return http.StatusOK }
func (b discoveryBody) GetBody() []byte { return b }

func TestDiscoveryResponseValidation(t *testing.T) {
	requestErr := errors.New("request failed")
	if _, err := listObject(nil, requestErr); !errors.Is(err, requestErr) {
		t.Fatalf("lost request error: %v", err)
	}
	if _, err := listObject(nil, nil); err == nil || !strings.Contains(err.Error(), "no response") {
		t.Fatalf("accepted missing response: %v", err)
	}
	for _, tt := range []struct {
		name string
		body any
		path []string
		item string
		want string
	}{
		{name: "null response", body: nil, want: "empty response"},
		{name: "invalid root", body: []any{}, want: "decode API response"},
		{
			name: "invalid nested collection", body: map[string]any{"tunnels": []any{}},
			path: []string{"tunnels", "item"}, want: "not an object",
		},
		{name: "non-object item", body: map[string]any{"rules": []any{nil}}, path: []string{"rules"}, want: "invalid item"},
		{
			name: "missing rule", body: map[string]any{"rules": []any{map[string]any{}}},
			path: []string{"rules"}, item: "rule", want: "invalid \"rule\"",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.body)
			if err != nil {
				t.Fatal(err)
			}
			_, err = listItems(discoveryBody(body), nil, tt.path, tt.item)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestRuleDiscoveryFailures(t *testing.T) {
	for _, tt := range []struct {
		name       string
		interfaces any
		rules      any
		status     int
		want       string
	}{
		{name: "interface request failure", status: http.StatusForbidden, want: "HTTP 403"},
		{name: "omitted interfaces", interfaces: map[string]any{}, want: "interface list"},
		{
			name: "invalid interface", interfaces: map[string]any{"interfaces": []string{"lan/wan"}},
			want: "invalid firewall interface",
		},
		{
			name: "duplicate interface", interfaces: map[string]any{"interfaces": []string{"lan", "lan"}},
			rules: map[string]any{"entries": []any{}}, want: "duplicate firewall interface",
		},
		{
			name: "missing entries", interfaces: map[string]any{"interfaces": []string{"lan"}},
			rules: map[string]any{}, want: "interface \"lan\"",
		},
		{
			name: "invalid read-only flag", interfaces: map[string]any{"interfaces": []string{"lan"}},
			rules: map[string]any{"entries": []any{map[string]any{"rule": map[string]any{"readonly": "invalid"}}}},
			want:  "invalid read-only flag",
		},
		{
			name: "missing rule identifier", interfaces: map[string]any{"interfaces": []string{"lan"}},
			rules: map[string]any{"entries": []any{map[string]any{"rule": map[string]any{}}}}, want: "identifier",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			l, _ := configuredList(t, "firewall_rule", func(w http.ResponseWriter, r *http.Request) {
				// Raw response bodies must also be validated when the API omits its JSON content type.
				w.Header().Set("Content-Type", "text/plain")
				body := tt.rules
				if r.URL.Path == "/firewall/rules/interface" {
					body = tt.interfaces
					if tt.status != 0 {
						w.WriteHeader(tt.status)
					}
				}

				if err := json.NewEncoder(w).Encode(body); err != nil {
					t.Error(err)
				}
			})
			_, _, err := l.ruleIdentifiers(context.Background())
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestListReportsDiscoveryFailures(t *testing.T) {
	for _, tt := range []struct {
		name         string
		omitClient   bool
		listStatus   int
		detailStatus int
		want         string
	}{
		{name: "missing configuration", omitClient: true, want: "not configured"},
		{name: "API failure", listStatus: http.StatusForbidden, want: "HTTP 403"},
		{name: "deleted during discovery", detailStatus: http.StatusNotFound, want: "disappeared during discovery"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			l, req := configuredList(t, "firewall_alias", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				body := discoveryResponse("firewall_alias")
				status := tt.listStatus
				if r.URL.Path == "/aliases/example" {
					status = tt.detailStatus
					body = map[string]any{"name": "example"}
				}
				if status != 0 {
					w.WriteHeader(status)
				}
				if err := json.NewEncoder(w).Encode(body); err != nil {
					t.Error(err)
				}
			})
			if tt.omitClient {
				l.client = nil
			}
			req.IncludeResource = true
			var stream list.ListResultsStream
			l.List(context.Background(), req, &stream)
			var count int
			for result := range stream.Results {
				count++
				if !result.Diagnostics.HasError() || !strings.Contains(result.Diagnostics.Errors()[0].Detail(), tt.want) {
					t.Fatalf("diagnostics = %v, want %q", result.Diagnostics, tt.want)
				}
			}
			if count != 1 {
				t.Fatalf("received %d results for failed discovery", count)
			}
		})
	}
}

func TestListStopsWhenConsumerStops(t *testing.T) {
	for _, readonly := range []bool{false, true} {
		l, req := configuredList(t, "firewall_rule", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			body := any(map[string]any{"interfaces": []string{"lan"}})
			if r.URL.Path != "/firewall/rules/interface" {
				body = map[string]any{"entries": []any{
					map[string]any{"rule": map[string]any{"id": "first", "readonly": readonly}},
					map[string]any{"rule": map[string]any{"id": "second"}},
				}}
			}
			if err := json.NewEncoder(w).Encode(body); err != nil {
				t.Error(err)
			}
		})
		var stream list.ListResultsStream
		l.List(context.Background(), req, &stream)
		var count int
		stream.Results(func(result list.ListResult) bool {
			count++
			if result.Diagnostics.HasError() || (readonly && len(result.Diagnostics.Warnings()) != 1) ||
				(!readonly && result.DisplayName != "lan/first") {
				t.Fatalf("unexpected first result: %#v", result)
			}
			return false
		})
		if count != 1 {
			t.Fatalf("stream ignored cancellation: %d results", count)
		}
	}
}

func TestListRejectsInvalidConfiguration(t *testing.T) {
	l, _ := configuredList(t, "firewall_alias", func(http.ResponseWriter, *http.Request) {
		t.Error("configuration should not contact the API")
	})
	for _, data := range []any{nil, "invalid"} {
		var resp resource.ConfigureResponse
		l.Configure(context.Background(), resource.ConfigureRequest{ProviderData: data}, &resp)
		if resp.Diagnostics.HasError() != (data != nil) {
			t.Fatalf("unexpected configuration diagnostics: %v", resp.Diagnostics)
		}
	}
}

func TestListRejectsUnsupportedTypeAndAmbiguousSettings(t *testing.T) {
	l, _ := configuredList(t, "wireguard_settings", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"config": []any{}}); err != nil {
			t.Error(err)
		}
	})
	for _, tt := range []struct {
		name string
		want string
	}{
		{name: "unsupported", want: "unsupported list resource"},
		{name: "wireguard_settings", want: "expected one WireGuard settings object"},
	} {
		_, err := l.identifiers(context.Background(), tt.name)
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Fatalf("error = %v, want %q", err, tt.want)
		}
	}
}
