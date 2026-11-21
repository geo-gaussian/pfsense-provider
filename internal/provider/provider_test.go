package provider

import (
	"context"
	"testing"

	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
)

func TestMetadataUsesBuildVersion(t *testing.T) {
	p := New("0.1.0")()
	var resp frameworkprovider.MetadataResponse
	p.Metadata(context.Background(), frameworkprovider.MetadataRequest{}, &resp)
	if resp.Version != "0.1.0" {
		t.Fatalf("provider version = %q, want 0.1.0", resp.Version)
	}
}
