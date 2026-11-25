package firewall_resources

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"reflect"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type stubResponse struct {
	status int
	body   []byte
}

type conversionErrorValue struct {
	typ tftypes.Type
}

func (conversionErrorValue) IsKnown() bool        { return true }
func (conversionErrorValue) IsNull() bool         { return false }
func (v conversionErrorValue) Type() tftypes.Type { return v.typ }
func (conversionErrorValue) As(any) error         { return errors.New("conversion failed") }

func (r stubResponse) StatusCode() int { return r.status }
func (r stubResponse) GetBody() []byte { return r.body }

func TestValueToGo(t *testing.T) {
	objectType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"name": tftypes.String}}
	mapType := tftypes.Map{ElementType: tftypes.String}
	listType := tftypes.List{ElementType: tftypes.String}
	setType := tftypes.Set{ElementType: tftypes.String}
	tupleType := tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String}}
	cases := []struct {
		name    string
		value   tftypes.Value
		want    any
		include bool
		wantErr bool
	}{
		{name: "unknown", value: tftypes.NewValue(tftypes.String, tftypes.UnknownValue)},
		{name: "null", value: tftypes.NewValue(tftypes.String, nil), include: true},
		{name: "string", value: tftypes.NewValue(tftypes.String, "value"), want: "value", include: true},
		{name: "boolean", value: tftypes.NewValue(tftypes.Bool, true), want: true, include: true},
		{name: "number", value: tftypes.NewValue(tftypes.Number, 2), include: true},
		{
			name: "object", value: tftypes.NewValue(objectType, map[string]tftypes.Value{
				"name": tftypes.NewValue(tftypes.String, "value"),
			}),
			want: map[string]any{"name": "value"}, include: true,
		},
		{
			name: "map", value: tftypes.NewValue(mapType, map[string]tftypes.Value{
				"name": tftypes.NewValue(tftypes.String, "value"),
			}),
			want: map[string]any{"name": "value"}, include: true,
		},
		{
			name: "list", value: tftypes.NewValue(listType, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "value"),
				tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
			}),
			want: []any{"value"}, include: true,
		},
		{
			name: "set", value: tftypes.NewValue(setType, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "value"),
			}),
			want: []any{"value"}, include: true,
		},
		{name: "unsupported", value: tftypes.NewValue(tupleType, []tftypes.Value{tftypes.NewValue(tftypes.String, "value")}), wantErr: true},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, include, err := valueToGo(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("valueToGo() error = %v, wantErr %v", err, tt.wantErr)
			}
			if include != tt.include {
				t.Fatalf("valueToGo() include = %v, want %v", include, tt.include)
			}
			if tt.name == "number" {
				number, ok := got.(*big.Float)
				if !ok || number.Cmp(big.NewFloat(2)) != 0 {
					t.Fatalf("valueToGo() number = %v, want 2", got)
				}
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("valueToGo() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestDecodeObjectAndCollection(t *testing.T) {
	for _, body := range [][]byte{nil, []byte("null"), []byte("[]"), []byte("{")} {
		if _, err := decodeObject(body); err == nil {
			t.Errorf("decodeObject(%q) unexpectedly succeeded", body)
		}
	}
	root, err := decodeObject([]byte(`{"items":[{"entry":{"id":"a"}},{"not_an_entry":true},null]}`))
	if err != nil {
		t.Fatal(err)
	}
	items, err := collection(root, []string{"items"}, "entry")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(items, []map[string]any{{"id": "a"}}) {
		t.Fatalf("collection() = %#v", items)
	}
	if _, err := collection(root, []string{"items", "nested"}, ""); err == nil {
		t.Fatal("collection() accepted a missing nested field")
	}
	if _, err := collection(map[string]any{"items": map[string]any{}}, []string{"items"}, ""); err == nil {
		t.Fatal("collection() accepted a non-list field")
	}
	if _, err := collection(map[string]any{"items": []any{"bad"}}, []string{"items"}, ""); err != nil {
		t.Fatalf("collection() rejected a list with no map entries: %v", err)
	}
}

func TestMatchAndStateHelpers(t *testing.T) {
	item := map[string]any{
		"name": "rule", "enabled": true,
		"source": map[string]any{"address": "192.0.2.1"},
		"ports":  []any{"80", "443"},
	}
	selector := map[string]any{
		"name":   "rule",
		"source": map[string]any{"address": "192.0.2.1", "alias": nil},
		"ports":  []any{"80", "443"},
	}
	if !matchesAttributes(item, selector, []string{"name", "source", "ports"}) {
		t.Fatal("matchesAttributes() did not match equivalent nested values")
	}
	if matchesAttributes(item, map[string]any{"name": "other"}, []string{"name"}) {
		t.Fatal("matchesAttributes() matched different values")
	}
	if matchesAttributes(item, map[string]any{"unknown": "value"}, []string{"unknown"}) {
		t.Fatal("matchesAttributes() matched a missing attribute")
	}
	if matchesAttributes(item, map[string]any{"name": nil}, []string{"name"}) {
		t.Fatal("matchesAttributes() matched without a usable selector")
	}
	if matchesValue("1", 1) || matchesValue([]any{"a"}, "a") {
		t.Fatal("matchesValue() matched incompatible values")
	}
	if got := completeMap(map[string]schema.Attribute{
		"name": schema.StringAttribute{},
		"nested": schema.SingleNestedAttribute{Attributes: map[string]schema.Attribute{
			"value": schema.StringAttribute{},
		}},
	}, map[string]any{"nested": map[string]any{"value": "new"}}, map[string]any{
		"name": "prior", "nested": map[string]any{"value": "old"},
	}); !reflect.DeepEqual(got, map[string]any{
		"name": "prior", "nested": map[string]any{"value": "new"},
	}) {
		t.Fatalf("completeMap() = %#v", got)
	}
	if got := completeAttribute(schema.ListNestedAttribute{
		NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{},
		}},
	}, []any{map[string]any{}}, []any{map[string]any{"name": "prior"}}); !reflect.DeepEqual(got, []any{map[string]any{"name": "prior"}}) {
		t.Fatalf("completeAttribute() list result = %#v", got)
	}
	if got := completeAttribute(schema.StringAttribute{}, "actual", nil); got != "actual" {
		t.Fatalf("completeAttribute() = %#v", got)
	}
	if got := completeAttribute(schema.ListNestedAttribute{}, []any{"not an object"}, nil); !reflect.DeepEqual(got, []any{"not an object"}) {
		t.Fatalf("completeAttribute() malformed list = %#v", got)
	}
	listAttribute := schema.ListNestedAttribute{NestedObject: schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{"name": schema.StringAttribute{}},
	}}
	if got := completeAttribute(listAttribute, []any{"invalid", map[string]any{}}, []any{"not a map"}); !reflect.DeepEqual(got, []any{"invalid", map[string]any{"name": nil}}) {
		t.Fatalf("completeAttribute() without a matching prior map = %#v", got)
	}
	if got := completeAttribute(listAttribute, "not a list", nil); got != "not a list" {
		t.Fatalf("completeAttribute() non-list = %#v", got)
	}
	if got := completeAttribute(schema.SingleNestedAttribute{}, "not an object", nil); got != "not an object" {
		t.Fatalf("completeAttribute() non-object = %#v", got)
	}
	if got := completeAttribute(schema.StringAttribute{}, nil, "fallback"); got != "fallback" {
		t.Fatalf("completeAttribute() nil fallback = %#v", got)
	}
	if matchesValue(make(chan int), make(chan int)) || matchesValue(make(chan int), "value") {
		t.Fatal("matchesValue() matched values that cannot be JSON-encoded")
	}
	if !matchesValue("ignored", nil) {
		t.Fatal("matchesValue() did not accept a nil expected value")
	}
	if matchesValue("not a map", map[string]any{"name": "rule"}) {
		t.Fatal("matchesValue() accepted a non-map actual value")
	}
	if matchesValue(map[string]any{}, map[string]any{"name": "rule"}) {
		t.Fatal("matchesValue() accepted a missing nested value")
	}
	if matchesValue([]any{"80"}, []any{"80", "443"}) {
		t.Fatal("matchesValue() accepted lists of different lengths")
	}
	if matchesValue([]any{"80"}, []any{"443"}) {
		t.Fatal("matchesValue() accepted different list values")
	}
}

