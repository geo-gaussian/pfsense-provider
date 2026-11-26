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
	parts, err := importParts(req.ID, attributes, r.def.name)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import identifier", err.Error())
		return
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
	if err := setImportValues(objectType, stateValues, attributes, parts); err != nil {
		resp.Diagnostics.AddError("Invalid import identifier", err.Error())
		return
	}
	resp.State.Raw = tftypes.NewValue(objectType, stateValues)
}

func setImportValues(
	objectType tftypes.Object, stateValues map[string]tftypes.Value, attributes, parts []string,
) error {
	for index, name := range attributes {
		attributeType, ok := objectType.AttributeTypes[name]
		if !ok {
			return fmt.Errorf("The firewall resource schema does not contain %q.", name)
		}
		if !attributeType.Is(tftypes.String) {
			return fmt.Errorf("The import attribute %q must be a string.", name)
		}
		stateValues[name] = tftypes.NewValue(attributeType, parts[index])
	}
	return nil
}

func importParts(id string, attributes []string, name string) ([]string, error) {
	parts := []string{id}
	if len(attributes) > 1 {
		parts = strings.Split(id, "/")
		if len(parts) != len(attributes) {
			return nil, fmt.Errorf("Expected %d slash-separated values for %q.", len(attributes), name)
		}
	}
	for index, name := range attributes {
		if strings.TrimSpace(parts[index]) == "" {
			return nil, fmt.Errorf("The import value for %q cannot be empty.", name)
		}
	}
	return parts, nil
}
