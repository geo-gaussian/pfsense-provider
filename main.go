package main

import (
	"context"
	"log"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/provider"
	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var version = "dev"

// These dependencies can be replaced to test startup without exiting the test process.
var serveProvider = frameworkprovider.Serve
var fatal = log.Fatal

func main() {
	ctx := context.Background()
	err := serveProvider(ctx, provider.New(version), frameworkprovider.ServeOpts{
		Address: "registry.terraform.io/geo-gaussian/pfsense-nexus",
	})
	if err != nil {
		fatal(err)
	}
}
