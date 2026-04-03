package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccVPNServer_wireguard_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVPNServerConfig_wireguard_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"name",
						"tfacc-wg-server",
					),
					resource.TestCheckResourceAttr("unifi_vpn_server.test", "enabled", "true"),
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"subnet",
						"10.100.0.1/24",
					),
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"wireguard.port",
						"51820",
					),
				),
			},
			{
				ResourceName:      "unifi_vpn_server.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"wireguard.private_key",
				},
			},
		},
	})
}

func TestAccVPNServer_wireguard_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVPNServerConfig_wireguard_update_before(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"name",
						"tfacc-wg-update",
					),
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"subnet",
						"10.101.0.1/24",
					),
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"wireguard.port",
						"51820",
					),
				),
			},
			{
				Config: testAccVPNServerConfig_wireguard_update_after(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"name",
						"tfacc-wg-update-renamed",
					),
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"subnet",
						"10.102.0.1/24",
					),
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"wireguard.port",
						"51821",
					),
				),
			},
		},
	})
}

func TestAccVPNServer_wireguard_with_dns(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVPNServerConfig_wireguard_with_dns(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_vpn_server.test", "name", "tfacc-wg-dns"),
					resource.TestCheckResourceAttr("unifi_vpn_server.test", "dns.enabled", "true"),
					resource.TestCheckResourceAttr("unifi_vpn_server.test", "dns.servers.#", "2"),
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"dns.servers.0",
						"8.8.8.8",
					),
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"dns.servers.1",
						"8.8.4.4",
					),
				),
			},
			{
				ResourceName:      "unifi_vpn_server.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"wireguard.private_key",
				},
			},
		},
	})
}

func TestAccVPNServer_wireguard_disabled(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVPNServerConfig_wireguard_disabled(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"name",
						"tfacc-wg-disabled",
					),
					resource.TestCheckResourceAttr("unifi_vpn_server.test", "enabled", "false"),
				),
			},
		},
	})
}

func TestAccVPNServer_l2tp_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVPNServerConfig_l2tp_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"name",
						"tfacc-l2tp-server",
					),
					resource.TestCheckResourceAttr("unifi_vpn_server.test", "enabled", "true"),
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"subnet",
						"10.110.0.1/24",
					),
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"l2tp.allow_weak_ciphers",
						"false",
					),
				),
			},
			{
				ResourceName:      "unifi_vpn_server.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"l2tp.pre_shared_key",
					"radiusprofile_id",
				},
			},
		},
	})
}

func TestAccVPNServer_l2tp_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVPNServerConfig_l2tp_update_before(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"name",
						"tfacc-l2tp-update",
					),
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"l2tp.allow_weak_ciphers",
						"false",
					),
				),
			},
			{
				Config: testAccVPNServerConfig_l2tp_update_after(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"name",
						"tfacc-l2tp-update-renamed",
					),
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"l2tp.allow_weak_ciphers",
						"true",
					),
				),
			},
		},
	})
}

func TestAccVPNServer_openvpn_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVPNServerConfig_openvpn_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"name",
						"tfacc-ovpn-server",
					),
					resource.TestCheckResourceAttr("unifi_vpn_server.test", "enabled", "true"),
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"subnet",
						"10.120.0.1/24",
					),
					resource.TestCheckResourceAttr("unifi_vpn_server.test", "openvpn.port", "1194"),
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"openvpn.mode",
						"server",
					),
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"openvpn.encryption_cipher",
						"AES_256_GCM",
					),
				),
			},
			{
				ResourceName:      "unifi_vpn_server.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"radiusprofile_id",
					"openvpn.server_crt",
					"openvpn.server_key",
					"openvpn.dh_key",
					"openvpn.shared_client_key",
					"openvpn.shared_client_crt",
					"openvpn.auth_key",
					"openvpn.ca_crt",
					"openvpn.ca_key",
				},
			},
		},
	})
}

func TestAccVPNServer_openvpn_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVPNServerConfig_openvpn_update_before(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"name",
						"tfacc-ovpn-update",
					),
					resource.TestCheckResourceAttr("unifi_vpn_server.test", "openvpn.port", "1194"),
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"openvpn.encryption_cipher",
						"AES_256_GCM",
					),
				),
			},
			{
				Config: testAccVPNServerConfig_openvpn_update_after(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"name",
						"tfacc-ovpn-update-renamed",
					),
					resource.TestCheckResourceAttr("unifi_vpn_server.test", "openvpn.port", "1195"),
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"openvpn.encryption_cipher",
						"AES_256_CBC",
					),
				),
			},
		},
	})
}

func TestAccVPNServer_wireguard_custom_wan(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVPNServerConfig_wireguard_custom_wan(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_vpn_server.test", "name", "tfacc-wg-wan"),
					resource.TestCheckResourceAttr("unifi_vpn_server.test", "wan.ip", "any"),
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test",
						"wan.interface",
						"wan2",
					),
				),
			},
			{
				ResourceName:      "unifi_vpn_server.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"wireguard.private_key",
				},
			},
		},
	})
}

// --- Config helper functions ---

func testAccVPNServerConfig_wireguard_basic() string {
	return `
resource "unifi_vpn_server" "test" {
  name   = "tfacc-wg-server"
  subnet = "10.100.0.1/24"

  wireguard = {
    private_key = "WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg="
  }
}
`
}

