package unifi

import (
	"context"
	"fmt"
	"os"
	"testing"

	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func testAccRadiusUserCheckDestroy(s *terraform.State) error {
	ctx := context.Background()
	apiURL := os.Getenv("UNIFI_API")
	if apiURL == "" {
		return nil
	}
	apiClient, err := unifi.New(ctx, &unifi.Config{
		BaseURL:       apiURL,
		Username:      os.Getenv("UNIFI_USERNAME"),
		Password:      os.Getenv("UNIFI_PASSWORD"),
		AllowInsecure: true,
	})
	if err != nil {
		return nil //nolint:nilerr // best-effort check; skip when no live client
	}
	c := &Client{ApiClient: apiClient, Site: "default"}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "unifi_radius_user" {
			continue
		}
		site := rs.Primary.Attributes["site"]
		if site == "" {
			site = c.Site
		}
		_, err := c.GetAccount(ctx, site, rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("unifi_radius_user %s still exists", rs.Primary.ID)
		}
		if _, ok := err.(*unifi.NotFoundError); !ok {
			return err
		}
	}
	return nil
}

func TestAccRadiusUser_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: providerFactories,
		CheckDestroy:             testAccRadiusUserCheckDestroy,
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
	r := NewRadiusUserResource()
	if r == nil {
		t.Fatal("NewRadiusUserResource() returned nil")
	}
	if _, ok := r.(fwresource.ResourceWithImportState); !ok {
		t.Error("expected ResourceWithImportState interface")
	}
}

func TestNewRadiusUserListResource(t *testing.T) {
	r := NewRadiusUserListResource()
	if r == nil {
		t.Fatal("NewRadiusUserListResource() returned nil")
	}
	if _, ok := r.(fwlist.ListResourceWithConfigure); !ok {
		t.Error("expected ListResourceWithConfigure interface")
	}
}

