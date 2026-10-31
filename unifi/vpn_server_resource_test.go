package unifi

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/cidrtypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/ubiquiti-community/go-unifi/unifi"
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
			{
				ResourceName:    "unifi_vpn_server.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
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
					"openvpn.server.crt",
					"openvpn.server.key",
					"openvpn.dh_key",
					"openvpn.shared_client.crt",
					"openvpn.shared_client.key",
					"openvpn.auth_key",
					"openvpn.ca.crt",
					"openvpn.ca.key",
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

func TestAccVPNServer_wireguard_write_only_private_key(t *testing.T) {
	const pub1 = "7B+2Z3odPbDNsfVr+F8invj6/mBKLVaolOHXZoCaBA0="
	noKeyInState := func(extra ...resource.TestCheckFunc) resource.TestCheckFunc {
		return resource.ComposeTestCheckFunc(append([]resource.TestCheckFunc{
			resource.TestCheckNoResourceAttr("unifi_vpn_server.test", "wireguard.private_key"),
			resource.TestCheckNoResourceAttr("unifi_vpn_server.test", "wireguard.private_key_wo"),
		}, extra...)...)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_11_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccVPNServerConfig_wireguard_write_only_private_key(
					1, "WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg=", 51820,
				),
				Check: noKeyInState(
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test", "wireguard.private_key_wo_version", "1"),
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test", "wireguard.public_key", pub1),
				),
			},
			{
				// Another change without a version bump keeps the key.
				Config: testAccVPNServerConfig_wireguard_write_only_private_key(
					1, "WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg=", 51821,
				),
				Check: noKeyInState(
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test", "wireguard.port", "51821"),
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test", "wireguard.public_key", pub1),
				),
			},
			{
				// A version bump rotates the key.
				Config: testAccVPNServerConfig_wireguard_write_only_private_key(
					2, "uGEwDKZ2Hf2s2Dg59c9K+qYzJEBN5s8fNWVTxZx9kUo=", 51821,
				),
				Check: noKeyInState(
					resource.TestCheckResourceAttr(
						"unifi_vpn_server.test", "wireguard.private_key_wo_version", "2"),
					resource.TestCheckResourceAttrWith(
						"unifi_vpn_server.test", "wireguard.public_key",
						func(v string) error {
							if v == "" || v == pub1 {
								return fmt.Errorf("public_key not rotated: %q", v)
							}
							return nil
						}),
				),
			},
		},
	})
}

// --- Config helper functions ---

func testAccVPNServerConfig_wireguard_write_only_private_key(
	version int,
	privateKey string,
	port int,
) string {
	return fmt.Sprintf(`
resource "unifi_vpn_server" "test" {
  name   = "tfacc-wg-server-wo"
  subnet = "10.100.2.1/24"

  wireguard = {
    private_key_wo         = %q
    private_key_wo_version = %d
    port                   = %d
  }
}
`, privateKey, version, port)
}

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

// TestGenerateWireGuardPrivateKey verifies the provider generates a valid
// base64 32-byte Curve25519 private key when the user omits one (#255).
func TestGenerateWireGuardPrivateKey(t *testing.T) {
	k1, err := generateWireGuardPrivateKey()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	raw, err := base64.StdEncoding.DecodeString(k1)
	if err != nil {
		t.Fatalf("key is not valid base64: %v", err)
	}
	if len(raw) != 32 {
		t.Errorf("key length = %d bytes, want 32", len(raw))
	}
	// Curve25519 clamping must be applied.
	if raw[0]&7 != 0 || raw[31]&128 != 0 || raw[31]&64 == 0 {
		t.Errorf("key is not WireGuard-clamped: %x", raw)
	}
	// Keys must be unique per call.
	k2, _ := generateWireGuardPrivateKey()
	if k1 == k2 {
		t.Error("two generated keys are identical")
	}
}

func TestNewVPNServerResource(t *testing.T) {
	r := NewVPNServerResource()
	if r == nil {
		t.Fatal("NewVPNServerResource() returned nil")
	}
	if _, ok := r.(fwresource.ResourceWithConfigure); !ok {
		t.Error("expected ResourceWithConfigure interface")
	}
	if _, ok := r.(fwresource.ResourceWithImportState); !ok {
		t.Error("expected ResourceWithImportState interface")
	}
}

func TestNewVPNServerListResource(t *testing.T) {
	r := NewVPNServerListResource()
	if r == nil {
		t.Fatal("NewVPNServerListResource() returned nil")
	}
}

func Test_vpnServerDNSModel_AttributeTypes(t *testing.T) {
	m := vpnServerDNSModel{}
	got := m.AttributeTypes()
	for _, key := range []string{"enabled", "servers"} {
		if _, ok := got[key]; !ok {
			t.Errorf("AttributeTypes() missing key %q", key)
		}
	}
	if got["enabled"] != types.BoolType {
		t.Errorf("enabled type = %v, want BoolType", got["enabled"])
	}
}

func Test_vpnServerWANModel_AttributeTypes(t *testing.T) {
	m := vpnServerWANModel{}
	got := m.AttributeTypes()
	for _, key := range []string{"ip", "interface"} {
		if _, ok := got[key]; !ok {
			t.Errorf("AttributeTypes() missing key %q", key)
		}
	}
}

func Test_vpnServerWireguardModel_AttributeTypes(t *testing.T) {
	m := vpnServerWireguardModel{}
	got := m.AttributeTypes()
	for _, key := range []string{
		"private_key", "private_key_wo", "private_key_wo_version", "public_key", "port",
	} {
		if _, ok := got[key]; !ok {
			t.Errorf("AttributeTypes() missing key %q", key)
		}
	}
	if got["port"] != types.Int64Type {
		t.Errorf("port type = %v, want Int64Type", got["port"])
	}
}

func Test_vpnServerL2TPModel_AttributeTypes(t *testing.T) {
	m := vpnServerL2TPModel{}
	got := m.AttributeTypes()
	for _, key := range []string{"allow_weak_ciphers", "pre_shared_key"} {
		if _, ok := got[key]; !ok {
			t.Errorf("AttributeTypes() missing key %q", key)
		}
	}
}

func Test_vpnServerOpenVPNModel_AttributeTypes(t *testing.T) {
	m := vpnServerOpenVPNModel{}
	got := m.AttributeTypes()
	for _, key := range []string{"port", "mode", "encryption_cipher", "server", "dh_key", "shared_client", "auth_key", "ca"} {
		if _, ok := got[key]; !ok {
			t.Errorf("AttributeTypes() missing key %q", key)
		}
	}
	for _, flat := range []string{"server_crt", "server_key", "shared_client_crt", "shared_client_key", "ca_crt", "ca_key"} {
		if _, ok := got[flat]; ok {
			t.Errorf("AttributeTypes() still has flat key %q", flat)
		}
	}
}

func Test_vpnServerCertPairModel_AttributeTypes(t *testing.T) {
	m := vpnServerCertPairModel{}
	got := m.AttributeTypes()
	for _, key := range []string{"crt", "key"} {
		if _, ok := got[key]; !ok {
			t.Errorf("AttributeTypes() missing key %q", key)
		}
	}
}

