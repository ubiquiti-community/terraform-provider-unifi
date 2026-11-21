package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi/settings"
)

// #709: the controller answers 200 but ignores ips_mode, enabled_categories and
// enabled_networks while advanced_filtering_preference is "disabled", so the
// write must carry "manual" whenever IPS is being turned on.
func TestSettingIpsAdvancedFilteringPreference(t *testing.T) {
	ctx := context.Background()
	r := &settingResource{}

	model := func(mode, pref types.String) *settingIpsModel {
		return &settingIpsModel{
			IPSMode:                             mode,
			AdvancedFilteringPreference:         pref,
			HoneypotEnabled:                     types.BoolNull(),
			ContentFilteringBlockingPageEnabled: types.BoolNull(),
			MemoryOptimized:                     types.BoolNull(),
			RestrictTorrents:                    types.BoolNull(),
			EnabledCategories:                   types.ListNull(types.StringType),
			EnabledNetworks:                     types.ListNull(types.StringType),
			Honeypot:                            types.ListNull(types.ObjectType{AttrTypes: ipsHoneypotAttrTypes}),
			Suppression:                         types.ObjectNull(ipsSuppressionAttrTypes),
		}
	}
	disabledLive := func() *settings.Ips {
		return &settings.Ips{IPsMode: "disabled", AdvancedFilteringPreference: "disabled"}
	}

	cases := []struct {
		name string
		m    *settingIpsModel
		live *settings.Ips
		want string
	}{
		{
			"enabling IPS while live preference is disabled sends manual",
			model(types.StringValue("ids"), types.StringNull()),
			disabledLive(),
			"manual",
		},
		{
			"unknown (computed) preference is treated as unset",
			model(types.StringValue("ips"), types.StringUnknown()),
			disabledLive(),
			"manual",
		},
		{
			"explicitly configured preference is respected",
			model(types.StringValue("ids"), types.StringValue("disabled")),
			disabledLive(),
			"disabled",
		},
		{
			"ips_mode disabled leaves the preference alone",
			model(types.StringValue("disabled"), types.StringNull()),
			disabledLive(),
			"disabled",
		},
		{
			"ips_mode not managed leaves the preference alone",
			model(types.StringNull(), types.StringNull()),
			disabledLive(),
			"disabled",
		},
		{
			"live manual stays manual",
			model(types.StringValue("ids"), types.StringNull()),
			&settings.Ips{IPsMode: "ids", AdvancedFilteringPreference: "manual"},
			"manual",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var diags diag.Diagnostics
			got, _ := r.ipsModelToSetting(ctx, c.m, &diags, c.live)
			if diags.HasError() {
				t.Fatalf("conversion: %v", diags)
			}
			if got.AdvancedFilteringPreference != c.want {
				t.Errorf("advanced_filtering_preference = %q, want %q",
					got.AdvancedFilteringPreference, c.want)
			}
		})
	}
}
