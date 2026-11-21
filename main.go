package main

import (
	"context"
	"log"

	"github.com/geo-gaussian/pfsense-provider/internal/provider"
	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	ctx := context.Background()
	err := frameworkprovider.Serve(ctx, provider.New, frameworkprovider.ServeOpts{
		Address: "registry.terraform.io/geo-gaussian/pfsense-nexus",
	})
	if err != nil {
		log.Fatal(err)
	}
}
