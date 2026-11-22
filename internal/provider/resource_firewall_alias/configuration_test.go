package resource_firewall_alias

import (
	"context"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestConfigurationAndImport(t *testing.T) {
	ctx := context.Background()
	r := New().(*aliasResource)
	var metadata resource.MetadataResponse
	r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "pfsense"}, &metadata)
	if metadata.TypeName != "pfsense_firewall_alias" {
		t.Fatalf("unexpected resource type %q", metadata.TypeName)
	}
	var response resource.ConfigureResponse
	r.Configure(ctx, resource.ConfigureRequest{}, &response)
	if response.Diagnostics.HasError() || r.client != nil {
		t.Fatalf("unexpected configuration: %v", response.Diagnostics)
	}
	response = resource.ConfigureResponse{}
	r.Configure(ctx, resource.ConfigureRequest{ProviderData: "invalid"}, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("accepted incorrect provider data")
	}
	client, err := api.NewClientWithResponses("https://example.test")
	if err != nil {
		t.Fatal(err)
	}
	response = resource.ConfigureResponse{}
	r.Configure(ctx, resource.ConfigureRequest{ProviderData: client}, &response)
	if response.Diagnostics.HasError() || r.client != client {
		t.Fatalf("client was not configured: %v", response.Diagnostics)
	}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	model := FirewallAliasModel{
		Name: types.StringNull(), Address: types.StringNull(), Descr: types.StringNull(),
		Detail: types.StringNull(), Tags: types.StringNull(), Type: types.StringNull(),
		Updatefreq: types.StringNull(),
		Targets:    types.ListNull(TargetsType{ObjectType: types.ObjectType{AttrTypes: TargetsValue{}.AttributeTypes(ctx)}}),
	}
	if diags := state.Set(ctx, &model); diags.HasError() {
		t.Fatal(diags)
	}
	importResp := resource.ImportStateResponse{State: state}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "example"}, &importResp)
	if importResp.Diagnostics.HasError() {
		t.Fatal(importResp.Diagnostics)
	}
	var imported FirewallAliasModel
	if diags := importResp.State.Get(ctx, &imported); diags.HasError() || imported.Name.ValueString() != "example" {
		t.Fatalf("unexpected import state: %v, %#v", diags, imported)
	}
}

func TestInvalidAliasRequestAndResponse(t *testing.T) {
	ctx := context.Background()
	model := FirewallAliasModel{Name: types.StringValue("invalid name")}
	if _, err := model.request(ctx); err == nil {
		t.Fatal("accepted invalid alias name")
	}
	if err := model.fromAPI(ctx, &api.FWAlias{}); err == nil {
		t.Fatal("accepted alias without a name")
	}
}
