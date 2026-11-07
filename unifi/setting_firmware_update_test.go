package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi/settings"
)

// firmware_update writes only the channels it is given, on top of the stored
// super_fwupdate document, so sso_enabled (a plain bool the block does not
// model) is never sent as false.
func TestFirmwareUpdateModelToSetting(t *testing.T) {
	r := &settingResource{}
	stored := func() *settings.SuperFwupdate {
		return &settings.SuperFwupdate{FirmwareChannel: "beta", SsoEnabled: true}
	}

	for _, tc := range []struct {
		name           string
		model          settingFirmwareUpdateModel
		wantFirmware   string
		wantController string
	}{
		{
			"sets firmware channel",
			settingFirmwareUpdateModel{
				FirmwareChannel:   types.StringValue("release"),
				ControllerChannel: types.StringNull(),
			},
			"release", "",
		},
		{
			"sets controller channel",
			settingFirmwareUpdateModel{
				FirmwareChannel:   types.StringUnknown(),
				ControllerChannel: types.StringValue("release-candidate"),
			},
			"beta", "release-candidate",
		},
		{
			"unset keeps stored",
			settingFirmwareUpdateModel{
				FirmwareChannel:   types.StringNull(),
				ControllerChannel: types.StringNull(),
			},
			"beta", "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := r.firmwareUpdateModelToSetting(&tc.model, stored())
			if got.FirmwareChannel != tc.wantFirmware {
				t.Errorf("firmware_channel = %q, want %q", got.FirmwareChannel, tc.wantFirmware)
			}
			if got.ControllerChannel != tc.wantController {
				t.Errorf(
					"controller_channel = %q, want %q",
					got.ControllerChannel,
					tc.wantController,
				)
			}
			if !got.SsoEnabled {
				t.Error("sso_enabled = false, want the stored true preserved")
			}
		})
	}
}

// A UniFi OS console stores no controller_channel; it reads back as null
// rather than "".
func TestFirmwareUpdateSettingToModel(t *testing.T) {
	r := &settingResource{}
	m := r.firmwareUpdateSettingToModel(&settings.SuperFwupdate{FirmwareChannel: "beta"})
	if m.FirmwareChannel.ValueString() != "beta" {
		t.Errorf("firmware_channel = %v, want beta", m.FirmwareChannel)
	}
	if !m.ControllerChannel.IsNull() {
		t.Errorf("controller_channel = %v, want null", m.ControllerChannel)
	}
}