func Test_vpnServerResource_Metadata(t *testing.T) {
	tests := []struct {
		providerTypeName, wantTypeName string
	}{
		{"unifi", "unifi_vpn_server"},
		{"test", "test_vpn_server"},
	}
	for _, tt := range tests {
		t.Run(tt.providerTypeName, func(t *testing.T) {
			r := &vpnServerResource{}
			resp := &fwresource.MetadataResponse{}
			r.Metadata(
				context.Background(),
				fwresource.MetadataRequest{ProviderTypeName: tt.providerTypeName},
				resp,
			)
			if resp.TypeName != tt.wantTypeName {
				t.Errorf("TypeName = %q, want %q", resp.TypeName, tt.wantTypeName)
			}
		})
	}
}

func Test_vpnServerResource_IdentitySchema(t *testing.T) {
	r := &vpnServerResource{}
	resp := &fwresource.IdentitySchemaResponse{}
	r.IdentitySchema(context.Background(), fwresource.IdentitySchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("IdentitySchema() produced errors: %v", resp.Diagnostics)
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Error("IdentitySchema missing 'id' attribute")
	}
	if _, ok := resp.IdentitySchema.Attributes["site"]; !ok {
		t.Error("IdentitySchema missing 'site' attribute")
	}
}

func Test_vpnServerResource_Schema(t *testing.T) {
	r := &vpnServerResource{}
	resp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() produced errors: %v", resp.Diagnostics)
	}
	for _, attr := range []string{"id", "site", "name", "enabled", "subnet", "dns", "wan", "radiusprofile_id", "wireguard", "l2tp", "openvpn"} {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("missing attribute %q", attr)
		}
	}
}

func Test_vpnServerResource_Configure(t *testing.T) {
	tests := []struct {
		name      string
		data      any
		wantError bool
	}{
		{"nil", nil, false},
		{"wrong type", "wrong", true},
		{"correct client", &Client{Site: "default"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &vpnServerResource{}
			resp := &fwresource.ConfigureResponse{}
			r.Configure(
				context.Background(),
				fwresource.ConfigureRequest{ProviderData: tt.data},
				resp,
			)
			if tt.wantError && !resp.Diagnostics.HasError() {
				t.Error("expected error")
			}
			if !tt.wantError && resp.Diagnostics.HasError() {
				t.Errorf("unexpected error: %v", resp.Diagnostics)
			}
		})
	}
}

// vpnServerImportHarness builds the empty state and null identity containers
// the framework hands to ImportState, so the import logic can be unit tested
// without a live controller.
func vpnServerImportHarness(t *testing.T) (tfsdk.State, tfsdk.ResourceIdentity) {
	t.Helper()
	ctx := context.Background()
	r := &vpnServerResource{}

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}

	var idResp fwresource.IdentitySchemaResponse
	r.IdentitySchema(ctx, fwresource.IdentitySchemaRequest{}, &idResp)
	if idResp.Diagnostics.HasError() {
		t.Fatalf("identity schema: %v", idResp.Diagnostics)
	}

	state := tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
	}
	identity := tfsdk.ResourceIdentity{
		Schema: idResp.IdentitySchema,
		Raw:    tftypes.NewValue(idResp.IdentitySchema.Type().TerraformType(ctx), nil),
	}
	return state, identity
}

// vpnServerIdentityValue builds a populated identity container for
// identity-based import requests. Pass "" to leave site null.
func vpnServerIdentityValue(t *testing.T, id, site string) tfsdk.ResourceIdentity {
	t.Helper()
	ctx := context.Background()
	_, identity := vpnServerImportHarness(t)

	siteVal := tftypes.NewValue(tftypes.String, nil)
	if site != "" {
		siteVal = tftypes.NewValue(tftypes.String, site)
	}
	identity.Raw = tftypes.NewValue(
		identity.Schema.Type().TerraformType(ctx),
		map[string]tftypes.Value{
			"id":   tftypes.NewValue(tftypes.String, id),
			"site": siteVal,
		},
	)
	return identity
}

func Test_vpnServerResource_ImportState(t *testing.T) {
	ctx := context.Background()
	const oid = "0123456789abcdef01234567"

	newRes := func() *vpnServerResource {
		return &vpnServerResource{client: &Client{Site: "default"}}
	}

	getString := func(t *testing.T, get func(context.Context, path.Path, any) diag.Diagnostics, p path.Path) types.String {
		t.Helper()
		var v types.String
		if d := get(ctx, p, &v); d.HasError() {
			t.Fatalf("get %s: %v", p, d)
		}
		return v
	}

	t.Run("identity import defaults omitted site to provider site", func(t *testing.T) {
		r := newRes()
		state, _ := vpnServerImportHarness(t)
		reqIdentity := vpnServerIdentityValue(t, oid, "")
		respIdentity := vpnServerIdentityValue(t, oid, "")
		resp := &fwresource.ImportStateResponse{State: state, Identity: &respIdentity}

		r.ImportState(ctx, fwresource.ImportStateRequest{ID: "", Identity: &reqIdentity}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diags: %v", resp.Diagnostics)
		}
		if got := getString(t, resp.State.GetAttribute, path.Root("id")); got.ValueString() != oid {
			t.Errorf("state id = %v, want %s", got, oid)
		}
		if got := getString(
			t,
			resp.State.GetAttribute,
			path.Root("site"),
		); got.ValueString() != "default" {
			t.Errorf("state site = %v, want default", got)
		}
		if got := getString(
			t,
			resp.Identity.GetAttribute,
			path.Root("site"),
		); got.ValueString() != "default" {
			t.Errorf("identity site = %v, want default", got)
		}
	})

	t.Run("string import by id mirrors identity", func(t *testing.T) {
		r := newRes()
		state, respIdentity := vpnServerImportHarness(t)
		resp := &fwresource.ImportStateResponse{State: state, Identity: &respIdentity}

		r.ImportState(ctx, fwresource.ImportStateRequest{ID: oid}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diags: %v", resp.Diagnostics)
		}
		if got := getString(t, resp.State.GetAttribute, path.Root("id")); got.ValueString() != oid {
			t.Errorf("state id = %v, want %s", got, oid)
		}
		if got := getString(
			t,
			resp.Identity.GetAttribute,
			path.Root("id"),
		); got.ValueString() != oid {
			t.Errorf("identity id = %v, want %s", got, oid)
		}
	})

	t.Run("string import with site prefix", func(t *testing.T) {
		r := newRes()
		state, respIdentity := vpnServerImportHarness(t)
		resp := &fwresource.ImportStateResponse{State: state, Identity: &respIdentity}

		r.ImportState(ctx, fwresource.ImportStateRequest{ID: "other:" + oid}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diags: %v", resp.Diagnostics)
		}
		if got := getString(
			t,
			resp.State.GetAttribute,
			path.Root("site"),
		); got.ValueString() != "other" {
			t.Errorf("state site = %v, want other", got)
		}
		if got := getString(
			t,
			resp.Identity.GetAttribute,
			path.Root("site"),
		); got.ValueString() != "other" {
			t.Errorf("identity site = %v, want other", got)
		}
	})

	t.Run("string import by name sets name only", func(t *testing.T) {
		r := newRes()
		state, respIdentity := vpnServerImportHarness(t)
		resp := &fwresource.ImportStateResponse{State: state, Identity: &respIdentity}

		r.ImportState(ctx, fwresource.ImportStateRequest{ID: "name=my-server"}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diags: %v", resp.Diagnostics)
		}
		if got := getString(
			t,
			resp.State.GetAttribute,
			path.Root("name"),
		); got.ValueString() != "my-server" {
			t.Errorf("state name = %v, want my-server", got)
		}
	})
}

