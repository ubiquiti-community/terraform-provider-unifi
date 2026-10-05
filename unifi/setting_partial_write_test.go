package unifi

import (
	"reflect"
	"testing"

	"github.com/ubiquiti-community/go-unifi/unifi/settings"
)

// A setting as UniFi Network 10.6.106 stores it: most optional booleans are
// absent rather than false.
func storedMgmt() map[string]any {
	return map[string]any{
		"_id":                         "66ec38ab34dbcb19dce4a66c",
		"site_id":                     "66ec388934dbcb19dce4a667",
		"key":                         "mgmt",
		"auto_upgrade":                true,
		"auto_upgrade_hour":           float64(3),
		"advanced_feature_enabled":    true,
		"debug_tools_enabled":         false,
		"unifi_idp_enabled":           true,
		"wifiman_enabled":             true,
		"x_ssh_enabled":               true,
		"x_ssh_auth_password_enabled": true,
		"x_ssh_password":              "secret",
	}
}

func Test_settingChanges_absentKeysAreNotWrittenAsFalse(t *testing.T) {
	// What the typed struct round-trips to: led_enabled, boot_sound, ... come
	// back as false because the struct has no way to say "absent".
	target := &settings.Mgmt{
		AutoUpgrade:                  true,
		AutoUpgradeHour:              ptrInt64(3),
		AdvancedFeatureEnabled:       true,
		UniFiIdentityProviderEnabled: true,
		WifimanEnabled:               true,
		SSHEnabled:                   true,
		SSHPassword:                  "secret",
	}
	got, err := settingChanges(storedMgmt(), target)
	if err != nil {
		t.Fatal(err)
	}
	// Only the real change: password auth switched off.
	want := map[string]any{"x_ssh_auth_password_enabled": false}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("changes = %v, want %v", got, want)
	}
}

func Test_settingChanges_unchangedSettingSendsNothing(t *testing.T) {
	target := &settings.Mgmt{
		AutoUpgrade:                  true,
		AutoUpgradeHour:              ptrInt64(3),
		AdvancedFeatureEnabled:       true,
		UniFiIdentityProviderEnabled: true,
		WifimanEnabled:               true,
		SSHEnabled:                   true,
		SSHAuthPasswordEnabled:       true,
		SSHPassword:                  "secret",
	}
	got, err := settingChanges(storedMgmt(), target)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("changes = %v, want none", got)
	}
}

func Test_settingChanges_newNonZeroKeyIsSent(t *testing.T) {
	target := &settings.Mgmt{AutoUpgrade: true, AutoUpgradeHour: ptrInt64(3), LedEnabled: true}
	got, err := settingChanges(storedMgmt(), target)
	if err != nil {
		t.Fatal(err)
	}
	if got["led_enabled"] != true {
		t.Errorf("led_enabled = %v, want true sent", got["led_enabled"])
	}
}

func Test_settingChanges_numberAndStringFormsAreEqual(t *testing.T) {
	// The controller stores country.code as "276"; go-unifi sends 276.
	got, err := settingChanges(map[string]any{"key": "country", "code": "276"}, &settings.Country{Code: ptrInt64(276)})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("changes = %v, want none", got)
	}
}

func Test_jsonValuesEqual(t *testing.T) {
	for _, tt := range []struct {
		a, b any
		want bool
	}{
		{[]any{"36", "40"}, []any{float64(36), float64(40)}, true},
		{[]any{"36"}, []any{float64(40)}, false},
		{map[string]any{"mode": "auto"}, map[string]any{"mode": "auto"}, true},
		{true, false, false},
		{"", float64(0), false},
	} {
		if got := jsonValuesEqual(tt.a, tt.b); got != tt.want {
			t.Errorf("jsonValuesEqual(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
