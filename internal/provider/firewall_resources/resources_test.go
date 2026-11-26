package firewall_resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/provider/resource_firewall_schedule"
	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/provider/resource_firewall_virtual_ip"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestScheduleResourceLifecycle(t *testing.T) {
	ctx := context.Background()
	stored := map[string]any{
		"id": "schedule-1", "name": "weekday", "descr": "office hours",
	}
	reloads := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case req.URL.Path == "/firewall/schedules" && req.Method == http.MethodPost:
			if err := json.NewDecoder(req.Body).Decode(&stored); err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			_, _ = fmt.Fprint(w, `{"msg":"ok"}`)
		case req.URL.Path == "/firewall/schedules" && req.Method == http.MethodGet:
			_, _ = fmt.Fprintf(w, `{"schedules":[%s]}`, marshalTestValue(t, stored))
		case req.URL.Path == "/firewall/schedules/schedule-1" && req.Method == http.MethodPut:
			if err := json.NewDecoder(req.Body).Decode(&stored); err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			_, _ = fmt.Fprint(w, `{"msg":"ok"}`)
		case req.URL.Path == "/firewall/schedules/schedule-1" && req.Method == http.MethodDelete:
			stored = nil
			_, _ = fmt.Fprint(w, `{"msg":"ok"}`)
		case req.URL.Path == "/system/filterreload/reload" && req.Method == http.MethodPost:
			reloads++
			_, _ = fmt.Fprint(w, `{"result":"started"}`)
		default:
			t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	r := NewSchedule()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	var configureResp resource.ConfigureResponse
	configuredResource, ok := r.(resource.ResourceWithConfigure)
	if !ok {
		t.Fatal("resource does not support provider configuration")
	}
	configuredResource.Configure(ctx, resource.ConfigureRequest{ProviderData: client}, &configureResp)
	if configureResp.Diagnostics.HasError() {
		t.Fatal(configureResp.Diagnostics)
	}

	timerangeType := resource_firewall_schedule.TimerangeType{
		ObjectType: types.ObjectType{AttrTypes: resource_firewall_schedule.TimerangeValue{}.AttributeTypes(ctx)},
	}
	model := resource_firewall_schedule.FirewallScheduleModel{
		Id:         types.StringValue("schedule-1"),
		Name:       types.StringValue("weekday"),
		Descr:      types.StringValue("office hours"),
		Schedlabel: types.StringNull(),
		Timerange:  types.ListNull(timerangeType),
	}
	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(ctx, &model); diags.HasError() {
		t.Fatal(diags)
	}
	createResp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatal(createResp.Diagnostics)
	}

	readResp := resource.ReadResponse{State: createResp.State}
	r.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatal(readResp.Diagnostics)
	}

	model.Descr = types.StringValue("updated office hours")
	if diags := plan.Set(ctx, &model); diags.HasError() {
		t.Fatal(diags)
	}
	updateResp := resource.UpdateResponse{State: tfsdk.State{Schema: s}}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: createResp.State}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatal(updateResp.Diagnostics)
	}
	if stored["descr"] != "updated office hours" {
		t.Fatalf("description = %v, want updated value", stored["descr"])
	}

	deleteResp := resource.DeleteResponse{}
	r.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatal(deleteResp.Diagnostics)
	}
	if stored != nil || reloads != 3 {
		t.Fatalf("stored = %#v, reloads = %d", stored, reloads)
	}
}