func Test_vpnServerResource_modelToNetwork(t *testing.T) {
	ctx := context.Background()

	t.Run("missing vpn type returns error", func(t *testing.T) {
		r := &vpnServerResource{}
		model := &vpnServerResourceModel{
			Name:      types.StringValue("test"),
			Enabled:   types.BoolValue(true),
			Subnet:    cidrtypes.NewIPv4PrefixValue("10.100.0.1/24"),
			Wireguard: types.ObjectNull(vpnServerWireguardModel{}.AttributeTypes()),
			L2TP:      types.ObjectNull(vpnServerL2TPModel{}.AttributeTypes()),
			OpenVPN:   types.ObjectNull(vpnServerOpenVPNModel{}.AttributeTypes()),
			DNS:       types.ObjectNull(vpnServerDNSModel{}.AttributeTypes()),
			WAN:       types.ObjectNull(vpnServerWANModel{}.AttributeTypes()),
		}
		got, diags := r.modelToNetwork(ctx, model)
		if !diags.HasError() {
			t.Error("expected error for missing VPN type")
		}
		if got != nil {
			t.Error("expected nil network for error case")
		}
	})

	t.Run("wireguard model sets vpn type", func(t *testing.T) {
		r := &vpnServerResource{}
		port := int64(51820)
		privKey := "WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg="
		wgModel := vpnServerWireguardModel{
			PrivateKey: types.StringValue(privKey),
			PublicKey:  types.StringNull(),
			Port:       types.Int64Value(port),
		}
		wgObj, d := types.ObjectValueFrom(ctx, vpnServerWireguardModel{}.AttributeTypes(), wgModel)
		if d.HasError() {
			t.Fatalf("building wireguard object: %v", d)
		}
		model := &vpnServerResourceModel{
			Name:      types.StringValue("wg-server"),
			Enabled:   types.BoolValue(true),
			Subnet:    cidrtypes.NewIPv4PrefixValue("10.100.0.1/24"),
			Wireguard: wgObj,
			L2TP:      types.ObjectNull(vpnServerL2TPModel{}.AttributeTypes()),
			OpenVPN:   types.ObjectNull(vpnServerOpenVPNModel{}.AttributeTypes()),
			DNS:       types.ObjectNull(vpnServerDNSModel{}.AttributeTypes()),
			WAN:       types.ObjectNull(vpnServerWANModel{}.AttributeTypes()),
		}
		got, diags := r.modelToNetwork(ctx, model)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if got == nil {
			t.Fatal("expected non-nil network")
		}
		if got.VPNType == nil || *got.VPNType != "wireguard-server" {
			t.Errorf("VPNType = %v, want wireguard-server", got.VPNType)
		}
		if got.WireguardPrivateKey == nil || *got.WireguardPrivateKey != privKey {
			t.Errorf("WireguardPrivateKey = %v, want %q", got.WireguardPrivateKey, privKey)
		}
		if got.LocalPort == nil || *got.LocalPort != port {
			t.Errorf("LocalPort = %v, want %d", got.LocalPort, port)
		}
	})

	t.Run("l2tp model sets vpn type", func(t *testing.T) {
		r := &vpnServerResource{}
		l2tpModel := vpnServerL2TPModel{
			AllowWeakCiphers: types.BoolValue(false),
			PreSharedKey:     types.StringValue("my-psk"),
		}
		l2tpObj, d := types.ObjectValueFrom(ctx, vpnServerL2TPModel{}.AttributeTypes(), l2tpModel)
		if d.HasError() {
			t.Fatalf("building l2tp object: %v", d)
		}
		model := &vpnServerResourceModel{
			Name:      types.StringValue("l2tp-server"),
			Enabled:   types.BoolValue(true),
			Subnet:    cidrtypes.NewIPv4PrefixValue("10.110.0.1/24"),
			Wireguard: types.ObjectNull(vpnServerWireguardModel{}.AttributeTypes()),
			L2TP:      l2tpObj,
			OpenVPN:   types.ObjectNull(vpnServerOpenVPNModel{}.AttributeTypes()),
			DNS:       types.ObjectNull(vpnServerDNSModel{}.AttributeTypes()),
			WAN:       types.ObjectNull(vpnServerWANModel{}.AttributeTypes()),
		}
		got, diags := r.modelToNetwork(ctx, model)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if got.VPNType == nil || *got.VPNType != "l2tp-server" {
			t.Errorf("VPNType = %v, want l2tp-server", got.VPNType)
		}
		if got.IPSecPreSharedKey == nil || *got.IPSecPreSharedKey != "my-psk" {
			t.Errorf("IPSecPreSharedKey = %v, want my-psk", got.IPSecPreSharedKey)
		}
	})
}