func TestRequiresReplaceSupportsSchemaAttributeKinds(t *testing.T) {
	nested := schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
		"name": schema.StringAttribute{Optional: true},
	}}
	attributes := []schema.Attribute{
		schema.StringAttribute{Optional: true},
		schema.BoolAttribute{Required: true},
		schema.Int64Attribute{Optional: true},
		schema.Float64Attribute{Optional: true},
		schema.NumberAttribute{Optional: true},
		schema.ListAttribute{Optional: true, ElementType: types.StringType},
		schema.ListNestedAttribute{Optional: true, NestedObject: nested},
		schema.SetAttribute{Optional: true, ElementType: types.StringType},
		schema.SetNestedAttribute{Optional: true, NestedObject: nested},
		schema.MapAttribute{Optional: true, ElementType: types.StringType},
		schema.MapNestedAttribute{Optional: true, NestedObject: nested},
		schema.SingleNestedAttribute{Optional: true, Attributes: nested.Attributes},
		schema.DynamicAttribute{Optional: true},
		schema.StringAttribute{Computed: true},
		schema.StringAttribute{},
	}
	for index, attribute := range attributes {
		updated := withRequiresReplace(attribute)
		if index < len(attributes)-3 {
			planModifiers := testPlanModifierCount(updated)
			if planModifiers == 0 {
				t.Errorf("attribute %T did not receive a replacement modifier", attribute)
			}
		} else if planModifiers := testPlanModifierCount(updated); planModifiers != 0 {
			t.Errorf("non-configurable attribute %T received %d plan modifiers", attribute, planModifiers)
		}
	}
}

