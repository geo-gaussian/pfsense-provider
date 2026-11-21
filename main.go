package main

import (
	"context"
	"log"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/provider"
	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var version = "dev"

func main() {
	ctx := context.Background()
	err := frameworkprovider.Serve(ctx, provider.New(version), frameworkprovider.ServeOpts{
		Address: "registry.terraform.io/geo-gaussian/pfsense-nexus",
	})
	if err != nil {
		log.Fatal(err)
	}
}
