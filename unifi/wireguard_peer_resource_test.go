package unifi

import (
	"context"
	"fmt"
	"testing"

	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/ubiquiti-community/go-unifi/unifi"
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

func TestNewWireguardPeerResource(t *testing.T) {
	r := NewWireguardPeerResource()
	if r == nil {
		t.Fatal("NewWireguardPeerResource() returned nil")
	}
	if _, ok := r.(fwresource.ResourceWithConfigure); !ok {
		t.Error("expected ResourceWithConfigure interface")
	}
	if _, ok := r.(fwresource.ResourceWithImportState); !ok {
		t.Error("expected ResourceWithImportState interface")
	}
	if _, ok := r.(fwresource.ResourceWithIdentity); !ok {
		t.Error("expected ResourceWithIdentity interface")
	}
}

func TestNewWireguardPeerListResource(t *testing.T) {
	r := NewWireguardPeerListResource()
	if r == nil {
		t.Fatal("NewWireguardPeerListResource() returned nil")
	}
	if _, ok := r.(fwlist.ListResourceWithConfigure); !ok {
		t.Error("expected fwlist.ListResourceWithConfigure interface")
	}
}

func Test_wireguardPeerResource_Metadata(t *testing.T) {
	for _, tt := range []struct{ p, w string }{
		{"unifi", "unifi_wireguard_peer"},
		{"test", "test_wireguard_peer"},
	} {
		t.Run(tt.p, func(t *testing.T) {
			r := &wireguardPeerResource{}
			resp := &fwresource.MetadataResponse{}
			r.Metadata(
				context.Background(),
				fwresource.MetadataRequest{ProviderTypeName: tt.p},
				resp,
			)
			if resp.TypeName != tt.w {
				t.Errorf("TypeName = %q, want %q", resp.TypeName, tt.w)
			}
		})
	}
}

func Test_wireguardPeerResource_IdentitySchema(t *testing.T) {
	r := &wireguardPeerResource{}
	resp := &fwresource.IdentitySchemaResponse{}
	r.IdentitySchema(context.Background(), fwresource.IdentitySchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("IdentitySchema() returned errors: %v", resp.Diagnostics)
	}
	if len(resp.IdentitySchema.Attributes) == 0 {
		t.Error("IdentitySchema() returned no attributes")
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Error("IdentitySchema() missing 'id' attribute")
	}
}

func Test_wireguardPeerResource_Schema(t *testing.T) {
	r := &wireguardPeerResource{}
	resp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() returned errors: %v", resp.Diagnostics)
	}
	for _, key := range []string{"id", "site", "network_id", "name", "interface_ip", "public_key", "allowed_ips", "timeouts"} {
		if _, ok := resp.Schema.Attributes[key]; !ok {
			t.Errorf("Schema() missing attribute %q", key)
		}
	}
}

func Test_wireguardPeerResource_Configure(t *testing.T) {
	for _, tt := range []struct {
		name string
		data any
		err  bool
	}{
		{"nil", nil, false},
		{"wrong", "wrong", true},
		{"ok", &Client{}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := &wireguardPeerResource{}
			resp := &fwresource.ConfigureResponse{}
			r.Configure(
				context.Background(),
				fwresource.ConfigureRequest{ProviderData: tt.data},
				resp,
			)
			if tt.err && !resp.Diagnostics.HasError() {
				t.Error("expected error in diagnostics")
			}
			if !tt.err && resp.Diagnostics.HasError() {
				t.Errorf("unexpected error: %v", resp.Diagnostics)
			}
		})
	}
}