func testPlanModifierCount(attribute schema.Attribute) int {
	switch value := attribute.(type) {
	case schema.StringAttribute:
		return len(value.PlanModifiers)
	case schema.BoolAttribute:
		return len(value.PlanModifiers)
	case schema.Int64Attribute:
		return len(value.PlanModifiers)
	case schema.Float64Attribute:
		return len(value.PlanModifiers)
	case schema.NumberAttribute:
		return len(value.PlanModifiers)
	case schema.ListAttribute:
		return len(value.PlanModifiers)
	case schema.ListNestedAttribute:
		return len(value.PlanModifiers)
	case schema.SetAttribute:
		return len(value.PlanModifiers)
	case schema.SetNestedAttribute:
		return len(value.PlanModifiers)
	case schema.MapAttribute:
		return len(value.PlanModifiers)
	case schema.MapNestedAttribute:
		return len(value.PlanModifiers)
	case schema.SingleNestedAttribute:
		return len(value.PlanModifiers)
	default:
		return 0
	}
}

func TestNestedValueConversionErrors(t *testing.T) {
	tupleType := tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String}}
	tupleValue := tftypes.NewValue(tupleType, []tftypes.Value{tftypes.NewValue(tftypes.String, "value")})
	if _, _, err := valueToGo(tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{"nested": tupleType},
	}, map[string]tftypes.Value{"nested": tupleValue})); err == nil {
		t.Fatal("valueToGo() accepted an unsupported nested Terraform type")
	}
	if _, _, err := nestedValues(map[string]tftypes.Value{"nested": tupleValue}); err == nil {
		t.Fatal("nestedValues() accepted an unsupported Terraform type")
	}
	if _, _, err := nestedList([]tftypes.Value{tupleValue}); err == nil {
		t.Fatal("nestedList() accepted an unsupported Terraform type")
	}
	if _, err := objectValues(tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{"nested": tupleType},
	}, map[string]tftypes.Value{"nested": tupleValue})); err == nil {
		t.Fatal("objectValues() accepted an unsupported nested Terraform type")
	}
	for _, typ := range []tftypes.Type{
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{}},
		tftypes.Map{ElementType: tftypes.String},
		tftypes.List{ElementType: tftypes.String},
		tftypes.Set{ElementType: tftypes.String},
	} {
		if _, _, err := valueToGo(conversionErrorValue{typ: typ}); err == nil {
			t.Errorf("valueToGo(%s) suppressed an extraction failure", typ)
		}
	}
	if _, err := decodeObject([]byte(`1`)); err == nil {
		t.Fatal("decodeObject() accepted a scalar JSON value")
	}
	if _, err := encodeBody[struct {
		Port int `json:"port"`
	}](map[string]any{"port": "not an integer"}); err == nil {
		t.Fatal("encodeBody() accepted an invalid field type")
	}
}