func Test_vpnServerResource_networkToModel(t *testing.T) {
	ctx := context.Background()

	t.Run("wireguard network populates wireguard block", func(t *testing.T) {
		r := &vpnServerResource{}
		vpnType := "wireguard-server"
		name := "wg-test"
		subnet := "10.100.0.1/24"
		port := int64(51820)
		privKey := "WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg="
		network := &unifi.Network{
			ID:                  "net-123",
			Name:                &name,
			Enabled:             new(true),
			IPSubnet:            &subnet,
			VPNType:             &vpnType,
			WireguardPrivateKey: &privKey,
			LocalPort:           &port,
		}
		var model vpnServerResourceModel
		diags := r.networkToModel(ctx, network, &model, "default", &vpnServerResourceModel{})
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if model.ID.ValueString() != "net-123" {
			t.Errorf("ID = %q, want net-123", model.ID.ValueString())
		}
		if model.Name.ValueString() != "wg-test" {
			t.Errorf("Name = %q, want wg-test", model.Name.ValueString())
		}
		if !model.Enabled.ValueBool() {
			t.Error("Enabled should be true")
		}
		if model.Wireguard.IsNull() {
			t.Fatal("Wireguard block should not be null")
		}
		var wg vpnServerWireguardModel
		if d := model.Wireguard.As(
			ctx,
			&wg,
			struct{ UnhandledNullAsEmpty, UnhandledUnknownAsEmpty bool }{},
		); d.HasError() {
			t.Fatalf("reading wireguard: %v", d)
		}
		if wg.PrivateKey.ValueString() != privKey {
			t.Errorf("PrivateKey = %q, want %q", wg.PrivateKey.ValueString(), privKey)
		}
		if wg.Port.ValueInt64() != port {
			t.Errorf("Port = %d, want %d", wg.Port.ValueInt64(), port)
		}
		// L2TP and OpenVPN should be null for a wireguard server
		if !model.L2TP.IsNull() {
			t.Error("L2TP should be null for wireguard server")
		}
		if !model.OpenVPN.IsNull() {
			t.Error("OpenVPN should be null for wireguard server")
		}
	})

	t.Run("l2tp network preserves psk from prior state", func(t *testing.T) {
		r := &vpnServerResource{}
		vpnType := "l2tp-server"
		name := "l2tp-test"
		subnet := "10.110.0.1/24"
		network := &unifi.Network{
			ID:       "net-456",
			Name:     &name,
			Enabled:  new(true),
			IPSubnet: &subnet,
			VPNType:  &vpnType,
			// IPSecPreSharedKey is nil (API does not return it on read)
		}

		// Prior state has the PSK
		priorL2TP := vpnServerL2TPModel{
			AllowWeakCiphers: types.BoolValue(false),
			PreSharedKey:     types.StringValue("stored-psk"),
		}
		priorL2TPObj, d := types.ObjectValueFrom(
			ctx,
			vpnServerL2TPModel{}.AttributeTypes(),
			priorL2TP,
		)
		if d.HasError() {
			t.Fatalf("building prior l2tp: %v", d)
		}
		priorState := &vpnServerResourceModel{
			L2TP: priorL2TPObj,
		}

		var model vpnServerResourceModel
		diags := r.networkToModel(ctx, network, &model, "default", priorState)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if model.L2TP.IsNull() {
			t.Fatal("L2TP block should not be null")
		}
		var l2tp vpnServerL2TPModel
		if d := model.L2TP.As(
			ctx,
			&l2tp,
			struct{ UnhandledNullAsEmpty, UnhandledUnknownAsEmpty bool }{},
		); d.HasError() {
			t.Fatalf("reading l2tp: %v", d)
		}
		// PSK should be preserved from prior state since the API doesn't return it
		if l2tp.PreSharedKey.ValueString() != "stored-psk" {
			t.Errorf(
				"PreSharedKey = %q, want stored-psk (preserved from prior state)",
				l2tp.PreSharedKey.ValueString(),
			)
		}
	})
}

func Test_vpnServerResource_ListResourceConfigSchema(t *testing.T) {
	r := &vpnServerResource{}
	resp := &fwlist.ListResourceSchemaResponse{}
	r.ListResourceConfigSchema(context.Background(), fwlist.ListResourceSchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("ListResourceConfigSchema() produced errors: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["site"]; !ok {
		t.Error("ListResourceConfigSchema missing 'site' attribute")
	}
}

func Test_generateWireGuardPrivateKey(t *testing.T) {
	got, err := generateWireGuardPrivateKey()
	if err != nil {
		t.Fatalf("generateWireGuardPrivateKey() error = %v", err)
	}
	if got == "" {
		t.Error("generateWireGuardPrivateKey() returned empty string")
	}
	// Must be valid base64
	raw, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		t.Fatalf("result is not valid base64: %v", err)
	}
	if len(raw) != 32 {
		t.Errorf("key length = %d bytes, want 32", len(raw))
	}
}

func TestAccVPNServerList_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccVPNServerConfig_wireguard_basic(),
			},
			{
				Query: true,
				Config: `
					provider "unifi" {}
					list "unifi_vpn_server" "test" {
						provider = unifi
						config {
							filter {
								name  = "name"
								value = "tfacc-wg-server"
						  }
					  }
					}
				`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLengthAtLeast("unifi_vpn_server.test", 1),
				},
			},
		},
	})
}

// TestVPNServerUpgradeState_v0NestsOpenVPNCertPairs guards the v0 -> v1 schema
// upgrade: the flat openvpn.{ca,server,shared_client}_{crt,key} attributes move
// into nested crt/key objects, while every other attribute (including the
// deliberately un-nested dh_key and auth_key) passes through.
func TestVPNServerUpgradeState_v0NestsOpenVPNCertPairs(t *testing.T) {
	ctx := context.Background()
	r := &vpnServerResource{}

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	prior := []byte(`{
		"id": "vpn-1", "site": "default", "name": "ovpn", "enabled": true,
		"subnet": "10.120.0.1/24",
		"dns": {"enabled": false, "servers": null},
		"wan": {"ip": "any", "interface": "wan"},
		"radiusprofile_id": null, "wireguard": null, "l2tp": null,
		"openvpn": {
			"port": 1194, "mode": "server", "encryption_cipher": "AES_256_GCM",
			"server_crt": "SERVER-CRT", "server_key": "SERVER-KEY",
			"dh_key": "DH-PARAMS",
			"shared_client_key": "CLIENT-KEY", "shared_client_crt": "CLIENT-CRT",
			"auth_key": "TA-KEY",
			"ca_crt": "CA-CRT", "ca_key": null
		},
		"timeouts": null
	}`)

	up, ok := r.UpgradeState(ctx)[0]
	if !ok {
		t.Fatal("no upgrader registered for schema version 0")
	}
	resp := &fwresource.UpgradeStateResponse{}
	up.StateUpgrader(ctx, fwresource.UpgradeStateRequest{
		RawState: &tfprotov6.RawState{JSON: prior},
	}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade failed: %v", resp.Diagnostics)
	}
	val, err := resp.DynamicValue.Unmarshal(schemaType)
	if err != nil {
		t.Fatalf("unmarshal upgraded value: %v", err)
	}

	var root map[string]tftypes.Value
	if err := val.As(&root); err != nil {
		t.Fatalf("as object: %v", err)
	}
	obj := func(v tftypes.Value, name string) map[string]tftypes.Value {
		t.Helper()
		var m map[string]tftypes.Value
		if err := v.As(&m); err != nil {
			t.Fatalf("%s: as object: %v (value %v)", name, err, v)
		}
		return m
	}
	str := func(v tftypes.Value, name, want string) {
		t.Helper()
		var s string
		if err := v.As(&s); err != nil || s != want {
			t.Errorf("%s = %v (%v), want %q", name, v, err, want)
		}
	}

	str(root["name"], "name", "ovpn")
	str(root["subnet"], "subnet", "10.120.0.1/24")
	if !root["wireguard"].IsNull() || !root["l2tp"].IsNull() {
		t.Errorf("wireguard/l2tp should stay null: %v %v", root["wireguard"], root["l2tp"])
	}

	openvpn := obj(root["openvpn"], "openvpn")
	for _, flat := range []string{
		"server_crt", "server_key", "shared_client_crt", "shared_client_key", "ca_crt", "ca_key",
	} {
		if _, exists := openvpn[flat]; exists {
			t.Errorf("flat attribute openvpn.%s survived the upgrade", flat)
		}
	}
	str(openvpn["mode"], "openvpn.mode", "server")
	str(openvpn["dh_key"], "openvpn.dh_key", "DH-PARAMS")
	str(openvpn["auth_key"], "openvpn.auth_key", "TA-KEY")

	server := obj(openvpn["server"], "openvpn.server")
	str(server["crt"], "openvpn.server.crt", "SERVER-CRT")
	str(server["key"], "openvpn.server.key", "SERVER-KEY")
	shared := obj(openvpn["shared_client"], "openvpn.shared_client")
	str(shared["crt"], "openvpn.shared_client.crt", "CLIENT-CRT")
	str(shared["key"], "openvpn.shared_client.key", "CLIENT-KEY")
	ca := obj(openvpn["ca"], "openvpn.ca")
	str(ca["crt"], "openvpn.ca.crt", "CA-CRT")
	if !ca["key"].IsNull() {
		t.Errorf("openvpn.ca.key = %v, want null (a null leaf is preserved)", ca["key"])
	}
}

