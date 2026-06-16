package unifi

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/cidrtypes"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	for _, key := range []string{"private_key", "public_key", "port"} {
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
	for _, key := range []string{"port", "mode", "encryption_cipher", "server_crt", "server_key", "dh_key", "ca_crt", "ca_key"} {
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

func Test_vpnServerResource_ImportState(t *testing.T) {
	t.Skip(
		"ImportState delegates to ImportStatePassthroughWithIdentity which requires full state schema setup",
	)
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
			Enabled:             true,
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
			Enabled:  true,
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
