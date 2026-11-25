package data_readonly

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestReadDataSource(t *testing.T) {
	tests := []struct {
		name        string
		dataSource  func() datasource.DataSource
		path        string
		configured  map[string]tftypes.Value
		status      int
		body        string
		contentType string
		wantJSON    string
		sensitive   bool
		wantError   bool
	}{
		{
			name:       "NPT rules",
			dataSource: NewFirewallNATNpt,
			path:       "/firewall/nat/npt",
			status:     http.StatusOK,
			body:       `{"rules":[]}`,
			wantJSON:   `{"rules":[]}`,
		},
		{
			name:       "one-to-one NAT rules",
			dataSource: NewFirewallNATOneToOne,
			path:       "/firewall/nat/onetoone",
			status:     http.StatusOK,
			body:       `{"rules":[]}`,
			wantJSON:   `{"rules":[]}`,
		},
		{
			name:       "outbound NAT rules",
			dataSource: NewFirewallNATOutbound,
			path:       "/firewall/nat/outbound",
			status:     http.StatusOK,
			body:       `{"rules":[]}`,
			wantJSON:   `{"rules":[]}`,
		},
		{
			name:       "NAT rules",
			dataSource: NewFirewallNATPortForward,
			path:       "/firewall/nat/port-forward",
			status:     http.StatusOK,
			body:       `{"rules":[]}`,
			wantJSON:   `{"rules":[]}`,
		},
		{
			name:       "firewall interfaces",
			dataSource: NewFirewallInterfaces,
			path:       "/firewall/rules/interface",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "firewall rules",
			dataSource: NewFirewallRules,
			path:       "/firewall/rules/interface/lan",
			configured: map[string]tftypes.Value{"interface": tftypes.NewValue(tftypes.String, "lan")},
			status:     http.StatusOK,
			body:       `{"entries":[]}`,
			wantJSON:   `{"entries":[]}`,
		},
		{
			name:       "firewall rules with unknown interface",
			dataSource: NewFirewallRules,
			path:       "/firewall/rules/interface/lan",
			configured: map[string]tftypes.Value{"interface": tftypes.NewValue(tftypes.String, tftypes.UnknownValue)},
			status:     http.StatusOK,
			body:       `{"entries":[]}`,
			wantError:  true,
		},
		{
			name:       "firewall schedules",
			dataSource: NewFirewallSchedules,
			path:       "/firewall/schedules",
			status:     http.StatusOK,
			body:       `{"schedules":[]}`,
			wantJSON:   `{"schedules":[]}`,
		},
		{
			name:       "traffic shapers",
			dataSource: NewFirewallTrafficShaper,
			path:       "/firewall/traffic-shaper",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "ALTQ",
			dataSource: NewFirewallALTQ,
			path:       "/firewall/traffic-shaper/altq",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "limiters",
			dataSource: NewFirewallLimiters,
			path:       "/firewall/traffic-shaper/limiter",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "packet flow settings",
			dataSource: NewFirewallPFlow,
			path:       "/firewall/pflow",
			status:     http.StatusOK,
			body:       `{"options":{"enable":true},"exporters":[]}`,
			wantJSON:   `{"options":{"enable":true},"exporters":[]}`,
		},
		{
			name:       "virtual IPs",
			dataSource: NewFirewallVirtualIPs,
			path:       "/firewall/virtualip",
			status:     http.StatusOK,
			body:       `{"virtualips":[]}`,
			wantJSON:   `{"virtualips":[]}`,
		},
		{
			name:       "network interfaces",
			dataSource: NewInterfaces,
			path:       "/interfaces",
			status:     http.StatusOK,
			body:       `{"interfaces":[]}`,
			wantJSON:   `{"interfaces":[]}`,
		},
		{
			name:       "interface assignments",
			dataSource: NewInterfaceAssignments,
			path:       "/interfaces/assigned",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "host interfaces",
			dataSource: NewHostInterfaces,
			path:       "/interfaces/by-device",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "interface descriptors",
			dataSource: NewInterfaceDescriptors,
			path:       "/interfaces/descriptors",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "bridge interfaces",
			dataSource: NewBridgeInterfaces,
			path:       "/interfaces/bridge",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "GIF interfaces",
			dataSource: NewGIFInterfaces,
			path:       "/interfaces/gif",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "GRE interfaces",
			dataSource: NewGREInterfaces,
			path:       "/interfaces/gre",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "interface groups",
			dataSource: NewInterfaceGroups,
			path:       "/interfaces/groups",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "LAGG interfaces",
			dataSource: NewLAGGInterfaces,
			path:       "/interfaces/lagg",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "QinQ interfaces",
			dataSource: NewQinQInterfaces,
			path:       "/interfaces/qinq",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "VLAN interfaces",
			dataSource: NewVLANInterfaces,
			path:       "/interfaces/vlan",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "VXLAN interfaces",
			dataSource: NewVXLANInterfaces,
			path:       "/interfaces/vxlan",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "gateways",
			dataSource: NewGateways,
			path:       "/system/gateways",
			status:     http.StatusOK,
			body:       `{"gateways":[]}`,
			wantJSON:   `{"gateways":[]}`,
		},
		{
			name:       "static routes",
			dataSource: NewStaticRoutes,
			path:       "/system/routes/static",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "DNS Forwarder",
			dataSource: NewDNSForwarder,
			path:       "/services/dnsmasq",
			status:     http.StatusOK,
			body:       `{"config":{"custom_options":"YQ=="}}`,
			wantJSON:   `{"config":{"custom_options":"YQ=="}}`,
			sensitive:  true,
		},
		{
			name:       "IGMP proxies",
			dataSource: NewIGMPProxies,
			path:       "/services/igmpproxy",
			status:     http.StatusOK,
			body:       `{"igmpentries":[]}`,
			wantJSON:   `{"igmpentries":[]}`,
		},
		{
			name:       "mDNS",
			dataSource: NewMDNS,
			path:       "/services/mdns",
			status:     http.StatusOK,
			body:       `{"enable":true}`,
			wantJSON:   `{"enable":true}`,
		},
		{
			name:       "DHCPv4 relay",
			dataSource: NewDHCPv4Relay,
			path:       "/services/dhcp-relay/v4",
			status:     http.StatusOK,
			body:       `{"enable":true}`,
			wantJSON:   `{"enable":true}`,
		},
		{
			name:       "DHCPv6 relay",
			dataSource: NewDHCPv6Relay,
			path:       "/services/dhcp-relay/v6",
			status:     http.StatusOK,
			body:       `{"enable":true}`,
			wantJSON:   `{"enable":true}`,
		},
		{
			name:       "router advertisements",
			dataSource: NewRouterAdvertisements,
			path:       "/services/radv",
			status:     http.StatusOK,
			body:       `{"intf_radvs":[]}`,
			wantJSON:   `{"intf_radvs":[]}`,
		},
		{
			name:       "IPsec widget",
			dataSource: NewIPSecWidget,
			path:       "/system/widgets/ipsec",
			status:     http.StatusOK,
			body:       `{"phase1s_active":0}`,
			wantJSON:   `{"phase1s_active":0}`,
		},
		{
			name:       "OpenVPN status",
			dataSource: NewOpenVPNStatus,
			path:       "/vpn/openvpn/status",
			status:     http.StatusOK,
			body:       `{"servers":[],"clients":[]}`,
			wantJSON:   `{"servers":[],"clients":[]}`,
		},
		{
			name:       "WireGuard configuration",
			dataSource: NewWireGuardConfig,
			path:       "/vpn/wg",
			status:     http.StatusOK,
			body:       `{"config":[{"enable":true}],"tunnels":{"item":[]},"peers":{"item":[]}}`,
			wantJSON:   `{"config":[{"enable":true}],"tunnels":{"item":[]},"peers":{"item":[]}}`,
			sensitive:  true,
		},
		{
			name:       "WireGuard tunnel configuration",
			dataSource: NewWireGuardTunnelConf,
			path:       "/vpn/wg/tunnel/wg0/conf",
			configured: map[string]tftypes.Value{"name": tftypes.NewValue(tftypes.String, "wg0")},
			status:     http.StatusOK,
			body:       `{"conf":"[Interface]\nPrivateKey = secret"}`,
			wantJSON:   `{"conf":"[Interface]\nPrivateKey = secret"}`,
			sensitive:  true,
		},
		{
			name:       "WireGuard status without tunnels",
			dataSource: NewWireGuardStatus,
			path:       "/vpn/wg/status",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "WireGuard status HTTP error",
			dataSource: NewWireGuardStatus,
			path:       "/vpn/wg/status",
			status:     http.StatusBadRequest,
			body:       `{"errmsg":"failure"}`,
			wantError:  true,
		},
		{
			name:        "WireGuard status with unexpected content type",
			dataSource:  NewWireGuardStatus,
			path:        "/vpn/wg/status",
			status:      http.StatusOK,
			contentType: "text/plain",
			body:        "not JSON",
			wantError:   true,
		},
		{
			name:       "DHCP leases",
			dataSource: NewDHCPLeases,
			path:       "/services/dhcp/leases",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
			sensitive:  true,
		},
		{
			name:       "DNS Resolver status",
			dataSource: NewDNSResolverStatus,
			path:       "/services/dnsresolver/status",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "NDP table",
			dataSource: NewNDPTable,
			path:       "/services/ndp",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
			sensitive:  true,
		},
		{
			name:       "service status",
			dataSource: NewServicesStatus,
			path:       "/services/status",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "UPnP configuration",
			dataSource: NewUPnPConfig,
			path:       "/services/upnp",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "UPnP status",
			dataSource: NewUPnPStatus,
			path:       "/services/upnp/status",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
			sensitive:  true,
		},
		{
			name:       "service watchdog",
			dataSource: NewServiceWatchdog,
			path:       "/services/watchdog",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "system summary",
			dataSource: NewSystemSummary,
			path:       "/system",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "certificate options",
			dataSource: NewCertificateOptions,
			path:       "/system/certoptions",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "dirty subsystems",
			dataSource: NewDirtySubsystems,
			path:       "/system/config/dirty",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "firmware information",
			dataSource: NewFirmwareInfo,
			path:       "/system/firmware",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "gateway priorities",
			dataSource: NewGatewayPriorities,
			path:       "/system/gateways/group_priorities",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "gateway groups",
			dataSource: NewGatewayGroups,
			path:       "/system/gateways/groups",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "gateway status",
			dataSource: NewGatewaysStatus,
			path:       "/system/gateways/status",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
			sensitive:  true,
		},
		{
			name:       "interface status",
			dataSource: NewInterfacesStatus,
			path:       "/system/interfaces/status",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
			sensitive:  true,
		},
		{
			name:       "monitoring datasets",
			dataSource: NewMonitoringDatasets,
			path:       "/system/monitoring/datasets",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "queue statistics",
			dataSource: NewQueueStats,
			path:       "/system/queue/stats",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "routing dirty state",
			dataSource: NewRoutesDirtyState,
			path:       "/system/routes",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "system status",
			dataSource: NewSystemStatus,
			path:       "/system/status",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
			sensitive:  true,
		},
		{
			name:       "interface statistics",
			dataSource: NewInterfaceStats,
			path:       "/system/status/ifstats",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
			sensitive:  true,
		},
		{
			name:       "interface widgets",
			dataSource: NewInterfaceWidgets,
			path:       "/system/widgets/interfaces",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
			sensitive:  true,
		},
		{
			name:       "interface statistics widget",
			dataSource: NewInterfaceStatisticsWidget,
			path:       "/system/widgets/interfacestats",
			status:     http.StatusOK,
			body:       `{}`,
			wantJSON:   `{}`,
		},
		{
			name:       "HTTP error",
			dataSource: NewFirewallNATPortForward,
			path:       "/firewall/nat/port-forward",
			status:     http.StatusBadRequest,
			body:       `{"errmsg":"failure"}`,
			wantError:  true,
		},
		{
			name:       "invalid JSON response",
			dataSource: NewFirewallNATPortForward,
			path:       "/firewall/nat/port-forward",
			status:     http.StatusOK,
			body:       `{`,
			wantError:  true,
		},
		{
			name:       "null JSON response",
			dataSource: NewFirewallNATPortForward,
			path:       "/firewall/nat/port-forward",
			status:     http.StatusOK,
			body:       `null`,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet || req.URL.Path != tt.path {
					t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				contentType := tt.contentType
				if contentType == "" {
					contentType = "application/json"
				}
				w.Header().Set("Content-Type", contentType)
				w.WriteHeader(tt.status)
				_, _ = fmt.Fprint(w, tt.body)
			}))
			defer server.Close()

			client, err := api.NewClientWithResponses(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			d := tt.dataSource()
			configureDataSource(t, d, client)
			dataSchema := dataSourceSchema(t, d)
			jsonAttribute, ok := dataSchema.Attributes["json"].(schema.StringAttribute)
			if !ok {
				t.Fatal("json attribute is not a string")
			}
			if jsonAttribute.Sensitive != tt.sensitive {
				t.Fatalf("json sensitivity = %t, want %t", jsonAttribute.Sensitive, tt.sensitive)
			}

			configValues := make(map[string]tftypes.Value, len(tt.configured)+1)
			for name, value := range tt.configured {
				configValues[name] = value
			}
			configValues["json"] = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
			config := tfsdk.Config{
				Schema: dataSchema,
				Raw:    tftypes.NewValue(dataSchema.Type().TerraformType(context.Background()), configValues),
			}
			resp := datasource.ReadResponse{State: tfsdk.State{Schema: dataSchema}}
			d.Read(context.Background(), datasource.ReadRequest{Config: config}, &resp)

			if resp.Diagnostics.HasError() != tt.wantError {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
			if tt.wantError {
				return
			}

			var state types.Object
			if diags := resp.State.Get(context.Background(), &state); diags.HasError() {
				t.Fatal(diags)
			}
			stateAttributes := state.Attributes()
			jsonValue, ok := stateAttributes["json"].(types.String)
			if !ok || jsonValue.ValueString() != tt.wantJSON {
				t.Fatalf("json = %#v, want %q", stateAttributes["json"], tt.wantJSON)
			}
			if input, ok := tt.configured["interface"]; ok {
				interfaceValue, exists := stateAttributes["interface"].(types.String)
				expected := "lan"
				if err := input.As(&expected); err != nil {
					t.Fatal(err)
				}
				if !exists || interfaceValue.ValueString() != expected {
					t.Fatalf("interface argument was not retained in state: %#v", stateAttributes["interface"])
				}
			}
		})
	}
}