// TestVPNServer_openvpnCertPairsRoundTrip checks the nested certificate/key
// pairs convert API -> model -> API without loss, and that null and unknown
// pairs reach the wire exactly as the flat leaves they replaced did.
func TestVPNServer_openvpnCertPairsRoundTrip(t *testing.T) {
	ctx := context.Background()
	r := &vpnServerResource{}
	str := func(v string) *string { return &v }
	port := int64(1194)

	api := &unifi.Network{
		ID:                      "vpn-1",
		Name:                    str("ovpn"),
		Purpose:                 unifi.PurposeUserVPN,
		Enabled:                 new(true),
		IPSubnet:                str("10.120.0.1/24"),
		VPNType:                 str("openvpn-server"),
		LocalPort:               &port,
		OpenVPNMode:             str("server"),
		OpenVPNEncryptionCipher: str("AES_256_GCM"),
		ServerCrt:               str("SERVER-CRT"),
		ServerKey:               str("SERVER-KEY"),
		DhKey:                   str("DH-PARAMS"),
		SharedClientCrt:         str("CLIENT-CRT"),
		SharedClientKey:         str("CLIENT-KEY"),
		AuthKey:                 str("TA-KEY"),
		CaCrt:                   str("CA-CRT"),
	}

	var model vpnServerResourceModel
	if d := r.networkToModel(ctx, api, &model, "default", &vpnServerResourceModel{}); d.HasError() {
		t.Fatalf("networkToModel: %v", d)
	}
	ovpn := model.OpenVPN.Attributes()
	server := attrAs[types.Object](t, ovpn["server"]).Attributes()
	if attrAs[types.String](t, server["crt"]).ValueString() != "SERVER-CRT" ||
		attrAs[types.String](t, server["key"]).ValueString() != "SERVER-KEY" {
		t.Errorf("openvpn.server = %v", server)
	}
	shared := attrAs[types.Object](t, ovpn["shared_client"]).Attributes()
	if attrAs[types.String](t, shared["crt"]).ValueString() != "CLIENT-CRT" ||
		attrAs[types.String](t, shared["key"]).ValueString() != "CLIENT-KEY" {
		t.Errorf("openvpn.shared_client = %v", shared)
	}
	ca := attrAs[types.Object](t, ovpn["ca"]).Attributes()
	if attrAs[types.String](t, ca["crt"]).ValueString() != "CA-CRT" || !ca["key"].IsNull() {
		t.Errorf("openvpn.ca = %v (a nil API key must be a null leaf)", ca)
	}
	if attrAs[types.String](t, ovpn["dh_key"]).ValueString() != "DH-PARAMS" ||
		attrAs[types.String](t, ovpn["auth_key"]).ValueString() != "TA-KEY" {
		t.Errorf("un-nested openvpn leaves not read: %v", ovpn)
	}

	back, d := r.modelToNetwork(ctx, &model)
	if d.HasError() {
		t.Fatalf("modelToNetwork: %v", d)
	}
	if back.VPNType == nil || *back.VPNType != "openvpn-server" {
		t.Fatalf("VPNType = %v, want openvpn-server", back.VPNType)
	}
	want := map[string]string{
		"ServerCrt": "SERVER-CRT", "ServerKey": "SERVER-KEY",
		"SharedClientCrt": "CLIENT-CRT", "SharedClientKey": "CLIENT-KEY",
		"CaCrt": "CA-CRT", "DhKey": "DH-PARAMS", "AuthKey": "TA-KEY",
	}
	for name, got := range map[string]*string{
		"ServerCrt": back.ServerCrt, "ServerKey": back.ServerKey,
		"SharedClientCrt": back.SharedClientCrt, "SharedClientKey": back.SharedClientKey,
		"CaCrt": back.CaCrt, "DhKey": back.DhKey, "AuthKey": back.AuthKey,
	} {
		if got == nil || *got != want[name] {
			t.Errorf("%s round trip = %v, want %q", name, got, want[name])
		}
	}
	if back.CaKey != nil {
		t.Errorf("CaKey = %q, want nil (a null leaf stays off the wire)", *back.CaKey)
	}

	// On the wire a pair behaves exactly like the flat leaves it replaced: a
	// null pair is omitted, and an unknown pair (create, before the controller
	// has generated it) sends "" for both leaves as the unknown flat leaves
	// did. The un-nested leaves are unaffected.
	planned := vpnServerOpenVPNModel{
		Port:             types.Int64Value(1194),
		Mode:             types.StringValue("server"),
		EncryptionCipher: types.StringValue("AES_256_GCM"),
		Server:           types.ObjectUnknown(vpnServerCertPairModel{}.AttributeTypes()),
		DhKey:            types.StringUnknown(),
		SharedClient:     types.ObjectNull(vpnServerCertPairModel{}.AttributeTypes()),
		AuthKey:          types.StringValue("TA-KEY"),
		Ca:               types.ObjectUnknown(vpnServerCertPairModel{}.AttributeTypes()),
	}
	model.OpenVPN, d = types.ObjectValueFrom(ctx, vpnServerOpenVPNModel{}.AttributeTypes(), planned)
	if d.HasError() {
		t.Fatalf("planned ObjectValueFrom: %v", d)
	}
	back, d = r.modelToNetwork(ctx, &model)
	if d.HasError() {
		t.Fatalf("modelToNetwork (unknown pairs): %v", d)
	}
	empty := func(p *string) bool { return p != nil && *p == "" }
	if !empty(back.ServerCrt) || !empty(back.ServerKey) || !empty(back.CaCrt) ||
		!empty(back.CaKey) || !empty(back.DhKey) {
		t.Errorf(
			"unknown pairs must send \"\" like the unknown flat leaves: server=%v/%v ca=%v/%v dh=%v",
			back.ServerCrt,
			back.ServerKey,
			back.CaCrt,
			back.CaKey,
			back.DhKey,
		)
	}
	if back.SharedClientCrt != nil || back.SharedClientKey != nil {
		t.Errorf(
			"null pair leaked into the request: %v %v",
			back.SharedClientCrt,
			back.SharedClientKey,
		)
	}
	if back.AuthKey == nil || *back.AuthKey != "TA-KEY" ||
		back.LocalPort == nil || *back.LocalPort != 1194 {
		t.Errorf("known openvpn leaves dropped: auth=%v port=%v", back.AuthKey, back.LocalPort)
	}
}