func TestSetStateRejectsInvalidInputs(t *testing.T) {
	ctx := context.Background()
	s := schema.Schema{Attributes: map[string]schema.Attribute{
		"id":   schema.StringAttribute{Optional: true, Computed: true},
		"name": schema.StringAttribute{Required: true},
	}}
	r := &resourceModel{def: definition{schema: func(context.Context) schema.Schema { return s }}}
	state := tfsdk.State{Schema: s}
	if diags := r.setState(ctx, &state, map[string]any{"name": make(chan int)}, nil); !diags.HasError() {
		t.Fatal("setState() accepted a value that cannot be JSON-encoded")
	}
	if diags := r.setState(ctx, &state, map[string]any{"name": map[string]any{}}, nil); !diags.HasError() {
		t.Fatal("setState() accepted a value incompatible with the Terraform schema")
	}
	state.Schema = nil
	if diags := r.setState(ctx, &state, map[string]any{"name": "valid"}, nil); !diags.HasError() {
		t.Fatal("setState() accepted a missing schema")
	}
}

func TestRequireStringOnlyChangesExistingStringAttributes(t *testing.T) {
	input := schema.Schema{Attributes: map[string]schema.Attribute{
		"name":  schema.StringAttribute{Optional: true},
		"count": schema.Int64Attribute{Optional: true},
	}}
	result := requireString(input, "name")
	name := result.Attributes["name"].(schema.StringAttribute)
	if !name.Required || name.Optional || name.Computed {
		t.Fatalf("required string attribute = %#v", name)
	}
	if _, ok := requireString(input, "missing").Attributes["missing"]; ok {
		t.Fatal("requireString() inserted a missing attribute")
	}
	if _, ok := requireString(input, "count").Attributes["count"].(schema.Int64Attribute); !ok {
		t.Fatal("requireString() changed a non-string attribute")
	}
}

