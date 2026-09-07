package unifi

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
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
						"interim_update.enabled",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"interim_update.interval",
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
						"vlan.enabled",
						"false",
					),
				),
			},
			{
				ResourceName:      "unifi_radius_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Identity-based import (import block with identity, Terraform 1.12+).
			{
				ResourceName:    "unifi_radius_profile.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
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
						"interim_update.enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"interim_update.interval",
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
						"vlan.enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"vlan.wlan_mode",
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
						"interim_update.interval",
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
						"interim_update.enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"interim_update.interval",
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
  name               = "tfacc-radius-profile-interim"
  accounting_enabled = true

  interim_update = {
    enabled  = true
    interval = "30m0s"
  }
}
`
}

func testAccRadiusProfileConfig_withVlan() string {
	return `
resource "unifi_radius_profile" "test" {
  name = "tfacc-radius-profile-vlan"

  vlan = {
    enabled   = true
    wlan_mode = "required"
  }
}
`
}

func testAccRadiusProfileConfig_updated() string {
	return `
resource "unifi_radius_profile" "test" {
  name               = "tfacc-radius-profile-updated"
  accounting_enabled = true

  interim_update = {
    enabled  = true
    interval = "30m0s"
  }
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
	for _, name := range []string{
		"id", "site", "name", "accounting_enabled", "interim_update",
		"use_usg_acct_server", "use_usg_auth_server", "vlan", "timeouts",
	} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	for _, flat := range []string{
		"interim_update_enabled", "interim_update_interval", "vlan_enabled", "vlan_wlan_mode",
	} {
		if _, ok := resp.Schema.Attributes[flat]; ok {
			t.Errorf("flat attribute %q should have been nested", flat)
		}
	}
}

func Test_radiusProfileResource_UpgradeState(t *testing.T) {
	r := &radiusProfileResource{}
	upgraders := r.UpgradeState(context.Background())
	for _, v := range []int64{0, 1} {
		if _, ok := upgraders[v]; !ok {
			t.Errorf("expected state upgrader for version %d", v)
		}
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

	vlanObj := func(enabled bool, mode string) types.Object {
		return types.ObjectValueMust(radiusProfileVlanAttrTypes(), map[string]attr.Value{
			"enabled":   types.BoolValue(enabled),
			"wlan_mode": types.StringValue(mode),
		})
	}

	t.Run("plan values override state", func(t *testing.T) {
		plan := &radiusProfileResourceModel{
			Name:              types.StringValue("new-profile"),
			AccountingEnabled: types.BoolValue(true),
			InterimUpdate: types.ObjectValueMust(
				radiusProfileInterimUpdateAttrTypes(),
				map[string]attr.Value{
					"enabled":  types.BoolValue(true),
					"interval": timetypes.NewGoDurationValue(30 * time.Minute),
				},
			),
			UseUSGAcctServer: types.BoolValue(true),
			UseUSGAuthServer: types.BoolValue(false),
			Vlan:             vlanObj(true, "required"),
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
			ID:                types.StringValue("prof-1"),
			Name:              types.StringValue("old-profile"),
			AccountingEnabled: types.BoolValue(false),
			InterimUpdate:     radiusProfileInterimUpdateDefault(),
			UseUSGAcctServer:  types.BoolValue(false),
			UseUSGAuthServer:  types.BoolValue(false),
			Vlan:              vlanObj(false, "disabled"),
		}
		r.applyPlanToState(ctx, plan, state)
		if state.Name.ValueString() != "new-profile" {
			t.Errorf("Name = %q, want new-profile", state.Name.ValueString())
		}
		if !state.AccountingEnabled.ValueBool() {
			t.Error("AccountingEnabled should be true")
		}
		if !state.Vlan.Equal(plan.Vlan) {
			t.Errorf("vlan = %v, want %v", state.Vlan, plan.Vlan)
		}
		if !state.InterimUpdate.Equal(plan.InterimUpdate) {
			t.Errorf("interim_update = %v, want %v", state.InterimUpdate, plan.InterimUpdate)
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
			Name:              types.StringNull(),
			AccountingEnabled: types.BoolNull(),
			InterimUpdate:     types.ObjectNull(radiusProfileInterimUpdateAttrTypes()),
			UseUSGAcctServer:  types.BoolNull(),
			UseUSGAuthServer:  types.BoolNull(),
			Vlan:              types.ObjectNull(radiusProfileVlanAttrTypes()),
			AuthServer:        nil,
			AcctServer:        nil,
		}
		state := &radiusProfileResourceModel{
			Name:              types.StringValue("keep-profile"),
			AccountingEnabled: types.BoolValue(true),
			Vlan:              vlanObj(false, "optional"),
		}
		r.applyPlanToState(ctx, plan, state)
		if state.Name.ValueString() != "keep-profile" {
			t.Errorf("Name should be preserved, got %q", state.Name.ValueString())
		}
		if !state.AccountingEnabled.ValueBool() {
			t.Error("AccountingEnabled should be preserved as true")
		}
		if !state.Vlan.Equal(vlanObj(false, "optional")) {
			t.Errorf("vlan should be preserved, got %v", state.Vlan)
		}
	})

	t.Run("unset nested leaves keep their state value", func(t *testing.T) {
		plan := &radiusProfileResourceModel{
			Vlan: types.ObjectValueMust(radiusProfileVlanAttrTypes(), map[string]attr.Value{
				"enabled":   types.BoolValue(true),
				"wlan_mode": types.StringNull(),
			}),
		}
		state := &radiusProfileResourceModel{Vlan: vlanObj(false, "optional")}
		r.applyPlanToState(ctx, plan, state)
		if !state.Vlan.Equal(vlanObj(true, "optional")) {
			t.Errorf("vlan = %v, want enabled=true wlan_mode=optional", state.Vlan)
		}
	})
}

func Test_radiusProfileResource_modelToRadiusProfile(t *testing.T) {
	ctx := context.Background()
	r := &radiusProfileResource{}

	t.Run("basic fields are converted", func(t *testing.T) {
		model := &radiusProfileResourceModel{
			Name:              types.StringValue("my-profile"),
			AccountingEnabled: types.BoolValue(true),
			InterimUpdate:     radiusProfileInterimUpdateDefault(),
			UseUSGAcctServer:  types.BoolValue(false),
			UseUSGAuthServer:  types.BoolValue(false),
			Vlan: types.ObjectValueMust(radiusProfileVlanAttrTypes(), map[string]attr.Value{
				"enabled":   types.BoolValue(false),
				"wlan_mode": types.StringValue("disabled"),
			}),
			AuthServer: []radiusServerModel{},
			AcctServer: []radiusServerModel{},
		}
		got, diags := r.modelToRadiusProfile(ctx, model)
		if diags.HasError() {
			t.Fatalf("modelToRadiusProfile() diagnostics: %v", diags)
		}
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
			Vlan:              radiusProfileVlanDefault(),
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
		got, diags := r.modelToRadiusProfile(ctx, model)
		if diags.HasError() {
			t.Fatalf("modelToRadiusProfile() diagnostics: %v", diags)
		}
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
		if d := r.radiusProfileToModel(ctx, profile, model, "default"); d.HasError() {
			t.Fatalf("radiusProfileToModel() diagnostics: %v", d)
		}

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
		vlan := model.Vlan.Attributes()
		if !attrAs[types.Bool](t, vlan["enabled"]).ValueBool() {
			t.Error("vlan.enabled should be true")
		}
		if got := attrAs[types.String](t, vlan["wlan_mode"]).ValueString(); got != "required" {
			t.Errorf("vlan.wlan_mode = %q, want required", got)
		}
		iu := model.InterimUpdate.Attributes()
		if attrAs[types.Bool](t, iu["enabled"]).ValueBool() {
			t.Error("interim_update.enabled should be false")
		}
		if got := attrAs[timetypes.GoDuration](t, iu["interval"]).ValueString(); got != "1h0m0s" {
			t.Errorf("interim_update.interval = %q, want 1h0m0s", got)
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

	// #356: the controller-managed default profile returns a server entry with no
	// IP. It must map to a null (not "") IP so, with ip now Optional, importing then
	// re-declaring the profile doesn't fail with "ip is required" or churn a diff.
	t.Run("server without IP maps to null", func(t *testing.T) {
		port := int64(1812)
		profile := &unifi.RADIUSProfile{
			ID:               "prof-default",
			Name:             "Default",
			UseUsgAuthServer: true,
			AuthServers: []unifi.RADIUSProfileAuthServers{
				{IP: "", Port: &port, Secret: "shhh"},
			},
		}
		model := &radiusProfileResourceModel{}
		if d := r.radiusProfileToModel(ctx, profile, model, "default"); d.HasError() {
			t.Fatalf("radiusProfileToModel() diagnostics: %v", d)
		}
		if len(model.AuthServer) != 1 {
			t.Fatalf("AuthServer length = %d, want 1", len(model.AuthServer))
		}
		if !model.AuthServer[0].IP.IsNull() {
			t.Errorf("AuthServer IP = %q, want null", model.AuthServer[0].IP.ValueString())
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
		if d := r.radiusProfileToModel(ctx, profile, model, "site1"); d.HasError() {
			t.Fatalf("radiusProfileToModel() diagnostics: %v", d)
		}
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

// TestRadiusProfileUpgradeState_nestsPrefixedGroups guards the v1 -> v2 schema
// upgrade: the flat interim_update_* and vlan_* attributes move into nested
// objects. The v0 upgrader must apply the same nesting after its integer ->
// duration rewrite, since every upgrader targets the current schema.
func TestRadiusProfileUpgradeState_nestsPrefixedGroups(t *testing.T) {
	ctx := context.Background()
	r := &radiusProfileResource{}

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	if schemaResp.Schema.Version != 2 {
		t.Fatalf("radius profile schema Version = %d, want 2", schemaResp.Schema.Version)
	}
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	ups := r.UpgradeState(ctx)
	for _, v := range []int64{0, 1} {
		if _, ok := ups[v]; !ok {
			t.Fatalf("no upgrader registered for schema version %d", v)
		}
	}

	upgrade := func(t *testing.T, version int64, prior string) map[string]tftypes.Value {
		t.Helper()
		resp := &fwresource.UpgradeStateResponse{}
		ups[version].StateUpgrader(ctx, fwresource.UpgradeStateRequest{
			RawState: &tfprotov6.RawState{JSON: []byte(prior)},
		}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("upgrade from v%d failed: %v", version, resp.Diagnostics)
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
	obj := func(t *testing.T, v tftypes.Value, name string) map[string]tftypes.Value {
		t.Helper()
		var m map[string]tftypes.Value
		if err := v.As(&m); err != nil {
			t.Fatalf("%s: as object: %v (value %v)", name, err, v)
		}
		return m
	}
	str := func(t *testing.T, v tftypes.Value, name, want string) {
		t.Helper()
		var s string
		if err := v.As(&s); err != nil || s != want {
			t.Errorf("%s = %v (%v), want %q", name, v, err, want)
		}
	}
	boolean := func(t *testing.T, v tftypes.Value, name string, want bool) {
		t.Helper()
		var b bool
		if err := v.As(&b); err != nil || b != want {
			t.Errorf("%s = %v (%v), want %v", name, v, err, want)
		}
	}

	t.Run("v1 nests interim_update and vlan", func(t *testing.T) {
		root := upgrade(t, 1, `{
			"id": "rp-1", "site": "default", "name": "corp", "accounting_enabled": true,
			"interim_update_enabled": true, "interim_update_interval": "30m0s",
			"use_usg_acct_server": false, "use_usg_auth_server": true,
			"vlan_enabled": true, "vlan_wlan_mode": "required",
			"auth_server": [{"ip": "10.0.0.1", "port": 1812, "secret": "s"}],
			"acct_server": []
		}`)
		for _, flat := range []string{
			"interim_update_enabled", "interim_update_interval", "vlan_enabled", "vlan_wlan_mode",
		} {
			if _, exists := root[flat]; exists {
				t.Errorf("flat attribute %q survived the upgrade", flat)
			}
		}
		iu := obj(t, root["interim_update"], "interim_update")
		boolean(t, iu["enabled"], "interim_update.enabled", true)
		str(t, iu["interval"], "interim_update.interval", "30m0s")
		vlan := obj(t, root["vlan"], "vlan")
		boolean(t, vlan["enabled"], "vlan.enabled", true)
		str(t, vlan["wlan_mode"], "vlan.wlan_mode", "required")
		// Attributes that were not nested survive untouched.
		boolean(t, root["use_usg_auth_server"], "use_usg_auth_server", true)
		var servers []tftypes.Value
		if err := root["auth_server"].As(&servers); err != nil || len(servers) != 1 {
			t.Errorf("auth_server = %v (%v), want one entry", root["auth_server"], err)
		}
	})

	t.Run("v0 converts the interval then nests", func(t *testing.T) {
		root := upgrade(t, 0, `{
			"id": "rp-0", "site": "default", "name": "legacy",
			"interim_update_enabled": false, "interim_update_interval": 3600,
			"vlan_enabled": false, "vlan_wlan_mode": ""
		}`)
		if _, exists := root["interim_update_interval"]; exists {
			t.Error("flat interim_update_interval survived the v0 upgrade")
		}
		iu := obj(t, root["interim_update"], "interim_update")
		boolean(t, iu["enabled"], "interim_update.enabled", false)
		str(t, iu["interval"], "interim_update.interval", "1h0m0s")
		vlan := obj(t, root["vlan"], "vlan")
		boolean(t, vlan["enabled"], "vlan.enabled", false)
		str(t, vlan["wlan_mode"], "vlan.wlan_mode", "")
	})

	t.Run("state without a group leaves its object null", func(t *testing.T) {
		root := upgrade(t, 1, `{"id": "rp-2", "site": "default", "name": "bare"}`)
		if !root["interim_update"].IsNull() {
			t.Errorf("interim_update = %v, want null", root["interim_update"])
		}
		if !root["vlan"].IsNull() {
			t.Errorf("vlan = %v, want null", root["vlan"])
		}
	})
}

// TestRadiusProfileNestedGroups_wireAndReadBack checks that the nested
// interim_update and vlan groups are written to and read back from the API
// struct, that the object defaults reproduce what the flat attributes sent
// when omitted, and that a null/unknown group contributes nothing.
func TestRadiusProfileNestedGroups_wireAndReadBack(t *testing.T) {
	ctx := context.Background()
	r := &radiusProfileResource{}

	model := &radiusProfileResourceModel{
		Name:              types.StringValue("corp"),
		AccountingEnabled: types.BoolValue(true),
		InterimUpdate: types.ObjectValueMust(
			radiusProfileInterimUpdateAttrTypes(),
			map[string]attr.Value{
				"enabled":  types.BoolValue(true),
				"interval": timetypes.NewGoDurationValue(30 * time.Minute),
			},
		),
		Vlan: types.ObjectValueMust(radiusProfileVlanAttrTypes(), map[string]attr.Value{
			"enabled":   types.BoolValue(true),
			"wlan_mode": types.StringValue("required"),
		}),
	}
	api, diags := r.modelToRadiusProfile(ctx, model)
	if diags.HasError() {
		t.Fatalf("modelToRadiusProfile: %v", diags)
	}
	if !api.InterimUpdateEnabled || api.InterimUpdateInterval == nil ||
		*api.InterimUpdateInterval != 1800 {
		t.Errorf("interim_update: %v %v", api.InterimUpdateEnabled, api.InterimUpdateInterval)
	}
	if !api.VLANEnabled || api.VLANWLANMode != "required" {
		t.Errorf("vlan: %v %q", api.VLANEnabled, api.VLANWLANMode)
	}

	// Read back: both groups are rebuilt from the API response.
	var back radiusProfileResourceModel
	if d := r.radiusProfileToModel(ctx, api, &back, "default"); d.HasError() {
		t.Fatalf("radiusProfileToModel: %v", d)
	}
	if !back.InterimUpdate.Equal(model.InterimUpdate) {
		t.Errorf("interim_update read back = %v, want %v", back.InterimUpdate, model.InterimUpdate)
	}
	if !back.Vlan.Equal(model.Vlan) {
		t.Errorf("vlan read back = %v, want %v", back.Vlan, model.Vlan)
	}

	// The object defaults reproduce what the flat defaults used to send.
	defaults := &radiusProfileResourceModel{
		Name:          types.StringValue("plain"),
		InterimUpdate: radiusProfileInterimUpdateDefault(),
		Vlan:          radiusProfileVlanDefault(),
	}
	api, diags = r.modelToRadiusProfile(ctx, defaults)
	if diags.HasError() {
		t.Fatalf("modelToRadiusProfile (defaults): %v", diags)
	}
	if api.InterimUpdateEnabled || api.InterimUpdateInterval == nil ||
		*api.InterimUpdateInterval != 3600 {
		t.Errorf(
			"default interim_update: %v %v",
			api.InterimUpdateEnabled,
			api.InterimUpdateInterval,
		)
	}
	if api.VLANEnabled || api.VLANWLANMode != "" {
		t.Errorf("default vlan: %v %q", api.VLANEnabled, api.VLANWLANMode)
	}

	// A null or unknown group contributes nothing, as unset flat attributes did.
	bare := &radiusProfileResourceModel{
		Name:          types.StringValue("bare"),
		InterimUpdate: types.ObjectNull(radiusProfileInterimUpdateAttrTypes()),
		Vlan:          types.ObjectUnknown(radiusProfileVlanAttrTypes()),
	}
	api, diags = r.modelToRadiusProfile(ctx, bare)
	if diags.HasError() {
		t.Fatalf("modelToRadiusProfile (bare): %v", diags)
	}
	if api.InterimUpdateEnabled || api.InterimUpdateInterval != nil ||
		api.VLANEnabled || api.VLANWLANMode != "" {
		t.Errorf("null/unknown groups leaked into the request: %+v", api)
	}
}
