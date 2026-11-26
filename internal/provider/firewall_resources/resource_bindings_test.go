package firewall_resources

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type firewallActionBinding struct {
	name     string
	resource resource.Resource
	values   map[string]any
	calls    []string
}

func firewallActionBindings() []firewallActionBinding {
	return []firewallActionBinding{
		{
			name: "NPT NAT", resource: NewNATNpt(),
			calls: []string{"PUT /firewall/nat/npt", "GET /firewall/nat/npt", "DELETE /firewall/nat/npt/rule-1"},
		},
		{
			name: "one-to-one NAT", resource: NewNATOneToOne(),
			calls: []string{"PUT /firewall/nat/onetoone", "GET /firewall/nat/onetoone", "DELETE /firewall/nat/onetoone/rule-1"},
		},
		{
			name: "port forward NAT", resource: NewNATPortForward(),
			calls: []string{
				"PUT /firewall/nat/port-forward", "GET /firewall/nat/port-forward",
				"POST /firewall/nat/port-forward/rule-1", "DELETE /firewall/nat/port-forward/rule-1",
			},
		},
		{
			name: "filter rule", resource: NewRule(), values: map[string]any{"interface": "lan"},
			calls: []string{
				"POST /firewall/rules/interface/lan", "GET /firewall/rules/interface/lan",
				"PATCH /firewall/rules/interface/lan/rule-1", "DELETE /firewall/rules/interface/lan/rule-1",
			},
		},
		{
			name: "schedule", resource: NewSchedule(), values: map[string]any{"name": "weekday"},
			calls: []string{
				"POST /firewall/schedules", "GET /firewall/schedules",
				"PUT /firewall/schedules/rule-1", "DELETE /firewall/schedules/rule-1",
			},
		},
		{
			name: "ALTQ", resource: NewALTQ(),
			calls: []string{
				"PUT /firewall/traffic-shaper/altq", "GET /firewall/traffic-shaper/altq",
				"POST /firewall/traffic-shaper/altq/rule-1", "DELETE /firewall/traffic-shaper/altq/rule-1",
			},
		},
		{
			name: "limiter", resource: NewLimiter(),
			calls: []string{
				"PUT /firewall/traffic-shaper/limiter", "GET /firewall/traffic-shaper/limiter",
				"POST /firewall/traffic-shaper/limiter/rule-1", "DELETE /firewall/traffic-shaper/limiter/rule-1",
			},
		},
		{
			name: "PFlow exporter", resource: NewPFlowExporter(),
			calls: []string{
				"POST /firewall/pflow/exporters", "GET /firewall/pflow",
				"PUT /firewall/pflow/exporters/rule-1", "DELETE /firewall/pflow/exporters/rule-1",
			},
		},
		{
			name: "virtual IP", resource: NewVirtualIP(),
			calls: []string{
				"POST /firewall/virtualip", "GET /firewall/virtualip",
				"PUT /firewall/virtualip/rule-1", "DELETE /firewall/virtualip/rule-1",
			},
		},
	}
}

func TestFirewallResourceAPIActionBindings(t *testing.T) {
	for _, tt := range firewallActionBindings() {
		t.Run(tt.name, func(t *testing.T) {
			remaining := make(map[string]int, len(tt.calls))
			for _, call := range tt.calls {
				remaining[call]++
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				call := req.Method + " " + req.URL.Path
				if remaining[call] == 0 {
					t.Errorf("unexpected API request %s", call)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				remaining[call]--
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, `{}`)
			}))
			defer server.Close()

			client, err := api.NewClientWithResponses(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			model := tt.resource.(*resourceModel)
			values := tt.values
			if values == nil {
				values = map[string]any{}
			}
			if _, err := model.def.create(context.Background(), client, values, ""); err != nil {
				t.Fatalf("create action: %v", err)
			}
			if _, err := model.def.read(context.Background(), client, values); err != nil {
				t.Fatalf("read action: %v", err)
			}
			if model.def.update != nil {
				if _, err := model.def.update(context.Background(), client, values, "rule-1"); err != nil {
					t.Fatalf("update action: %v", err)
				}
			}
			if _, err := model.def.delete(context.Background(), client, values, "rule-1"); err != nil {
				t.Fatalf("delete action: %v", err)
			}
			for call, count := range remaining {
				if count != 0 {
					t.Errorf("API request %q was not made", call)
				}
			}
		})
	}
}

func TestFirewallResourceActionValidationErrors(t *testing.T) {
	ctx := context.Background()
	client, err := api.NewClientWithResponses("http://127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		action action
		values map[string]any
	}{
		{
			name:   "body encoding",
			action: NewNATNpt().(*resourceModel).def.create,
			values: map[string]any{"invalid": make(chan int)},
		},
		{
			name:   "required interface",
			action: NewRule().(*resourceModel).def.create,
			values: map[string]any{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := test.action(ctx, client, test.values, ""); err == nil {
				t.Fatal("expected the action to reject invalid values")
			}
		})
	}
	if _, err := NewRule().(*resourceModel).def.read(ctx, client, nil); err == nil {
		t.Fatal("rule read accepted a missing interface")
	}
	if _, err := NewRule().(*resourceModel).def.update(ctx, client, map[string]any{}, "rule-1"); err == nil {
		t.Fatal("rule update accepted a missing interface")
	}
	if _, err := NewRule().(*resourceModel).def.delete(ctx, client, nil, "rule-1"); err == nil {
		t.Fatal("rule delete accepted a missing interface")
	}

	for _, firewallResource := range []resource.Resource{
		NewNATOneToOne(), NewNATPortForward(), NewRule(), NewSchedule(),
		NewALTQ(), NewLimiter(), NewPFlowExporter(), NewVirtualIP(),
	} {
		model := firewallResource.(*resourceModel)
		invalidBody := map[string]any{"invalid": make(chan int)}
		if _, err := model.def.create(ctx, client, invalidBody, ""); err == nil {
			t.Errorf("%s create accepted a non-serializable body", model.def.name)
		}
		if model.def.update != nil {
			if _, err := model.def.update(ctx, client, invalidBody, "rule-1"); err == nil {
				t.Errorf("%s update accepted a non-serializable body", model.def.name)
			}
		}
	}
}
