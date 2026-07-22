package unifi

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/ubiquiti-community/go-unifi/unifi/settings"
)

func TestAccSettingResource_mgmt(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSettingConfig_mgmt(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"mgmt.auto_upgrade",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"mgmt.ssh_enabled",
						"false",
					),
				),
			},
			{
				ResourceName:      "unifi_setting.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"mgmt.%",
					"mgmt.auto_upgrade",
					"mgmt.ssh_enabled",
				},
			},
		},
	})
}

func TestAccSettingResource_radius(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSettingConfig_radius(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"radius.accounting_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"radius.auth_port",
						"1812",
					),
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"radius.acct_port",
						"1813",
					),
				),
			},
			{
				ResourceName:      "unifi_setting.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"radius.secret", // Secret is sensitive and won't be in state after import
					"radius.%",
					"radius.accounting_enabled",
					"radius.acct_port",
					"radius.auth_port",
					"radius.interim_update_interval",
				},
			},
		},
	})
}

func TestAccSettingResource_usg(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSettingConfig_usg(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"usg.ftp_module",
						"true",
					),
				),
			},
			{
				ResourceName:      "unifi_setting.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"usg.%",
					"usg.ftp_module",
				},
			},
		},
	})
}

func TestAccSettingResource_combined(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSettingConfig_combined(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"mgmt.auto_upgrade",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"mgmt.ssh_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"radius.accounting_enabled",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"usg.ftp_module",
						"false",
					),
				),
			},
			{
				Config: testAccSettingConfig_combinedUpdate(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"mgmt.auto_upgrade",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"mgmt.ssh_enabled",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"radius.accounting_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"usg.ftp_module",
						"true",
					),
				),
			},
		},
	})
}

func TestAccSettingResource_sshKeys(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSettingConfig_sshKeys(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"mgmt.ssh_enabled",
						"true",
					),
					resource.TestCheckResourceAttr("unifi_setting.test", "mgmt.ssh_keys.#", "1"),
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"mgmt.ssh_keys.0.name",
						"test-key",
					),
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"mgmt.ssh_keys.0.type",
						"ssh-rsa",
					),
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"mgmt.ssh_keys.0.comment",
						"Test SSH Key",
					),
				),
			},
			{
				Config: testAccSettingConfig_sshKeysUpdate(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_setting.test", "mgmt.ssh_keys.#", "2"),
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"mgmt.ssh_keys.0.name",
						"test-key",
					),
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"mgmt.ssh_keys.1.name",
						"test-key-2",
					),
				),
			},
		},
	})
}

func testAccSettingConfig_mgmt() string {
	return `
resource "unifi_setting" "test" {
  mgmt = {
    auto_upgrade = true
    ssh_enabled  = false
  }
}
`
}

func testAccSettingConfig_radius() string {
	return `
resource "unifi_setting" "test" {
  radius = {
    accounting_enabled = true
    auth_port          = 1812
    acct_port          = 1813
    secret             = "test-secret-123"
  }
}
`
}

func testAccSettingConfig_usg() string {
	return `
resource "unifi_setting" "test" {
  usg = {
    ftp_module = true
  }
}
`
}

func testAccSettingConfig_combined() string {
	return `
resource "unifi_setting" "test" {
  mgmt = {
    auto_upgrade = true
    ssh_enabled  = true
  }

  radius = {
    accounting_enabled = false
  }

  usg = {
    ftp_module = false
  }
}
`
}

func testAccSettingConfig_combinedUpdate() string {
	return `
resource "unifi_setting" "test" {
  mgmt = {
    auto_upgrade = false
    ssh_enabled  = false
  }

  radius = {
    accounting_enabled = true
  }

  usg = {
    ftp_module = true
  }
}
`
}

func testAccSettingConfig_sshKeys() string {
	return `
resource "unifi_setting" "test" {
  mgmt = {
    ssh_enabled = true
    ssh_keys = [{
      name    = "test-key"
      type    = "ssh-rsa"
      key     = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDTest123"
      comment = "Test SSH Key"
    }]
  }
}
`
}

func TestAccSettingResource_doh(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSettingConfig_dohAuto(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"doh.state",
						"auto",
					),
				),
			},
			{
				ResourceName:      "unifi_setting.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"doh.%",
					"doh.state",
				},
			},
			{
				Config: testAccSettingConfig_dohOff(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"doh.state",
						"off",
					),
				),
			},
		},
	})
}

func TestAccSettingResource_dohCustomServers(t *testing.T) {
	// custom_servers requires controller support beyond simulation/demo mode;
	// the simulation controller returns DohCustomServersUnsupported (400).
	// Run only against a real controller (UNIFI_SKIP_CONTAINER bypasses the
	// docker simulation and targets the pre-set UNIFI_* endpoint).
	if os.Getenv("UNIFI_SKIP_CONTAINER") == "" {
		t.Skip("custom DoH servers require a real controller; set UNIFI_SKIP_CONTAINER to run")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSettingConfig_dohCustomServers(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"doh.state",
						"custom",
					),
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"doh.custom_servers.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"doh.custom_servers.0.server_name",
						"my-resolver",
					),
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"doh.custom_servers.0.enabled",
						"true",
					),
				),
			},
			{
				ResourceName:      "unifi_setting.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"doh.%",
					"doh.state",
					"doh.custom_servers.#",
					"doh.custom_servers.0.server_name",
					"doh.custom_servers.0.sdns_stamp",
					"doh.custom_servers.0.enabled",
				},
			},
		},
	})
}

