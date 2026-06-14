package unifi

import (
	"context"
	"encoding/base64"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
	tests := []struct {
		name string
		want fwresource.Resource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewVPNServerResource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewVPNServerResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewVPNServerListResource(t *testing.T) {
	tests := []struct {
		name string
		want fwlist.ListResource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewVPNServerListResource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewVPNServerListResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_vpnServerDNSModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    vpnServerDNSModel
		want map[string]attr.Type
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("vpnServerDNSModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_vpnServerWANModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    vpnServerWANModel
		want map[string]attr.Type
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("vpnServerWANModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_vpnServerWireguardModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    vpnServerWireguardModel
		want map[string]attr.Type
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("vpnServerWireguardModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_vpnServerL2TPModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    vpnServerL2TPModel
		want map[string]attr.Type
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("vpnServerL2TPModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_vpnServerOpenVPNModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    vpnServerOpenVPNModel
		want map[string]attr.Type
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("vpnServerOpenVPNModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_vpnServerResource_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.MetadataRequest
		resp *fwresource.MetadataResponse
	}
	tests := []struct {
		name string
		r    *vpnServerResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Metadata(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_vpnServerResource_IdentitySchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwresource.IdentitySchemaRequest
		resp *fwresource.IdentitySchemaResponse
	}
	tests := []struct {
		name string
		r    *vpnServerResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.IdentitySchema(tt.args.in0, tt.args.in1, tt.args.resp)
		})
	}
}

func Test_vpnServerResource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.SchemaRequest
		resp *fwresource.SchemaResponse
	}
	tests := []struct {
		name string
		r    *vpnServerResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Schema(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_vpnServerResource_Configure(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ConfigureRequest
		resp *fwresource.ConfigureResponse
	}
	tests := []struct {
		name string
		r    *vpnServerResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Configure(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_vpnServerResource_Create(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.CreateRequest
		resp *fwresource.CreateResponse
	}
	tests := []struct {
		name string
		r    *vpnServerResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Create(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_vpnServerResource_Read(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ReadRequest
		resp *fwresource.ReadResponse
	}
	tests := []struct {
		name string
		r    *vpnServerResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Read(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_vpnServerResource_Update(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.UpdateRequest
		resp *fwresource.UpdateResponse
	}
	tests := []struct {
		name string
		r    *vpnServerResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Update(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_vpnServerResource_Delete(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.DeleteRequest
		resp *fwresource.DeleteResponse
	}
	tests := []struct {
		name string
		r    *vpnServerResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Delete(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_vpnServerResource_ImportState(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ImportStateRequest
		resp *fwresource.ImportStateResponse
	}
	tests := []struct {
		name string
		r    *vpnServerResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.ImportState(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_vpnServerResource_modelToNetwork(t *testing.T) {
	type args struct {
		ctx   context.Context
		model *vpnServerResourceModel
	}
	tests := []struct {
		name  string
		r     *vpnServerResource
		args  args
		want  *unifi.Network
		want1 diag.Diagnostics
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.r.modelToNetwork(tt.args.ctx, tt.args.model)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("vpnServerResource.modelToNetwork() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("vpnServerResource.modelToNetwork() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_vpnServerResource_networkToModel(t *testing.T) {
	type args struct {
		ctx        context.Context
		network    *unifi.Network
		model      *vpnServerResourceModel
		site       string
		priorState *vpnServerResourceModel
	}
	tests := []struct {
		name string
		r    *vpnServerResource
		args args
		want diag.Diagnostics
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.networkToModel(
				tt.args.ctx,
				tt.args.network,
				tt.args.model,
				tt.args.site,
				tt.args.priorState,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("vpnServerResource.networkToModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_vpnServerResource_ListResourceConfigSchema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwlist.ListResourceSchemaRequest
		resp *fwlist.ListResourceSchemaResponse
	}
	tests := []struct {
		name string
		r    *vpnServerResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.ListResourceConfigSchema(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_vpnServerResource_List(t *testing.T) {
	type args struct {
		ctx    context.Context
		req    fwlist.ListRequest
		stream *fwlist.ListResultsStream
	}
	tests := []struct {
		name string
		r    *vpnServerResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.List(tt.args.ctx, tt.args.req, tt.args.stream)
		})
	}
}

func Test_generateWireGuardPrivateKey(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateWireGuardPrivateKey()
			if (err != nil) != tt.wantErr {
				t.Errorf("generateWireGuardPrivateKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("generateWireGuardPrivateKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
