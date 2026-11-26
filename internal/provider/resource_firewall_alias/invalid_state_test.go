package resource_firewall_alias

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestLifecycleRejectsInvalidInput(t *testing.T) {
	ctx := context.Background()
	r := &aliasResource{}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	model := FirewallAliasModel{
		Name: types.StringValue("invalid name"), Type: types.StringNull(),
		Address: types.StringNull(), Descr: types.StringNull(), Detail: types.StringNull(),
		Tags: types.StringNull(), Updatefreq: types.StringNull(),
		Targets: types.ListNull(TargetsType{ObjectType: types.ObjectType{AttrTypes: TargetsValue{}.AttributeTypes(ctx)}}),
	}
	plan := tfsdk.Plan{Schema: s}
	state := tfsdk.State{Schema: s}
	if diags := plan.Set(ctx, &model); diags.HasError() {
		t.Fatal(diags)
	}
	if diags := state.Set(ctx, &model); diags.HasError() {
		t.Fatal(diags)
	}
	for _, invalidRaw := range []bool{false, true} {
		t.Run(map[bool]string{false: "invalid name", true: "invalid state type"}[invalidRaw], func(t *testing.T) {
			testPlan, testState := plan, state
			if invalidRaw {
				testPlan.Raw = tftypes.NewValue(tftypes.String, "invalid")
				testState.Raw = tftypes.NewValue(tftypes.String, "invalid")
			}
			create := resource.CreateResponse{State: state}
			r.Create(ctx, resource.CreateRequest{Plan: testPlan}, &create)
			update := resource.UpdateResponse{State: state}
			r.Update(ctx, resource.UpdateRequest{Plan: testPlan, State: state}, &update)
			remove := resource.DeleteResponse{State: testState}
			r.Delete(ctx, resource.DeleteRequest{State: testState}, &remove)
			if !create.Diagnostics.HasError() || !update.Diagnostics.HasError() || !remove.Diagnostics.HasError() {
				t.Fatalf(
					"invalid input accepted: create=%v update=%v delete=%v",
					create.Diagnostics,
					update.Diagnostics,
					remove.Diagnostics,
				)
			}
			if invalidRaw {
				read := resource.ReadResponse{State: testState}
				r.Read(ctx, resource.ReadRequest{State: testState}, &read)
				if !read.Diagnostics.HasError() {
					t.Fatal("invalid read state accepted")
				}
			}
		})
	}
}