func TestAccSettingResource_ips(t *testing.T) {
	// ips_mode ids/ips/ipsInline requires a real UniFi gateway (UDM/USG) to take effect;
	// the simulation controller accepts the PUT but reverts ips_mode to "disabled" on read-back.
	// This test covers fields that work without gateway hardware.
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSettingConfig_ips(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"ips.ips_mode",
						"disabled",
					),
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"ips.restrict_torrents",
						"true",
					),
				),
			},
			{
				ResourceName:      "unifi_setting.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"ips.%",
					"ips.ips_mode",
					"ips.restrict_torrents",
				},
			},
			{
				Config: testAccSettingConfig_ipsDisabled(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"ips.ips_mode",
						"disabled",
					),
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"ips.restrict_torrents",
						"false",
					),
				),
			},
		},
	})
}

func TestAccSettingResource_ipsHoneypot(t *testing.T) {
	// Honeypot requires a UDM-class gateway; the simulation controller presents as a USG,
	// which returns HoneypotIsNotSupportedInUsg (400).
	// honeypot is not supported on USG-class/simulation controllers; it
	// requires a UDM-class device. Run only against a real controller.
	if os.Getenv("UNIFI_SKIP_CONTAINER") == "" {
		t.Skip("honeypot requires a real UDM-class controller; set UNIFI_SKIP_CONTAINER to run")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSettingConfig_ipsHoneypot(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"ips.honeypot_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"ips.honeypot.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"ips.honeypot.0.ip_address",
						"10.1.10.254",
					),
					resource.TestCheckResourceAttr(
						"unifi_setting.test",
						"ips.honeypot.0.version",
						"v4",
					),
				),
			},
		},
	})
}

func testAccSettingConfig_dohAuto() string {
	return `
resource "unifi_setting" "test" {
  doh = {
    state = "auto"
  }
}
`
}

func testAccSettingConfig_dohOff() string {
	return `
resource "unifi_setting" "test" {
  doh = {
    state = "off"
  }
}
`
}

func testAccSettingConfig_dohCustomServers() string {
	return `
resource "unifi_setting" "test" {
  doh = {
    state = "custom"
    custom_servers = [{
      server_name = "my-resolver"
      sdns_stamp  = "sdns://AgcAAAAAAAAACTEyNy4wLjAuMQA"
      enabled     = true
    }]
  }
}
`
}

func testAccSettingConfig_ips() string {
	return `
resource "unifi_setting" "test" {
  ips = {
    ips_mode          = "disabled"
    restrict_torrents = true
  }
}
`
}

func testAccSettingConfig_ipsDisabled() string {
	return `
resource "unifi_setting" "test" {
  ips = {
    ips_mode          = "disabled"
    restrict_torrents = false
  }
}
`
}

func testAccSettingConfig_ipsHoneypot() string {
	return `
resource "unifi_network" "test" {
  name   = "test-honeypot-network"
  subnet = "10.1.10.1/24"
  vlan   = 10
}

resource "unifi_setting" "test" {
  ips = {
    honeypot_enabled = true
    honeypot = [{
      ip_address = "10.1.10.254"
      network_id = unifi_network.test.id
      version    = "v4"
    }]
  }
}
`
}

func testAccSettingConfig_sshKeysUpdate() string {
	return `
resource "unifi_setting" "test" {
  mgmt = {
    ssh_enabled = true
    ssh_keys = [
      {
        name    = "test-key"
        type    = "ssh-rsa"
        key     = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDTest123"
        comment = "Test SSH Key"
      },
      {
        name    = "test-key-2"
        type    = "ssh-ed25519"
        key     = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest456"
        comment = "Second Test Key"
      }
    ]
  }
}
`
}

func TestNewSettingResource(t *testing.T) {
	r := NewSettingResource()
	if r == nil {
		t.Fatal("NewSettingResource() returned nil")
	}
	if _, ok := r.(fwresource.ResourceWithConfigure); !ok {
		t.Error("expected ResourceWithConfigure interface")
	}
	if _, ok := r.(fwresource.ResourceWithImportState); !ok {
		t.Error("expected ResourceWithImportState interface")
	}
}

