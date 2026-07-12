package unifi

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/cidrtypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/ubiquiti-community/go-unifi/unifi"
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
						"wireguard.interface",
						"wan",
					),
					resource.TestCheckResourceAttrSet(
						"unifi_vpn_client.test",
						"wireguard.configuration.content",
					),
					resource.TestCheckResourceAttrSet(
						"unifi_vpn_client.test",
						"wireguard.configuration.filename",
					),
				),
			},
			{
				ResourceName:      "unifi_vpn_client.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"wireguard.private_key",
					"wireguard.configuration",
					"wireguard.configuration.content",
					"wireguard.configuration.filename",
					"wireguard.preshared_key",
					"wireguard.peer",
					"wireguard.peer.ip",
					"wireguard.peer.port",
					"wireguard.peer.public_key",
					"wireguard.dns_servers",
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

func TestAccVPNClient_write_only_private_key(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVPNClientConfig_write_only_private_key(1, "WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg="),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_vpn_client.test", "enabled", "false"),
					resource.TestCheckNoResourceAttr("unifi_vpn_client.test", "wireguard.private_key"),
					resource.TestCheckNoResourceAttr("unifi_vpn_client.test", "wireguard.private_key_wo"),
					resource.TestCheckResourceAttr("unifi_vpn_client.test", "wireguard.private_key_wo_version", "1"),
				),
			},
			{
				Config: testAccVPNClientConfig_write_only_private_key(2, "uGEwDKZ2Hf2s2Dg59c9K+qYzJEBN5s8fNWVTxZx9kUo="),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("unifi_vpn_client.test", "wireguard.private_key"),
					resource.TestCheckNoResourceAttr("unifi_vpn_client.test", "wireguard.private_key_wo"),
					resource.TestCheckResourceAttr("unifi_vpn_client.test", "wireguard.private_key_wo_version", "2"),
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
      content  = "W0ludGVyZmFjZV0KUHJpdmF0ZUtleSA9IFdQaUJhL0FrMVcrOFNwOEw1eXZieWhIZVJPMm81a0p2aWhxMlZ0SitrRmc9CkFkZHJlc3MgPSAxMC4wLjAuMi8yNApETlMgPSA4LjguOC44LCA4LjguNC40CgpbUGVlcl0KUHVibGljS2V5ID0gN0IrMlozb2RQYkROc2ZWcitGOGludmo2L21CS0xWYW9sT0hYWm9DYUJBMD0KRW5kcG9pbnQgPSAxOTIuMC4yLjE6NTE4MjAKQWxsb3dlZElQcyA9IDAuMC4wLjAvMAo="
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

func testAccVPNClientConfig_write_only_private_key(version int, privateKey string) string {
	return fmt.Sprintf(`
resource "unifi_vpn_client" "test" {
  name          = "test-wireguard-write-only"
  enabled       = false
  subnet        = "10.0.3.2/24"
  default_route = false
  pull_dns      = false

  wireguard = {
    private_key_wo         = %q
    private_key_wo_version = %d
    interface              = "wan"
    dns_servers            = ["8.8.8.8"]

    peer = {
      ip         = "192.0.2.1"
      port       = 51820
      public_key = "7B+2Z3odPbDNsfVr+F8invj6/mBKLVaolOHXZoCaBA0="
    }
  }
}
`, privateKey, version)
}

func TestNewVPNClientResource(t *testing.T) {
	r := NewVPNClientResource()
	if r == nil {
		t.Fatal("NewVPNClientResource() returned nil")
	}
	if _, ok := r.(fwresource.ResourceWithImportState); !ok {
		t.Error("expected ResourceWithImportState interface")
	}
}

func TestNewVPNClientListResource(t *testing.T) {
	r := NewVPNClientListResource()
	if r == nil {
		t.Fatal("NewVPNClientListResource() returned nil")
	}
	if _, ok := r.(fwlist.ListResourceWithConfigure); !ok {
		t.Error("expected ListResourceWithConfigure interface")
	}
}

func Test_wireguardConfigurationModel_AttributeTypes(t *testing.T) {
	m := wireguardConfigurationModel{}
	got := m.AttributeTypes()
	want := map[string]attr.Type{
		"content":  types.StringType,
		"filename": types.StringType,
	}
	if len(got) != len(want) {
		t.Errorf("AttributeTypes() returned %d entries, want %d", len(got), len(want))
	}
	for k, wantType := range want {
		if gotType, ok := got[k]; !ok {
			t.Errorf("missing key %q", k)
		} else if gotType != wantType {
			t.Errorf("key %q: got %v, want %v", k, gotType, wantType)
		}
	}
}

func Test_wireguardPeerModel_AttributeTypes(t *testing.T) {
	m := wireguardPeerModel{}
	got := m.AttributeTypes()
	want := map[string]attr.Type{
		"ip":         types.StringType,
		"port":       types.Int64Type,
		"public_key": types.StringType,
	}
	if len(got) != len(want) {
		t.Errorf("AttributeTypes() returned %d entries, want %d", len(got), len(want))
	}
	for k, wantType := range want {
		if gotType, ok := got[k]; !ok {
			t.Errorf("missing key %q", k)
		} else if gotType != wantType {
			t.Errorf("key %q: got %v, want %v", k, gotType, wantType)
		}
	}
}

func Test_wireguardModel_AttributeTypes(t *testing.T) {
	m := wireguardModel{}
	got := m.AttributeTypes()
	for _, key := range []string{
		"private_key", "private_key_wo", "private_key_wo_version", "configuration", "peer",
		"preshared_key_enabled", "preshared_key", "interface", "dns_servers",
	} {
		if _, ok := got[key]; !ok {
			t.Errorf("missing key %q in AttributeTypes()", key)
		}
	}
}

func Test_vpnClientResource_Metadata(t *testing.T) {
	for _, tt := range []struct{ provider, want string }{
		{"unifi", "unifi_vpn_client"},
		{"test", "test_vpn_client"},
	} {
		t.Run(tt.provider, func(t *testing.T) {
			r := &vpnClientResource{}
			resp := &fwresource.MetadataResponse{}
			r.Metadata(
				context.Background(),
				fwresource.MetadataRequest{ProviderTypeName: tt.provider},
				resp,
			)
			if resp.TypeName != tt.want {
				t.Errorf("got %q, want %q", resp.TypeName, tt.want)
			}
		})
	}
}

func Test_vpnClientResource_IdentitySchema(t *testing.T) {
	r := &vpnClientResource{}
	resp := &fwresource.IdentitySchemaResponse{}
	r.IdentitySchema(context.Background(), fwresource.IdentitySchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("IdentitySchema() produced errors: %v", resp.Diagnostics)
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Error("IdentitySchema missing 'id' attribute")
	}
}

func Test_vpnClientResource_Schema(t *testing.T) {
	r := &vpnClientResource{}
	resp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() produced errors: %v", resp.Diagnostics)
	}
	for _, attr := range []string{"id", "site", "name", "enabled", "subnet", "default_route", "pull_dns", "wireguard", "timeouts"} {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("missing attribute %q", attr)
		}
	}
}

