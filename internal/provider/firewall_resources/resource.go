package firewall_resources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/numberplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type response interface {
	StatusCode() int
	GetBody() []byte
}

type action func(context.Context, *api.ClientWithResponses, map[string]any, string) (response, error)
type readAction func(context.Context, *api.ClientWithResponses, map[string]any) (response, error)

type terraformValue interface {
	IsKnown() bool
	IsNull() bool
	Type() tftypes.Type
	As(any) error
}

type definition struct {
	name                string
	description         string
	keyAttribute        string
	importAttributes    []string
	replaceAttributes   []string
	responsePath        []string
	itemPath            string
	matchAttributes     []string
	sensitiveAttributes []string
	schema              func(context.Context) schema.Schema
	create              action
	read                readAction
	update              action
	delete              action
}

type resourceModel struct {
	client *api.ClientWithResponses
	def    definition
}

var errItemNotFound = errors.New("firewall configuration not found")

var _ resource.Resource = (*resourceModel)(nil)
var _ resource.ResourceWithConfigure = (*resourceModel)(nil)
var _ resource.ResourceWithImportState = (*resourceModel)(nil)

func newResource(def definition) resource.Resource {
	return &resourceModel{def: def}
}

func (r *resourceModel) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.def.name
}

func (r *resourceModel) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.def.schema(ctx)
	resp.Schema.Description = r.def.description
	if key, ok := resp.Schema.Attributes[r.def.keyAttribute]; ok {
		resp.Schema.Attributes[r.def.keyAttribute] = withRequiresReplace(key)
	}
	if r.def.update == nil {
		for name, attribute := range resp.Schema.Attributes {
			if name == r.def.keyAttribute {
				continue
			}
			resp.Schema.Attributes[name] = withRequiresReplace(attribute)
		}
	}
	for _, name := range r.def.replaceAttributes {
		if attribute, ok := resp.Schema.Attributes[name]; ok {
			resp.Schema.Attributes[name] = withRequiresReplace(attribute)
		}
	}
	for _, name := range r.def.sensitiveAttributes {
		if attribute, ok := resp.Schema.Attributes[name].(schema.StringAttribute); ok {
			attribute.Sensitive = true
			resp.Schema.Attributes[name] = attribute
		}
	}
}

func withRequiresReplace(attribute schema.Attribute) schema.Attribute {
	switch value := attribute.(type) {
	case schema.StringAttribute:
		if value.Optional || value.Required {
			value.PlanModifiers = append(value.PlanModifiers, stringplanmodifier.RequiresReplace())
		}
		return value
	case schema.BoolAttribute:
		if value.Optional || value.Required {
			value.PlanModifiers = append(value.PlanModifiers, boolplanmodifier.RequiresReplace())
		}
		return value
	case schema.Int64Attribute:
		if value.Optional || value.Required {
			value.PlanModifiers = append(value.PlanModifiers, int64planmodifier.RequiresReplace())
		}
		return value
	case schema.Float64Attribute:
		if value.Optional || value.Required {
			value.PlanModifiers = append(value.PlanModifiers, float64planmodifier.RequiresReplace())
		}
		return value
	case schema.NumberAttribute:
		if value.Optional || value.Required {
			value.PlanModifiers = append(value.PlanModifiers, numberplanmodifier.RequiresReplace())
		}
		return value
	case schema.ListAttribute:
		if value.Optional || value.Required {
			value.PlanModifiers = append(value.PlanModifiers, listplanmodifier.RequiresReplace())
		}
		return value
	case schema.ListNestedAttribute:
		if value.Optional || value.Required {
			value.PlanModifiers = append(value.PlanModifiers, listplanmodifier.RequiresReplace())
		}
		return value
	case schema.SetAttribute:
		if value.Optional || value.Required {
			value.PlanModifiers = append(value.PlanModifiers, setplanmodifier.RequiresReplace())
		}
		return value
	case schema.SetNestedAttribute:
		if value.Optional || value.Required {
			value.PlanModifiers = append(value.PlanModifiers, setplanmodifier.RequiresReplace())
		}
		return value
	case schema.MapAttribute:
		if value.Optional || value.Required {
			value.PlanModifiers = append(value.PlanModifiers, mapplanmodifier.RequiresReplace())
		}
		return value
	case schema.MapNestedAttribute:
		if value.Optional || value.Required {
			value.PlanModifiers = append(value.PlanModifiers, mapplanmodifier.RequiresReplace())
		}
		return value
	case schema.SingleNestedAttribute:
		if value.Optional || value.Required {
			value.PlanModifiers = append(value.PlanModifiers, objectplanmodifier.RequiresReplace())
		}
		return value
	default:
		return attribute
	}
}