func TestVirtualIPRetainsMaskedPasswords(t *testing.T) {
	ctx := context.Background()
	const password = "sensitive-password"
	var gotPassword string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case req.URL.Path == "/firewall/virtualip" && req.Method == http.MethodPost:
			var body map[string]any
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			gotPassword, _ = body["password"].(string)
			_, _ = fmt.Fprint(w, `{"id":"vip-1"}`)
		case req.URL.Path == "/firewall/virtualip" && req.Method == http.MethodGet:
			_, _ = fmt.Fprint(
				w,
				`{"virtualips":[{"id":"vip-1","mode":"carp","interface":"wan","subnet":"198.51.100.10/32","descr":"VIP"}]}`,
			)
		case req.URL.Path == "/system/filterreload/reload" && req.Method == http.MethodPost:
			_, _ = fmt.Fprint(w, `{"result":"started"}`)
		default:
			t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	r := NewVirtualIP()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	for _, name := range []string{"password", "password_confirm"} {
		attribute, ok := schemaResp.Schema.Attributes[name].(schema.StringAttribute)
		if !ok || !attribute.Sensitive {
			t.Fatalf("%q is not a sensitive string attribute", name)
		}
	}
	configuredResource := r.(resource.ResourceWithConfigure)
	var configureResp resource.ConfigureResponse
	configuredResource.Configure(ctx, resource.ConfigureRequest{ProviderData: client}, &configureResp)
	if configureResp.Diagnostics.HasError() {
		t.Fatal(configureResp.Diagnostics)
	}

	model := resource_firewall_virtual_ip.FirewallVirtualIpModel{
		Advbase:         types.Int64Null(),
		Advskew:         types.Int64Null(),
		CarpMode:        types.StringNull(),
		CarpPeer:        types.StringNull(),
		Descr:           types.StringValue("VIP"),
		Id:              types.StringNull(),
		Interface:       types.StringValue("wan"),
		Mode:            types.StringValue("carp"),
		Password:        types.StringValue(password),
		PasswordConfirm: types.StringValue(password),
		Subnet:          types.StringValue("198.51.100.10/32"),
		Uniqid:          types.StringNull(),
		Vhid:            types.Int64Null(),
	}
	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := plan.Set(ctx, &model); diags.HasError() {
		t.Fatal(diags)
	}
	createResp := resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatal(createResp.Diagnostics)
	}
	if gotPassword != password {
		t.Fatalf("API password = %q, want configured value", gotPassword)
	}
	var state resource_firewall_virtual_ip.FirewallVirtualIpModel
	if diags := createResp.State.Get(ctx, &state); diags.HasError() {
		t.Fatal(diags)
	}
	if state.Password.ValueString() != password || state.PasswordConfirm.ValueString() != password {
		t.Fatal("masked API response overwrote the sensitive password state")
	}

	readResp := resource.ReadResponse{State: createResp.State}
	r.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatal(readResp.Diagnostics)
	}
	if diags := readResp.State.Get(ctx, &state); diags.HasError() {
		t.Fatal(diags)
	}
	if state.Password.ValueString() != password || state.PasswordConfirm.ValueString() != password {
		t.Fatal("refresh did not retain the sensitive password state")
	}
}

func TestFirewallRuleImportUsesInterfaceAndRuleID(t *testing.T) {
	ctx := context.Background()
	r := NewRule()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	importer, ok := r.(resource.ResourceWithImportState)
	if !ok {
		t.Fatal("resource does not support import")
	}
	resp := resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	importer.ImportState(ctx, resource.ImportStateRequest{ID: "lan/rule-1"}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	values, err := objectValues(resp.State.Raw)
	if err != nil {
		t.Fatal(err)
	}
	if values["interface"] != "lan" || values["id"] != "rule-1" {
		t.Fatalf("import values = %#v", values)
	}
}

func TestFirewallResourceSchemasValidate(t *testing.T) {
	ctx := context.Background()
	resources := []resource.Resource{
		NewNATNpt(),
		NewNATOneToOne(),
		NewNATPortForward(),
		NewRule(),
		NewSchedule(),
		NewALTQ(),
		NewLimiter(),
		NewPFlowExporter(),
		NewVirtualIP(),
	}
	seen := make(map[string]struct{}, len(resources))
	for _, firewallResource := range resources {
		var metadata resource.MetadataResponse
		firewallResource.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "pfsense"}, &metadata)
		if _, exists := seen[metadata.TypeName]; exists {
			t.Fatalf("duplicate resource type name %q", metadata.TypeName)
		}
		seen[metadata.TypeName] = struct{}{}
		var schemaResp resource.SchemaResponse
		firewallResource.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
		if diags := schemaResp.Schema.ValidateImplementation(ctx); diags.HasError() {
			t.Errorf("%s schema is invalid: %v", metadata.TypeName, diags)
		}
	}
}

func marshalTestValue(t *testing.T, value any) []byte {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return body
}
