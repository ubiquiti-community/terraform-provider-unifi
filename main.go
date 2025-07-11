// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi"
)

//go:generate go tool github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate -provider-name unifi
func main() {
	var debug bool

	flag.BoolVar(
		&debug,
		"debug",
		false,
		"set to true to run the provider with support for debuggers like delve",
	)
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address:         "registry.terraform.io/ubiquiti-community/unifi",
		Debug:           debug,
		ProtocolVersion: 6,
	}

	err := providerserver.Serve(context.Background(), unifi.New, opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
