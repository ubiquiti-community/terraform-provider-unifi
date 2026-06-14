package unifi

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
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
	tests := []struct {
		name string
		want fwresource.Resource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewWireguardPeerResource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewWireguardPeerResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewWireguardPeerListResource(t *testing.T) {
	tests := []struct {
		name string
		want fwlist.ListResource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewWireguardPeerListResource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewWireguardPeerListResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_wireguardPeerResource_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.MetadataRequest
		resp *fwresource.MetadataResponse
	}
	tests := []struct {
		name string
		r    *wireguardPeerResource
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

func Test_wireguardPeerResource_IdentitySchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwresource.IdentitySchemaRequest
		resp *fwresource.IdentitySchemaResponse
	}
	tests := []struct {
		name string
		r    *wireguardPeerResource
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

func Test_wireguardPeerResource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.SchemaRequest
		resp *fwresource.SchemaResponse
	}
	tests := []struct {
		name string
		r    *wireguardPeerResource
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

func Test_wireguardPeerResource_Configure(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ConfigureRequest
		resp *fwresource.ConfigureResponse
	}
	tests := []struct {
		name string
		r    *wireguardPeerResource
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

func Test_wireguardPeerResource_Create(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.CreateRequest
		resp *fwresource.CreateResponse
	}
	tests := []struct {
		name string
		r    *wireguardPeerResource
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

func Test_wireguardPeerResource_Read(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ReadRequest
		resp *fwresource.ReadResponse
	}
	tests := []struct {
		name string
		r    *wireguardPeerResource
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

func Test_wireguardPeerResource_Update(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.UpdateRequest
		resp *fwresource.UpdateResponse
	}
	tests := []struct {
		name string
		r    *wireguardPeerResource
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

func Test_wireguardPeerResource_Delete(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.DeleteRequest
		resp *fwresource.DeleteResponse
	}
	tests := []struct {
		name string
		r    *wireguardPeerResource
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

func Test_wireguardPeerResource_ImportState(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ImportStateRequest
		resp *fwresource.ImportStateResponse
	}
	tests := []struct {
		name string
		r    *wireguardPeerResource
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

func Test_wireguardPeerResource_modelToPeer(t *testing.T) {
	type args struct {
		ctx   context.Context
		model *wireguardPeerResourceModel
	}
	tests := []struct {
		name  string
		r     *wireguardPeerResource
		args  args
		want  *unifi.WireGuardPeer
		want1 diag.Diagnostics
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.r.modelToPeer(tt.args.ctx, tt.args.model)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("wireguardPeerResource.modelToPeer() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("wireguardPeerResource.modelToPeer() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_wireguardPeerResource_peerToModel(t *testing.T) {
	type args struct {
		ctx   context.Context
		peer  *unifi.WireGuardPeer
		model *wireguardPeerResourceModel
		site  string
	}
	tests := []struct {
		name string
		r    *wireguardPeerResource
		args args
		want diag.Diagnostics
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.peerToModel(tt.args.ctx, tt.args.peer, tt.args.model, tt.args.site); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("wireguardPeerResource.peerToModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_wireguardPeerResource_ListResourceConfigSchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwlist.ListResourceSchemaRequest
		resp *fwlist.ListResourceSchemaResponse
	}
	tests := []struct {
		name string
		r    *wireguardPeerResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.ListResourceConfigSchema(tt.args.in0, tt.args.in1, tt.args.resp)
		})
	}
}

func Test_wireguardPeerResource_List(t *testing.T) {
	type args struct {
		ctx    context.Context
		req    fwlist.ListRequest
		stream *fwlist.ListResultsStream
	}
	tests := []struct {
		name string
		r    *wireguardPeerResource
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