func Test_vpnClientResource_Configure(t *testing.T) {
	for _, tt := range []struct {
		name    string
		data    any
		wantErr bool
	}{
		{"nil", nil, false},
		{"wrong type", "wrong", true},
		{"correct", &Client{Site: "default"}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := &vpnClientResource{}
			resp := &fwresource.ConfigureResponse{}
			r.Configure(
				context.Background(),
				fwresource.ConfigureRequest{ProviderData: tt.data},
				resp,
			)
			if tt.wantErr && !resp.Diagnostics.HasError() {
				t.Error("expected error")
			}
			if !tt.wantErr && resp.Diagnostics.HasError() {
				t.Errorf("unexpected: %v", resp.Diagnostics)
			}
		})
	}
}

func Test_vpnClientResource_modelToNetwork(t *testing.T) {
	ctx := context.Background()
	r := &vpnClientResource{}

	t.Run("basic manual mode fields", func(t *testing.T) {
		peerObj, d := types.ObjectValueFrom(
			ctx,
			wireguardPeerModel{}.AttributeTypes(),
			wireguardPeerModel{
				IP:        types.StringValue("1.2.3.4"),
				Port:      types.Int64Value(51820),
				PublicKey: types.StringValue("pubkey=="),
			},
		)
		if d.HasError() {
			t.Fatalf("building peer: %v", d)
		}
		wg := wireguardModel{
			PrivateKey:          types.StringValue("privkey=="),
			Configuration:       types.ObjectNull(wireguardConfigurationModel{}.AttributeTypes()),
			Peer:                peerObj,
			PresharedKeyEnabled: types.BoolValue(false),
			PresharedKey:        types.StringNull(),
			Interface:           types.StringValue("wan"),
			DnsServers:          types.ListNull(types.StringType),
		}
		wgObj, d := types.ObjectValueFrom(ctx, wg.AttributeTypes(), wg)
		if d.HasError() {
			t.Fatalf("building wireguard: %v", d)
		}

		from := cidrtypes.NewIPv4PrefixValue("10.0.0.2/24")
		model := &vpnClientResourceModel{
			Name:         types.StringValue("test-vpn"),
			Enabled:      types.BoolValue(true),
			Subnet:       from,
			DefaultRoute: types.BoolValue(false),
			PullDNS:      types.BoolValue(false),
			Wireguard:    wgObj,
		}
		network, diags := r.modelToNetwork(ctx, model)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if network.Purpose != unifi.PurposeVPNClient {
			t.Errorf("Purpose = %q, want vpn-client", network.Purpose)
		}
		if network.VPNType == nil || *network.VPNType != "wireguard-client" {
			t.Errorf("VPNType = %v, want wireguard-client", network.VPNType)
		}
		if network.WireguardClientMode == nil || *network.WireguardClientMode != "manual" {
			t.Errorf("WireguardClientMode = %v, want manual", network.WireguardClientMode)
		}
	})

	t.Run("null wireguard produces basic network", func(t *testing.T) {
		from := cidrtypes.NewIPv4PrefixValue("10.0.0.1/24")
		model := &vpnClientResourceModel{
			Name:      types.StringValue("min-vpn"),
			Enabled:   types.BoolValue(true),
			Subnet:    from,
			Wireguard: types.ObjectNull(wireguardModel{}.AttributeTypes()),
		}
		network, diags := r.modelToNetwork(ctx, model)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if network == nil {
			t.Fatal("expected non-nil network")
		}
		if network.Purpose != unifi.PurposeVPNClient {
			t.Errorf("Purpose = %q, want vpn-client", network.Purpose)
		}
	})
}

