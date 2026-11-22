package data_readonly

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestReadDataSource(t *testing.T) {
	tests := []struct {
		name       string
		dataSource func() datasource.DataSource
		path       string
		configured map[string]tftypes.Value
		status     int
		body       string
		wantJSON   string
		wantError  bool
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
				w.Header().Set("Content-Type", "application/json")
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
			schema := dataSourceSchema(t, d)

			configValues := make(map[string]tftypes.Value, len(tt.configured)+1)
			for name, value := range tt.configured {
				configValues[name] = value
			}
			configValues["json"] = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
			config := tfsdk.Config{
				Schema: schema,
				Raw:    tftypes.NewValue(schema.Type().TerraformType(context.Background()), configValues),
			}
			resp := datasource.ReadResponse{State: tfsdk.State{Schema: schema}}
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