func TestDataSourceMetadata(t *testing.T) {
	d := NewFirewallNATNpt()
	var resp datasource.MetadataResponse
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "pfsense"}, &resp)
	if resp.TypeName != "pfsense_firewall_nat_npt" {
		t.Fatalf("type name = %q, want pfsense_firewall_nat_npt", resp.TypeName)
	}
}

func TestConfigureDataSourceHandlesMissingAndInvalidClients(t *testing.T) {
	tests := []struct {
		name         string
		providerData any
		wantError    bool
	}{
		{name: "missing client"},
		{name: "unexpected client type", providerData: "not an API client", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configurable := NewFirewallNATNpt().(datasource.DataSourceWithConfigure)
			var resp datasource.ConfigureResponse
			configurable.Configure(
				context.Background(),
				datasource.ConfigureRequest{ProviderData: tt.providerData},
				&resp,
			)
			if resp.Diagnostics.HasError() != tt.wantError {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
		})
	}
}

func TestReadDataSourceRequiresConfiguredClient(t *testing.T) {
	d := NewFirewallNATNpt()
	dataSchema := dataSourceSchema(t, d)
	config := tfsdk.Config{
		Schema: dataSchema,
		Raw: tftypes.NewValue(dataSchema.Type().TerraformType(context.Background()), map[string]tftypes.Value{
			"json": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		}),
	}
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: dataSchema}}
	d.Read(context.Background(), datasource.ReadRequest{Config: config}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error when the API client is not configured")
	}
}

