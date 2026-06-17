package unifi

import (
	"context"
	"testing"

	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func TestAccRadiusProfile_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_radius_profile.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"name",
						"tfacc-radius-profile",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"accounting_enabled",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"interim_update_enabled",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"interim_update_interval",
						"1h0m0s",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"use_usg_acct_server",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"use_usg_auth_server",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"vlan_enabled",
						"false",
					),
				),
			},
			{
				ResourceName:      "unifi_radius_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRadiusProfile_withAuthServer(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_withAuthServer(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_radius_profile.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"name",
						"tfacc-radius-profile-auth",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"auth_server.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"auth_server.0.ip",
						"192.168.1.100",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"auth_server.0.port",
						"1812",
					),
				),
			},
			{
				ResourceName:      "unifi_radius_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Secrets are not returned by the API on read
				ImportStateVerifyIgnore: []string{"auth_server.0.secret"},
			},
		},
	})
}

func TestAccRadiusProfile_withAuthServerCustomPort(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_withAuthServerCustomPort(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"auth_server.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"auth_server.0.ip",
						"10.0.0.1",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"auth_server.0.port",
						"1822",
					),
				),
			},
		},
	})
}

func TestAccRadiusProfile_withAcctServer(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_withAcctServer(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_radius_profile.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"name",
						"tfacc-radius-profile-acct",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"accounting_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"acct_server.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"acct_server.0.ip",
						"192.168.1.101",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"acct_server.0.port",
						"1813",
					),
				),
			},
			{
				ResourceName:      "unifi_radius_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Secrets are not returned by the API on read
				ImportStateVerifyIgnore: []string{"acct_server.0.secret"},
			},
		},
	})
}

func TestAccRadiusProfile_withAuthAndAcctServers(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_withAuthAndAcctServers(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_radius_profile.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"auth_server.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"acct_server.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"accounting_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"auth_server.0.ip",
						"192.168.1.100",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"acct_server.0.ip",
						"192.168.1.101",
					),
				),
			},
			{
				ResourceName:      "unifi_radius_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"auth_server.0.secret",
					"acct_server.0.secret",
				},
			},
		},
	})
}

func TestAccRadiusProfile_withInterimUpdate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_withInterimUpdate(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"interim_update_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"interim_update_interval",
						"30m0s",
					),
				),
			},
			{
				ResourceName:      "unifi_radius_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRadiusProfile_withVlan(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_withVlan(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"vlan_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"vlan_wlan_mode",
						"required",
					),
				),
			},
			{
				ResourceName:      "unifi_radius_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRadiusProfile_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"name",
						"tfacc-radius-profile",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"accounting_enabled",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"interim_update_interval",
						"1h0m0s",
					),
				),
			},
			{
				Config: testAccRadiusProfileConfig_updated(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"name",
						"tfacc-radius-profile-updated",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"accounting_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"interim_update_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"interim_update_interval",
						"30m0s",
					),
				),
			},
		},
	})
}

func TestAccRadiusProfile_importWithSite(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_radius_profile.test", "id"),
					resource.TestCheckResourceAttrSet("unifi_radius_profile.test", "site"),
				),
			},
			{
				ResourceName:      "unifi_radius_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Import using "site:id" format verified via ImportStateIdFunc below
			},
		},
	})
}

func testAccRadiusProfileConfig_basic() string {
	return `
resource "unifi_radius_profile" "test" {
  name = "tfacc-radius-profile"
}
`
}

func testAccRadiusProfileConfig_withAuthServer() string {
	return `
resource "unifi_radius_profile" "test" {
  name = "tfacc-radius-profile-auth"

  auth_server {
    ip     = "192.168.1.100"
    secret = "test-auth-secret"
  }
}
`
}

func testAccRadiusProfileConfig_withAuthServerCustomPort() string {
	return `
resource "unifi_radius_profile" "test" {
  name = "tfacc-radius-profile-auth-port"

  auth_server {
    ip     = "10.0.0.1"
    port   = 1822
    secret = "test-auth-secret"
  }
}
`
}

func testAccRadiusProfileConfig_withAcctServer() string {
	return `
resource "unifi_radius_profile" "test" {
  name               = "tfacc-radius-profile-acct"
  accounting_enabled = true

  acct_server {
    ip     = "192.168.1.101"
    secret = "test-acct-secret"
  }
}
`
}

