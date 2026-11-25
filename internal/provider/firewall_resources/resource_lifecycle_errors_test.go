package firewall_resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type lifecycleOptions struct {
	createErr     error
	createResult  response
	updateErr     error
	updateResult  response
	deleteErr     error
	deleteResult  response
	readErrAt     int
	readResult    response
	reloadStatus  int
	withoutUpdate bool
}

type lifecycleHarness struct {
	model  *resourceModel
	schema schema.Schema
	reads  int
}

type failingRoundTripper struct{}

func (failingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("connection failed")
}

func newLifecycleHarness(t *testing.T, options lifecycleOptions) *lifecycleHarness {
	t.Helper()
	resourceSchema := schema.Schema{Attributes: map[string]schema.Attribute{
		"id":   schema.StringAttribute{Optional: true, Computed: true},
		"name": schema.StringAttribute{Required: true},
		"note": schema.StringAttribute{Optional: true},
	}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/system/filterreload/reload" {
			t.Errorf("unexpected API request %s %s", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		status := options.reloadStatus
		if status == 0 {
			status = http.StatusOK
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = fmt.Fprint(w, `{"result":"started"}`)
	}))
	t.Cleanup(server.Close)
	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	harness := &lifecycleHarness{schema: resourceSchema}
	def := definition{
		name: "test_resource", keyAttribute: "id", responsePath: []string{"items"},
		matchAttributes: []string{"name"},
		schema:          func(context.Context) schema.Schema { return resourceSchema },
		create: func(context.Context, *api.ClientWithResponses, map[string]any, string) (response, error) {
			if options.createErr != nil {
				return nil, options.createErr
			}
			if options.createResult != nil {
				return options.createResult, nil
			}
			return stubResponse{status: http.StatusOK, body: []byte(`{"msg":"ok","id":"id-1"}`)}, nil
		},
		read: func(context.Context, *api.ClientWithResponses, map[string]any) (response, error) {
			harness.reads++
			if options.readErrAt == harness.reads {
				return nil, errors.New("read failed")
			}
			if options.readResult != nil {
				return options.readResult, nil
			}
			return stubResponse{status: http.StatusOK, body: []byte(`{"items":[{"id":"id-1","name":"example"}]}`)}, nil
		},
		delete: func(context.Context, *api.ClientWithResponses, map[string]any, string) (response, error) {
			if options.deleteErr != nil {
				return nil, options.deleteErr
			}
			if options.deleteResult != nil {
				return options.deleteResult, nil
			}
			return stubResponse{status: http.StatusOK, body: []byte(`{"msg":"ok"}`)}, nil
		},
	}
	if !options.withoutUpdate {
		def.update = func(context.Context, *api.ClientWithResponses, map[string]any, string) (response, error) {
			if options.updateErr != nil {
				return nil, options.updateErr
			}
			if options.updateResult != nil {
				return options.updateResult, nil
			}
			return stubResponse{status: http.StatusOK, body: []byte(`{"msg":"ok"}`)}, nil
		}
	}
	harness.model = &resourceModel{client: client, def: def}
	return harness
}

func lifecycleValue(t *testing.T, s schema.Schema, name, id string) tftypes.Value {
	t.Helper()
	values := map[string]any{"id": id, "name": name, "note": nil}
	body, err := jsonMarshal(values)
	if err != nil {
		t.Fatal(err)
	}
	value, err := tftypes.ValueFromJSON(body, s.Type().TerraformType(context.Background()))
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func jsonMarshal(value any) ([]byte, error) {
	return json.Marshal(value)
}

func TestResourceLifecycleCreateFailures(t *testing.T) {
	missingClient := &resourceModel{def: definition{keyAttribute: "id"}}
	missingClientResp := resource.CreateResponse{}
	missingClient.Create(context.Background(), resource.CreateRequest{}, &missingClientResp)
	if !missingClientResp.Diagnostics.HasError() {
		t.Fatal("Create() accepted a missing API client")
	}

	tests := []struct {
		name     string
		options  lifecycleOptions
		invalid  bool
		noSchema bool
	}{
		{name: "action error", options: lifecycleOptions{createErr: errors.New("create failed")}},
		{name: "invalid response", options: lifecycleOptions{createResult: stubResponse{status: http.StatusBadRequest, body: []byte(`{}`)}}},
		{name: "missing created item", options: lifecycleOptions{readResult: stubResponse{status: http.StatusOK, body: []byte(`{"items":[]}`)}}},
		{name: "reload error", options: lifecycleOptions{reloadStatus: http.StatusInternalServerError}},
		{name: "post-reload read error", options: lifecycleOptions{readErrAt: 2}},
		{name: "invalid plan", invalid: true},
		{name: "missing state schema", noSchema: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newLifecycleHarness(t, tt.options)
			plan := lifecycleValue(t, h.schema, "example", "")
			if tt.invalid {
				plan = tftypes.NewValue(tftypes.String, "not an object")
			}
			state := tfsdk.State{Schema: h.schema}
			if tt.noSchema {
				state.Schema = nil
			}
			resp := resource.CreateResponse{State: state}
			h.model.Create(context.Background(), resource.CreateRequest{
				Plan: tfsdk.Plan{Schema: h.schema, Raw: plan},
			}, &resp)
			if !resp.Diagnostics.HasError() {
				t.Fatal("Create() did not report the expected failure")
			}
		})
	}
}

