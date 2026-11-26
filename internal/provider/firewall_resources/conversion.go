package firewall_resources

import (
	"bytes"

	"encoding/json"

	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

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
		switch kind.(type) {
		case tftypes.Object, tftypes.Map:
			return convertFields(value)
		case tftypes.List, tftypes.Set:
			return convertList(value)
		}
	}
	return nil, false, fmt.Errorf("unsupported Terraform type %s", value.Type())
}

func convertFields(value terraformValue) (any, bool, error) {
	fields := make(map[string]tftypes.Value)
	if err := value.As(&fields); err != nil {
		return nil, false, err
	}
	return nestedValues(fields)
}

func convertList(value terraformValue) (any, bool, error) {
	values := make([]tftypes.Value, 0)
	if err := value.As(&values); err != nil {
		return nil, false, err
	}
	return nestedList(values)
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