// The controller returns no public key; it is derived from the private key,
// as `wg pubkey` does.
func Test_vpnServerResource_wireguardPublicKey(t *testing.T) {
	const priv = "WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg="
	const pub = "7B+2Z3odPbDNsfVr+F8invj6/mBKLVaolOHXZoCaBA0=" // from `wg pubkey`

	if got, err := wireGuardPublicKey(priv); err != nil || got != pub {
		t.Errorf("wireGuardPublicKey = %q, %v, want %q", got, err, pub)
	}

	ctx := context.Background()
	r := &vpnServerResource{}
	var model vpnServerResourceModel
	if d := r.networkToModel(ctx, &unifi.Network{
		ID:                  "net-1",
		Name:                new("wg"),
		Enabled:             new(true),
		IPSubnet:            new("10.100.0.1/24"),
		VPNType:             new("wireguard-server"),
		WireguardPrivateKey: new(priv),
		LocalPort:           new(int64(51820)),
	}, &model, "default", &vpnServerResourceModel{}); d.HasError() {
		t.Fatalf("unexpected diags: %v", d)
	}
	var wg vpnServerWireguardModel
	if d := model.Wireguard.As(
		ctx,
		&wg,
		struct{ UnhandledNullAsEmpty, UnhandledUnknownAsEmpty bool }{},
	); d.HasError() {
		t.Fatalf("reading wireguard: %v", d)
	}
	if wg.PublicKey.ValueString() != pub {
		t.Errorf("public_key = %q, want %q", wg.PublicKey.ValueString(), pub)
	}
}

// The IPv6 fields as the controller holds them for a WireGuard server created
// in the UI with "IPv6 Gateway/Subnet" and the WAN's IPv6 address (Network
// 10.x): ipv6_subnet in gateway form, and the listening family.
func Test_vpnServerResource_wireguardIPv6(t *testing.T) {
	ctx := context.Background()
	r := &vpnServerResource{}
	readWG := func(t *testing.T, model vpnServerResourceModel) vpnServerWireguardModel {
		t.Helper()
		var wg vpnServerWireguardModel
		if d := model.Wireguard.As(
			ctx,
			&wg,
			struct{ UnhandledNullAsEmpty, UnhandledUnknownAsEmpty bool }{},
		); d.HasError() {
			t.Fatalf("reading wireguard: %v", d)
		}
		return wg
	}

	t.Run("read: subnet_ipv6 and ip_version from the controller", func(t *testing.T) {
		network := &unifi.Network{
			ID:                                     "net-v6",
			Name:                                   new("wg-v6"),
			Enabled:                                new(true),
			IPSubnet:                               new("192.0.2.1/24"),
			IPV6Subnet:                             new("fd00:db8::1/64"),
			VPNType:                                new("wireguard-server"),
			LocalPort:                              new(int64(51821)),
			WireguardInterfaceBindingModeIPVersion: new("v6"),
		}
		var model vpnServerResourceModel
		if d := r.networkToModel(
			ctx,
			network,
			&model,
			"default",
			&vpnServerResourceModel{},
		); d.HasError() {
			t.Fatalf("unexpected diags: %v", d)
		}
		if got := model.SubnetIPv6.ValueString(); got != "fd00:db8::1/64" {
			t.Errorf("SubnetIPv6 = %q, want fd00:db8::1/64", got)
		}
		if got := readWG(t, model).IPVersion.ValueString(); got != "v6" {
			t.Errorf("IPVersion = %q, want v6", got)
		}
	})

	t.Run("read: a server without them is IPv4 only", func(t *testing.T) {
		network := &unifi.Network{
			ID:        "net-v4",
			Name:      new("wg-v4"),
			Enabled:   new(true),
			IPSubnet:  new("198.51.100.1/24"),
			VPNType:   new("wireguard-server"),
			LocalPort: new(int64(51820)),
		}
		var model vpnServerResourceModel
		if d := r.networkToModel(
			ctx,
			network,
			&model,
			"default",
			&vpnServerResourceModel{},
		); d.HasError() {
			t.Fatalf("unexpected diags: %v", d)
		}
		if !model.SubnetIPv6.IsNull() {
			t.Errorf("SubnetIPv6 = %q, want null", model.SubnetIPv6.ValueString())
		}
		if got := readWG(t, model).IPVersion.ValueString(); got != "v4" {
			t.Errorf("IPVersion = %q, want v4", got)
		}
	})

	t.Run("write: both reach the API fields; unset leaves them off", func(t *testing.T) {
		build := func(subnet6 cidrtypes.IPv6Prefix, ipVersion types.String) *unifi.Network {
			t.Helper()
			wgObj, d := types.ObjectValueFrom(
				ctx,
				vpnServerWireguardModel{}.AttributeTypes(),
				vpnServerWireguardModel{
					PrivateKey: types.StringValue("WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg="),
					PublicKey:  types.StringNull(),
					Port:       types.Int64Value(51821),
					IPVersion:  ipVersion,
				},
			)
			if d.HasError() {
				t.Fatalf("building wireguard object: %v", d)
			}
			got, diags := r.modelToNetwork(ctx, &vpnServerResourceModel{
				Name:       types.StringValue("wg-v6"),
				Enabled:    types.BoolValue(true),
				Subnet:     cidrtypes.NewIPv4PrefixValue("192.0.2.1/24"),
				SubnetIPv6: subnet6,
				Wireguard:  wgObj,
				L2TP:       types.ObjectNull(vpnServerL2TPModel{}.AttributeTypes()),
				OpenVPN:    types.ObjectNull(vpnServerOpenVPNModel{}.AttributeTypes()),
				DNS:        types.ObjectNull(vpnServerDNSModel{}.AttributeTypes()),
				WAN:        types.ObjectNull(vpnServerWANModel{}.AttributeTypes()),
			})
			if diags.HasError() {
				t.Fatalf("unexpected diags: %v", diags)
			}
			return got
		}

		got := build(cidrtypes.NewIPv6PrefixValue("fd00:db8::1/64"), types.StringValue("v6"))
		if got.IPV6Subnet == nil || *got.IPV6Subnet != "fd00:db8::1/64" {
			t.Errorf("IPV6Subnet = %v, want fd00:db8::1/64", got.IPV6Subnet)
		}
		if v := got.WireguardInterfaceBindingModeIPVersion; v == nil || *v != "v6" {
			t.Errorf("WireguardInterfaceBindingModeIPVersion = %v, want v6", v)
		}

		got = build(cidrtypes.NewIPv6PrefixNull(), types.StringNull())
		if got.IPV6Subnet != nil {
			t.Errorf("IPV6Subnet = %q, want nil", *got.IPV6Subnet)
		}
		if v := got.WireguardInterfaceBindingModeIPVersion; v != nil {
			t.Errorf("WireguardInterfaceBindingModeIPVersion = %q, want nil", *v)
		}
	})
}

func Test_vpnServerResource_privateKeyWOSchema(t *testing.T) {
	r := &vpnServerResource{}
	resp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	wg, ok := resp.Schema.Attributes["wireguard"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("wireguard is not a single nested attribute")
	}
	wo, ok := wg.Attributes["private_key_wo"].(schema.StringAttribute)
	if !ok || !wo.WriteOnly || !wo.Sensitive || !wo.Optional || wo.Computed {
		t.Fatalf("private_key_wo must be an optional, sensitive, write-only string: %+v", wo)
	}
	version, ok := wg.Attributes["private_key_wo_version"].(schema.Int64Attribute)
	if !ok || !version.Optional || version.WriteOnly {
		t.Fatalf("private_key_wo_version must be an optional int64: %+v", version)
	}
}