func testAccVPNServerConfig_wireguard_update_before() string {
	return `
resource "unifi_vpn_server" "test" {
  name   = "tfacc-wg-update"
  subnet = "10.101.0.1/24"

  wireguard = {
    private_key = "WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg="
    port        = 51820
  }
}
`
}

func testAccVPNServerConfig_wireguard_update_after() string {
	return `
resource "unifi_vpn_server" "test" {
  name   = "tfacc-wg-update-renamed"
  subnet = "10.102.0.1/24"

  wireguard = {
    private_key = "WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg="
    port        = 51821
  }
}
`
}

func testAccVPNServerConfig_wireguard_with_dns() string {
	return `
resource "unifi_vpn_server" "test" {
  name   = "tfacc-wg-dns"
  subnet = "10.103.0.1/24"

  dns = {
    servers = ["8.8.8.8", "8.8.4.4"]
  }

  wireguard = {
    private_key = "WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg="
  }
}
`
}

func testAccVPNServerConfig_wireguard_disabled() string {
	return `
resource "unifi_vpn_server" "test" {
  name    = "tfacc-wg-disabled"
  subnet  = "10.104.0.1/24"
  enabled = false

  wireguard = {
    private_key = "WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg="
  }
}
`
}

func testAccVPNServerConfig_l2tp_basic() string {
	return `
resource "unifi_radius_profile" "test" {
  name = "tfacc-l2tp-radius"

  auth_server {
    ip       = "192.168.1.100"
    port     = 1812
    secret   = "radius-secret"
  }
}

resource "unifi_vpn_server" "test" {
  name             = "tfacc-l2tp-server"
  subnet           = "10.110.0.1/24"
  radiusprofile_id = unifi_radius_profile.test.id

  l2tp = {
    pre_shared_key = "tfacc-l2tp-psk-secret"
  }
}
`
}

func testAccVPNServerConfig_l2tp_update_before() string {
	return `
resource "unifi_radius_profile" "test" {
  name = "tfacc-l2tp-upd-radius"

  auth_server {
    ip       = "192.168.1.100"
    port     = 1812
    secret   = "radius-secret"
  }
}

resource "unifi_vpn_server" "test" {
  name             = "tfacc-l2tp-update"
  subnet           = "10.111.0.1/24"
  radiusprofile_id = unifi_radius_profile.test.id

  l2tp = {
    pre_shared_key     = "tfacc-l2tp-psk-secret"
    allow_weak_ciphers = false
  }
}
`
}

func testAccVPNServerConfig_l2tp_update_after() string {
	return `
resource "unifi_radius_profile" "test" {
  name = "tfacc-l2tp-upd-radius"

  auth_server {
    ip       = "192.168.1.100"
    port     = 1812
    secret   = "radius-secret"
  }
}

resource "unifi_vpn_server" "test" {
  name             = "tfacc-l2tp-update-renamed"
  subnet           = "10.111.0.1/24"
  radiusprofile_id = unifi_radius_profile.test.id

  l2tp = {
    pre_shared_key     = "tfacc-l2tp-psk-secret"
    allow_weak_ciphers = true
  }
}
`
}

func testAccVPNServerConfig_openvpn_basic() string {
	return `
resource "unifi_radius_profile" "test" {
  name = "tfacc-ovpn-radius"

  auth_server {
    ip       = "192.168.1.100"
    port     = 1812
    secret   = "radius-secret"
  }
}

resource "unifi_vpn_server" "test" {
  name              = "tfacc-ovpn-server"
  subnet            = "10.120.0.1/24"
  radiusprofile_id  = unifi_radius_profile.test.id

  openvpn = {}
}
`
}

func testAccVPNServerConfig_openvpn_update_before() string {
	return `
resource "unifi_radius_profile" "test" {
  name = "tfacc-ovpn-upd-radius"

  auth_server {
    ip       = "192.168.1.100"
    port     = 1812
    secret   = "radius-secret"
  }
}

resource "unifi_vpn_server" "test" {
  name              = "tfacc-ovpn-update"
  subnet            = "10.121.0.1/24"
  radiusprofile_id  = unifi_radius_profile.test.id

  openvpn = {
    port              = 1194
    encryption_cipher = "AES_256_GCM"
  }
}
`
}

func testAccVPNServerConfig_openvpn_update_after() string {
	return `
resource "unifi_radius_profile" "test" {
  name = "tfacc-ovpn-upd-radius"

  auth_server {
    ip       = "192.168.1.100"
    port     = 1812
    secret   = "radius-secret"
  }
}

resource "unifi_vpn_server" "test" {
  name              = "tfacc-ovpn-update-renamed"
  subnet            = "10.121.0.1/24"
  radiusprofile_id  = unifi_radius_profile.test.id

  openvpn = {
    port              = 1195
    encryption_cipher = "AES_256_CBC"
  }
}
`
}

func testAccVPNServerConfig_wireguard_custom_wan() string {
	return `
resource "unifi_vpn_server" "test" {
  name   = "tfacc-wg-wan"
  subnet = "10.105.0.1/24"

  wan = {
    interface = "wan2"
  }

  wireguard = {
    private_key = "WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg="
  }
}
`
}