func Test_settingResource_Metadata(t *testing.T) {
	tests := []struct {
		providerTypeName, wantTypeName string
	}{
		{"unifi", "unifi_setting"},
		{"test", "test_setting"},
	}
	for _, tt := range tests {
		t.Run(tt.providerTypeName, func(t *testing.T) {
			r := &settingResource{}
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

func Test_settingResource_Schema(t *testing.T) {
	r := &settingResource{}
	resp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() produced errors: %v", resp.Diagnostics)
	}
	for _, attr := range []string{"id", "site", "mgmt", "radius", "usg", "igmp_snooping", "doh", "ips"} {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("missing attribute %q", attr)
		}
	}
}

// TestSettingNtpServersUseStateForUnknown guards #382: omitted
// Optional+Computed server fields must retain their prior values instead of
// repeatedly planning as "known after apply" when the NTP block is configured.
func TestSettingNtpServersUseStateForUnknown(t *testing.T) {
	resp := &fwresource.SchemaResponse{}
	(&settingResource{}).Schema(context.Background(), fwresource.SchemaRequest{}, resp)

	ntp, ok := resp.Schema.Attributes["ntp"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("ntp is not a SingleNestedAttribute")
	}
	for _, key := range []string{"ntp_server_1", "ntp_server_2", "ntp_server_3", "ntp_server_4"} {
		server, ok := ntp.Attributes[key].(schema.StringAttribute)
		if !ok {
			t.Errorf("ntp.%s is not a StringAttribute", key)
			continue
		}
		if !server.Optional || !server.Computed {
			t.Errorf("ntp.%s must remain Optional+Computed", key)
		}
		if len(server.PlanModifiers) == 0 {
			t.Errorf("ntp.%s must use UseStateForUnknown (#382)", key)
			continue
		}

		req := planmodifier.StringRequest{
			ConfigValue: types.StringNull(),
			PlanValue:   types.StringUnknown(),
			State: tfsdk.State{
				Raw: tftypes.NewValue(tftypes.String, ""),
			},
			StateValue: types.StringValue(""),
		}
		modified := &planmodifier.StringResponse{PlanValue: req.PlanValue}
		server.PlanModifiers[0].PlanModifyString(context.Background(), req, modified)
		if modified.Diagnostics.HasError() {
			t.Errorf("ntp.%s plan modifier returned errors: %v", key, modified.Diagnostics)
		}
		if modified.PlanValue.IsNull() || modified.PlanValue.IsUnknown() ||
			modified.PlanValue.ValueString() != "" {
			t.Errorf("ntp.%s plan = %v, want prior known empty state", key, modified.PlanValue)
		}
	}
}

func Test_settingResource_UpgradeState(t *testing.T) {
	r := &settingResource{}
	ctx := context.Background()
	got := r.UpgradeState(ctx)
	if got == nil {
		t.Fatal("UpgradeState() returned nil")
	}
	if _, ok := got[0]; !ok {
		t.Error("UpgradeState() map should contain version key 0")
	}
}

func Test_settingResource_Configure(t *testing.T) {
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
			r := &settingResource{}
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

func Test_settingResource_ImportState(t *testing.T) {
	t.Skip(
		"ImportState delegates to ImportStatePassthroughID which requires full state schema setup",
	)
}

func Test_settingResource_mgmtModelToSetting(t *testing.T) {
	r := &settingResource{}
	ctx := context.Background()

	t.Run("nil model returns empty setting", func(t *testing.T) {
		// mgmtModelToSetting does not accept nil (it dereferences the pointer);
		// test zero-value model produces a zero-value settings.Mgmt.
		model := &settingMgmtModel{
			AutoUpgrade: types.BoolNull(),
			SSHEnabled:  types.BoolNull(),
			SSHKeys:     types.ListNull(types.StringType),
		}
		got := r.mgmtModelToSetting(ctx, model, &settings.Mgmt{})
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		if got.AutoUpgrade {
			t.Error("AutoUpgrade should be false for null input")
		}
		if got.SSHEnabled {
			t.Error("SSHEnabled should be false for null input")
		}
	})

	t.Run("basic fields set", func(t *testing.T) {
		model := &settingMgmtModel{
			AutoUpgrade: types.BoolValue(true),
			SSHEnabled:  types.BoolValue(false),
			SSHKeys:     types.ListNull(types.StringType),
		}
		got := r.mgmtModelToSetting(ctx, model, &settings.Mgmt{})
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		if !got.AutoUpgrade {
			t.Error("AutoUpgrade should be true")
		}
		if got.SSHEnabled {
			t.Error("SSHEnabled should be false")
		}
	})
}

func Test_settingResource_mgmtSettingToModel(t *testing.T) {
	r := &settingResource{}
	ctx := context.Background()

	t.Run("null plan fields produce null model fields", func(t *testing.T) {
		setting := &settings.Mgmt{
			AutoUpgrade: true,
			SSHEnabled:  true,
		}
		plan := &settingMgmtModel{
			AutoUpgrade: types.BoolNull(),
			SSHEnabled:  types.BoolNull(),
			SSHKeys:     types.ListNull(types.StringType),
		}
		got := r.mgmtSettingToModel(ctx, setting, plan)
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		if !got.AutoUpgrade.IsNull() {
			t.Error("AutoUpgrade should be null when plan is null")
		}
		if !got.SSHEnabled.IsNull() {
			t.Error("SSHEnabled should be null when plan is null")
		}
	})

	t.Run("non-null plan fields reflect remote value", func(t *testing.T) {
		setting := &settings.Mgmt{
			AutoUpgrade: true,
			SSHEnabled:  false,
		}
		plan := &settingMgmtModel{
			AutoUpgrade: types.BoolValue(false), // plan had a value configured
			SSHEnabled:  types.BoolValue(true),
			SSHKeys:     types.ListNull(types.StringType),
		}
		got := r.mgmtSettingToModel(ctx, setting, plan)
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		if !got.AutoUpgrade.ValueBool() {
			t.Error("AutoUpgrade should reflect remote value (true)")
		}
		if got.SSHEnabled.ValueBool() {
			t.Error("SSHEnabled should reflect remote value (false)")
		}
	})
}

func Test_settingResource_radiusModelToSetting(t *testing.T) {
	r := &settingResource{}

	t.Run("null fields leave base unchanged", func(t *testing.T) {
		authPort := int64(1812)
		base := &settings.Radius{
			AccountingEnabled: true,
			AuthPort:          &authPort,
		}
		model := &settingRadiusModel{
			AccountingEnabled:     types.BoolNull(),
			AcctPort:              types.Int64Null(),
			AuthPort:              types.Int64Null(),
			InterimUpdateInterval: timetypes.NewGoDurationNull(),
			Secret:                types.StringNull(),
		}
		got := r.radiusModelToSetting(context.Background(), model, base)
		// radiusModelToSetting starts from base and only overlays non-null fields.
		// Null AccountingEnabled means the base value (true) is left in place.
		if !got.AccountingEnabled {
			t.Error("AccountingEnabled should remain true (from base)")
		}
	})

	t.Run("non-null fields overlay base", func(t *testing.T) {
		base := &settings.Radius{}
		model := &settingRadiusModel{
			AccountingEnabled:     types.BoolValue(true),
			AcctPort:              types.Int64Value(1813),
			AuthPort:              types.Int64Value(1812),
			InterimUpdateInterval: timetypes.NewGoDurationNull(),
			Secret:                types.StringValue("mysecret"),
		}
		got := r.radiusModelToSetting(context.Background(), model, base)
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		if !got.AccountingEnabled {
			t.Error("AccountingEnabled should be true")
		}
		if got.AuthPort == nil || *got.AuthPort != 1812 {
			t.Errorf("AuthPort = %v, want 1812", got.AuthPort)
		}
		if got.Secret != "mysecret" {
			t.Errorf("Secret = %q, want mysecret", got.Secret)
		}
	})
}

func Test_settingResource_radiusSettingToModel(t *testing.T) {
	r := &settingResource{}

	t.Run("nil secret plan produces null secret model", func(t *testing.T) {
		authPort := int64(1812)
		acctPort := int64(1813)
		setting := &settings.Radius{
			AccountingEnabled: true,
			AuthPort:          &authPort,
			AcctPort:          &acctPort,
			Secret:            "remote-secret",
		}
		plan := &settingRadiusModel{
			Secret: types.StringNull(),
		}
		got := r.radiusSettingToModel(context.Background(), setting, plan)
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		if !got.AccountingEnabled.ValueBool() {
			t.Error("AccountingEnabled should be true")
		}
		// When plan.Secret is null, model.Secret should be null regardless of remote value.
		if !got.Secret.IsNull() {
			t.Errorf(
				"Secret should be null when plan secret is null, got %q",
				got.Secret.ValueString(),
			)
		}
	})

	t.Run("non-null secret plan reflects remote value", func(t *testing.T) {
		setting := &settings.Radius{Secret: "the-secret"}
		plan := &settingRadiusModel{Secret: types.StringValue("old")}
		got := r.radiusSettingToModel(context.Background(), setting, plan)
		if got.Secret.ValueString() != "the-secret" {
			t.Errorf("Secret = %q, want the-secret", got.Secret.ValueString())
		}
	})
}

func Test_settingResource_usgModelToSetting(t *testing.T) {
	r := &settingResource{}
	ctx := context.Background()

	t.Run("null fields produce zero-value setting", func(t *testing.T) {
		model := &settingUSGModel{
			FtpModule:       types.BoolNull(),
			BroadcastPing:   types.BoolNull(),
			DNSVerification: types.ObjectNull(nil),
		}
		got := r.usgModelToSetting(ctx, model)
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		if got.FtpModule {
			t.Error("FtpModule should be false for null input")
		}
	})

	t.Run("ftp_module set to true", func(t *testing.T) {
		model := &settingUSGModel{
			FtpModule:       types.BoolValue(true),
			BroadcastPing:   types.BoolNull(),
			DNSVerification: types.ObjectNull(nil),
		}
		got := r.usgModelToSetting(ctx, model)
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		if !got.FtpModule {
			t.Error("FtpModule should be true")
		}
	})
}

func Test_settingResource_usgSettingToModel(t *testing.T) {
	r := &settingResource{}
	ctx := context.Background()

	t.Run("null plan fields produce null model fields", func(t *testing.T) {
		setting := &settings.Usg{FtpModule: true, SipModule: true}
		plan := &settingUSGModel{
			FtpModule: types.BoolNull(),
			SipModule: types.BoolNull(),
		}
		got := r.usgSettingToModel(ctx, setting, plan)
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		if !got.FtpModule.IsNull() {
			t.Error("FtpModule should be null when plan is null")
		}
	})

	t.Run("non-null plan fields reflect remote value", func(t *testing.T) {
		setting := &settings.Usg{FtpModule: true, GreModule: false}
		plan := &settingUSGModel{
			FtpModule: types.BoolValue(false),
			GreModule: types.BoolValue(true),
		}
		got := r.usgSettingToModel(ctx, setting, plan)
		if !got.FtpModule.ValueBool() {
			t.Error("FtpModule should be true (remote value)")
		}
		if got.GreModule.ValueBool() {
			t.Error("GreModule should be false (remote value)")
		}
	})
}

func Test_settingResource_igmpSnoopingModelToSetting(t *testing.T) {
	r := &settingResource{}
	ctx := context.Background()

	t.Run("enabled overlaid onto base", func(t *testing.T) {
		base := &settings.IgmpSnooping{Enabled: false, QuerierMode: "AUTO"}
		model := &settingIgmpSnoopingModel{
			Enabled:    types.BoolValue(true),
			NetworkIDs: types.ListNull(types.StringType),
		}
		var diags diag.Diagnostics
		got := r.igmpSnoopingModelToSetting(ctx, model, base, &diags)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if !got.Enabled {
			t.Error("Enabled should be true")
		}
		// Advanced fields on base must be preserved.
		if got.QuerierMode != "AUTO" {
			t.Errorf("QuerierMode = %q, want AUTO", got.QuerierMode)
		}
	})

	t.Run("network_ids overlaid onto base", func(t *testing.T) {
		base := &settings.IgmpSnooping{NetworkIDs: []string{"old-net"}}
		nids, d := types.ListValueFrom(ctx, types.StringType, []string{"net-1", "net-2"})
		if d.HasError() {
			t.Fatalf("building list: %v", d)
		}
		model := &settingIgmpSnoopingModel{
			Enabled:    types.BoolNull(),
			NetworkIDs: nids,
		}
		var diags diag.Diagnostics
		got := r.igmpSnoopingModelToSetting(ctx, model, base, &diags)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if len(got.NetworkIDs) != 2 || got.NetworkIDs[0] != "net-1" {
			t.Errorf("NetworkIDs = %v, want [net-1 net-2]", got.NetworkIDs)
		}
	})
}

func Test_settingResource_igmpSnoopingSettingToModel(t *testing.T) {
	r := &settingResource{}
	ctx := context.Background()

	t.Run("basic fields mapped", func(t *testing.T) {
		setting := &settings.IgmpSnooping{
			Enabled:    true,
			NetworkIDs: []string{"net-a", "net-b"},
		}
		var diags diag.Diagnostics
		got := r.igmpSnoopingSettingToModel(ctx, setting, &diags)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		if !got.Enabled.ValueBool() {
			t.Error("Enabled should be true")
		}
		var ids []string
		if d := got.NetworkIDs.ElementsAs(ctx, &ids, false); d.HasError() {
			t.Fatalf("reading network_ids: %v", d)
		}
		if len(ids) != 2 {
			t.Errorf("NetworkIDs len = %d, want 2", len(ids))
		}
	})

	t.Run("empty network ids", func(t *testing.T) {
		setting := &settings.IgmpSnooping{Enabled: false, NetworkIDs: nil}
		var diags diag.Diagnostics
		got := r.igmpSnoopingSettingToModel(ctx, setting, &diags)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if got.Enabled.ValueBool() {
			t.Error("Enabled should be false")
		}
	})
}

func Test_settingResource_dohModelToSetting(t *testing.T) {
	r := &settingResource{}
	ctx := context.Background()

	t.Run("null fields produce empty setting", func(t *testing.T) {
		model := &settingDohModel{
			State:         types.StringNull(),
			ServerNames:   types.ListNull(types.StringType),
			CustomServers: types.ListNull(types.StringType),
		}
		var diags diag.Diagnostics
		got := r.dohModelToSetting(ctx, model, &diags)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		if got.State != "" {
			t.Errorf("State should be empty, got %q", got.State)
		}
	})

	t.Run("state set", func(t *testing.T) {
		model := &settingDohModel{
			State:         types.StringValue("auto"),
			ServerNames:   types.ListNull(types.StringType),
			CustomServers: types.ListNull(types.StringType),
		}
		var diags diag.Diagnostics
		got := r.dohModelToSetting(ctx, model, &diags)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if got.State != "auto" {
			t.Errorf("State = %q, want auto", got.State)
		}
	})
}

func Test_settingResource_dohSettingToModel(t *testing.T) {
	r := &settingResource{}
	ctx := context.Background()

	t.Run("null plan state produces null model state", func(t *testing.T) {
		setting := &settings.Doh{State: "auto"}
		plan := &settingDohModel{
			State:         types.StringNull(),
			ServerNames:   types.ListNull(types.StringType),
			CustomServers: types.ListNull(types.StringType),
		}
		var diags diag.Diagnostics
		got := r.dohSettingToModel(ctx, setting, plan, &diags)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		if !got.State.IsNull() {
			t.Errorf("State should be null when plan is null, got %q", got.State.ValueString())
		}
	})

	t.Run("non-null plan state reflects remote value", func(t *testing.T) {
		setting := &settings.Doh{State: "off"}
		plan := &settingDohModel{
			State:         types.StringValue("auto"),
			ServerNames:   types.ListNull(types.StringType),
			CustomServers: types.ListNull(types.StringType),
		}
		var diags diag.Diagnostics
		got := r.dohSettingToModel(ctx, setting, plan, &diags)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if got.State.ValueString() != "off" {
			t.Errorf("State = %q, want off", got.State.ValueString())
		}
	})
}

func Test_settingResource_ipsModelToSetting(t *testing.T) {
	r := &settingResource{}
	ctx := context.Background()

	t.Run("null fields produce empty setting", func(t *testing.T) {
		model := &settingIpsModel{
			IPSMode:          types.StringNull(),
			HoneypotEnabled:  types.BoolNull(),
			RestrictTorrents: types.BoolNull(),
		}
		var diags diag.Diagnostics
		got := r.ipsModelToSetting(ctx, model, &diags)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		if got.IPsMode != "" {
			t.Errorf("IPsMode should be empty, got %q", got.IPsMode)
		}
	})

	t.Run("ips_mode and restrict_torrents set", func(t *testing.T) {
		model := &settingIpsModel{
			IPSMode:          types.StringValue("disabled"),
			RestrictTorrents: types.BoolValue(true),
			HoneypotEnabled:  types.BoolNull(),
		}
		var diags diag.Diagnostics
		got := r.ipsModelToSetting(ctx, model, &diags)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if got.IPsMode != "disabled" {
			t.Errorf("IPsMode = %q, want disabled", got.IPsMode)
		}
		if !got.RestrictTorrents {
			t.Error("RestrictTorrents should be true")
		}
	})
}

func Test_settingResource_ipsSettingToModel(t *testing.T) {
	r := &settingResource{}
	ctx := context.Background()

	t.Run("null plan ips_mode produces null model ips_mode", func(t *testing.T) {
		setting := &settings.Ips{IPsMode: "ips"}
		plan := &settingIpsModel{
			IPSMode: types.StringNull(),
		}
		var diags diag.Diagnostics
		got := r.ipsSettingToModel(ctx, setting, plan, &diags)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		if !got.IPSMode.IsNull() {
			t.Errorf("IPSMode should be null when plan is null, got %q", got.IPSMode.ValueString())
		}
	})

	t.Run("non-null plan reflects remote value", func(t *testing.T) {
		setting := &settings.Ips{IPsMode: "disabled", RestrictTorrents: true}
		plan := &settingIpsModel{
			IPSMode:          types.StringValue("ips"),
			RestrictTorrents: types.BoolValue(false),
		}
		var diags diag.Diagnostics
		got := r.ipsSettingToModel(ctx, setting, plan, &diags)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if got.IPSMode.ValueString() != "disabled" {
			t.Errorf("IPSMode = %q, want disabled", got.IPSMode.ValueString())
		}
		if !got.RestrictTorrents.ValueBool() {
			t.Error("RestrictTorrents should be true (remote value)")
		}
	})
}

// TestIgmpSnoopingModelMerge guards #164: the site-level igmp_snooping setting
// exposes only enabled + network_ids, and the model->setting conversion must
// overlay those onto the current remote setting so advanced querier/flood
// fields configured in the UI are preserved across an update.
func TestIgmpSnoopingModelMerge(t *testing.T) {
	ctx := context.Background()
	r := &settingResource{}
	var diags diag.Diagnostics

	// Current remote setting with advanced fields that must survive.
	base := &settings.IgmpSnooping{
		Enabled:             false,
		QuerierMode:         "CUSTOM",
		QuerierSwitches:     []string{"aa:bb:cc:dd:ee:ff"},
		FloodKnownProtocols: true,
	}
	nids, d := types.ListValueFrom(ctx, types.StringType, []string{"net-1", "net-2"})
	if d.HasError() {
		t.Fatalf("building network_ids: %v", d)
	}
	model := &settingIgmpSnoopingModel{
		Enabled:    types.BoolValue(true),
		NetworkIDs: nids,
	}

	out := r.igmpSnoopingModelToSetting(ctx, model, base, &diags)
	if diags.HasError() {
		t.Fatalf("igmpSnoopingModelToSetting: %v", diags)
	}
	if !out.Enabled {
		t.Error("Enabled not applied from model")
	}
	if len(out.NetworkIDs) != 2 || out.NetworkIDs[0] != "net-1" {
		t.Errorf("NetworkIDs = %v, want [net-1 net-2]", out.NetworkIDs)
	}
	// Advanced fields must be preserved from base (not dropped).
	if out.QuerierMode != "CUSTOM" || len(out.QuerierSwitches) != 1 || !out.FloodKnownProtocols {
		t.Errorf("advanced fields not preserved: querier_mode=%q querier_switches=%v flood=%v",
			out.QuerierMode, out.QuerierSwitches, out.FloodKnownProtocols)
	}

	// Read-back conversion.
	m := r.igmpSnoopingSettingToModel(ctx, out, &diags)
	if diags.HasError() {
		t.Fatalf("igmpSnoopingSettingToModel: %v", diags)
	}
	if !m.Enabled.ValueBool() {
		t.Error("model Enabled = false, want true")
	}
	var ids []string
	if d := m.NetworkIDs.ElementsAs(ctx, &ids, false); d.HasError() {
		t.Fatalf("reading model network_ids: %v", d)
	}
	if len(ids) != 2 {
		t.Errorf("model network_ids = %v, want 2", ids)
	}
}

// TestAutoSpeedtestSettingRoundTrip is a unit round-trip for the auto_speedtest
// setting block (#272): model -> go-unifi setting -> model preserves the fields.
func TestAutoSpeedtestSettingRoundTrip(t *testing.T) {
	r := &settingResource{}
	in := &settingAutoSpeedtestModel{
		Enabled:  types.BoolValue(true),
		CronExpr: types.StringValue("0 3 * * *"),
	}
	setting := r.autoSpeedtestModelToSetting(in)
	if !setting.Enabled || setting.CronExpr != "0 3 * * *" {
		t.Fatalf("modelToSetting = %+v, want enabled cron=0 3 * * *", setting)
	}
	out := r.autoSpeedtestSettingToModel(setting)
	if !out.Enabled.ValueBool() || out.CronExpr.ValueString() != "0 3 * * *" {
		t.Errorf("settingToModel = %+v, want enabled cron preserved", out)
	}
}

// TestNtpSettingStateNormalization guards #382: the controller represents
// unset NTP servers as "", which is a valid configured value and must not be
// rewritten to null during the post-apply read.
func TestNtpSettingStateNormalization(t *testing.T) {
	r := &settingResource{}
	apiSetting := r.ntpModelToSetting(&settingNtpModel{
		NtpServer1:        types.StringValue("pool.ntp.org"),
		NtpServer2:        types.StringValue(""),
		NtpServer3:        types.StringNull(),
		NtpServer4:        types.StringNull(),
		SettingPreference: types.StringValue("manual"),
	})

	state := r.ntpSettingToModel(apiSetting)
	if state.NtpServer1.ValueString() != "pool.ntp.org" {
		t.Errorf("ntp_server_1 = %q, want pool.ntp.org", state.NtpServer1.ValueString())
	}
	for key, value := range map[string]types.String{
		"ntp_server_2": state.NtpServer2,
		"ntp_server_3": state.NtpServer3,
		"ntp_server_4": state.NtpServer4,
	} {
		if value.IsNull() || value.IsUnknown() || value.ValueString() != "" {
			t.Errorf("%s = %v, want known empty string", key, value)
		}
	}
}

// TestSettingBlocksRoundTrip covers the model<->go-unifi conversions for the
// settings added in #273 (a representative scalar block and the list-bearing one).
func TestSettingBlocksRoundTrip(t *testing.T) {
	ctx := context.Background()
	r := &settingResource{}

	t.Run("ntp", func(t *testing.T) {
		in := &settingNtpModel{
			NtpServer1:        types.StringValue("pool.ntp.org"),
			SettingPreference: types.StringValue("manual"),
		}
		out := r.ntpSettingToModel(r.ntpModelToSetting(in))
		if out.NtpServer1.ValueString() != "pool.ntp.org" ||
			out.SettingPreference.ValueString() != "manual" {
			t.Errorf("ntp round-trip mismatch: %+v", out)
		}
	})

	t.Run("syslog", func(t *testing.T) {
		var diags diag.Diagnostics
		contents, _ := types.ListValueFrom(ctx, types.StringType, []string{"device", "client"})
		in := &settingSyslogModel{
			Enabled:  types.BoolValue(true),
			IP:       types.StringValue("10.0.0.9"),
			Port:     types.Int64Value(514),
			Contents: contents,
		}
		setting := r.syslogModelToSetting(ctx, in, &diags)
		if diags.HasError() {
			t.Fatalf("modelToSetting: %v", diags)
		}
		if !setting.Enabled || setting.IP != "10.0.0.9" || setting.Port == nil ||
			*setting.Port != 514 || len(setting.Contents) != 2 {
			t.Fatalf("syslog modelToSetting mismatch: %+v", setting)
		}
		out := r.syslogSettingToModel(ctx, setting, &diags)
		if diags.HasError() {
			t.Fatalf("settingToModel: %v", diags)
		}
		var gotContents []string
		out.Contents.ElementsAs(ctx, &gotContents, false)
		if out.IP.ValueString() != "10.0.0.9" || len(gotContents) != 2 {
			t.Errorf("syslog round-trip mismatch: %+v", out)
		}
	})
}

// TestMgmtNewFields guards #274: the new mgmt fields overlay onto the current
// remote setting (read-base, so unmanaged fields aren't clobbered) and the
// secret ssh_password is preserved from the plan (the controller never echoes it).
func TestMgmtNewFields(t *testing.T) {
	ctx := context.Background()
	r := &settingResource{}

	// Base has a field the user does NOT manage; it must survive.
	base := &settings.Mgmt{WifimanEnabled: true}
	model := &settingMgmtModel{
		SSHUsername:            types.StringValue("admin"),
		SSHPassword:            types.StringValue("s3cret"),
		SSHAuthPasswordEnabled: types.BoolValue(true),
		AdvancedFeatureEnabled: types.BoolValue(true),
	}
	setting := r.mgmtModelToSetting(ctx, model, base)
	if !setting.WifimanEnabled {
		t.Error("read-base field WifimanEnabled was clobbered")
	}
	if setting.SSHUsername != "admin" || setting.SSHPassword != "s3cret" ||
		!setting.SSHAuthPasswordEnabled || !setting.AdvancedFeatureEnabled {
		t.Errorf("overlay missing: %+v", setting)
	}

	// On read, ssh_password is preserved from the plan (API returns no plaintext).
	plan := &settingMgmtModel{
		SSHUsername: types.StringValue("admin"),
		SSHPassword: types.StringValue("s3cret"),
	}
	out := r.mgmtSettingToModel(ctx, &settings.Mgmt{SSHUsername: "admin"}, plan)
	if out.SSHPassword.ValueString() != "s3cret" {
		t.Errorf("ssh_password not preserved: %q", out.SSHPassword.ValueString())
	}
	if out.SSHUsername.ValueString() != "admin" {
		t.Errorf("ssh_username = %q, want admin", out.SSHUsername.ValueString())
	}
	// An unconfigured field stays null (no drift on unmanaged settings).
	if !out.WifimanEnabled.IsNull() {
		t.Error("unconfigured wifiman_enabled should be null")
	}
}

// TestIpsSuppressionAlertsRoundTrip guards #275: signature alert suppression
// (incl. gid/id pointers and the nested tracking list) round-trips model<->setting.
func TestIpsSuppressionAlertsRoundTrip(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	r := &settingResource{}

	tracking, _ := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: ipsTrackingAttrTypes},
		[]settingIpsTrackingModel{{
			Direction: types.StringValue("both"),
			Mode:      types.StringValue("ip"),
			Value:     types.StringValue("10.0.0.5"),
		}})
	alerts, _ := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: ipsAlertAttrTypes},
		[]settingIpsAlertModel{{
			Category:  types.StringValue("malware"),
			Gid:       types.Int64Value(1),
			ID:        types.Int64Value(2001),
			Signature: types.StringValue("ET MALWARE"),
			Type:      types.StringValue("track"),
			Tracking:  tracking,
		}})

	model := &settingIpsModel{
		EnabledCategories:    types.ListNull(types.StringType),
		EnabledNetworks:      types.ListNull(types.StringType),
		Honeypot:             types.ListNull(types.ObjectType{AttrTypes: ipsHoneypotAttrTypes}),
		SuppressionWhitelist: types.ListNull(types.ObjectType{AttrTypes: ipsWhitelistAttrTypes}),
		SuppressionAlerts:    alerts,
	}
	setting := r.ipsModelToSetting(ctx, model, &diags)
	if diags.HasError() {
		t.Fatalf("modelToSetting: %v", diags)
	}
	if setting.Suppression == nil || len(setting.Suppression.Alerts) != 1 {
		t.Fatalf("alerts not built: %+v", setting.Suppression)
	}
	a := setting.Suppression.Alerts[0]
	if a.Category != "malware" || a.Gid == nil || *a.Gid != 1 || a.ID == nil || *a.ID != 2001 ||
		a.Type != "track" || len(a.Tracking) != 1 || a.Tracking[0].Value != "10.0.0.5" {
		t.Fatalf("alert mismatch: %+v", a)
	}

	out := r.ipsSettingToModel(ctx, setting, model, &diags)
	if diags.HasError() {
		t.Fatalf("settingToModel: %v", diags)
	}
	var outAlerts []settingIpsAlertModel
	out.SuppressionAlerts.ElementsAs(ctx, &outAlerts, false)
	if len(outAlerts) != 1 || outAlerts[0].Signature.ValueString() != "ET MALWARE" ||
		outAlerts[0].Gid.ValueInt64() != 1 {
		t.Errorf("read-back alerts mismatch: %+v", outAlerts)
	}
}