// vpnServerTestWireguard builds a wireguard object for the unit tests below.
func vpnServerTestWireguard(
	t *testing.T,
	privateKey, privateKeyWO types.String,
	version types.Int64,
) types.Object {
	t.Helper()
	obj, d := types.ObjectValueFrom(
		context.Background(),
		vpnServerWireguardModel{}.AttributeTypes(),
		vpnServerWireguardModel{
			PrivateKey:          privateKey,
			PrivateKeyWO:        privateKeyWO,
			PrivateKeyWOVersion: version,
			PublicKey:           types.StringNull(),
			Port:                types.Int64Value(51820),
			IPVersion:           types.StringValue("v4"),
		},
	)
	if d.HasError() {
		t.Fatalf("building wireguard object: %v", d)
	}
	return obj
}

// The controller echoes the private key on every read. With private_key_wo in
// use (private_key null in prior state) it must not reach the state, while the
// public key is still derived from it and the version is carried over.
func Test_vpnServerResource_networkToModel_privateKeyWO(t *testing.T) {
	const priv = "WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg="
	const pub = "7B+2Z3odPbDNsfVr+F8invj6/mBKLVaolOHXZoCaBA0="
	ctx := context.Background()
	r := &vpnServerResource{}
	network := &unifi.Network{
		ID:                  "net-1",
		Name:                new("wg"),
		Enabled:             new(true),
		IPSubnet:            new("10.100.0.1/24"),
		VPNType:             new("wireguard-server"),
		WireguardPrivateKey: new(priv),
		LocalPort:           new(int64(51820)),
	}
	read := func(t *testing.T, prior *vpnServerResourceModel) vpnServerWireguardModel {
		t.Helper()
		var model vpnServerResourceModel
		if d := r.networkToModel(ctx, network, &model, "default", prior); d.HasError() {
			t.Fatalf("unexpected diags: %v", d)
		}
		var wg vpnServerWireguardModel
		if d := model.Wireguard.As(
			ctx,
			&wg,
			struct{ UnhandledNullAsEmpty, UnhandledUnknownAsEmpty bool }{},
		); d.HasError() {
			t.Fatalf("reading wireguard: %v", d)
		}
		return wg
	}

	t.Run("write-only: neither key in state", func(t *testing.T) {
		wg := read(t, &vpnServerResourceModel{Wireguard: vpnServerTestWireguard(
			t, types.StringNull(), types.StringNull(), types.Int64Value(3),
		)})
		if !wg.PrivateKey.IsNull() {
			t.Errorf("private_key = %q, want null", wg.PrivateKey.ValueString())
		}
		if !wg.PrivateKeyWO.IsNull() {
			t.Errorf("private_key_wo = %q, want null", wg.PrivateKeyWO.ValueString())
		}
		if wg.PrivateKeyWOVersion.ValueInt64() != 3 {
			t.Errorf("private_key_wo_version = %v, want 3", wg.PrivateKeyWOVersion)
		}
		if wg.PublicKey.ValueString() != pub {
			t.Errorf("public_key = %q, want %q", wg.PublicKey.ValueString(), pub)
		}
	})

	t.Run("private_key: echoed key kept in state", func(t *testing.T) {
		wg := read(t, &vpnServerResourceModel{Wireguard: vpnServerTestWireguard(
			t, types.StringValue(priv), types.StringNull(), types.Int64Null(),
		)})
		if wg.PrivateKey.ValueString() != priv {
			t.Errorf("private_key = %q, want %q", wg.PrivateKey.ValueString(), priv)
		}
		if !wg.PrivateKeyWOVersion.IsNull() {
			t.Errorf("private_key_wo_version = %v, want null", wg.PrivateKeyWOVersion)
		}
	})

	t.Run("create without a key: generated key read back", func(t *testing.T) {
		wg := read(t, &vpnServerResourceModel{Wireguard: vpnServerTestWireguard(
			t, types.StringUnknown(), types.StringNull(), types.Int64Null(),
		)})
		if wg.PrivateKey.ValueString() != priv {
			t.Errorf("private_key = %q, want %q", wg.PrivateKey.ValueString(), priv)
		}
	})

	t.Run("import: key read back", func(t *testing.T) {
		wg := read(t, &vpnServerResourceModel{
			Wireguard: types.ObjectNull(vpnServerWireguardModel{}.AttributeTypes()),
		})
		if wg.PrivateKey.ValueString() != priv {
			t.Errorf("private_key = %q, want %q", wg.PrivateKey.ValueString(), priv)
		}
	})
}

func Test_vpnServerSendPrivateKeyWO(t *testing.T) {
	ctx := context.Background()
	const priv = "WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg="
	model := func(wg types.Object) *vpnServerResourceModel {
		return &vpnServerResourceModel{Wireguard: wg}
	}
	wo := func(pk types.String, v types.Int64) types.Object {
		return vpnServerTestWireguard(t, pk, types.StringNull(), v)
	}
	null := types.ObjectNull(vpnServerWireguardModel{}.AttributeTypes())

	tests := []struct {
		name        string
		plan, prior *vpnServerResourceModel
		want        bool
	}{
		{
			"same version keeps the controller key",
			model(wo(types.StringNull(), types.Int64Value(1))),
			model(wo(types.StringNull(), types.Int64Value(1))),
			false,
		},
		{
			"no version on either side keeps the controller key",
			model(wo(types.StringNull(), types.Int64Null())),
			model(wo(types.StringNull(), types.Int64Null())),
			false,
		},
		{
			"version bump sends the key",
			model(wo(types.StringNull(), types.Int64Value(2))),
			model(wo(types.StringNull(), types.Int64Value(1))),
			true,
		},
		{
			"switch from private_key sends the key",
			model(wo(types.StringNull(), types.Int64Null())),
			model(wo(types.StringValue(priv), types.Int64Null())),
			true,
		},
		{
			"no prior wireguard block sends the key",
			model(wo(types.StringNull(), types.Int64Null())),
			model(null),
			true,
		},
		{
			"no wireguard block in plan",
			model(null),
			model(wo(types.StringNull(), types.Int64Value(1))),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, d := vpnServerSendPrivateKeyWO(ctx, tt.plan, tt.prior)
			if d.HasError() {
				t.Fatalf("unexpected diags: %v", d)
			}
			if got != tt.want {
				t.Errorf("vpnServerSendPrivateKeyWO = %t, want %t", got, tt.want)
			}
		})
	}
}

