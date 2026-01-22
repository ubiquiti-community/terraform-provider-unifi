package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccVPNClient_file_mode(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVPNClientConfig_file_mode(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_vpn_client.test",
						"name",
						"test-wireguard-vpn",
					),
					resource.TestCheckResourceAttr("unifi_vpn_client.test", "enabled", "true"),
					resource.TestCheckResourceAttr(
						"unifi_vpn_client.test",
						"default_route",
						"true",
					),
					resource.TestCheckResourceAttr("unifi_vpn_client.test", "pull_dns", "false"),
					resource.TestCheckResourceAttr(
						"unifi_vpn_client.test",
						"wireguard_config.mode",
						"file",
					),
					resource.TestCheckResourceAttr(
						"unifi_vpn_client.test",
						"wireguard_config.interface",
						"wan",
					),
					resource.TestCheckResourceAttrSet(
						"unifi_vpn_client.test",
						"wireguard_config.configuration_file",
					),
					resource.TestCheckResourceAttrSet(
						"unifi_vpn_client.test",
						"wireguard_config.configuration_filename",
					),
				),
			},
			{
				ResourceName:      "unifi_vpn_client.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"wireguard.private_key",
					"wireguard.configuration.content",
					"wireguard.preshared_key",
				},
			},
		},
	})
}

func TestAccVPNClient_manual_mode(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVPNClientConfig_manual_mode(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_vpn_client.test",
						"name",
						"test-wireguard-manual",
					),
					resource.TestCheckResourceAttr("unifi_vpn_client.test", "enabled", "true"),
					resource.TestCheckResourceAttr(
						"unifi_vpn_client.test",
						"default_route",
						"false",
					),
					resource.TestCheckResourceAttr("unifi_vpn_client.test", "pull_dns", "true"),
					resource.TestCheckResourceAttr(
						"unifi_vpn_client.test",
						"wireguard.peer.ip",
						"192.0.2.1",
					),
					resource.TestCheckResourceAttr(
						"unifi_vpn_client.test",
						"wireguard.peer.port",
						"51820",
					),
					resource.TestCheckResourceAttrSet(
						"unifi_vpn_client.test",
						"wireguard.peer.public_key",
					),
				),
			},
			{
				ResourceName:      "unifi_vpn_client.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"wireguard.private_key",
					"wireguard.peer.public_key",
					"wireguard.preshared_key",
				},
			},
		},
	})
}

func TestAccVPNClient_with_preshared_key(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVPNClientConfig_with_preshared_key(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_vpn_client.test",
						"name",
						"test-wireguard-psk",
					),
					resource.TestCheckResourceAttr(
						"unifi_vpn_client.test",
						"wireguard.preshared_key_enabled",
						"true",
					),
					resource.TestCheckResourceAttrSet(
						"unifi_vpn_client.test",
						"wireguard.preshared_key",
					),
				),
			},
		},
	})
}

func testAccVPNClientConfig_file_mode() string {
	return `
resource "unifi_vpn_client" "test" {
  name          = "test-wireguard-vpn"
  enabled       = true
  subnet        = "10.0.0.2/24"
  default_route = true
  pull_dns      = false

  wireguard = {
    private_key = "WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg="
    interface   = "wan"
    
    configuration = {
      content  = "W0ludGVyZmFjZV0KUHJpdmF0ZUtleSA9IFdQaUJhL0FrMVcrOFNwOEw1eXZieWhIZVJPMm81a0p2aWhxMlZ0SitrRmc9CkFkZHJlc3MgPSAxMC4wLjAuMi8yNAoKW1BlZXJdClB1YmxpY0tleSA9IDdCKzJaM29kUGJETnNmVnIrRjhpbnZqNi9tQktMVmFvbE9IWFpvQ2FCQTA9CkVuZHBvaW50ID0gMTkyLjAuMi4xOjUxODIwCg=="
      filename = "wireguard.conf"
    }
  }
}
`
}

func testAccVPNClientConfig_manual_mode() string {
	return `
resource "unifi_vpn_client" "test" {
  name          = "test-wireguard-manual"
  enabled       = true
  subnet        = "10.0.1.2/24"
  default_route = false
  pull_dns      = true

  wireguard = {
    private_key = "WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg="
    interface   = "wan"
    dns_servers = ["8.8.8.8", "8.8.4.4"]
    
    peer = {
      ip         = "192.0.2.1"
      port       = 51820
      public_key = "7B+2Z3odPbDNsfVr+F8invj6/mBKLVaolOHXZoCaBA0="
    }
  }
}
`
}

func testAccVPNClientConfig_with_preshared_key() string {
	return `
resource "unifi_vpn_client" "test" {
  name          = "test-wireguard-psk"
  enabled       = true
  subnet        = "10.0.2.2/24"
  default_route = true
  pull_dns      = false

  wireguard = {
    private_key            = "WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg="
    preshared_key_enabled  = true
    preshared_key          = "F3JcsRyn9Hywwyhl4EznlV4ZThatbB5Hi4U9b3emM+g="
    interface              = "wan"
    dns_servers            = ["8.8.8.8", "8.8.4.4"]
    
    peer = {
      ip         = "192.0.2.1"
      port       = 51820
      public_key = "7B+2Z3odPbDNsfVr+F8invj6/mBKLVaolOHXZoCaBA0="
    }
  }
}
`
}