func Test_wireguardPeerResource_modelToPeer(t *testing.T) {
	ctx := context.Background()
	r := &wireguardPeerResource{}

	t.Run("maps all fields", func(t *testing.T) {
		allowedIPs, _ := types.ListValueFrom(
			ctx,
			types.StringType,
			[]string{"10.0.0.0/8", "192.168.1.0/24"},
		)
		model := &wireguardPeerResourceModel{
			Name:        types.StringValue("my-peer"),
			InterfaceIP: types.StringValue("10.0.0.2"),
			PublicKey:   types.StringValue("abc123=="),
			AllowedIPs:  allowedIPs,
		}
		got, diags := r.modelToPeer(ctx, model)
		if diags.HasError() {
			t.Fatalf("modelToPeer() errors: %v", diags)
		}
		if got == nil {
			t.Fatal("modelToPeer() returned nil")
		}
		if got.Name != "my-peer" {
			t.Errorf("Name = %q, want my-peer", got.Name)
		}
		if got.InterfaceIP != "10.0.0.2" {
			t.Errorf("InterfaceIP = %q, want 10.0.0.2", got.InterfaceIP)
		}
		if got.PublicKey != "abc123==" {
			t.Errorf("PublicKey = %q, want abc123==", got.PublicKey)
		}
		if len(got.AllowedIPs) != 2 {
			t.Errorf("AllowedIPs len = %d, want 2", len(got.AllowedIPs))
		}
	})

	t.Run("null allowed_ips gives empty slice", func(t *testing.T) {
		model := &wireguardPeerResourceModel{
			Name:        types.StringValue("peer2"),
			InterfaceIP: types.StringValue("10.0.0.3"),
			PublicKey:   types.StringValue("xyz=="),
			AllowedIPs:  types.ListNull(types.StringType),
		}
		got, diags := r.modelToPeer(ctx, model)
		if diags.HasError() {
			t.Fatalf("modelToPeer() errors: %v", diags)
		}
		if got.AllowedIPs == nil {
			t.Error("AllowedIPs should be non-nil empty slice, got nil")
		}
		if len(got.AllowedIPs) != 0 {
			t.Errorf("AllowedIPs should be empty, got %v", got.AllowedIPs)
		}
	})
}

func Test_wireguardPeerResource_peerToModel(t *testing.T) {
	ctx := context.Background()
	r := &wireguardPeerResource{}

	t.Run("populates all model fields", func(t *testing.T) {
		peer := &unifi.WireGuardPeer{
			ID:          "peer-abc",
			NetworkID:   "net-xyz",
			Name:        "my-peer",
			InterfaceIP: "10.0.0.5",
			PublicKey:   "pubkey==",
			AllowedIPs:  []string{"172.16.0.0/12"},
		}
		var model wireguardPeerResourceModel
		diags := r.peerToModel(ctx, peer, &model, "default")
		if diags.HasError() {
			t.Fatalf("peerToModel() errors: %v", diags)
		}
		if model.ID.ValueString() != "peer-abc" {
			t.Errorf("ID = %q, want peer-abc", model.ID.ValueString())
		}
		if model.Site.ValueString() != "default" {
			t.Errorf("Site = %q, want default", model.Site.ValueString())
		}
		if model.NetworkID.ValueString() != "net-xyz" {
			t.Errorf("NetworkID = %q, want net-xyz", model.NetworkID.ValueString())
		}
		if model.Name.ValueString() != "my-peer" {
			t.Errorf("Name = %q, want my-peer", model.Name.ValueString())
		}
		if model.InterfaceIP.ValueString() != "10.0.0.5" {
			t.Errorf("InterfaceIP = %q, want 10.0.0.5", model.InterfaceIP.ValueString())
		}
		if model.PublicKey.ValueString() != "pubkey==" {
			t.Errorf("PublicKey = %q, want pubkey==", model.PublicKey.ValueString())
		}
		if len(model.AllowedIPs.Elements()) != 1 {
			t.Errorf("AllowedIPs len = %d, want 1", len(model.AllowedIPs.Elements()))
		}
	})

	t.Run("empty allowed_ips becomes empty list", func(t *testing.T) {
		peer := &unifi.WireGuardPeer{
			ID:          "peer-empty",
			NetworkID:   "net-1",
			Name:        "p",
			InterfaceIP: "10.0.0.1",
			PublicKey:   "k==",
			AllowedIPs:  []string{},
		}
		var model wireguardPeerResourceModel
		diags := r.peerToModel(ctx, peer, &model, "site1")
		if diags.HasError() {
			t.Fatalf("peerToModel() errors: %v", diags)
		}
		if model.AllowedIPs.IsNull() {
			t.Error("AllowedIPs should not be null for empty slice")
		}
		if len(model.AllowedIPs.Elements()) != 0 {
			t.Errorf(
				"AllowedIPs should be empty, got %d elements",
				len(model.AllowedIPs.Elements()),
			)
		}
	})
}

func Test_wireguardPeerResource_ListResourceConfigSchema(t *testing.T) {
	r := &wireguardPeerResource{}
	resp := &fwlist.ListResourceSchemaResponse{}
	r.ListResourceConfigSchema(context.Background(), fwlist.ListResourceSchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ListResourceConfigSchema() returned errors: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["site"]; !ok {
		t.Error("ListResourceConfigSchema() missing 'site' attribute")
	}
	if _, ok := resp.Schema.Attributes["network_id"]; !ok {
		t.Error("ListResourceConfigSchema() missing 'network_id' attribute")
	}
}
