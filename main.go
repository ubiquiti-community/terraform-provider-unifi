package main // import "github.com/ubiquiti-community/terraform-provider-unifi"

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	
	// Plugin Framework provider
	frameworkProvider "github.com/ubiquiti-community/terraform-provider-unifi/unifi"
)

// Generate docs for website
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

// these will be set by the goreleaser configuration
// to appropriate values for the compiled binary.
var version string = "dev"

// goreleaser can also pass the specific commit if you want
// commit  string = "".

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/ubiquiti-community/unifi",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), frameworkProvider.New, opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