func (r *resourceModel) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*api.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError("Invalid provider configuration", "The firewall resource received an unexpected API client.")
		return
	}
	r.client = client
}

func (r *resourceModel) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.configured(&resp.Diagnostics) {
		return
	}
	plan, err := objectValues(req.Plan.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Invalid firewall configuration", err.Error())
		return
	}
	result, err := r.def.create(ctx, r.client, plan, "")
	if err != nil {
		resp.Diagnostics.AddError("Unable to create firewall configuration", err.Error())
		return
	}
	if err := validateResponse(result); err != nil {
		resp.Diagnostics.AddError("Unable to create firewall configuration", err.Error())
		return
	}
	if created, err := decodeObject(result.GetBody()); err == nil {
		if key, ok := created[r.def.keyAttribute]; ok && key != nil && valueString(plan[r.def.keyAttribute]) == "" {
			plan[r.def.keyAttribute] = key
		}
	}

	item, err := r.readItem(ctx, plan, true)
	if err != nil {
		resp.Diagnostics.AddError("Unable to verify created firewall configuration", err.Error())
		return
	}
	if diags := r.setState(ctx, &resp.State, item, plan); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	if err := r.reload(ctx); err != nil {
		resp.Diagnostics.AddError("Firewall configuration created but reload failed", err.Error())
		return
	}
	item, err = r.readItem(ctx, item, false)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read created firewall configuration", err.Error())
		return
	}
	resp.Diagnostics.Append(r.setState(ctx, &resp.State, item, plan)...)
}

func (r *resourceModel) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.configured(&resp.Diagnostics) {
		return
	}
	state, err := objectValues(req.State.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read firewall configuration", err.Error())
		return
	}
	item, err := r.readItem(ctx, state, false)
	if err != nil {
		if errors.Is(err, errItemNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read firewall configuration", err.Error())
		return
	}
	resp.Diagnostics.Append(r.setState(ctx, &resp.State, item, state)...)
}

func (r *resourceModel) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.configured(&resp.Diagnostics) {
		return
	}
	if r.def.update == nil {
		resp.Diagnostics.AddError("Unsupported firewall update", "This firewall API does not support in-place updates; Terraform must replace the resource.")
		return
	}
	prior, err := objectValues(req.State.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Invalid firewall state", err.Error())
		return
	}
	plan, err := objectValues(req.Plan.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Invalid firewall configuration", err.Error())
		return
	}
	key, err := resourceKey(prior, r.def.keyAttribute)
	if err != nil {
		resp.Diagnostics.AddError("Invalid firewall state", err.Error())
		return
	}
	result, err := r.def.update(ctx, r.client, plan, key)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update firewall configuration", err.Error())
		return
	}
	if err := validateResponse(result); err != nil {
		resp.Diagnostics.AddError("Unable to update firewall configuration", err.Error())
		return
	}
	item, err := r.readItem(ctx, prior, false)
	if err != nil {
		resp.Diagnostics.AddError("Unable to verify updated firewall configuration", err.Error())
		return
	}
	if diags := r.setState(ctx, &resp.State, item, plan); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	if err := r.reload(ctx); err != nil {
		resp.Diagnostics.AddError("Firewall configuration updated but reload failed", err.Error())
		return
	}
	item, err = r.readItem(ctx, item, false)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read updated firewall configuration", err.Error())
		return
	}
	resp.Diagnostics.Append(r.setState(ctx, &resp.State, item, plan)...)
}