// The update path must never generate a new key or drop the current one when
// neither private_key nor a (sent) private_key_wo is available; only create
// generates one.
func Test_vpnServerResource_resolveWireguardPrivateKey(t *testing.T) {
	ctx := context.Background()
	r := &vpnServerResource{}
	wgNetwork := func(key *string) *unifi.Network {
		return &unifi.Network{VPNType: new("wireguard-server"), WireguardPrivateKey: key}
	}

	n := wgNetwork(nil)
	if d := r.resolveWireguardPrivateKey(
		ctx, n, "default", "", types.StringValue("wo-key"), true,
	); d.HasError() {
		t.Fatalf("unexpected diags: %v", d)
	}
	if n.WireguardPrivateKey == nil || *n.WireguardPrivateKey != "wo-key" {
		t.Errorf("private_key_wo not sent: %v", n.WireguardPrivateKey)
	}

	n = wgNetwork(new("explicit"))
	if d := r.resolveWireguardPrivateKey(
		ctx, n, "default", "id", types.StringNull(), false,
	); d.HasError() {
		t.Fatalf("unexpected diags: %v", d)
	}
	if *n.WireguardPrivateKey != "explicit" {
		t.Errorf("explicit private_key replaced: %q", *n.WireguardPrivateKey)
	}

	n = wgNetwork(nil)
	if d := r.resolveWireguardPrivateKey(
		ctx, n, "default", "", types.StringNull(), true,
	); d.HasError() {
		t.Fatalf("unexpected diags: %v", d)
	}
	if n.WireguardPrivateKey == nil {
		t.Fatal("create without a key did not generate one")
	}
	if raw, err := base64.StdEncoding.DecodeString(*n.WireguardPrivateKey); err != nil ||
		len(raw) != 32 {
		t.Errorf("generated key is not a 32-byte base64 key: %q", *n.WireguardPrivateKey)
	}

	other := &unifi.Network{VPNType: new("l2tp-server")}
	if d := r.resolveWireguardPrivateKey(
		ctx, other, "default", "", types.StringValue("wo-key"), true,
	); d.HasError() {
		t.Fatalf("unexpected diags: %v", d)
	}
	if other.WireguardPrivateKey != nil {
		t.Errorf("non-WireGuard server got a key: %q", *other.WireguardPrivateKey)
	}
}

// vpnServerTestTFState returns raw resource data (as for config, plan or
// state) holding only the given wireguard object.
func vpnServerTestTFState(t *testing.T, s schema.Schema, wg types.Object) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	st := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
	if d := st.SetAttribute(ctx, path.Root("wireguard"), wg); d.HasError() {
		t.Fatalf("building state: %v", d)
	}
	return st
}

func Test_vpnServerPrivateKeyPlanModifiers(t *testing.T) {
	ctx := context.Background()
	const priv = "WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg="
	const pub = "7B+2Z3odPbDNsfVr+F8invj6/mBKLVaolOHXZoCaBA0="
	r := &vpnServerResource{}
	schemaResp := &fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, schemaResp)
	s := schemaResp.Schema
	keyPath := path.Root("wireguard").AtName("private_key")
	pubPath := path.Root("wireguard").AtName("public_key")
	wg := func(pk, wo types.String, v types.Int64) types.Object {
		return vpnServerTestWireguard(t, pk, wo, v)
	}

	planPrivateKey := func(
		t *testing.T,
		config, state types.Object,
		plan types.String,
	) types.String {
		t.Helper()
		cfg := vpnServerTestTFState(t, s, config)
		var configValue, stateValue types.String
		cfg.GetAttribute(ctx, keyPath, &configValue)
		st := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
		if !state.IsNull() {
			st = vpnServerTestTFState(t, s, state)
			st.GetAttribute(ctx, keyPath, &stateValue)
		} else {
			stateValue = types.StringNull()
		}
		req := planmodifier.StringRequest{
			Path:        keyPath,
			Config:      tfsdk.Config{Schema: s, Raw: cfg.Raw},
			ConfigValue: configValue,
			State:       st,
			StateValue:  stateValue,
			PlanValue:   plan,
		}
		resp := &planmodifier.StringResponse{PlanValue: plan}
		vpnServerPrivateKeyNullWithWO{}.PlanModifyString(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diags: %v", resp.Diagnostics)
		}
		return resp.PlanValue
	}
	nullWG := types.ObjectNull(vpnServerWireguardModel{}.AttributeTypes())

	t.Run("private_key_wo configured: planned null", func(t *testing.T) {
		got := planPrivateKey(t,
			wg(types.StringNull(), types.StringValue(priv), types.Int64Value(1)),
			nullWG, types.StringUnknown())
		if !got.IsNull() {
			t.Errorf("plan = %v, want null", got)
		}
	})
	t.Run("switch to private_key_wo: stored key planned null", func(t *testing.T) {
		got := planPrivateKey(t,
			wg(types.StringNull(), types.StringValue(priv), types.Int64Null()),
			wg(types.StringValue(priv), types.StringNull(), types.Int64Null()),
			types.StringValue(priv))
		if !got.IsNull() {
			t.Errorf("plan = %v, want null", got)
		}
	})
	t.Run("private_key_wo removed: stays out of state", func(t *testing.T) {
		got := planPrivateKey(t,
			wg(types.StringNull(), types.StringNull(), types.Int64Null()),
			wg(types.StringNull(), types.StringNull(), types.Int64Value(1)),
			types.StringUnknown())
		if !got.IsNull() {
			t.Errorf("plan = %v, want null", got)
		}
	})
	t.Run("no key on create: left unknown for generation", func(t *testing.T) {
		got := planPrivateKey(t,
			wg(types.StringNull(), types.StringNull(), types.Int64Null()),
			nullWG, types.StringUnknown())
		if !got.IsUnknown() {
			t.Errorf("plan = %v, want unknown", got)
		}
	})

	planPublicKey := func(t *testing.T, plan, state types.Object) types.String {
		t.Helper()
		pl := vpnServerTestTFState(t, s, plan)
		st := vpnServerTestTFState(t, s, state)
		req := planmodifier.StringRequest{
			Path:       pubPath,
			Plan:       tfsdk.Plan{Schema: s, Raw: pl.Raw},
			State:      st,
			StateValue: types.StringValue(pub),
			PlanValue:  types.StringValue(pub),
		}
		resp := &planmodifier.StringResponse{PlanValue: types.StringValue(pub)}
		vpnServerPublicKeyUnknownOnKeyChange{}.PlanModifyString(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diags: %v", resp.Diagnostics)
		}
		return resp.PlanValue
	}
	t.Run("same version: public key kept", func(t *testing.T) {
		v1 := wg(types.StringNull(), types.StringNull(), types.Int64Value(1))
		if got := planPublicKey(t, v1, v1); got.ValueString() != pub {
			t.Errorf("plan = %v, want %q", got, pub)
		}
	})
	t.Run("version bump: public key unknown", func(t *testing.T) {
		got := planPublicKey(t,
			wg(types.StringNull(), types.StringNull(), types.Int64Value(2)),
			wg(types.StringNull(), types.StringNull(), types.Int64Value(1)))
		if !got.IsUnknown() {
			t.Errorf("plan = %v, want unknown", got)
		}
	})
	t.Run("private_key change: public key unknown", func(t *testing.T) {
		got := planPublicKey(t,
			wg(types.StringValue("other"), types.StringNull(), types.Int64Null()),
			wg(types.StringValue(priv), types.StringNull(), types.Int64Null()))
		if !got.IsUnknown() {
			t.Errorf("plan = %v, want unknown", got)
		}
	})
}