func TestResponseAndIdentityHelpers(t *testing.T) {
	if statusCode(nil) != 0 || statusCode(stubResponse{status: http.StatusOK}) != http.StatusOK {
		t.Fatal("statusCode() returned an unexpected status")
	}
	for _, tt := range []struct {
		name    string
		result  response
		wantErr bool
	}{
		{name: "nil", wantErr: true},
		{name: "http error", result: stubResponse{status: http.StatusBadRequest, body: []byte(`{}`)}, wantErr: true},
		{name: "invalid json", result: stubResponse{status: http.StatusOK, body: []byte("{")}, wantErr: true},
		{name: "error message", result: stubResponse{status: http.StatusOK, body: []byte(`{"errmsg":"denied"}`)}, wantErr: true},
		{name: "empty message", result: stubResponse{status: http.StatusOK, body: []byte(`{"msg":""}`)}, wantErr: true},
		{name: "error status", result: stubResponse{status: http.StatusOK, body: []byte(`{"status":"Failed"}`)}, wantErr: true},
		{name: "success", result: stubResponse{status: http.StatusOK, body: []byte(`{"msg":"ok"}`)}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateResponse(tt.result); (err != nil) != tt.wantErr {
				t.Fatalf("validateResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	if _, err := resourceKey(map[string]any{}, "id"); err == nil {
		t.Fatal("resourceKey() accepted a missing ID")
	}
	if _, err := resourceKey(map[string]any{"id": ""}, "id"); err == nil {
		t.Fatal("resourceKey() accepted an empty ID")
	}
	if got, err := resourceKey(map[string]any{"id": json.Number("1")}, "id"); err != nil || got != "1" {
		t.Fatalf("resourceKey() = %q, %v", got, err)
	}
	if valueString(nil) != "" || valueString("value") != "value" {
		t.Fatal("valueString() returned an unexpected value")
	}
	if _, err := stringValue(map[string]any{}, "interface"); err == nil {
		t.Fatal("stringValue() accepted a missing interface")
	}
	if _, err := stringValue(map[string]any{"interface": nil}, "interface"); err == nil {
		t.Fatal("stringValue() accepted a null interface")
	}
	if got, err := stringValue(map[string]any{"interface": "lan"}, "interface"); err != nil || got != "lan" {
		t.Fatalf("stringValue() = %q, %v", got, err)
	}
}

func TestReadItemSelectionAndErrors(t *testing.T) {
	ctx := context.Background()
	goodResponse := stubResponse{status: http.StatusOK, body: []byte(`{"items":[{"item":{"id":"one","name":"rule","source":{"address":"192.0.2.1"}}}]}`)}
	tests := []struct {
		name         string
		result       response
		readErr      error
		selector     map[string]any
		match        bool
		wantErr      bool
		wantNotFound bool
	}{
		{name: "key lookup", result: goodResponse, selector: map[string]any{"id": "one"}, wantErr: false},
		{name: "attribute lookup", result: goodResponse, selector: map[string]any{"name": "rule", "source": map[string]any{"address": "192.0.2.1"}}, match: true},
		{name: "read error", readErr: errors.New("offline"), wantErr: true},
		{name: "nil response", wantErr: true},
		{name: "not found response", result: stubResponse{status: http.StatusNotFound}, wantErr: true, wantNotFound: true},
		{name: "http error", result: stubResponse{status: http.StatusBadRequest}, wantErr: true},
		{name: "invalid json", result: stubResponse{status: http.StatusOK, body: []byte("{")}, wantErr: true},
		{name: "invalid collection", result: stubResponse{status: http.StatusOK, body: []byte(`{}`)}, wantErr: true},
		{name: "missing selector item", result: goodResponse, selector: map[string]any{"id": "missing"}, wantErr: true, wantNotFound: true},
		{name: "no attribute match", result: goodResponse, selector: map[string]any{"name": "other"}, match: true, wantErr: true},
		{
			name: "ambiguous attribute match",
			result: stubResponse{status: http.StatusOK, body: []byte(`{"items":[
				{"item":{"id":"one","name":"rule"}},{"item":{"id":"two","name":"rule"}}
			]}`)},
			selector: map[string]any{"name": "rule"}, match: true, wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &resourceModel{def: definition{
				keyAttribute: "id", responsePath: []string{"items"}, itemPath: "item",
				matchAttributes: []string{"name", "source"},
				read: func(context.Context, *api.ClientWithResponses, map[string]any) (response, error) {
					return tt.result, tt.readErr
				},
			}}
			item, err := r.readItem(ctx, tt.selector, tt.match)
			if (err != nil) != tt.wantErr {
				t.Fatalf("readItem() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantNotFound && !errors.Is(err, errItemNotFound) {
				t.Fatalf("readItem() error = %v, want errItemNotFound", err)
			}
			if err == nil && item == nil {
				t.Fatal("readItem() returned a nil item without an error")
			}
		})
	}
}
