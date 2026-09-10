package unifi

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
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
					resource.TestCheckResourceAttr("unifi_radius_user.test", "tunnel.type", "3"),
					resource.TestCheckResourceAttr(
						"unifi_radius_user.test",
						"tunnel.medium_type",
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
			// Identity-based import (import block with identity, Terraform 1.12+).
			{
				ResourceName:    "unifi_radius_user.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
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
						"tunnel.config_type",
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
	name     = "test-account-vlan"
	password = "test-password"
	vlan     = 100

	tunnel = {
		config_type = "802.1x"
	}
}
`
}

// TestAccRadiusUser_tunnelType13 verifies that tunnel.type accepts 13 (VLAN),
// which the controller allows (1-13) but the provider previously capped at 12.
func TestAccRadiusUser_tunnelType13(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusUserConfig_tunnelType13(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_radius_user.tt13", "tunnel.type", "13"),
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
	name     = "test-account-tt13"
	password = "test-password"

	tunnel = {
		type = 13
	}
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
	if _, ok := r.(fwresource.ResourceWithUpgradeState); !ok {
		t.Error("expected ResourceWithUpgradeState interface")
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
	for _, name := range []string{
		"id", "site", "name", "password", "tunnel", "network_id", "vlan", "timeouts",
	} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	for _, flat := range []string{"tunnel_type", "tunnel_medium_type", "tunnel_config_type"} {
		if _, ok := resp.Schema.Attributes[flat]; ok {
			t.Errorf("flat attribute %q should have been nested", flat)
		}
	}
	if resp.Schema.Version != radiusUserSchemaVersion {
		t.Errorf("Schema.Version = %d, want %d", resp.Schema.Version, radiusUserSchemaVersion)
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

// testRadiusUserTunnel builds a `tunnel` object for unit tests.
func testRadiusUserTunnel(typ, medium types.Int64, config types.String) types.Object {
	return types.ObjectValueMust(radiusUserTunnelAttrTypes(), map[string]attr.Value{
		"type":        typ,
		"medium_type": medium,
		"config_type": config,
	})
}

func Test_radiusUserResource_applyPlanToState(t *testing.T) {
	ctx := context.Background()
	r := &radiusUserResource{}

	t.Run("plan values override state", func(t *testing.T) {
		plan := &radiusUserResourceModel{
			Name:     types.StringValue("new-name"),
			Password: types.StringValue("new-pass"),
			Tunnel: testRadiusUserTunnel(
				types.Int64Value(13),
				types.Int64Value(6),
				types.StringValue("802.1x"),
			),
			NetworkID: types.StringValue("net-xyz"),
			VLAN:      types.Int64Value(200),
		}
		state := &radiusUserResourceModel{
			ID:        types.StringValue("existing-id"),
			Name:      types.StringValue("old-name"),
			Password:  types.StringValue("old-pass"),
			Tunnel:    radiusUserTunnelDefault(),
			NetworkID: types.StringNull(),
			VLAN:      types.Int64Null(),
		}
		r.applyPlanToState(ctx, plan, state)
		if state.Name.ValueString() != "new-name" {
			t.Errorf("Name = %q, want new-name", state.Name.ValueString())
		}
		if !state.Tunnel.Equal(plan.Tunnel) {
			t.Errorf("tunnel = %v, want %v", state.Tunnel, plan.Tunnel)
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
			Name:      types.StringNull(),
			Password:  types.StringNull(),
			Tunnel:    types.ObjectNull(radiusUserTunnelAttrTypes()),
			NetworkID: types.StringNull(),
			VLAN:      types.Int64Null(),
		}
		state := &radiusUserResourceModel{
			Name:   types.StringValue("keep-name"),
			Tunnel: radiusUserTunnelDefault(),
		}
		r.applyPlanToState(ctx, plan, state)
		if state.Name.ValueString() != "keep-name" {
			t.Errorf("Name should be preserved, got %q", state.Name.ValueString())
		}
		if !state.Tunnel.Equal(radiusUserTunnelDefault()) {
			t.Errorf("tunnel should be preserved, got %v", state.Tunnel)
		}
	})

	t.Run("unset tunnel leaves keep their state value", func(t *testing.T) {
		plan := &radiusUserResourceModel{
			Tunnel: testRadiusUserTunnel(
				types.Int64Value(13),
				types.Int64Null(),
				types.StringNull(),
			),
		}
		state := &radiusUserResourceModel{
			Tunnel: testRadiusUserTunnel(
				types.Int64Value(3),
				types.Int64Value(6),
				types.StringValue("802.1x"),
			),
		}
		r.applyPlanToState(ctx, plan, state)
		want := testRadiusUserTunnel(
			types.Int64Value(13),
			types.Int64Value(6),
			types.StringValue("802.1x"),
		)
		if !state.Tunnel.Equal(want) {
			t.Errorf("tunnel = %v, want %v", state.Tunnel, want)
		}
	})
}

func Test_radiusUserResource_modelToRadiusUser(t *testing.T) {
	ctx := context.Background()
	r := &radiusUserResource{}

	t.Run("basic fields are set", func(t *testing.T) {
		model := &radiusUserResourceModel{
			Name:      types.StringValue("alice"),
			Password:  types.StringValue("secret"),
			Tunnel:    radiusUserTunnelDefault(),
			NetworkID: types.StringNull(),
			VLAN:      types.Int64Null(),
		}
		got, diags := r.modelToRadiusUser(ctx, model)
		if diags.HasError() {
			t.Fatalf("modelToRadiusUser() diagnostics: %v", diags)
		}
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
		if got.TunnelConfigType != "" {
			t.Errorf("TunnelConfigType = %q, want empty", got.TunnelConfigType)
		}
	})

	t.Run("optional fields are populated when set", func(t *testing.T) {
		model := &radiusUserResourceModel{
			Name:     types.StringValue("bob"),
			Password: types.StringValue("pass"),
			Tunnel: testRadiusUserTunnel(
				types.Int64Value(13),
				types.Int64Value(6),
				types.StringValue("802.1x"),
			),
			NetworkID: types.StringValue("net-abc"),
			VLAN:      types.Int64Value(100),
		}
		got, diags := r.modelToRadiusUser(ctx, model)
		if diags.HasError() {
			t.Fatalf("modelToRadiusUser() diagnostics: %v", diags)
		}
		if got.TunnelType == nil || *got.TunnelType != 13 {
			t.Errorf("TunnelType = %v, want 13", got.TunnelType)
		}
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
		if d := r.radiusUserToModel(ctx, account, model, "default"); d.HasError() {
			t.Fatalf("radiusUserToModel() diagnostics: %v", d)
		}

		if model.ID.ValueString() != "acc-1" {
			t.Errorf("ID = %q, want acc-1", model.ID.ValueString())
		}
		if model.Site.ValueString() != "default" {
			t.Errorf("Site = %q, want default", model.Site.ValueString())
		}
		if model.Name.ValueString() != "alice" {
			t.Errorf("Name = %q, want alice", model.Name.ValueString())
		}
		want := testRadiusUserTunnel(
			types.Int64Value(3),
			types.Int64Value(6),
			types.StringValue("802.1x"),
		)
		if !model.Tunnel.Equal(want) {
			t.Errorf("tunnel = %v, want %v", model.Tunnel, want)
		}
		if model.NetworkID.ValueString() != "net-abc" {
			t.Errorf("NetworkID = %q, want net-abc", model.NetworkID.ValueString())
		}
		if model.VLAN.ValueInt64() != 100 {
			t.Errorf("VLAN = %d, want 100", model.VLAN.ValueInt64())
		}
	})

	t.Run("empty strings become null", func(t *testing.T) {
		account := &unifi.Account{
			ID:               "acc-2",
			NetworkID:        "",
			TunnelConfigType: "",
		}
		model := &radiusUserResourceModel{}
		if d := r.radiusUserToModel(ctx, account, model, "site1"); d.HasError() {
			t.Fatalf("radiusUserToModel() diagnostics: %v", d)
		}

		if !model.NetworkID.IsNull() {
			t.Errorf(
				"NetworkID should be null for empty string, got %q",
				model.NetworkID.ValueString(),
			)
		}
		if got := model.Tunnel.Attributes()["config_type"]; !got.IsNull() {
			t.Errorf("tunnel.config_type should be null for empty string, got %v", got)
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

// TestRadiusUserUpgradeState_v0NestsTunnel guards the v0 -> v1 schema upgrade:
// the flat tunnel_type, tunnel_medium_type and tunnel_config_type attributes
// move into the nested tunnel object. unifi_account shares this schema and
// upgrader.
func TestRadiusUserUpgradeState_v0NestsTunnel(t *testing.T) {
	ctx := context.Background()
	r := &radiusUserResource{}

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	if schemaResp.Schema.Version != 1 {
		t.Fatalf("radius user schema Version = %d, want 1", schemaResp.Schema.Version)
	}
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	ups := r.UpgradeState(ctx)
	if _, ok := ups[0]; !ok {
		t.Fatal("no upgrader registered for schema version 0")
	}

	upgrade := func(t *testing.T, prior string) map[string]tftypes.Value {
		t.Helper()
		resp := &fwresource.UpgradeStateResponse{}
		ups[0].StateUpgrader(ctx, fwresource.UpgradeStateRequest{
			RawState: &tfprotov6.RawState{JSON: []byte(prior)},
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
		return root
	}
	num := func(t *testing.T, v tftypes.Value, name string, want int64) {
		t.Helper()
		var f big.Float
		if err := v.As(&f); err != nil {
			t.Errorf("%s = %v: %v", name, v, err)
			return
		}
		if got, _ := f.Int64(); got != want {
			t.Errorf("%s = %d, want %d", name, got, want)
		}
	}

	t.Run("flat tunnel attributes move under tunnel", func(t *testing.T) {
		root := upgrade(t, `{
			"id": "acc-1", "site": "default", "name": "alice", "password": "secret",
			"tunnel_type": 13, "tunnel_medium_type": 6, "tunnel_config_type": "802.1x",
			"network_id": "net-1", "vlan": 100,
			"timeouts": null
		}`)
		for _, flat := range []string{"tunnel_type", "tunnel_medium_type", "tunnel_config_type"} {
			if _, exists := root[flat]; exists {
				t.Errorf("flat attribute %q survived the upgrade", flat)
			}
		}
		var tunnel map[string]tftypes.Value
		if err := root["tunnel"].As(&tunnel); err != nil {
			t.Fatalf("tunnel: as object: %v (value %v)", err, root["tunnel"])
		}
		num(t, tunnel["type"], "tunnel.type", 13)
		num(t, tunnel["medium_type"], "tunnel.medium_type", 6)
		var cfg string
		if err := tunnel["config_type"].As(&cfg); err != nil || cfg != "802.1x" {
			t.Errorf("tunnel.config_type = %v (%v), want 802.1x", tunnel["config_type"], err)
		}
		// Attributes that were not nested survive untouched.
		num(t, root["vlan"], "vlan", 100)
	})

	t.Run("null config_type stays null inside tunnel", func(t *testing.T) {
		root := upgrade(t, `{
			"id": "acc-2", "site": "default", "name": "bob", "password": "secret",
			"tunnel_type": 3, "tunnel_medium_type": 6, "tunnel_config_type": null
		}`)
		var tunnel map[string]tftypes.Value
		if err := root["tunnel"].As(&tunnel); err != nil {
			t.Fatalf("tunnel: as object: %v (value %v)", err, root["tunnel"])
		}
		num(t, tunnel["type"], "tunnel.type", 3)
		if !tunnel["config_type"].IsNull() {
			t.Errorf("tunnel.config_type = %v, want null", tunnel["config_type"])
		}
	})

	t.Run("state without tunnel attributes leaves tunnel null", func(t *testing.T) {
		root := upgrade(t, `{"id": "acc-3", "site": "default", "name": "bare", "password": "x"}`)
		if !root["tunnel"].IsNull() {
			t.Errorf("tunnel = %v, want null", root["tunnel"])
		}
	})
}

// TestRadiusUserNestedTunnel_wireAndReadBack checks that the nested tunnel
// group is written to and read back from the API struct, that the object
// default reproduces what the flat defaults sent when omitted, and that a
// null/unknown group contributes nothing.
func TestRadiusUserNestedTunnel_wireAndReadBack(t *testing.T) {
	ctx := context.Background()
	r := &radiusUserResource{}

	model := &radiusUserResourceModel{
		Name:     types.StringValue("alice"),
		Password: types.StringValue("secret"),
		Tunnel: testRadiusUserTunnel(
			types.Int64Value(13),
			types.Int64Value(6),
			types.StringValue("802.1x"),
		),
	}
	api, diags := r.modelToRadiusUser(ctx, model)
	if diags.HasError() {
		t.Fatalf("modelToRadiusUser: %v", diags)
	}
	if api.TunnelType == nil || *api.TunnelType != 13 ||
		api.TunnelMediumType == nil || *api.TunnelMediumType != 6 ||
		api.TunnelConfigType != "802.1x" {
		t.Errorf("tunnel: %v %v %q", api.TunnelType, api.TunnelMediumType, api.TunnelConfigType)
	}

	// Read back: the group is rebuilt from the API response.
	var back radiusUserResourceModel
	if d := r.radiusUserToModel(ctx, api, &back, "default"); d.HasError() {
		t.Fatalf("radiusUserToModel: %v", d)
	}
	if !back.Tunnel.Equal(model.Tunnel) {
		t.Errorf("tunnel read back = %v, want %v", back.Tunnel, model.Tunnel)
	}

	// The object default reproduces what the flat defaults used to send:
	// tunnel_type 3, tunnel_medium_type 6 and no tunnel_config_type.
	api, diags = r.modelToRadiusUser(ctx, &radiusUserResourceModel{
		Name:     types.StringValue("plain"),
		Password: types.StringValue("secret"),
		Tunnel:   radiusUserTunnelDefault(),
	})
	if diags.HasError() {
		t.Fatalf("modelToRadiusUser (default): %v", diags)
	}
	if api.TunnelType == nil || *api.TunnelType != 3 ||
		api.TunnelMediumType == nil || *api.TunnelMediumType != 6 ||
		api.TunnelConfigType != "" {
		t.Errorf(
			"default tunnel: %v %v %q",
			api.TunnelType,
			api.TunnelMediumType,
			api.TunnelConfigType,
		)
	}

	// A null or unknown group contributes nothing, as unset flat attributes did.
	for name, obj := range map[string]types.Object{
		"null":    types.ObjectNull(radiusUserTunnelAttrTypes()),
		"unknown": types.ObjectUnknown(radiusUserTunnelAttrTypes()),
	} {
		api, diags = r.modelToRadiusUser(ctx, &radiusUserResourceModel{
			Name:     types.StringValue("bare"),
			Password: types.StringValue("secret"),
			Tunnel:   obj,
		})
		if diags.HasError() {
			t.Fatalf("modelToRadiusUser (%s): %v", name, diags)
		}
		if api.TunnelType != nil || api.TunnelMediumType != nil || api.TunnelConfigType != "" {
			t.Errorf("%s tunnel leaked into the request: %+v", name, api)
		}
	}
}
