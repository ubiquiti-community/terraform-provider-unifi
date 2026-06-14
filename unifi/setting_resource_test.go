package unifi

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	tests := []struct {
		name string
		want fwresource.Resource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewSettingResource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSettingResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_settingResource_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.MetadataRequest
		resp *fwresource.MetadataResponse
	}
	tests := []struct {
		name string
		r    *settingResource
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

func Test_settingResource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.SchemaRequest
		resp *fwresource.SchemaResponse
	}
	tests := []struct {
		name string
		r    *settingResource
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

func Test_settingResource_UpgradeState(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		r    *settingResource
		args args
		want map[int64]fwresource.StateUpgrader
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.UpgradeState(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("settingResource.UpgradeState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_settingResource_Configure(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ConfigureRequest
		resp *fwresource.ConfigureResponse
	}
	tests := []struct {
		name string
		r    *settingResource
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

func Test_settingResource_Create(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.CreateRequest
		resp *fwresource.CreateResponse
	}
	tests := []struct {
		name string
		r    *settingResource
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

func Test_settingResource_Read(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ReadRequest
		resp *fwresource.ReadResponse
	}
	tests := []struct {
		name string
		r    *settingResource
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

func Test_settingResource_Update(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.UpdateRequest
		resp *fwresource.UpdateResponse
	}
	tests := []struct {
		name string
		r    *settingResource
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

func Test_settingResource_Delete(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.DeleteRequest
		resp *fwresource.DeleteResponse
	}
	tests := []struct {
		name string
		r    *settingResource
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

func Test_settingResource_ImportState(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ImportStateRequest
		resp *fwresource.ImportStateResponse
	}
	tests := []struct {
		name string
		r    *settingResource
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

func Test_settingResource_readSettings(t *testing.T) {
	type args struct {
		ctx   context.Context
		site  string
		data  *settingResourceModel
		diags *diag.Diagnostics
	}
	tests := []struct {
		name string
		r    *settingResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.readSettings(tt.args.ctx, tt.args.site, tt.args.data, tt.args.diags)
		})
	}
}

func Test_settingResource_mgmtModelToSetting(t *testing.T) {
	type args struct {
		ctx   context.Context
		model *settingMgmtModel
	}
	tests := []struct {
		name string
		r    *settingResource
		args args
		want *settings.Mgmt
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.mgmtModelToSetting(
				tt.args.ctx,
				tt.args.model,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("settingResource.mgmtModelToSetting() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_settingResource_mgmtSettingToModel(t *testing.T) {
	type args struct {
		ctx     context.Context
		setting *settings.Mgmt
		plan    *settingMgmtModel
	}
	tests := []struct {
		name string
		r    *settingResource
		args args
		want *settingMgmtModel
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.mgmtSettingToModel(
				tt.args.ctx,
				tt.args.setting,
				tt.args.plan,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("settingResource.mgmtSettingToModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_settingResource_radiusModelToSetting(t *testing.T) {
	type args struct {
		in0   context.Context
		model *settingRadiusModel
		base  *settings.Radius
	}
	tests := []struct {
		name string
		r    *settingResource
		args args
		want *settings.Radius
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.radiusModelToSetting(
				tt.args.in0,
				tt.args.model,
				tt.args.base,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("settingResource.radiusModelToSetting() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_settingResource_radiusSettingToModel(t *testing.T) {
	type args struct {
		in0     context.Context
		setting *settings.Radius
		plan    *settingRadiusModel
	}
	tests := []struct {
		name string
		r    *settingResource
		args args
		want *settingRadiusModel
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.radiusSettingToModel(
				tt.args.in0,
				tt.args.setting,
				tt.args.plan,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("settingResource.radiusSettingToModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_settingResource_usgModelToSetting(t *testing.T) {
	type args struct {
		ctx   context.Context
		model *settingUSGModel
	}
	tests := []struct {
		name string
		r    *settingResource
		args args
		want *settings.Usg
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.usgModelToSetting(
				tt.args.ctx,
				tt.args.model,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("settingResource.usgModelToSetting() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_settingResource_usgSettingToModel(t *testing.T) {
	type args struct {
		ctx     context.Context
		setting *settings.Usg
		plan    *settingUSGModel
	}
	tests := []struct {
		name string
		r    *settingResource
		args args
		want *settingUSGModel
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.usgSettingToModel(
				tt.args.ctx,
				tt.args.setting,
				tt.args.plan,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("settingResource.usgSettingToModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_settingResource_igmpSnoopingModelToSetting(t *testing.T) {
	type args struct {
		ctx   context.Context
		model *settingIgmpSnoopingModel
		base  *settings.IgmpSnooping
		diags *diag.Diagnostics
	}
	tests := []struct {
		name string
		r    *settingResource
		args args
		want *settings.IgmpSnooping
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.igmpSnoopingModelToSetting(
				tt.args.ctx,
				tt.args.model,
				tt.args.base,
				tt.args.diags,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("settingResource.igmpSnoopingModelToSetting() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_settingResource_igmpSnoopingSettingToModel(t *testing.T) {
	type args struct {
		ctx     context.Context
		setting *settings.IgmpSnooping
		diags   *diag.Diagnostics
	}
	tests := []struct {
		name string
		r    *settingResource
		args args
		want *settingIgmpSnoopingModel
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.igmpSnoopingSettingToModel(
				tt.args.ctx,
				tt.args.setting,
				tt.args.diags,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("settingResource.igmpSnoopingSettingToModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_settingResource_dohModelToSetting(t *testing.T) {
	type args struct {
		ctx   context.Context
		model *settingDohModel
		diags *diag.Diagnostics
	}
	tests := []struct {
		name string
		r    *settingResource
		args args
		want *settings.Doh
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.dohModelToSetting(
				tt.args.ctx,
				tt.args.model,
				tt.args.diags,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("settingResource.dohModelToSetting() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_settingResource_dohSettingToModel(t *testing.T) {
	type args struct {
		ctx     context.Context
		setting *settings.Doh
		plan    *settingDohModel
		diags   *diag.Diagnostics
	}
	tests := []struct {
		name string
		r    *settingResource
		args args
		want *settingDohModel
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.dohSettingToModel(
				tt.args.ctx,
				tt.args.setting,
				tt.args.plan,
				tt.args.diags,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("settingResource.dohSettingToModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_settingResource_ipsModelToSetting(t *testing.T) {
	type args struct {
		ctx   context.Context
		model *settingIpsModel
		diags *diag.Diagnostics
	}
	tests := []struct {
		name string
		r    *settingResource
		args args
		want *settings.Ips
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.ipsModelToSetting(
				tt.args.ctx,
				tt.args.model,
				tt.args.diags,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("settingResource.ipsModelToSetting() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_settingResource_ipsSettingToModel(t *testing.T) {
	type args struct {
		ctx     context.Context
		setting *settings.Ips
		plan    *settingIpsModel
		diags   *diag.Diagnostics
	}
	tests := []struct {
		name string
		r    *settingResource
		args args
		want *settingIpsModel
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.ipsSettingToModel(
				tt.args.ctx,
				tt.args.setting,
				tt.args.plan,
				tt.args.diags,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("settingResource.ipsSettingToModel() = %v, want %v", got, tt.want)
			}
		})
	}
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
