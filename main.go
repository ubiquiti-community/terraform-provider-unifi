// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
	"github.com/ubiquiti-community/terraform-provider-unifi/internal/provider"
)

const (
	unifiProviderName = "registry.terraform.io/ubiquiti-community/unifi"
)

func main() {
	ctx := context.Background()

	// Remove any date and time prefix in log package function output to
	// prevent duplicate timestamp and incorrect log level setting
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))

	debugFlag := flag.Bool("debug", false, "Start provider in debug mode.")
	flag.Parse()

	var serveOpts []tf6server.ServeOpt

	if *debugFlag {
		serveOpts = append(serveOpts, tf6server.WithManagedDebug())
	}

	upgradedSdkProvider, err := tf5to6server.UpgradeServer(
		context.Background(),
		provider.Provider().GRPCProvider,
	)

	providers := []func() tfprotov6.ProviderServer{
		func() tfprotov6.ProviderServer {
			return upgradedSdkProvider
		},
		providerserver.NewProtocol6(provider.NewFrameworkProvider()),
	}

	muxServer, err := tf6muxserver.NewMuxServer(ctx, providers...)
	if err != nil {
		tflog.Error(ctx, "Failed to create MuxServer", map[string]any{
			"error": err,
		})
		os.Exit(1)
	}

	err = tf6server.Serve(unifiProviderName, muxServer.ProviderServer, serveOpts...)
	if err != nil {
		log.Printf("[ERROR] Could not start serving the ProviderServer: %v", err)
		os.Exit(1)
	}
}
