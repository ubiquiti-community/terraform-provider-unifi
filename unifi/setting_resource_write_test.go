package unifi

import (
	"context"
	"reflect"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
)

// A configuration touching every setting block that goes through writeSetting.
func everySettingBlock(dpiEnabled bool) map[string]any {
	return map[string]any{
		"auto_speedtest": map[string]any{"enabled": false, "cron_expr": "0 2 * * *"},
		"country":        map[string]any{"code": 276},
		"doh": map[string]any{
			"state":        "auto",
			"server_names": []any{"cloudflare", "google"},
		},
		"dpi": map[string]any{
			"enabled":                dpiEnabled,
			"fingerprinting_enabled": true,
		},
		"igmp_snooping": map[string]any{"enabled": true, "network_ids": []any{"net-a"}},
		"ips":           map[string]any{"ips_mode": "disabled", "memory_optimized": true},
		"lcm": map[string]any{
			"enabled":      true,
			"brightness":   80,
			"idle_timeout": 300,
		},
		"mgmt":                 map[string]any{"auto_upgrade": true, "auto_upgrade_hour": 3},
		"network_optimization": map[string]any{"enabled": true},
		"ntp": map[string]any{
			"setting_preference": "manual",
			"ntp_server_1":       "0.pool.ntp.org",
		},
		"radius": map[string]any{"accounting_enabled": false, "auth_port": 1812, "acct_port": 1813},
		"syslog": map[string]any{"enabled": true, "this_controller": true},
		"usg":    map[string]any{"broadcast_ping": true, "upnp_enabled": false},
	}
}

// A controller stores every setting from the start, even if only with its key.
func storedSettingKeys() []map[string]any {
	keys := []string{
		"auto_speedtest", "country", "doh", "dpi", "igmp_snooping", "ips", "lcm",
		"mgmt", "network_optimization", "ntp", "radius", "rsyslogd", "usg",
	}
	stored := make([]map[string]any, len(keys))
	for i, k := range keys {
		stored[i] = map[string]any{"key": k}
	}
	return stored
}

func settingCreate(
	t *testing.T,
	r *settingResource,
	attrs map[string]any,
) *fwresource.CreateResponse {
	t.Helper()
	s := resourceSchema(t, r)
	resp := &fwresource.CreateResponse{State: tfState(t, s, nil), Identity: tfIdentity(t, r)}
	r.Create(context.Background(), fwresource.CreateRequest{
		Plan: tfPlan(t, s, attrs), Config: tfConfig(t, s, attrs),
	}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
	return resp
}

func settingUpdate(
	t *testing.T,
	r *settingResource,
	state fwresource.CreateResponse,
	attrs map[string]any,
) {
	t.Helper()
	s := resourceSchema(t, r)
	resp := &fwresource.UpdateResponse{State: state.State, Identity: state.Identity}
	r.Update(context.Background(), fwresource.UpdateRequest{
		Plan: tfPlan(t, s, attrs), Config: tfConfig(t, s, attrs), State: state.State,
	}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", resp.Diagnostics)
	}
}

// Create writes every configured block, and an apply with nothing changed
// writes nothing: each block goes through writeSetting, which compares against
// the stored setting.
func Test_settingResource_createThenUnchangedUpdateWritesNothing(t *testing.T) {
	r, puts := newSettingsFakeController(t, storedSettingKeys())

	created := settingCreate(t, r, everySettingBlock(true))
	written := puts()
	for _, key := range []string{
		"auto_speedtest", "country", "doh", "dpi", "igmp_snooping", "ips", "lcm",
		"mgmt", "network_optimization", "ntp", "radius", "rsyslogd", "usg",
	} {
		if _, ok := written[key]; !ok {
			t.Errorf("create did not write %s", key)
		}
	}

	settingUpdate(t, r, *created, everySettingBlock(true))
	if got := puts(); len(got) != 0 {
		t.Errorf("unchanged update wrote %v", got)
	}
}

// An update sends only the block that changed, and only its changed key.
func Test_settingResource_updateWritesOnlyTheChange(t *testing.T) {
	r, puts := newSettingsFakeController(t, storedSettingKeys())
	created := settingCreate(t, r, everySettingBlock(true))
	puts()

	settingUpdate(t, r, *created, everySettingBlock(false))
	want := map[string]map[string]any{"dpi": {"key": "dpi", "enabled": false}}
	if got := puts(); !reflect.DeepEqual(got, want) {
		t.Errorf("update wrote %v, want %v", got, want)
	}
}
