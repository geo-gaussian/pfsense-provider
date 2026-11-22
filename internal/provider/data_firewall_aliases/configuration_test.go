package data_firewall_aliases

import (
	"context"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

func TestConfiguration(t *testing.T) {
	ctx := context.Background()
	d := New().(*aliasesDataSource)
	var metadata datasource.MetadataResponse
	d.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "pfsense"}, &metadata)
	if metadata.TypeName != "pfsense_firewall_aliases" {
		t.Fatalf("unexpected data source type %q", metadata.TypeName)
	}
	var resp datasource.ConfigureResponse
	d.Configure(ctx, datasource.ConfigureRequest{}, &resp)
	if resp.Diagnostics.HasError() || d.client != nil {
		t.Fatalf("unexpected configuration: %v", resp.Diagnostics)
	}
	resp = datasource.ConfigureResponse{}
	d.Configure(ctx, datasource.ConfigureRequest{ProviderData: "invalid"}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("accepted incorrect provider data")
	}
	client, err := api.NewClientWithResponses("https://example.test")
	if err != nil {
		t.Fatal(err)
	}
	resp = datasource.ConfigureResponse{}
	d.Configure(ctx, datasource.ConfigureRequest{ProviderData: client}, &resp)
	if resp.Diagnostics.HasError() || d.client != client {
		t.Fatalf("client was not configured: %v", resp.Diagnostics)
	}
}

func TestReadTransportFailure(t *testing.T) {
	client, err := api.NewClientWithResponses("https://example.test")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d := &aliasesDataSource{client: client}
	var schemaResp datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(ctx, datasource.ReadRequest{}, &resp)
	if len(resp.Diagnostics.Errors()) == 0 || resp.Diagnostics.Errors()[0].Summary() != "Unable to read firewall aliases" {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
}
