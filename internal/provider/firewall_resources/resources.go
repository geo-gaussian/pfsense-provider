package firewall_resources

import (
	"context"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/provider/resource_firewall_altq"
	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/provider/resource_firewall_limiter"
	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/provider/resource_firewall_nat_npt"
	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/provider/resource_firewall_nat_one_to_one"
	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/provider/resource_firewall_nat_port_forward"
	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/provider/resource_firewall_pflow_exporter"
	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/provider/resource_firewall_rule"
	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/provider/resource_firewall_schedule"
	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/provider/resource_firewall_virtual_ip"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func NewNATNpt() resource.Resource {
	return newResource(definition{
		name: "firewall_nat_npt", description: "Manages an IPv6 Network Prefix Translation rule.",
		keyAttribute: "id", responsePath: []string{"rules"},
		matchAttributes: []string{"interface", "descr", "disabled", "source", "destination"},
		schema:          resource_firewall_nat_npt.FirewallNatNptResourceSchema,
		create: func(
			ctx context.Context, client *api.ClientWithResponses, values map[string]any, _ string,
		) (response, error) {
			body, err := encodeBody[api.NATNptRule](values)
			if err != nil {
				return nil, err
			}
			return client.FirewallCreateNATNptWithResponse(ctx, body)
		},
		read: func(ctx context.Context, client *api.ClientWithResponses, _ map[string]any) (response, error) {
			return client.FirewallGetNATNptWithResponse(ctx)
		},
		delete: func(ctx context.Context, client *api.ClientWithResponses, _ map[string]any, id string) (response, error) {
			return client.FirewallDeleteNATNptRulesWithResponse(ctx, id)
		},
	})
}

func NewNATOneToOne() resource.Resource {
	return newResource(definition{
		name: "firewall_nat_one_to_one", description: "Manages a one-to-one NAT rule.",
		keyAttribute: "id", responsePath: []string{"rules"},
		matchAttributes: []string{"interface", "descr", "external", "destination", "source", "ipprotocol"},
		schema:          resource_firewall_nat_one_to_one.FirewallNatOneToOneResourceSchema,
		create: func(
			ctx context.Context, client *api.ClientWithResponses, values map[string]any, _ string,
		) (response, error) {
			body, err := encodeBody[api.NAT1To1Rule](values)
			if err != nil {
				return nil, err
			}
			return client.FirewallCreateNAT1To1RuleWithResponse(ctx, body)
		},
		read: func(ctx context.Context, client *api.ClientWithResponses, _ map[string]any) (response, error) {
			return client.FirewallGetNAT1To1RulesWithResponse(ctx)
		},
		delete: func(ctx context.Context, client *api.ClientWithResponses, _ map[string]any, id string) (response, error) {
			return client.FirewallDeleteNAT1To1RulesWithResponse(ctx, id)
		},
	})
}

func NewNATPortForward() resource.Resource {
	return newResource(definition{
		name: "firewall_nat_port_forward", description: "Manages a firewall port-forward NAT rule.",
		keyAttribute: "id", responsePath: []string{"rules"}, itemPath: "rule",
		matchAttributes: []string{"interface", "descr", "protocol", "destination", "source", "target"},
		schema:          resource_firewall_nat_port_forward.FirewallNatPortForwardResourceSchema,
		create: func(
			ctx context.Context, client *api.ClientWithResponses, values map[string]any, _ string,
		) (response, error) {
			body, err := encodeBody[api.NATRule](values)
			if err != nil {
				return nil, err
			}
			return client.FirewallCreateNATWithResponse(ctx, body)
		},
		read: func(ctx context.Context, client *api.ClientWithResponses, _ map[string]any) (response, error) {
			return client.FirewallGetNATWithResponse(ctx)
		},
		update: func(
			ctx context.Context, client *api.ClientWithResponses, values map[string]any, id string,
		) (response, error) {
			body, err := encodeBody[api.NATRule](values)
			if err != nil {
				return nil, err
			}
			return client.FirewallUpdateNATWithResponse(ctx, id, body)
		},
		delete: func(ctx context.Context, client *api.ClientWithResponses, _ map[string]any, id string) (response, error) {
			return client.FirewallDeleteNATWithResponse(ctx, id)
		},
	})
}