func Test_vpnClientResource_privateKeyWriteOnlySchema(t *testing.T) {
	ctx := context.Background()
	r := &vpnClientResource{}
	var resp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	wg, ok := resp.Schema.Attributes["wireguard"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("wireguard is not a single nested attribute")
	}
	privateKeyWO, ok := wg.Attributes["private_key_wo"].(schema.StringAttribute)
	if !ok {
		t.Fatal("wireguard.private_key_wo is not a string attribute")
	}
	privateKeyWOVersion, ok := wg.Attributes["private_key_wo_version"].(schema.Int64Attribute)
	if !ok || !privateKeyWOVersion.Optional {
		t.Fatal("wireguard.private_key_wo_version is not an optional int64 attribute")
	}
	if !privateKeyWO.WriteOnly || !privateKeyWO.Sensitive || !privateKeyWO.Optional {
		t.Fatalf(
			"private_key_wo flags = write-only:%t sensitive:%t optional:%t",
			privateKeyWO.WriteOnly,
			privateKeyWO.Sensitive,
			privateKeyWO.Optional,
		)
	}
}

func Test_vpnClientResource_networkToModel(t *testing.T) {
	ctx := context.Background()
	r := &vpnClientResource{}

	t.Run("manual mode populates peer", func(t *testing.T) {
		mode := "manual"
		ip := "1.2.3.4"
		port := int64(51820)
		pubKey := "pubkey=="
		subnet := "10.0.0.2/24"
		name := "test-vpn"
		network := &unifi.Network{
			ID:                           "net-1",
			Name:                         &name,
			Enabled:                      true,
			IPSubnet:                     &subnet,
			WireguardClientMode:          &mode,
			WireguardClientPeerIP:        &ip,
			WireguardClientPeerPort:      &port,
			WireguardClientPeerPublicKey: &pubKey,
		}
		model := &vpnClientResourceModel{}
		priorState := &vpnClientResourceModel{}
		diags := r.networkToModel(ctx, network, model, "default", priorState)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if model.ID.ValueString() != "net-1" {
			t.Errorf("ID = %q, want net-1", model.ID.ValueString())
		}
		if model.Site.ValueString() != "default" {
			t.Errorf("Site = %q, want default", model.Site.ValueString())
		}
		if model.Wireguard.IsNull() {
			t.Error("Wireguard should not be null")
		}
	})

	t.Run("write-only private key is not copied from API into state", func(t *testing.T) {
		mode := "manual"
		ip := "1.2.3.4"
		port := int64(51820)
		pubKey := "pubkey=="
		apiPrivateKey := "must-not-enter-state=="
		subnet := "10.0.0.2/24"
		name := "write-only-vpn"
		network := &unifi.Network{
			ID:                           "net-wo",
			Name:                         &name,
			Enabled:                      true,
			IPSubnet:                     &subnet,
			WireguardClientMode:          &mode,
			WireguardClientPeerIP:        &ip,
			WireguardClientPeerPort:      &port,
			WireguardClientPeerPublicKey: &pubKey,
			WireguardPrivateKey:          &apiPrivateKey,
		}

		priorWG := wireguardModel{
			PrivateKey:          types.StringNull(),
			PrivateKeyWO:        types.StringNull(),
			PrivateKeyWOVersion: types.Int64Value(7),
			Configuration:       types.ObjectNull(wireguardConfigurationModel{}.AttributeTypes()),
			Peer:                types.ObjectNull(wireguardPeerModel{}.AttributeTypes()),
			PresharedKeyEnabled: types.BoolValue(false),
			PresharedKey:        types.StringNull(),
			Interface:           types.StringValue("wan"),
			DnsServers:          types.ListNull(types.StringType),
		}
		priorWGObj, d := types.ObjectValueFrom(ctx, priorWG.AttributeTypes(), priorWG)
		if d.HasError() {
			t.Fatalf("building prior wireguard state: %v", d)
		}
		prior := &vpnClientResourceModel{Wireguard: priorWGObj}

		var model vpnClientResourceModel
		diags := r.networkToModel(ctx, network, &model, "default", prior)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		var gotWG wireguardModel
		diags = model.Wireguard.As(ctx, &gotWG, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			t.Fatalf("reading wireguard state: %v", diags)
		}
		if !gotWG.PrivateKey.IsNull() {
			t.Fatalf("private key entered state: %q", gotWG.PrivateKey.ValueString())
		}
		if gotWG.PrivateKeyWOVersion.ValueInt64() != 7 {
			t.Fatalf("private key version = %d, want 7", gotWG.PrivateKeyWOVersion.ValueInt64())
		}
	})

	t.Run("no mode produces null peer and null configuration", func(t *testing.T) {
		name := "no-mode-vpn"
		network := &unifi.Network{
			ID:   "net-2",
			Name: &name,
		}
		model := &vpnClientResourceModel{}
		priorState := &vpnClientResourceModel{}
		diags := r.networkToModel(ctx, network, model, "default", priorState)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if model.Wireguard.IsNull() {
			t.Error("Wireguard object should not be null even with no mode")
		}
	})
}

func Test_vpnClientResource_ListResourceConfigSchema(t *testing.T) {
	r := &vpnClientResource{}
	resp := &fwlist.ListResourceSchemaResponse{}
	r.ListResourceConfigSchema(context.Background(), fwlist.ListResourceSchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("ListResourceConfigSchema() produced errors: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["site"]; !ok {
		t.Error("ListResourceConfigSchema missing 'site' attribute")
	}
}

func TestAccVPNClientList_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccVPNClientConfig_file_mode(),
			},
			{
				Query: true,
				Config: `
					provider "unifi" {}
					list "unifi_vpn_client" "test" {
						provider = unifi
						config {
							filter {
								name  = "name"
								value = "test-wireguard-vpn"
						  }
					  }
					}
				`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLengthAtLeast("unifi_vpn_client.test", 1),
				},
			},
		},
	})
}