func testAccRadiusProfileConfig_withAuthAndAcctServers() string {
	return `
resource "unifi_radius_profile" "test" {
  name               = "tfacc-radius-profile-full"
  accounting_enabled = true

  auth_server {
    ip     = "192.168.1.100"
    secret = "test-auth-secret"
  }

  acct_server {
    ip     = "192.168.1.101"
    secret = "test-acct-secret"
  }
}
`
}

func testAccRadiusProfileConfig_withInterimUpdate() string {
	return `
resource "unifi_radius_profile" "test" {
  name                    = "tfacc-radius-profile-interim"
  accounting_enabled      = true
  interim_update_enabled  = true
  interim_update_interval = "30m0s"
}
`
}

func testAccRadiusProfileConfig_withVlan() string {
	return `
resource "unifi_radius_profile" "test" {
  name           = "tfacc-radius-profile-vlan"
  vlan_enabled   = true
  vlan_wlan_mode = "required"
}
`
}

func testAccRadiusProfileConfig_updated() string {
	return `
resource "unifi_radius_profile" "test" {
  name                    = "tfacc-radius-profile-updated"
  accounting_enabled      = true
  interim_update_enabled  = true
  interim_update_interval = "30m0s"
}
`
}

func TestNewRadiusProfileResource(t *testing.T) {
	r := NewRadiusProfileResource()
	if r == nil {
		t.Fatal("NewRadiusProfileResource() returned nil")
	}
	if _, ok := r.(fwresource.ResourceWithImportState); !ok {
		t.Error("expected ResourceWithImportState interface")
	}
	if _, ok := r.(fwresource.ResourceWithUpgradeState); !ok {
		t.Error("expected ResourceWithUpgradeState interface")
	}
}

func TestNewRadiusProfileListResource(t *testing.T) {
	r := NewRadiusProfileListResource()
	if r == nil {
		t.Fatal("NewRadiusProfileListResource() returned nil")
	}
	if _, ok := r.(fwlist.ListResourceWithConfigure); !ok {
		t.Error("expected ListResourceWithConfigure interface")
	}
}