func TestReadDataSourceRejectsInvalidRequestConfig(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	d := NewFirewallNATNpt()
	configureDataSource(t, d, client)
	dataSchema := dataSourceSchema(t, d)
	config := tfsdk.Config{Schema: dataSchema, Raw: tftypes.NewValue(tftypes.String, "invalid")}
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: dataSchema}}
	d.Read(context.Background(), datasource.ReadRequest{Config: config}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error for an invalid request config")
	}
}

func TestReadDataSourceHandlesReadFailures(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		read readFunc
	}{
		{
			name: "read error",
			read: func(context.Context, *api.ClientWithResponses, map[string]string) (apiResponse, error) {
				return nil, errors.New("mock read failure")
			},
		},
		{
			name: "nil response",
			read: func(context.Context, *api.ClientWithResponses, map[string]string) (apiResponse, error) {
				return nil, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newDataSource("test", "test data source", nil, tt.read)
			configureDataSource(t, d, client)
			dataSchema := dataSourceSchema(t, d)
			config := tfsdk.Config{
				Schema: dataSchema,
				Raw: tftypes.NewValue(dataSchema.Type().TerraformType(context.Background()), map[string]tftypes.Value{
					"json": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
				}),
			}
			resp := datasource.ReadResponse{State: tfsdk.State{Schema: dataSchema}}
			d.Read(context.Background(), datasource.ReadRequest{Config: config}, &resp)
			if !resp.Diagnostics.HasError() {
				t.Fatal("expected a read error")
			}
		})
	}
}