func NewRule() resource.Resource {
	return newResource(definition{
		name: "firewall_rule", description: "Manages a firewall filter rule on an interface.",
		keyAttribute: "id", importAttributes: []string{"interface", "id"}, replaceAttributes: []string{"interface"},
		responsePath: []string{"entries"}, itemPath: "rule",
		matchAttributes: []string{"interface", "tracker", "type", "protocol", "descr", "source", "destination"},
		schema: func(ctx context.Context) schema.Schema {
			return requireString(resource_firewall_rule.FirewallRuleResourceSchema(ctx), "interface")
		},
		create: func(
			ctx context.Context, client *api.ClientWithResponses, values map[string]any, _ string,
		) (response, error) {
			body, err := encodeBody[api.FWFilterRule](values)
			if err != nil {
				return nil, err
			}
			iface, err := stringValue(values, "interface")
			if err != nil {
				return nil, err
			}
			return client.AppendFirewallRuleWithResponse(ctx, iface, body)
		},
		read: func(ctx context.Context, client *api.ClientWithResponses, values map[string]any) (response, error) {
			iface, err := stringValue(values, "interface")
			if err != nil {
				return nil, err
			}
			return client.GetFirewallRulesWithResponse(ctx, iface)
		},
		update: func(
			ctx context.Context, client *api.ClientWithResponses, values map[string]any, id string,
		) (response, error) {
			body, err := encodeBody[api.FWFilterRule](values)
			if err != nil {
				return nil, err
			}
			iface, err := stringValue(values, "interface")
			if err != nil {
				return nil, err
			}
			return client.UpdateFirewallRuleWithResponse(ctx, iface, id, body)
		},
		delete: func(
			ctx context.Context, client *api.ClientWithResponses, values map[string]any, id string,
		) (response, error) {
			iface, err := stringValue(values, "interface")
			if err != nil {
				return nil, err
			}
			return client.DeleteFirewallRuleWithResponse(ctx, iface, id)
		},
	})
}

func NewSchedule() resource.Resource {
	return newResource(definition{
		name: "firewall_schedule", description: "Manages a firewall schedule.",
		keyAttribute: "id", responsePath: []string{"schedules"},
		matchAttributes: []string{"id", "name"},
		schema: func(ctx context.Context) schema.Schema {
			return requireString(resource_firewall_schedule.FirewallScheduleResourceSchema(ctx), "name")
		},
		create: func(
			ctx context.Context, client *api.ClientWithResponses, values map[string]any, _ string,
		) (response, error) {
			body, err := encodeBody[api.FWSchedule](values)
			if err != nil {
				return nil, err
			}
			return client.FirewallAddScheduleWithResponse(ctx, body)
		},
		read: func(ctx context.Context, client *api.ClientWithResponses, _ map[string]any) (response, error) {
			return client.FirewallGetSchedulesWithResponse(ctx)
		},
		update: func(
			ctx context.Context, client *api.ClientWithResponses, values map[string]any, id string,
		) (response, error) {
			body, err := encodeBody[api.FWSchedule](values)
			if err != nil {
				return nil, err
			}
			return client.FirewallUpdateScheduleWithResponse(ctx, id, body)
		},
		delete: func(ctx context.Context, client *api.ClientWithResponses, _ map[string]any, id string) (response, error) {
			return client.FirewallDeleteScheduleWithResponse(ctx, id)
		},
	})
}

func NewALTQ() resource.Resource {
	return newResource(definition{
		name: "firewall_altq", description: "Manages an ALTQ root queue and its child queues.",
		keyAttribute: "name", responsePath: []string{"altq"},
		matchAttributes: []string{"name", "if_ident"},
		schema:          resource_firewall_altq.FirewallAltqResourceSchema,
		create: func(
			ctx context.Context, client *api.ClientWithResponses, values map[string]any, _ string,
		) (response, error) {
			body, err := encodeBody[api.ALTQRootQueue](values)
			if err != nil {
				return nil, err
			}
			return client.FirewallAddALTQWithResponse(ctx, body)
		},
		read: func(ctx context.Context, client *api.ClientWithResponses, _ map[string]any) (response, error) {
			return client.FirewallGetALTQWithResponse(ctx)
		},
		update: func(
			ctx context.Context, client *api.ClientWithResponses, values map[string]any, name string,
		) (response, error) {
			body, err := encodeBody[api.ALTQRootQueue](values)
			if err != nil {
				return nil, err
			}
			return client.FirewallUpdateALTQWithResponse(ctx, name, body)
		},
		delete: func(ctx context.Context, client *api.ClientWithResponses, _ map[string]any, name string) (response, error) {
			return client.FirewallDeleteALTQWithResponse(ctx, name)
		},
	})
}