func Test_radiusUserResource_Metadata(t *testing.T) {
	for _, tt := range []struct{ provider, want string }{
		{"unifi", "unifi_radius_user"},
		{"test", "test_radius_user"},
	} {
		t.Run(tt.provider, func(t *testing.T) {
			r := &radiusUserResource{}
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

func Test_radiusUserResource_IdentitySchema(t *testing.T) {
	r := &radiusUserResource{}
	resp := &fwresource.IdentitySchemaResponse{}
	r.IdentitySchema(context.Background(), fwresource.IdentitySchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("IdentitySchema() produced errors: %v", resp.Diagnostics)
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Error("IdentitySchema missing 'id' attribute")
	}
}

func Test_radiusUserResource_Schema(t *testing.T) {
	r := &radiusUserResource{}
	resp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() produced errors: %v", resp.Diagnostics)
	}
	for _, attr := range []string{
		"id", "site", "name", "password", "tunnel_type",
		"tunnel_medium_type", "network_id", "vlan", "tunnel_config_type", "timeouts",
	} {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("missing attribute %q", attr)
		}
	}
}

func Test_radiusUserResource_Configure(t *testing.T) {
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
			r := &radiusUserResource{}
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

func Test_radiusUserResource_IdentitySchemaStub(t *testing.T) {
	// Already covered by Test_radiusUserResource_IdentitySchema above.
}

func Test_radiusUserResource_applyPlanToState(t *testing.T) {
	ctx := context.Background()
	r := &radiusUserResource{}

	t.Run("plan values override state", func(t *testing.T) {
		plan := &radiusUserResourceModel{
			Name:             types.StringValue("new-name"),
			Password:         types.StringValue("new-pass"),
			TunnelType:       types.Int64Value(13),
			TunnelMediumType: types.Int64Value(6),
			NetworkID:        types.StringValue("net-xyz"),
			VLAN:             types.Int64Value(200),
			TunnelConfigType: types.StringValue("802.1x"),
		}
		state := &radiusUserResourceModel{
			ID:               types.StringValue("existing-id"),
			Name:             types.StringValue("old-name"),
			Password:         types.StringValue("old-pass"),
			TunnelType:       types.Int64Value(3),
			TunnelMediumType: types.Int64Value(6),
			NetworkID:        types.StringNull(),
			VLAN:             types.Int64Null(),
			TunnelConfigType: types.StringNull(),
		}
		r.applyPlanToState(ctx, plan, state)
		if state.Name.ValueString() != "new-name" {
			t.Errorf("Name = %q, want new-name", state.Name.ValueString())
		}
		if state.TunnelType.ValueInt64() != 13 {
			t.Errorf("TunnelType = %d, want 13", state.TunnelType.ValueInt64())
		}
		if state.VLAN.ValueInt64() != 200 {
			t.Errorf("VLAN = %d, want 200", state.VLAN.ValueInt64())
		}
		// ID must be preserved (applyPlanToState doesn't touch it)
		if state.ID.ValueString() != "existing-id" {
			t.Errorf("ID was modified, want existing-id, got %q", state.ID.ValueString())
		}
	})

	t.Run("null plan values leave state unchanged", func(t *testing.T) {
		plan := &radiusUserResourceModel{
			Name:             types.StringNull(),
			Password:         types.StringNull(),
			TunnelType:       types.Int64Null(),
			TunnelMediumType: types.Int64Null(),
			NetworkID:        types.StringNull(),
			VLAN:             types.Int64Null(),
			TunnelConfigType: types.StringNull(),
		}
		state := &radiusUserResourceModel{
			Name:             types.StringValue("keep-name"),
			TunnelType:       types.Int64Value(3),
			TunnelMediumType: types.Int64Value(6),
		}
		r.applyPlanToState(ctx, plan, state)
		if state.Name.ValueString() != "keep-name" {
			t.Errorf("Name should be preserved, got %q", state.Name.ValueString())
		}
		if state.TunnelType.ValueInt64() != 3 {
			t.Errorf("TunnelType should be preserved, got %d", state.TunnelType.ValueInt64())
		}
	})
}

func Test_radiusUserResource_modelToRadiusUser(t *testing.T) {
	ctx := context.Background()
	r := &radiusUserResource{}

	t.Run("basic fields are set", func(t *testing.T) {
		tt3 := int64(3)
		tt6 := int64(6)
		model := &radiusUserResourceModel{
			Name:             types.StringValue("alice"),
			Password:         types.StringValue("secret"),
			TunnelType:       types.Int64Value(tt3),
			TunnelMediumType: types.Int64Value(tt6),
			NetworkID:        types.StringNull(),
			VLAN:             types.Int64Null(),
			TunnelConfigType: types.StringNull(),
		}
		got := r.modelToRadiusUser(ctx, model)
		if got == nil {
			t.Fatal("modelToRadiusUser() returned nil")
		}
		if got.Name != "alice" {
			t.Errorf("Name = %q, want alice", got.Name)
		}
		if got.Password != "secret" {
			t.Errorf("Password = %q, want secret", got.Password)
		}
		if got.TunnelType == nil || *got.TunnelType != 3 {
			t.Errorf("TunnelType = %v, want 3", got.TunnelType)
		}
		if got.TunnelMediumType == nil || *got.TunnelMediumType != 6 {
			t.Errorf("TunnelMediumType = %v, want 6", got.TunnelMediumType)
		}
		if got.NetworkID != "" {
			t.Errorf("NetworkID = %q, want empty", got.NetworkID)
		}
	})

	t.Run("optional fields are populated when set", func(t *testing.T) {
		model := &radiusUserResourceModel{
			Name:             types.StringValue("bob"),
			Password:         types.StringValue("pass"),
			TunnelType:       types.Int64Value(13),
			TunnelMediumType: types.Int64Value(6),
			NetworkID:        types.StringValue("net-abc"),
			VLAN:             types.Int64Value(100),
			TunnelConfigType: types.StringValue("802.1x"),
		}
		got := r.modelToRadiusUser(ctx, model)
		if got.NetworkID != "net-abc" {
			t.Errorf("NetworkID = %q, want net-abc", got.NetworkID)
		}
		if got.VLAN == nil || *got.VLAN != 100 {
			t.Errorf("VLAN = %v, want 100", got.VLAN)
		}
		if got.TunnelConfigType != "802.1x" {
			t.Errorf("TunnelConfigType = %q, want 802.1x", got.TunnelConfigType)
		}
	})
}

func Test_radiusUserResource_resolveVLAN(t *testing.T) {
	// client-independent branches (no network lookup needed)
	ctx := context.Background()
	r := &radiusUserResource{}

	t.Run("explicit vlan wins", func(t *testing.T) {
		model := &radiusUserResourceModel{
			VLAN:      types.Int64Value(100),
			NetworkID: types.StringValue("net-abc"),
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

func Test_radiusUserResource_radiusUserToModel(t *testing.T) {
	ctx := context.Background()
	r := &radiusUserResource{}

	t.Run("all fields are mapped from API", func(t *testing.T) {
		tt := int64(3)
		mt := int64(6)
		vlan := int64(100)
		account := &unifi.Account{
			ID:               "acc-1",
			Name:             "alice",
			Password:         "secret",
			TunnelType:       &tt,
			TunnelMediumType: &mt,
			NetworkID:        "net-abc",
			VLAN:             &vlan,
			TunnelConfigType: "802.1x",
		}
		model := &radiusUserResourceModel{}
		r.radiusUserToModel(ctx, account, model, "default")

		if model.ID.ValueString() != "acc-1" {
			t.Errorf("ID = %q, want acc-1", model.ID.ValueString())
		}
		if model.Site.ValueString() != "default" {
			t.Errorf("Site = %q, want default", model.Site.ValueString())
		}
		if model.Name.ValueString() != "alice" {
			t.Errorf("Name = %q, want alice", model.Name.ValueString())
		}
		if model.TunnelType.ValueInt64() != 3 {
			t.Errorf("TunnelType = %d, want 3", model.TunnelType.ValueInt64())
		}
		if model.NetworkID.ValueString() != "net-abc" {
			t.Errorf("NetworkID = %q, want net-abc", model.NetworkID.ValueString())
		}
		if model.VLAN.ValueInt64() != 100 {
			t.Errorf("VLAN = %d, want 100", model.VLAN.ValueInt64())
		}
		if model.TunnelConfigType.ValueString() != "802.1x" {
			t.Errorf("TunnelConfigType = %q, want 802.1x", model.TunnelConfigType.ValueString())
		}
	})

	t.Run("empty strings become null", func(t *testing.T) {
		account := &unifi.Account{
			ID:               "acc-2",
			NetworkID:        "",
			TunnelConfigType: "",
		}
		model := &radiusUserResourceModel{}
		r.radiusUserToModel(ctx, account, model, "site1")

		if !model.NetworkID.IsNull() {
			t.Errorf(
				"NetworkID should be null for empty string, got %q",
				model.NetworkID.ValueString(),
			)
		}
		if !model.TunnelConfigType.IsNull() {
			t.Errorf(
				"TunnelConfigType should be null for empty string, got %q",
				model.TunnelConfigType.ValueString(),
			)
		}
	})
}

func Test_radiusUserResource_ListResourceConfigSchema(t *testing.T) {
	r := &radiusUserResource{}
	resp := &fwlist.ListResourceSchemaResponse{}
	r.ListResourceConfigSchema(context.Background(), fwlist.ListResourceSchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("ListResourceConfigSchema() produced errors: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["site"]; !ok {
		t.Error("ListResourceConfigSchema missing 'site' attribute")
	}
}

func TestAccRadiusUserList_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusUserConfig_basic(),
			},
			{
				Query: true,
				Config: `
					provider "unifi" {}
					list "unifi_radius_user" "test" {
						provider = unifi
						config {
							filter {
								name  = "name"
								value = "test-account"
						  }
					  }
					}
				`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLengthAtLeast("unifi_radius_user.test", 1),
				},
			},
		},
	})
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
