package unifi

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func TestAccRadiusUser_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: providerFactories,
		CheckDestroy:             nil, // TODO: implement check destroy
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusUserConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_radius_user.test",
						"name",
						"test-account",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_user.test",
						"password",
						"test-password",
					),
					resource.TestCheckResourceAttr("unifi_radius_user.test", "tunnel_type", "3"),
					resource.TestCheckResourceAttr(
						"unifi_radius_user.test",
						"tunnel_medium_type",
						"6",
					),
				),
			},
			{
				ResourceName:            "unifi_radius_user.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"}, // Password is not returned by API
			},
		},
	})
}

func testAccRadiusUserConfig_basic() string {
	return `
resource "unifi_radius_user" "test" {
	name     = "test-account"
	password = "test-password"
}
`
}

func TestAccRadiusUser_vlan(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusUserConfig_vlan(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_radius_user.vlan", "vlan", "100"),
					resource.TestCheckResourceAttr(
						"unifi_radius_user.vlan",
						"tunnel_config_type",
						"802.1x",
					),
				),
			},
			{
				ResourceName:            "unifi_radius_user.vlan",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

func testAccRadiusUserConfig_vlan() string {
	return `
resource "unifi_radius_user" "vlan" {
	name               = "test-account-vlan"
	password           = "test-password"
	vlan               = 100
	tunnel_config_type = "802.1x"
}
`
}

// TestAccRadiusUser_tunnelType13 verifies that tunnel_type accepts 13 (VLAN),
// which the controller allows (1-13) but the provider previously capped at 12.
func TestAccRadiusUser_tunnelType13(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusUserConfig_tunnelType13(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_radius_user.tt13", "tunnel_type", "13"),
				),
			},
			{
				ResourceName:            "unifi_radius_user.tt13",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

func testAccRadiusUserConfig_tunnelType13() string {
	return `
resource "unifi_radius_user" "tt13" {
	name        = "test-account-tt13"
	password    = "test-password"
	tunnel_type = 13
}
`
}

// TestAccRadiusUser_moveFromAccount exercises the ResourceWithMoveState support
// (#222): a deprecated unifi_account can be migrated to unifi_radius_user with a
// `moved` block, in place. The move is proven by the target keeping the source's
// ID — a destroy/recreate would assign a new one.
func TestAccRadiusUser_moveFromAccount(t *testing.T) {
	var accountID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			// Create the deprecated resource and capture its ID.
			{
				Config: testAccRadiusUserConfig_accountForMove(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_account.move", "name", "move-account"),
					testAccCaptureResourceID("unifi_account.move", &accountID),
				),
			},
			// Move it to unifi_radius_user; the underlying object (ID) must survive.
			{
				Config: testAccRadiusUserConfig_radiusUserAfterMove(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_radius_user.move",
						"name",
						"move-account",
					),
					resource.TestCheckResourceAttrPtr("unifi_radius_user.move", "id", &accountID),
				),
			},
		},
	})
}

func testAccRadiusUserConfig_accountForMove() string {
	return `
resource "unifi_account" "move" {
	name     = "move-account"
	password = "move-password"
}
`
}

func testAccRadiusUserConfig_radiusUserAfterMove() string {
	return `
resource "unifi_radius_user" "move" {
	name     = "move-account"
	password = "move-password"
}

moved {
	from = unifi_account.move
	to   = unifi_radius_user.move
}
`
}

// testAccCaptureResourceID stores the primary ID of a resource into dst so a
// later step can assert it is unchanged.
func testAccCaptureResourceID(name string, dst *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("resource not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("resource %s has no ID set", name)
		}
		*dst = rs.Primary.ID
		return nil
	}
}