func NewLimiter() resource.Resource {
	return newResource(definition{
		name: "firewall_limiter", description: "Manages a firewall limiter and its queues.",
		keyAttribute: "name", responsePath: []string{"limiter"},
		matchAttributes: []string{"name"},
		schema:          resource_firewall_limiter.FirewallLimiterResourceSchema,
		create: func(
			ctx context.Context, client *api.ClientWithResponses, values map[string]any, _ string,
		) (response, error) {
			body, err := encodeBody[api.Limiter](values)
			if err != nil {
				return nil, err
			}
			return client.FirewallAddLimiterWithResponse(ctx, body)
		},
		read: func(ctx context.Context, client *api.ClientWithResponses, _ map[string]any) (response, error) {
			return client.FirewallGetLimiterWithResponse(ctx)
		},
		update: func(
			ctx context.Context, client *api.ClientWithResponses, values map[string]any, name string,
		) (response, error) {
			body, err := encodeBody[api.Limiter](values)
			if err != nil {
				return nil, err
			}
			return client.FirewallUpdateLimiterWithResponse(ctx, name, body)
		},
		delete: func(ctx context.Context, client *api.ClientWithResponses, _ map[string]any, name string) (response, error) {
			return client.FirewallDeleteLimiterWithResponse(ctx, name)
		},
	})
}

func NewPFlowExporter() resource.Resource {
	return newResource(definition{
		name: "firewall_pflow_exporter", description: "Manages a packet-flow exporter.",
		keyAttribute: "id", responsePath: []string{"exporters"},
		matchAttributes: []string{"id", "src", "srcport", "dst", "dstport", "proto", "domain"},
		schema:          resource_firewall_pflow_exporter.FirewallPflowExporterResourceSchema,
		create: func(
			ctx context.Context, client *api.ClientWithResponses, values map[string]any, _ string,
		) (response, error) {
			body, err := encodeBody[api.PFlowExporter](values)
			if err != nil {
				return nil, err
			}
			return client.FirewallAddPFlowExporterWithResponse(ctx, body)
		},
		read: func(ctx context.Context, client *api.ClientWithResponses, _ map[string]any) (response, error) {
			return client.FirewallGetPFlowWithResponse(ctx)
		},
		update: func(
			ctx context.Context, client *api.ClientWithResponses, values map[string]any, id string,
		) (response, error) {
			body, err := encodeBody[api.PFlowExporter](values)
			if err != nil {
				return nil, err
			}
			return client.FirewallUpdatePFlowExporterWithResponse(ctx, id, body)
		},
		delete: func(ctx context.Context, client *api.ClientWithResponses, _ map[string]any, id string) (response, error) {
			return client.FirewallDeletePFlowExporterWithResponse(ctx, id)
		},
	})
}

func NewVirtualIP() resource.Resource {
	return newResource(definition{
		name: "firewall_virtual_ip", description: "Manages a firewall virtual IP.",
		keyAttribute: "id", responsePath: []string{"virtualips"},
		matchAttributes:     []string{"id", "uniqid", "mode", "interface", "subnet"},
		sensitiveAttributes: []string{"password", "password_confirm"},
		schema:              resource_firewall_virtual_ip.FirewallVirtualIpResourceSchema,
		create: func(
			ctx context.Context, client *api.ClientWithResponses, values map[string]any, _ string,
		) (response, error) {
			body, err := encodeBody[api.VirtualIP](values)
			if err != nil {
				return nil, err
			}
			return client.FirewallSetVirtualIPWithResponse(ctx, body)
		},
		read: func(ctx context.Context, client *api.ClientWithResponses, _ map[string]any) (response, error) {
			return client.FirewallGetVirtualIPsWithResponse(ctx)
		},
		update: func(
			ctx context.Context, client *api.ClientWithResponses, values map[string]any, id string,
		) (response, error) {
			body, err := encodeBody[api.VirtualIP](values)
			if err != nil {
				return nil, err
			}
			return client.FirewallUpdateVirtualIPWithResponse(ctx, id, body)
		},
		delete: func(ctx context.Context, client *api.ClientWithResponses, _ map[string]any, id string) (response, error) {
			return client.FirewallDeleteVirtualIPWithResponse(ctx, id)
		},
	})
}

func requireString(s schema.Schema, name string) schema.Schema {
	attribute, ok := s.Attributes[name].(schema.StringAttribute)
	if !ok {
		return s
	}
	attribute.Required = true
	attribute.Optional = false
	attribute.Computed = false
	s.Attributes[name] = attribute
	return s
}