func Test_radiusProfileResource_Metadata(t *testing.T) {
	for _, tt := range []struct{ provider, want string }{
		{"unifi", "unifi_radius_profile"},
		{"test", "test_radius_profile"},
	} {
		t.Run(tt.provider, func(t *testing.T) {
			r := &radiusProfileResource{}
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

func Test_radiusProfileResource_IdentitySchema(t *testing.T) {
	r := &radiusProfileResource{}
	resp := &fwresource.IdentitySchemaResponse{}
	r.IdentitySchema(context.Background(), fwresource.IdentitySchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("IdentitySchema() produced errors: %v", resp.Diagnostics)
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Error("IdentitySchema missing 'id' attribute")
	}
}

func Test_radiusProfileResource_Schema(t *testing.T) {
	r := &radiusProfileResource{}
	resp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() produced errors: %v", resp.Diagnostics)
	}
	for _, attr := range []string{
		"id", "site", "name", "accounting_enabled", "interim_update_enabled",
		"interim_update_interval", "use_usg_acct_server", "use_usg_auth_server",
		"vlan_enabled", "vlan_wlan_mode", "timeouts",
	} {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("missing attribute %q", attr)
		}
	}
}

func Test_radiusProfileResource_UpgradeState(t *testing.T) {
	r := &radiusProfileResource{}
	upgraders := r.UpgradeState(context.Background())
	if _, ok := upgraders[0]; !ok {
		t.Error("expected state upgrader for version 0")
	}
}

func Test_radiusProfileResource_Configure(t *testing.T) {
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
			r := &radiusProfileResource{}
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

func Test_radiusProfileResource_applyPlanToState(t *testing.T) {
	ctx := context.Background()
	r := &radiusProfileResource{}

	t.Run("plan values override state", func(t *testing.T) {
		plan := &radiusProfileResourceModel{
			Name:                 types.StringValue("new-profile"),
			AccountingEnabled:    types.BoolValue(true),
			InterimUpdateEnabled: types.BoolValue(true),
			UseUSGAcctServer:     types.BoolValue(true),
			UseUSGAuthServer:     types.BoolValue(false),
			VlanEnabled:          types.BoolValue(true),
			VlanWlanMode:         types.StringValue("required"),
			AuthServer: []radiusServerModel{
				{
					IP:     types.StringValue("1.2.3.4"),
					Port:   types.Int64Value(1812),
					Secret: types.StringValue("s"),
				},
			},
			AcctServer: []radiusServerModel{},
		}
		state := &radiusProfileResourceModel{
			ID:                   types.StringValue("prof-1"),
			Name:                 types.StringValue("old-profile"),
			AccountingEnabled:    types.BoolValue(false),
			InterimUpdateEnabled: types.BoolValue(false),
			UseUSGAcctServer:     types.BoolValue(false),
			UseUSGAuthServer:     types.BoolValue(false),
			VlanEnabled:          types.BoolValue(false),
			VlanWlanMode:         types.StringValue("disabled"),
		}
		r.applyPlanToState(ctx, plan, state)
		if state.Name.ValueString() != "new-profile" {
			t.Errorf("Name = %q, want new-profile", state.Name.ValueString())
		}
		if !state.AccountingEnabled.ValueBool() {
			t.Error("AccountingEnabled should be true")
		}
		if state.VlanWlanMode.ValueString() != "required" {
			t.Errorf("VlanWlanMode = %q, want required", state.VlanWlanMode.ValueString())
		}
		if len(state.AuthServer) != 1 {
			t.Errorf("AuthServer length = %d, want 1", len(state.AuthServer))
		}
		// ID must be preserved
		if state.ID.ValueString() != "prof-1" {
			t.Errorf("ID was modified, want prof-1, got %q", state.ID.ValueString())
		}
	})

	t.Run("null plan values leave state unchanged", func(t *testing.T) {
		plan := &radiusProfileResourceModel{
			Name:                 types.StringNull(),
			AccountingEnabled:    types.BoolNull(),
			InterimUpdateEnabled: types.BoolNull(),
			UseUSGAcctServer:     types.BoolNull(),
			UseUSGAuthServer:     types.BoolNull(),
			VlanEnabled:          types.BoolNull(),
			VlanWlanMode:         types.StringNull(),
			AuthServer:           nil,
			AcctServer:           nil,
		}
		state := &radiusProfileResourceModel{
			Name:              types.StringValue("keep-profile"),
			AccountingEnabled: types.BoolValue(true),
			VlanWlanMode:      types.StringValue("optional"),
		}
		r.applyPlanToState(ctx, plan, state)
		if state.Name.ValueString() != "keep-profile" {
			t.Errorf("Name should be preserved, got %q", state.Name.ValueString())
		}
		if !state.AccountingEnabled.ValueBool() {
			t.Error("AccountingEnabled should be preserved as true")
		}
	})
}

func Test_radiusProfileResource_modelToRadiusProfile(t *testing.T) {
	ctx := context.Background()
	r := &radiusProfileResource{}

	t.Run("basic fields are converted", func(t *testing.T) {
		model := &radiusProfileResourceModel{
			Name:                 types.StringValue("my-profile"),
			AccountingEnabled:    types.BoolValue(true),
			InterimUpdateEnabled: types.BoolValue(false),
			UseUSGAcctServer:     types.BoolValue(false),
			UseUSGAuthServer:     types.BoolValue(false),
			VlanEnabled:          types.BoolValue(false),
			VlanWlanMode:         types.StringValue("disabled"),
			AuthServer:           []radiusServerModel{},
			AcctServer:           []radiusServerModel{},
		}
		got := r.modelToRadiusProfile(ctx, model)
		if got == nil {
			t.Fatal("modelToRadiusProfile() returned nil")
		}
		if got.Name != "my-profile" {
			t.Errorf("Name = %q, want my-profile", got.Name)
		}
		if !got.AccountingEnabled {
			t.Error("AccountingEnabled should be true")
		}
		if got.VLANWLANMode != "disabled" {
			t.Errorf("VLANWLANMode = %q, want disabled", got.VLANWLANMode)
		}
	})

	t.Run("auth and acct servers are appended", func(t *testing.T) {
		port := int64(1812)
		model := &radiusProfileResourceModel{
			Name:              types.StringValue("prof-with-servers"),
			AccountingEnabled: types.BoolValue(false),
			VlanWlanMode:      types.StringValue(""),
			AuthServer: []radiusServerModel{
				{
					IP:     types.StringValue("10.0.0.1"),
					Port:   types.Int64Value(port),
					Secret: types.StringValue("auth-secret"),
				},
			},
			AcctServer: []radiusServerModel{
				{
					IP:     types.StringValue("10.0.0.2"),
					Port:   types.Int64Value(1813),
					Secret: types.StringValue("acct-secret"),
				},
			},
		}
		got := r.modelToRadiusProfile(ctx, model)
		if len(got.AuthServers) != 1 {
			t.Fatalf("AuthServers length = %d, want 1", len(got.AuthServers))
		}
		if got.AuthServers[0].IP != "10.0.0.1" {
			t.Errorf("AuthServer IP = %q, want 10.0.0.1", got.AuthServers[0].IP)
		}
		if len(got.AcctServers) != 1 {
			t.Fatalf("AcctServers length = %d, want 1", len(got.AcctServers))
		}
		if got.AcctServers[0].IP != "10.0.0.2" {
			t.Errorf("AcctServer IP = %q, want 10.0.0.2", got.AcctServers[0].IP)
		}
	})
}

func Test_radiusProfileResource_radiusProfileToModel(t *testing.T) {
	ctx := context.Background()
	r := &radiusProfileResource{}

	t.Run("all fields are mapped from API", func(t *testing.T) {
		port1812 := int64(1812)
		port1813 := int64(1813)
		interval := int64(3600)
		profile := &unifi.RADIUSProfile{
			ID:                    "prof-1",
			Name:                  "my-profile",
			AccountingEnabled:     true,
			InterimUpdateEnabled:  false,
			InterimUpdateInterval: &interval,
			UseUsgAcctServer:      false,
			UseUsgAuthServer:      true,
			VLANEnabled:           true,
			VLANWLANMode:          "required",
			AuthServers: []unifi.RADIUSProfileAuthServers{
				{IP: "1.2.3.4", Port: &port1812, Secret: ""},
			},
			AcctServers: []unifi.RADIUSProfileAcctServers{
				{IP: "5.6.7.8", Port: &port1813, Secret: ""},
			},
		}
		model := &radiusProfileResourceModel{}
		r.radiusProfileToModel(ctx, profile, model, "default")

		if model.ID.ValueString() != "prof-1" {
			t.Errorf("ID = %q, want prof-1", model.ID.ValueString())
		}
		if model.Site.ValueString() != "default" {
			t.Errorf("Site = %q, want default", model.Site.ValueString())
		}
		if model.Name.ValueString() != "my-profile" {
			t.Errorf("Name = %q, want my-profile", model.Name.ValueString())
		}
		if !model.AccountingEnabled.ValueBool() {
			t.Error("AccountingEnabled should be true")
		}
		if !model.UseUSGAuthServer.ValueBool() {
			t.Error("UseUSGAuthServer should be true")
		}
		if !model.VlanEnabled.ValueBool() {
			t.Error("VlanEnabled should be true")
		}
		if model.VlanWlanMode.ValueString() != "required" {
			t.Errorf("VlanWlanMode = %q, want required", model.VlanWlanMode.ValueString())
		}
		if len(model.AuthServer) != 1 {
			t.Fatalf("AuthServer length = %d, want 1", len(model.AuthServer))
		}
		if model.AuthServer[0].IP.ValueString() != "1.2.3.4" {
			t.Errorf("AuthServer IP = %q, want 1.2.3.4", model.AuthServer[0].IP.ValueString())
		}
		if len(model.AcctServer) != 1 {
			t.Fatalf("AcctServer length = %d, want 1", len(model.AcctServer))
		}
	})

	t.Run("empty servers produce empty slices not nil", func(t *testing.T) {
		profile := &unifi.RADIUSProfile{
			ID:          "prof-2",
			Name:        "empty-prof",
			AuthServers: nil,
			AcctServers: nil,
		}
		model := &radiusProfileResourceModel{}
		r.radiusProfileToModel(ctx, profile, model, "site1")
		if model.AuthServer == nil {
			t.Error("AuthServer should be empty slice, not nil")
		}
		if model.AcctServer == nil {
			t.Error("AcctServer should be empty slice, not nil")
		}
	})
}

func Test_radiusProfileResource_ListResourceConfigSchema(t *testing.T) {
	r := &radiusProfileResource{}
	resp := &fwlist.ListResourceSchemaResponse{}
	r.ListResourceConfigSchema(context.Background(), fwlist.ListResourceSchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("ListResourceConfigSchema() produced errors: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["site"]; !ok {
		t.Error("ListResourceConfigSchema missing 'site' attribute")
	}
}

func TestAccRadiusProfileList_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_basic(),
			},
			{
				Query: true,
				Config: `
					provider "unifi" {}
					list "unifi_radius_profile" "test" {
						provider = unifi
						config {
							filter {
								name  = "name"
								value = "tfacc-radius-profile"
						  }
					  }
					}
				`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLengthAtLeast("unifi_radius_profile.test", 1),
				},
			},
		},
	})
}
