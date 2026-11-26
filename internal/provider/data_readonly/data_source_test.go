package data_readonly

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestReadDataSource(t *testing.T) {

	for _, tt := range readDataSourceCases {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet || req.URL.Path != tt.path {
					t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				contentType := tt.contentType
				if contentType == "" {
					contentType = "application/json"
				}
				w.Header().Set("Content-Type", contentType)
				w.WriteHeader(tt.status)
				_, _ = fmt.Fprint(w, tt.body)
			}))
			defer server.Close()

			client, err := api.NewClientWithResponses(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			d := tt.dataSource()
			configureDataSource(t, d, client)
			dataSchema := dataSourceSchema(t, d)
			jsonAttribute, ok := dataSchema.Attributes["json"].(schema.StringAttribute)
			if !ok {
				t.Fatal("json attribute is not a string")
			}
			if jsonAttribute.Sensitive != tt.sensitive {
				t.Fatalf("json sensitivity = %t, want %t", jsonAttribute.Sensitive, tt.sensitive)
			}

			configValues := make(map[string]tftypes.Value, len(tt.configured)+1)
			for name, value := range tt.configured {
				configValues[name] = value
			}
			configValues["json"] = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
			config := tfsdk.Config{
				Schema: dataSchema,
				Raw:    tftypes.NewValue(dataSchema.Type().TerraformType(context.Background()), configValues),
			}
			resp := datasource.ReadResponse{State: tfsdk.State{Schema: dataSchema}}
			d.Read(context.Background(), datasource.ReadRequest{Config: config}, &resp)

			if resp.Diagnostics.HasError() != tt.wantError {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
			if tt.wantError {
				return
			}

			var state types.Object
			if diags := resp.State.Get(context.Background(), &state); diags.HasError() {
				t.Fatal(diags)
			}
			stateAttributes := state.Attributes()
			jsonValue, ok := stateAttributes["json"].(types.String)
			if !ok || jsonValue.ValueString() != tt.wantJSON {
				t.Fatalf("json = %#v, want %q", stateAttributes["json"], tt.wantJSON)
			}
			if input, ok := tt.configured["interface"]; ok {
				interfaceValue, exists := stateAttributes["interface"].(types.String)
				expected := "lan"
				if err := input.As(&expected); err != nil {
					t.Fatal(err)
				}
				if !exists || interfaceValue.ValueString() != expected {
					t.Fatalf("interface argument was not retained in state: %#v", stateAttributes["interface"])
				}
			}
		})
	}
}

func TestDataSourceMetadata(t *testing.T) {
	d := NewFirewallNATNpt()
	var resp datasource.MetadataResponse
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "pfsense"}, &resp)
	if resp.TypeName != "pfsense_firewall_nat_npt" {
		t.Fatalf("type name = %q, want pfsense_firewall_nat_npt", resp.TypeName)
	}
}

func TestConfigureDataSourceHandlesMissingAndInvalidClients(t *testing.T) {
	tests := []struct {
		name         string
		providerData any
		wantError    bool
	}{
		{name: "missing client"},
		{name: "unexpected client type", providerData: "not an API client", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configurable := NewFirewallNATNpt().(datasource.DataSourceWithConfigure)
			var resp datasource.ConfigureResponse
			configurable.Configure(
				context.Background(),
				datasource.ConfigureRequest{ProviderData: tt.providerData},
				&resp,
			)
			if resp.Diagnostics.HasError() != tt.wantError {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
		})
	}
}

func TestReadDataSourceRequiresConfiguredClient(t *testing.T) {
	d := NewFirewallNATNpt()
	dataSchema := dataSourceSchema(t, d)
	config := tfsdk.Config{
		Schema: dataSchema,
		Raw: tftypes.NewValue(dataSchema.Type().TerraformType(context.Background()), map[string]tftypes.Value{
			"json": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		}),
	}
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: dataSchema}}
	d.Read(context.Background(), datasource.ReadRequest{Config: config}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error when the API client is not configured")
	}
}

func TestReadDataSourceRejectsInvalidRequestConfig(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	d := NewFirewallNATNpt()
	configureDataSource(t, d, client)
	dataSchema := dataSourceSchema(t, d)
	config := tfsdk.Config{Schema: dataSchema, Raw: tftypes.NewValue(tftypes.String, "invalid")}
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: dataSchema}}
	d.Read(context.Background(), datasource.ReadRequest{Config: config}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error for an invalid request config")
	}
}