func TestResourceLifecycleReadAndConfigurationFailures(t *testing.T) {
	ctx := context.Background()
	missingClient := &resourceModel{def: definition{keyAttribute: "id"}}
	missingClientRead := resource.ReadResponse{}
	missingClient.Read(ctx, resource.ReadRequest{}, &missingClientRead)
	if !missingClientRead.Diagnostics.HasError() {
		t.Fatal("Read() accepted a missing API client")
	}

	h := newLifecycleHarness(t, lifecycleOptions{
		readResult: stubResponse{status: http.StatusNotFound, body: []byte(`{}`)},
	})
	state := tfsdk.State{Schema: h.schema, Raw: lifecycleValue(t, h.schema, "example", "id-1")}
	resp := resource.ReadResponse{State: state}
	h.model.Read(ctx, resource.ReadRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported an error for an out-of-band deletion: %v", resp.Diagnostics)
	}

	h = newLifecycleHarness(t, lifecycleOptions{})
	resp = resource.ReadResponse{State: tfsdk.State{Schema: h.schema}}
	h.model.Read(ctx, resource.ReadRequest{State: tfsdk.State{Raw: tftypes.NewValue(tftypes.String, "bad")}}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("Read() accepted an invalid state value")
	}
	h = newLifecycleHarness(t, lifecycleOptions{
		readResult: stubResponse{status: http.StatusInternalServerError, body: []byte(`{}`)},
	})
	state = tfsdk.State{Schema: h.schema, Raw: lifecycleValue(t, h.schema, "example", "id-1")}
	resp = resource.ReadResponse{State: state}
	h.model.Read(ctx, resource.ReadRequest{State: state}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("Read() ignored an API failure")
	}

	if _, ok := any(h.model).(resource.ResourceWithConfigure); !ok {
		t.Fatal("resource does not implement Configure")
	}
	for _, providerData := range []any{nil, "unexpected"} {
		var configureResp resource.ConfigureResponse
		h.model.Configure(ctx, resource.ConfigureRequest{ProviderData: providerData}, &configureResp)
		if providerData != nil && !configureResp.Diagnostics.HasError() {
			t.Fatal("Configure() accepted unexpected provider data")
		}
	}
}

func TestResourceLifecycleUpdateFailures(t *testing.T) {
	missingClient := &resourceModel{def: definition{keyAttribute: "id"}}
	missingClientResp := resource.UpdateResponse{}
	missingClient.Update(context.Background(), resource.UpdateRequest{}, &missingClientResp)
	if !missingClientResp.Diagnostics.HasError() {
		t.Fatal("Update() accepted a missing API client")
	}

	tests := []struct {
		name     string
		options  lifecycleOptions
		invalid  string
		noSchema bool
	}{
		{name: "unsupported", options: lifecycleOptions{withoutUpdate: true}},
		{name: "invalid prior state", invalid: "state"},
		{name: "invalid plan", invalid: "plan"},
		{name: "missing identity", invalid: "identity"},
		{name: "action error", options: lifecycleOptions{updateErr: errors.New("update failed")}},
		{name: "invalid response", options: lifecycleOptions{updateResult: stubResponse{status: http.StatusBadRequest, body: []byte(`{}`)}}},
		{name: "verify read error", options: lifecycleOptions{readErrAt: 1}},
		{name: "reload error", options: lifecycleOptions{reloadStatus: http.StatusInternalServerError}},
		{name: "post-reload read error", options: lifecycleOptions{readErrAt: 2}},
		{name: "missing state schema", noSchema: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newLifecycleHarness(t, tt.options)
			prior := lifecycleValue(t, h.schema, "example", "id-1")
			plan := lifecycleValue(t, h.schema, "updated", "id-1")
			if tt.invalid == "state" {
				prior = tftypes.NewValue(tftypes.String, "not an object")
			} else if tt.invalid == "plan" {
				plan = tftypes.NewValue(tftypes.String, "not an object")
			} else if tt.invalid == "identity" {
				prior = lifecycleValue(t, h.schema, "example", "")
			}
			state := tfsdk.State{Schema: h.schema, Raw: prior}
			if tt.noSchema {
				state.Schema = nil
			}
			resp := resource.UpdateResponse{State: tfsdk.State{Schema: h.schema}}
			if tt.noSchema {
				resp.State = tfsdk.State{}
			}
			h.model.Update(context.Background(), resource.UpdateRequest{
				State: state, Plan: tfsdk.Plan{Schema: h.schema, Raw: plan},
			}, &resp)
			if !resp.Diagnostics.HasError() {
				t.Fatal("Update() did not report the expected failure")
			}
		})
	}
}

