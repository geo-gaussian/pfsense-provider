package firewall_resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func (r *resourceModel) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	attributes := r.def.importAttributes
	if len(attributes) == 0 {
		attributes = []string{r.def.keyAttribute}
	}
	parts := []string{req.ID}
	if len(attributes) > 1 {
		parts = strings.Split(req.ID, "/")
		if len(parts) != len(attributes) {
			resp.Diagnostics.AddError(
				"Invalid import identifier",
				fmt.Sprintf("Expected %d slash-separated values for %q.", len(attributes), r.def.name),
			)
			return
		}
	}
	for index, name := range attributes {
		if strings.TrimSpace(parts[index]) == "" {
			resp.Diagnostics.AddError(
				"Invalid import identifier",
				fmt.Sprintf("The import value for %q cannot be empty.", name),
			)
			return
		}
	}
	schemaValue := r.def.schema(ctx)
	if resp.State.Schema == nil {
		resp.State.Schema = schemaValue
	}
	objectType := schemaValue.Type().TerraformType(ctx).(tftypes.Object)
	stateValues := make(map[string]tftypes.Value, len(objectType.AttributeTypes))
	for name, attributeType := range objectType.AttributeTypes {
		stateValues[name] = tftypes.NewValue(attributeType, nil)
	}
	for index, name := range attributes {
		attributeType, ok := objectType.AttributeTypes[name]
		if !ok {
			resp.Diagnostics.AddError(
				"Invalid import identifier",
				fmt.Sprintf("The firewall resource schema does not contain %q.", name),
			)
			return
		}
		if !attributeType.Is(tftypes.String) {
			resp.Diagnostics.AddError(
				"Invalid import identifier",
				fmt.Sprintf("The import attribute %q must be a string.", name),
			)
			return
		}
		stateValues[name] = tftypes.NewValue(attributeType, parts[index])
	}
	resp.State.Raw = tftypes.NewValue(objectType, stateValues)
}
