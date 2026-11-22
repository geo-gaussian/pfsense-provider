package provider

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestConfigureRejectsMalformedConfiguration(t *testing.T) {
	ctx := context.Background()
	p := New("test")()
	var schemaResp frameworkprovider.SchemaResponse
	p.Schema(ctx, frameworkprovider.SchemaRequest{}, &schemaResp)
	var resp frameworkprovider.ConfigureResponse
	p.Configure(ctx, frameworkprovider.ConfigureRequest{Config: tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(tftypes.String, "invalid"),
	}}, &resp)
	if !resp.Diagnostics.HasError() || resp.ResourceData != nil || resp.DataSourceData != nil {
		t.Fatalf("malformed configuration accepted: %v", resp.Diagnostics)
	}
}

func TestConfigureReportsClientFactoryFailure(t *testing.T) {
	ctx := context.Background()
	p := New("test")().(*pfsenseProvider)
	var schemaResp frameworkprovider.SchemaResponse
	p.Schema(ctx, frameworkprovider.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	raw := tftypes.NewValue(s.Type().TerraformType(ctx), map[string]tftypes.Value{
		"url":      tftypes.NewValue(tftypes.String, "https://example.test"),
		"username": tftypes.NewValue(tftypes.String, "test-user"),
		"password": tftypes.NewValue(tftypes.String, strings.Repeat("x", 12)),
		"insecure": tftypes.NewValue(tftypes.Bool, false),
	})
	var resp frameworkprovider.ConfigureResponse
	p.configure(ctx, frameworkprovider.ConfigureRequest{Config: tfsdk.Config{Schema: s, Raw: raw}}, &resp,
		func(server string, _ ...api.ClientOption) (*api.ClientWithResponses, error) {
			if server != "https://example.test/api/device/pfsense/localhost/api" {
				t.Errorf("unexpected API server %q", server)
			}
			return nil, errors.New("factory failure")
		})
	if len(resp.Diagnostics.Errors()) == 0 || resp.Diagnostics.Errors()[0].Summary() != "Unable to configure API client" ||
		resp.ResourceData != nil || resp.DataSourceData != nil {
		t.Fatalf("factory failure not reported: %v", resp.Diagnostics)
	}
}