// TestSyslogOmitsUnsetPorts guards #303: an unset port / netconsole_port must be
// omitted (nil pointer), not serialized as 0 — the controller rejects port 0.
func TestSyslogOmitsUnsetPorts(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	r := &settingResource{}

	m := &settingSyslogModel{
		Enabled:        types.BoolValue(true),
		IP:             types.StringValue("10.0.10.15"),
		Port:           types.Int64Value(1514),
		NetconsolePort: types.Int64Null(), // netconsole disabled / unset
		Contents:       types.ListNull(types.StringType),
	}
	setting := r.syslogModelToSetting(ctx, m, &diags)
	if diags.HasError() {
		t.Fatalf("modelToSetting: %v", diags)
	}
	if setting.NetconsolePort != nil {
		t.Errorf("netconsole_port must be omitted when unset, got %d", *setting.NetconsolePort)
	}
	if setting.Port == nil || *setting.Port != 1514 {
		t.Errorf("port = %v, want 1514", setting.Port)
	}

	// Unknown (Optional+Computed at create) must also omit, not send 0.
	m.Port = types.Int64Unknown()
	setting = r.syslogModelToSetting(ctx, m, &diags)
	if setting.Port != nil {
		t.Errorf("unknown port must be omitted, got %d", *setting.Port)
	}
}

// TestLcmOmitsUnsetInts guards the same #303 pattern for the lcm block.
func TestLcmOmitsUnsetInts(t *testing.T) {
	r := &settingResource{}
	setting := r.lcmModelToSetting(&settingLcmModel{
		Enabled:     types.BoolValue(true),
		Brightness:  types.Int64Null(),
		IdleTimeout: types.Int64Unknown(),
	})
	if setting.Brightness != nil || setting.IDleTimeout != nil {
		t.Errorf("unset lcm ints must be omitted: brightness=%v idle=%v",
			setting.Brightness, setting.IDleTimeout)
	}
}
