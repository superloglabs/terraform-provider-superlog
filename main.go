package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/superloglabs/terraform-provider-superlog/internal/provider"
)

// Set by goreleaser.
var version = "dev"

//go:generate go tool tfplugindocs generate --provider-name superlog

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "Run the provider with support for debuggers such as delve.")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/superloglabs/superlog",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