func TestSetReadStateReportsObjectConstructionErrors(t *testing.T) {
	d := NewFirewallNATNpt()
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: dataSourceSchema(t, d)}}
	setReadState(context.Background(), map[string]attr.Value{
		"json": types.BoolValue(true),
	}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error for an invalid state value")
	}
}

func TestWireGuardStatusOmitsSecretMaterial(t *testing.T) {
	const responseBody = `{"config":[{"enable":true}],"status":[{"private_key":"tunnel-private","public_key":"tunnel-public","listen_port":"51820","status":"up","transfer_rx":10,"transfer_tx":20,"config":{"privatekey":"nested-private"},"peers":[{"public_key":"peer-public","preshared_key":"peer-secret","endpoint":"peer.example:51820","allowed_ips":"10.0.0.0/24","latest_handshake":"now","transfer_rx":"30","transfer_tx":"40","config":{"presharedkey":"nested-secret"}}]}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.Path != "/vpn/wg/status" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, responseBody)
	}))
	defer server.Close()

	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	d := NewWireGuardStatus()
	configureDataSource(t, d, client)
	schema := dataSourceSchema(t, d)
	configValues := map[string]tftypes.Value{
		"json": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	}
	config := tfsdk.Config{
		Schema: schema,
		Raw:    tftypes.NewValue(schema.Type().TerraformType(context.Background()), configValues),
	}
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: schema}}
	d.Read(context.Background(), datasource.ReadRequest{Config: config}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}

	var state types.Object
	if diags := resp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatal(diags)
	}
	jsonValue, ok := state.Attributes()["json"].(types.String)
	if !ok {
		t.Fatal("json state attribute is not a string")
	}
	got := jsonValue.ValueString()
	for _, secret := range []string{
		"private_key",
		"preshared_key",
		`"config"`,
		"tunnel-private",
		"peer-secret",
		"nested-private",
		"nested-secret",
	} {
		if strings.Contains(got, secret) {
			t.Fatalf("WireGuard output contains secret material %q: %s", secret, got)
		}
	}
	for _, safeValue := range []string{"tunnel-public", "peer-public", "latest_handshake", "transfer_rx"} {
		if !strings.Contains(got, safeValue) {
			t.Fatalf("WireGuard output omitted expected status value %q: %s", safeValue, got)
		}
	}
}

func TestWireGuardStatusHandlesMarshalError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"status":[]}`)
	}))
	defer server.Close()

	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	d := newWireGuardStatusDataSource(func(any) ([]byte, error) {
		return nil, errors.New("mock marshal failure")
	})
	configureDataSource(t, d, client)
	dataSchema := dataSourceSchema(t, d)
	config := tfsdk.Config{
		Schema: dataSchema,
		Raw: tftypes.NewValue(dataSchema.Type().TerraformType(context.Background()), map[string]tftypes.Value{
			"json": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		}),
	}
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: dataSchema}}
	d.Read(context.Background(), datasource.ReadRequest{Config: config}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error when WireGuard JSON encoding fails")
	}
	if !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "mock marshal failure") {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
}

