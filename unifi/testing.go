package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var providerFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"unifi": providerserver.NewProtocol6WithError(New()),
}

var testAccProtoV6ProviderFactories = providerFactories

func preCheck(t *testing.T) {
	if v := testing.Verbose(); v {
		t.Log("Running pre-checks for Unifi provider")
	}
}