func TestNewRadiusUserResource(t *testing.T) {
	tests := []struct {
		name string
		want fwresource.Resource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewRadiusUserResource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRadiusUserResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewRadiusUserListResource(t *testing.T) {
	tests := []struct {
		name string
		want fwlist.ListResource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewRadiusUserListResource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRadiusUserListResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_radiusUserResource_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.MetadataRequest
		resp *fwresource.MetadataResponse
	}
	tests := []struct {
		name string
		r    *radiusUserResource
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

func Test_radiusUserResource_IdentitySchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwresource.IdentitySchemaRequest
		resp *fwresource.IdentitySchemaResponse
	}
	tests := []struct {
		name string
		r    *radiusUserResource
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

func Test_radiusUserResource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.SchemaRequest
		resp *fwresource.SchemaResponse
	}
	tests := []struct {
		name string
		r    *radiusUserResource
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

func Test_radiusUserResource_Configure(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ConfigureRequest
		resp *fwresource.ConfigureResponse
	}
	tests := []struct {
		name string
		r    *radiusUserResource
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

func Test_radiusUserResource_Create(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.CreateRequest
		resp *fwresource.CreateResponse
	}
	tests := []struct {
		name string
		r    *radiusUserResource
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

func Test_radiusUserResource_Read(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ReadRequest
		resp *fwresource.ReadResponse
	}
	tests := []struct {
		name string
		r    *radiusUserResource
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

func Test_radiusUserResource_Update(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.UpdateRequest
		resp *fwresource.UpdateResponse
	}
	tests := []struct {
		name string
		r    *radiusUserResource
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

func Test_radiusUserResource_Delete(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.DeleteRequest
		resp *fwresource.DeleteResponse
	}
	tests := []struct {
		name string
		r    *radiusUserResource
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

func Test_radiusUserResource_ImportState(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ImportStateRequest
		resp *fwresource.ImportStateResponse
	}
	tests := []struct {
		name string
		r    *radiusUserResource
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

func Test_radiusUserResource_applyPlanToState(t *testing.T) {
	type args struct {
		in0   context.Context
		plan  *radiusUserResourceModel
		state *radiusUserResourceModel
	}
	tests := []struct {
		name string
		r    *radiusUserResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.applyPlanToState(tt.args.in0, tt.args.plan, tt.args.state)
		})
	}
}

func Test_radiusUserResource_modelToRadiusUser(t *testing.T) {
	type args struct {
		in0   context.Context
		model *radiusUserResourceModel
	}
	tests := []struct {
		name string
		r    *radiusUserResource
		args args
		want *unifi.Account
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.modelToRadiusUser(
				tt.args.in0,
				tt.args.model,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("radiusUserResource.modelToRadiusUser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_radiusUserResource_resolveVLAN(t *testing.T) {
	type args struct {
		ctx   context.Context
		model *radiusUserResourceModel
		site  string
	}
	tests := []struct {
		name  string
		r     *radiusUserResource
		args  args
		want  *int64
		want1 diag.Diagnostics
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.r.resolveVLAN(tt.args.ctx, tt.args.model, tt.args.site)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("radiusUserResource.resolveVLAN() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("radiusUserResource.resolveVLAN() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_radiusUserResource_radiusUserToModel(t *testing.T) {
	type args struct {
		in0     context.Context
		account *unifi.Account
		model   *radiusUserResourceModel
		site    string
	}
	tests := []struct {
		name string
		r    *radiusUserResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.radiusUserToModel(tt.args.in0, tt.args.account, tt.args.model, tt.args.site)
		})
	}
}

func Test_radiusUserResource_ListResourceConfigSchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwlist.ListResourceSchemaRequest
		resp *fwlist.ListResourceSchemaResponse
	}
	tests := []struct {
		name string
		r    *radiusUserResource
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

func Test_radiusUserResource_List(t *testing.T) {
	type args struct {
		ctx    context.Context
		req    fwlist.ListRequest
		stream *fwlist.ListResultsStream
	}
	tests := []struct {
		name string
		r    *radiusUserResource
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

// TestResolveVLAN_DeterministicBranches covers the paths of resolveVLAN that do
// not touch the controller (#67): an explicit vlan is returned as-is, and with
// neither vlan nor network_id the result is nil (untagged fallback). The
// network_id-derivation branch calls GetNetwork and is exercised by acceptance
// tests against a real controller.
func TestResolveVLAN_DeterministicBranches(t *testing.T) {
	ctx := context.Background()
	r := &radiusUserResource{} // client is nil; these branches never use it

	t.Run("explicit vlan wins", func(t *testing.T) {
		model := &radiusUserResourceModel{
			VLAN:      types.Int64Value(100),
			NetworkID: types.StringValue("net-abc"), // ignored when vlan is set
		}
		vlan, diags := r.resolveVLAN(ctx, model, "default")
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if vlan == nil || *vlan != 100 {
			t.Fatalf("vlan = %v, want 100", vlan)
		}
	})

	t.Run("no vlan and no network_id yields nil", func(t *testing.T) {
		model := &radiusUserResourceModel{
			VLAN:      types.Int64Null(),
			NetworkID: types.StringNull(),
		}
		vlan, diags := r.resolveVLAN(ctx, model, "default")
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if vlan != nil {
			t.Fatalf("vlan = %v, want nil", *vlan)
		}
	})

	t.Run("empty network_id string yields nil", func(t *testing.T) {
		model := &radiusUserResourceModel{
			VLAN:      types.Int64Null(),
			NetworkID: types.StringValue(""),
		}
		vlan, diags := r.resolveVLAN(ctx, model, "default")
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if vlan != nil {
			t.Fatalf("vlan = %v, want nil", *vlan)
		}
	})
}