func TestWireGuardStatusHandlesTransportError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			t.Error("HTTP response writer does not support connection hijacking")
			return
		}
		connection, _, err := hijacker.Hijack()
		if err != nil {
			t.Errorf("hijack connection: %v", err)
			return
		}
		_ = connection.Close()
	}))
	defer server.Close()

	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	d := NewWireGuardStatus()
	configureDataSource(t, d, client)
	dataSchema := dataSourceSchema(t, d)
	config := tfsdk.Config{
		Schema: dataSchema,
		Raw: tftypes.NewValue(dataSchema.Type().TerraformType(context.Background()), map[string]tftypes.Value{
			"json": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		}),
	}
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: dataSchema}}
	d.Read(context.Background(), datasource.ReadRequest{Config: config}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error when the API connection fails")
	}
}

func configureDataSource(t *testing.T, d datasource.DataSource, client *api.ClientWithResponses) {
	t.Helper()
	var resp datasource.ConfigureResponse
	configurable, ok := d.(datasource.DataSourceWithConfigure)
	if !ok {
		t.Fatal("data source does not support Configure")
	}
	configurable.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: client}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
}

func dataSourceSchema(t *testing.T, d datasource.DataSource) schema.Schema {
	t.Helper()
	var resp datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return resp.Schema
}
