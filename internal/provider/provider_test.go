package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestMetadataUsesBuildVersion(t *testing.T) {
	p := New("0.1.0")()
	var resp frameworkprovider.MetadataResponse
	p.Metadata(context.Background(), frameworkprovider.MetadataRequest{}, &resp)
	if resp.Version != "0.1.0" {
		t.Fatalf("provider version = %q, want 0.1.0", resp.Version)
	}
}

func TestConfigure(t *testing.T) {
	ctx := context.Background()
	p := New("test")()
	var schemaResp frameworkprovider.SchemaResponse
	p.Schema(ctx, frameworkprovider.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	if !s.Attributes["password"].IsSensitive() {
		t.Fatal("password must be sensitive")
	}
	tests := []struct {
		name        string
		url         interface{}
		username    interface{}
		password    interface{}
		insecure    interface{}
		wantSummary string
	}{
		{name: "valid", url: "https://example.test", username: "test-user", password: strings.Repeat("x", 12), insecure: false},
		{name: "unknown URL", url: tftypes.UnknownValue, username: "test-user", password: "x", insecure: false, wantSummary: "Unknown provider configuration"},
		{name: "invalid URL", url: "http://example.test", username: "test-user", password: "x", insecure: false, wantSummary: "Invalid provider configuration"},
		{name: "empty password", url: "https://example.test", username: "test-user", password: "", insecure: false, wantSummary: "Invalid provider configuration"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := tftypes.NewValue(s.Type().TerraformType(ctx), map[string]tftypes.Value{
				"url":      tftypes.NewValue(tftypes.String, tt.url),
				"username": tftypes.NewValue(tftypes.String, tt.username),
				"password": tftypes.NewValue(tftypes.String, tt.password),
				"insecure": tftypes.NewValue(tftypes.Bool, tt.insecure),
			})
			var resp frameworkprovider.ConfigureResponse
			p.Configure(ctx, frameworkprovider.ConfigureRequest{Config: tfsdk.Config{Schema: s, Raw: raw}}, &resp)
			if tt.wantSummary != "" {
				if len(resp.Diagnostics.Errors()) == 0 || resp.Diagnostics.Errors()[0].Summary() != tt.wantSummary {
					t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
				}
				return
			}
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			client, ok := resp.ResourceData.(*api.ClientWithResponses)
			if !ok || client == nil || resp.ResourceData != resp.DataSourceData {
				t.Fatalf("unexpected configured clients: %#v %#v", resp.ResourceData, resp.DataSourceData)
			}
		})
	}
	if len(p.Resources(ctx)) != 1 || len(p.DataSources(ctx)) != 25 {
		t.Fatal("expected alias resource and twenty-five data sources")
	}
}