func TestReadDataSourceHandlesReadFailures(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		read readFunc
	}{
		{
			name: "read error",
			read: func(context.Context, *api.ClientWithResponses, map[string]string) (apiResponse, error) {
				return nil, errors.New("mock read failure")
			},
		},
		{
			name: "nil response",
			read: func(context.Context, *api.ClientWithResponses, map[string]string) (apiResponse, error) {
				return nil, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newDataSource("test", "test data source", nil, tt.read)
			configureDataSource(t, d, client)
			dataSchema := dataSourceSchema(t, d)
			config := tfsdk.Config{
				Schema: dataSchema,
				Raw: tftypes.NewValue(dataSchema.Type().TerraformType(context.Background()), map[string]tftypes.Value{
					"json": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
				}),
			}
			resp := datasource.ReadResponse{State: tfsdk.State{Schema: dataSchema}}
			d.Read(context.Background(), datasource.ReadRequest{Config: config}, &resp)
			if !resp.Diagnostics.HasError() {
				t.Fatal("expected a read error")
			}
		})
	}
}

func TestSetReadStateReportsObjectConstructionErrors(t *testing.T) {
	d := NewFirewallNATNpt()
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: dataSourceSchema(t, d)}}
	setReadState(context.Background(), map[string]attr.Value{
		"json": types.BoolValue(true),
	}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error for an invalid state value")
	}
}

func TestWireGuardStatusOmitsSecretMaterial(t *testing.T) {
	responseBody, err := os.ReadFile("../../../tests/fixtures/wireguard-status.json")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.Path != "/vpn/wg/status" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(responseBody)
	}))
	defer server.Close()

	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	d := NewWireGuardStatus()
	configureDataSource(t, d, client)
	schema := dataSourceSchema(t, d)
	configValues := map[string]tftypes.Value{
		"json": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	}
	config := tfsdk.Config{
		Schema: schema,
		Raw:    tftypes.NewValue(schema.Type().TerraformType(context.Background()), configValues),
	}
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: schema}}
	d.Read(context.Background(), datasource.ReadRequest{Config: config}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}

	var state types.Object
	if diags := resp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatal(diags)
	}
	jsonValue, ok := state.Attributes()["json"].(types.String)
	if !ok {
		t.Fatal("json state attribute is not a string")
	}
	got := jsonValue.ValueString()
	for _, secret := range []string{
		"private_key",
		"preshared_key",
		`"config"`,
		"tunnel-private",
		"peer-secret",
		"nested-private",
		"nested-secret",
	} {
		if strings.Contains(got, secret) {
			t.Fatalf("WireGuard output contains secret material %q: %s", secret, got)
		}
	}
	for _, safeValue := range []string{"tunnel-public", "peer-public", "latest_handshake", "transfer_rx"} {
		if !strings.Contains(got, safeValue) {
			t.Fatalf("WireGuard output omitted expected status value %q: %s", safeValue, got)
		}
	}
}

func TestWireGuardStatusHandlesMarshalError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"status":[]}`)
	}))
	defer server.Close()

	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	d := newWireGuardStatusDataSource(func(any) ([]byte, error) {
		return nil, errors.New("mock marshal failure")
	})
	configureDataSource(t, d, client)
	dataSchema := dataSourceSchema(t, d)
	config := tfsdk.Config{
		Schema: dataSchema,
		Raw: tftypes.NewValue(dataSchema.Type().TerraformType(context.Background()), map[string]tftypes.Value{
			"json": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		}),
	}
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: dataSchema}}
	d.Read(context.Background(), datasource.ReadRequest{Config: config}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error when WireGuard JSON encoding fails")
	}
	if !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "mock marshal failure") {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
}

func TestWireGuardStatusHandlesTransportError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			t.Error("HTTP response writer does not support connection hijacking")
			return
		}
		connection, _, err := hijacker.Hijack()
		if err != nil {
			t.Errorf("hijack connection: %v", err)
			return
		}
		_ = connection.Close()
	}))
	defer server.Close()

	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	d := NewWireGuardStatus()
	configureDataSource(t, d, client)
	dataSchema := dataSourceSchema(t, d)
	config := tfsdk.Config{
		Schema: dataSchema,
		Raw: tftypes.NewValue(dataSchema.Type().TerraformType(context.Background()), map[string]tftypes.Value{
			"json": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		}),
	}
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: dataSchema}}
	d.Read(context.Background(), datasource.ReadRequest{Config: config}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error when the API connection fails")
	}
}

func configureDataSource(t *testing.T, d datasource.DataSource, client *api.ClientWithResponses) {
	t.Helper()
	var resp datasource.ConfigureResponse
	configurable, ok := d.(datasource.DataSourceWithConfigure)
	if !ok {
		t.Fatal("data source does not support Configure")
	}
	configurable.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: client}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
}

func dataSourceSchema(t *testing.T, d datasource.DataSource) schema.Schema {
	t.Helper()
	var resp datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return resp.Schema
}