func TestResourceLifecycleDeleteFailuresAndNotFound(t *testing.T) {
	tests := []struct {
		name    string
		options lifecycleOptions
		invalid bool
	}{
		{name: "action error", options: lifecycleOptions{deleteErr: errors.New("delete failed")}},
		{name: "invalid response", options: lifecycleOptions{deleteResult: stubResponse{status: http.StatusBadRequest}}},
		{name: "reload error", options: lifecycleOptions{reloadStatus: http.StatusInternalServerError}},
		{name: "still present", options: lifecycleOptions{}},
		{name: "verification read error", options: lifecycleOptions{readErrAt: 1}},
		{name: "not found is success", options: lifecycleOptions{
			deleteResult: stubResponse{status: http.StatusNotFound},
			readResult:   stubResponse{status: http.StatusNotFound},
		}},
		{name: "invalid state", invalid: true},
		{name: "missing identity", invalid: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newLifecycleHarness(t, tt.options)
			state := tfsdk.State{Schema: h.schema, Raw: lifecycleValue(t, h.schema, "example", "id-1")}
			if tt.name == "invalid state" {
				state.Raw = tftypes.NewValue(tftypes.String, "not an object")
			} else if tt.name == "missing identity" {
				state.Raw = lifecycleValue(t, h.schema, "example", "")
			}
			resp := resource.DeleteResponse{}
			h.model.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
			if tt.name == "not found is success" {
				if resp.Diagnostics.HasError() {
					t.Fatalf("Delete() treated 404 as an error: %v", resp.Diagnostics)
				}
				return
			}
			if !resp.Diagnostics.HasError() {
				t.Fatal("Delete() did not report the expected failure")
			}
		})
	}

	missingClient := &resourceModel{def: definition{keyAttribute: "id"}}
	resp := resource.DeleteResponse{}
	missingClient.Delete(context.Background(), resource.DeleteRequest{}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("Delete() accepted a missing API client")
	}
}

func TestImportStateRejectsInvalidIdentifiers(t *testing.T) {
	ctx := context.Background()
	r := NewRule().(resource.ResourceWithImportState)
	for _, id := range []string{"", "lan/", "/rule-1", "lan/rule/extra"} {
		var resp resource.ImportStateResponse
		r.ImportState(ctx, resource.ImportStateRequest{ID: id}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("ImportState(%q) did not reject the identifier", id)
		}
	}
	altq := NewALTQ().(resource.ResourceWithImportState)
	var resp resource.ImportStateResponse
	altq.ImportState(ctx, resource.ImportStateRequest{ID: "root-queue"}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("single-attribute import failed: %v", resp.Diagnostics)
	}
	values, err := objectValues(resp.State.Raw)
	if err != nil {
		t.Fatal(err)
	}
	if values["name"] != "root-queue" {
		t.Fatalf("imported root queue name = %#v", values["name"])
	}

	badDefinition := &resourceModel{def: definition{
		name: "invalid_import", keyAttribute: "id", importAttributes: []string{"unknown"},
		schema: func(context.Context) schema.Schema {
			return schema.Schema{Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{Optional: true, Computed: true},
			}}
		},
	}}
	var invalidResp resource.ImportStateResponse
	badDefinition.ImportState(ctx, resource.ImportStateRequest{ID: "value"}, &invalidResp)
	if !invalidResp.Diagnostics.HasError() {
		t.Fatal("ImportState() accepted an attribute not present in the schema")
	}
	wrongTypeDefinition := &resourceModel{def: definition{
		name: "invalid_import_type", keyAttribute: "id", importAttributes: []string{"id"},
		schema: func(context.Context) schema.Schema {
			return schema.Schema{Attributes: map[string]schema.Attribute{
				"id": schema.Int64Attribute{Optional: true, Computed: true},
			}}
		},
	}}
	invalidResp = resource.ImportStateResponse{}
	wrongTypeDefinition.ImportState(ctx, resource.ImportStateRequest{ID: "1"}, &invalidResp)
	if !invalidResp.Diagnostics.HasError() {
		t.Fatal("ImportState() accepted a non-string identity attribute")
	}
}

func TestReloadPropagatesRequestError(t *testing.T) {
	client, err := api.NewClientWithResponses("http://firewall.invalid",
		api.WithHTTPClient(&http.Client{Transport: failingRoundTripper{}}),
	)
	if err != nil {
		t.Fatal(err)
	}
	r := &resourceModel{client: client}
	if err := r.reload(context.Background()); err == nil {
		t.Fatal("reload() suppressed the HTTP transport error")
	}
}