func (r *resourceModel) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.configured(&resp.Diagnostics) {
		return
	}
	state, err := objectValues(req.State.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Invalid firewall state", err.Error())
		return
	}
	key, err := resourceKey(state, r.def.keyAttribute)
	if err != nil {
		resp.Diagnostics.AddError("Invalid firewall state", err.Error())
		return
	}
	result, err := r.def.delete(ctx, r.client, state, key)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete firewall configuration", err.Error())
		return
	}
	if result == nil || (result.StatusCode() != http.StatusOK && result.StatusCode() != http.StatusNotFound) {
		resp.Diagnostics.AddError("Unable to delete firewall configuration", fmt.Sprintf("The API returned HTTP %d.", statusCode(result)))
		return
	}
	if err := r.reload(ctx); err != nil {
		resp.Diagnostics.AddError("Firewall configuration deleted but reload failed", err.Error())
		return
	}
	if _, err := r.readItem(ctx, state, false); err == nil {
		resp.Diagnostics.AddError("Unable to verify deleted firewall configuration", "The configuration is still present after the delete request.")
	} else if !errors.Is(err, errItemNotFound) {
		resp.Diagnostics.AddError("Unable to verify deleted firewall configuration", err.Error())
	}
}

func (r *resourceModel) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
			resp.Diagnostics.AddError("Invalid import identifier", fmt.Sprintf("The import value for %q cannot be empty.", name))
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
			resp.Diagnostics.AddError("Invalid import identifier", fmt.Sprintf("The firewall resource schema does not contain %q.", name))
			return
		}
		if !attributeType.Is(tftypes.String) {
			resp.Diagnostics.AddError("Invalid import identifier", fmt.Sprintf("The import attribute %q must be a string.", name))
			return
		}
		stateValues[name] = tftypes.NewValue(attributeType, parts[index])
	}
	resp.State.Raw = tftypes.NewValue(objectType, stateValues)
}

func (r *resourceModel) configured(diags interface{ AddError(string, string) }) bool {
	if r.client == nil {
		diags.AddError("Unable to manage firewall configuration", "The API client is not configured.")
		return false
	}
	return true
}

func (r *resourceModel) reload(ctx context.Context) error {
	result, err := r.client.DoFilterReloadWithResponse(ctx)
	if err != nil {
		return fmt.Errorf("initiate firewall reload: %w", err)
	}
	if result.StatusCode() != http.StatusOK || result.JSON200 == nil {
		return fmt.Errorf("initiate firewall reload: HTTP %d or missing JSON response", result.StatusCode())
	}
	return nil
}

func (r *resourceModel) readItem(ctx context.Context, selector map[string]any, match bool) (map[string]any, error) {
	result, err := r.def.read(ctx, r.client, selector)
	if err != nil {
		return nil, fmt.Errorf("read firewall configuration: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("read firewall configuration: missing API response")
	}
	if result.StatusCode() == http.StatusNotFound {
		return nil, errItemNotFound
	}
	if result.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("read firewall configuration: HTTP %d", statusCode(result))
	}
	root, err := decodeObject(result.GetBody())
	if err != nil {
		return nil, fmt.Errorf("decode firewall configuration: %w", err)
	}
	items, err := collection(root, r.def.responsePath, r.def.itemPath)
	if err != nil {
		return nil, err
	}
	targetKey, keyErr := resourceKey(selector, r.def.keyAttribute)
	if keyErr == nil {
		for _, item := range items {
			if fmt.Sprint(item[r.def.keyAttribute]) == targetKey {
				return item, nil
			}
		}
	}
	if !match {
		return nil, fmt.Errorf("%w: %q", errItemNotFound, valueString(selector[r.def.keyAttribute]))
	}

	var matches []map[string]any
	for _, item := range items {
		if matchesAttributes(item, selector, r.def.matchAttributes) {
			matches = append(matches, item)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("firewall configuration matches %d API entries; set its %q field explicitly to identify it", len(matches), r.def.keyAttribute)
	}
	return nil, fmt.Errorf("created firewall configuration was not present in the API response")
}

func (r *resourceModel) setState(ctx context.Context, state *tfsdk.State, item, fallback map[string]any) diag.Diagnostics {
	for _, name := range r.def.sensitiveAttributes {
		current, currentOK := item[name]
		prior, priorOK := fallback[name]
		if priorOK && prior != nil && (!currentOK || current == nil || current == "") {
			item[name] = prior
		}
	}
	completed := completeMap(r.def.schema(ctx).Attributes, item, fallback)
	body, err := json.Marshal(completed)
	if err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Unable to set firewall state", err.Error())}
	}
	if state.Schema == nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Unable to set firewall state", "The Terraform resource schema is unavailable.")}
	}
	raw, err := tftypes.ValueFromJSON(body, state.Schema.Type().TerraformType(ctx))
	if err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Unable to set firewall state", err.Error())}
	}
	state.Raw = raw
	return nil
}

