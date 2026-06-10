package unifi

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccWireguardPeer_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWireguardPeerConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_wireguard_peer.test",
						"name",
						"tfacc-wg-peer",
					),
					resource.TestCheckResourceAttr(
						"unifi_wireguard_peer.test",
						"interface_ip",
						"192.0.2.10",
					),
					resource.TestCheckResourceAttr(
						"unifi_wireguard_peer.test",
						"public_key",
						"ZmFrZS10ZXN0LXdpcmVndWFyZC1wdWJrZXkAAAAAAAA=",
					),
					resource.TestCheckResourceAttr(
						"unifi_wireguard_peer.test",
						"allowed_ips.#",
						"0",
					),
					resource.TestCheckResourceAttrSet("unifi_wireguard_peer.test", "id"),
					resource.TestCheckResourceAttrPair(
						"unifi_wireguard_peer.test",
						"network_id",
						"unifi_vpn_server.test",
						"id",
					),
				),
			},
			{
				ResourceName:      "unifi_wireguard_peer.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["unifi_wireguard_peer.test"]
					if !ok {
						return "", fmt.Errorf("resource not found in state")
					}
					return rs.Primary.Attributes["network_id"] + ":" + rs.Primary.ID, nil
				},
			},
		},
	})
}

func TestAccWireguardPeer_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWireguardPeerConfig_basic(),
				Check: resource.TestCheckResourceAttr(
					"unifi_wireguard_peer.test",
					"interface_ip",
					"192.0.2.10",
				),
			},
			{
				Config: testAccWireguardPeerConfig_update(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_wireguard_peer.test",
						"interface_ip",
						"192.0.2.20",
					),
					resource.TestCheckResourceAttr(
						"unifi_wireguard_peer.test",
						"allowed_ips.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_wireguard_peer.test",
						"allowed_ips.0",
						"198.51.100.0/24",
					),
				),
			},
		},
	})
}

func testAccWireguardPeerConfig_basic() string {
	return `
resource "unifi_vpn_server" "test" {
  name   = "tfacc-wg-peer-server"
  subnet = "192.0.2.1/24"

  wireguard = {
    private_key = "WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg="
    port        = 51822
  }
}

resource "unifi_wireguard_peer" "test" {
  network_id   = unifi_vpn_server.test.id
  name         = "tfacc-wg-peer"
  interface_ip = "192.0.2.10"
  public_key   = "ZmFrZS10ZXN0LXdpcmVndWFyZC1wdWJrZXkAAAAAAAA="
}
`
}

func testAccWireguardPeerConfig_update() string {
	return `
resource "unifi_vpn_server" "test" {
  name   = "tfacc-wg-peer-server"
  subnet = "192.0.2.1/24"

  wireguard = {
    private_key = "WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg="
    port        = 51822
  }
}

resource "unifi_wireguard_peer" "test" {
  network_id   = unifi_vpn_server.test.id
  name         = "tfacc-wg-peer"
  interface_ip = "192.0.2.20"
  public_key   = "ZmFrZS10ZXN0LXdpcmVndWFyZC1wdWJrZXkAAAAAAAA="
  allowed_ips  = ["198.51.100.0/24"]
}
`
}