func validateResponse(result response) error {
	if result == nil || result.StatusCode() != http.StatusOK {
		return fmt.Errorf("the API returned HTTP %d", statusCode(result))
	}
	body, err := decodeObject(result.GetBody())
	if err != nil {
		return fmt.Errorf("the API returned an invalid JSON response: %w", err)
	}
	if message, ok := body["errmsg"].(string); ok && message != "" {
		return fmt.Errorf("the API reported an error: %s", message)
	}
	if message, ok := body["msg"].(string); ok && message == "" {
		return fmt.Errorf("the API returned an empty result message")
	}
	if status, ok := body["status"].(string); ok &&
		(strings.EqualFold(status, "error") || strings.EqualFold(status, "failed")) {
		return fmt.Errorf("the API reported status %q", status)
	}
	return nil
}

func statusCode(result response) int {
	if result == nil {
		return 0
	}
	return result.StatusCode()
}

func objectValues(value tftypes.Value) (map[string]any, error) {
	values := make(map[string]tftypes.Value)
	if err := value.As(&values); err != nil {
		return nil, err
	}
	result := make(map[string]any, len(values))
	for key, value := range values {
		converted, include, err := valueToGo(value)
		if err != nil {
			return nil, fmt.Errorf("convert planned value %q: %w", key, err)
		}
		if include {
			result[key] = converted
		}
	}
	return result, nil
}

func valueToGo(value terraformValue) (any, bool, error) {
	if !value.IsKnown() {
		return nil, false, nil
	}
	if value.IsNull() {
		return nil, true, nil
	}
	switch kind := value.Type().(type) {
	case interface{ Is(tftypes.Type) bool }:
		if kind.Is(tftypes.String) {
			var result string
			err := value.As(&result)
			return result, true, err
		}
		if kind.Is(tftypes.Bool) {
			var result bool
			err := value.As(&result)
			return result, true, err
		}
		if kind.Is(tftypes.Number) {
			result := new(big.Float)
			err := value.As(result)
			return result, true, err
		}
		if object, ok := kind.(tftypes.Object); ok {
			_ = object
			fields := make(map[string]tftypes.Value)
			if err := value.As(&fields); err != nil {
				return nil, false, err
			}
			return nestedValues(fields)
		}
		if _, ok := kind.(tftypes.Map); ok {
			fields := make(map[string]tftypes.Value)
			if err := value.As(&fields); err != nil {
				return nil, false, err
			}
			return nestedValues(fields)
		}
		if _, ok := kind.(tftypes.List); ok {
			values := make([]tftypes.Value, 0)
			if err := value.As(&values); err != nil {
				return nil, false, err
			}
			return nestedList(values)
		}
		if _, ok := kind.(tftypes.Set); ok {
			values := make([]tftypes.Value, 0)
			if err := value.As(&values); err != nil {
				return nil, false, err
			}
			return nestedList(values)
		}
	}
	return nil, false, fmt.Errorf("unsupported Terraform type %s", value.Type())
}

func nestedValues(values map[string]tftypes.Value) (any, bool, error) {
	result := make(map[string]any, len(values))
	for key, value := range values {
		converted, include, err := valueToGo(value)
		if err != nil {
			return nil, false, err
		}
		if include {
			result[key] = converted
		}
	}
	return result, true, nil
}

func nestedList(values []tftypes.Value) (any, bool, error) {
	result := make([]any, 0, len(values))
	for _, value := range values {
		converted, include, err := valueToGo(value)
		if err != nil {
			return nil, false, err
		}
		if include {
			result = append(result, converted)
		}
	}
	return result, true, nil
}

func decodeObject(body []byte) (map[string]any, error) {
	if !json.Valid(body) || bytes.Equal(bytes.TrimSpace(body), []byte("null")) {
		return nil, fmt.Errorf("empty JSON response")
	}
	var result map[string]any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func collection(root map[string]any, responsePath []string, itemPath string) ([]map[string]any, error) {
	var value any = root
	for _, part := range responsePath {
		object, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("API response field %q is not an object", part)
		}
		value = object[part]
	}
	values, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("API response collection %q is missing or invalid", strings.Join(responsePath, "."))
	}
	result := make([]map[string]any, 0, len(values))
	for _, value := range values {
		item, ok := value.(map[string]any)
		if !ok {
			continue
		}
		if itemPath != "" {
			item, ok = item[itemPath].(map[string]any)
			if !ok {
				continue
			}
		}
		result = append(result, item)
	}
	return result, nil
}

func matchesAttributes(item, selector map[string]any, attributes []string) bool {
	used := false
	for _, name := range attributes {
		expected, ok := selector[name]
		if !ok || expected == nil {
			continue
		}
		used = true
		actual, ok := item[name]
		if !ok || !matchesValue(actual, expected) {
			return false
		}
	}
	return used
}

func matchesValue(actual, expected any) bool {
	if expected == nil {
		return true
	}
	switch expectedValue := expected.(type) {
	case map[string]any:
		actualValue, ok := actual.(map[string]any)
		if !ok {
			return false
		}
		for key, expectedField := range expectedValue {
			if expectedField == nil {
				continue
			}
			actualField, ok := actualValue[key]
			if !ok || !matchesValue(actualField, expectedField) {
				return false
			}
		}
		return true
	case []any:
		actualValue, ok := actual.([]any)
		if !ok || len(actualValue) != len(expectedValue) {
			return false
		}
		for index, expectedItem := range expectedValue {
			if !matchesValue(actualValue[index], expectedItem) {
				return false
			}
		}
		return true
	default:
		expectedJSON, err := json.Marshal(expected)
		if err != nil {
			return false
		}
		actualJSON, err := json.Marshal(actual)
		return err == nil && bytes.Equal(expectedJSON, actualJSON)
	}
}

func resourceKey(object map[string]any, keyAttribute string) (string, error) {
	value, ok := object[keyAttribute]
	if !ok || value == nil || valueString(value) == "" {
		return "", fmt.Errorf("resource identity attribute %q is not set", keyAttribute)
	}
	return valueString(value), nil
}

func valueString(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func completeMap(attributes map[string]schema.Attribute, actual, fallback map[string]any) map[string]any {
	result := make(map[string]any, len(attributes))
	for name, attribute := range attributes {
		value, ok := actual[name]
		if !ok {
			value, ok = fallback[name]
		}
		if !ok {
			value = nil
		}
		result[name] = completeAttribute(attribute, value, fallback[name])
	}
	return result
}

func completeAttribute(attribute schema.Attribute, value, fallback any) any {
	if value == nil {
		return fallback
	}
	switch attr := attribute.(type) {
	case schema.ListNestedAttribute:
		items, ok := value.([]any)
		if !ok {
			return value
		}
		result := make([]any, 0, len(items))
		for index, item := range items {
			object, ok := item.(map[string]any)
			if !ok {
				result = append(result, item)
				continue
			}
			var prior map[string]any
			if priorItems, ok := fallback.([]any); ok && index < len(priorItems) {
				prior, _ = priorItems[index].(map[string]any)
			}
			result = append(result, completeMap(attr.NestedObject.Attributes, object, prior))
		}
		return result
	case schema.SingleNestedAttribute:
		object, ok := value.(map[string]any)
		if !ok {
			return value
		}
		prior, _ := fallback.(map[string]any)
		return completeMap(attr.Attributes, object, prior)
	default:
		return value
	}
}

func encodeBody[T any](values map[string]any) (T, error) {
	var result T
	body, err := json.Marshal(values)
	if err != nil {
		return result, err
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return result, err
	}
	return result, nil
}

func stringValue(object map[string]any, name string) (string, error) {
	value, ok := object[name]
	if !ok || value == nil || fmt.Sprint(value) == "" {
		return "", fmt.Errorf("required path attribute %q is not set", name)
	}
	return fmt.Sprint(value), nil
}
