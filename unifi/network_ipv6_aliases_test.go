package unifi

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/cidrtypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func ipv6WithAliases(settingPreference types.String, aliases types.List) types.Object {
	return types.ObjectValueMust(networkIPv6AttrTypes(), map[string]attr.Value{
		"interface_type":            types.StringValue("static"),
		"client_address_assignment": types.StringNull(),
		"static_subnet":             types.StringValue("fd00:699::1/64"),
		"setting_preference":        settingPreference,
		"aliases":                   aliases,
		"ra":                        types.ObjectNull(networkIPv6RAAttrTypes()),
		"pd":                        types.ObjectNull(networkIPv6PDAttrTypes()),
	})
}

// #699: ipv6.aliases and ipv6.setting_preference reach the controller. A null
// or unknown aliases list leaves IPV6Aliases nil, which go-unifi omits so the
// controller keeps its aliases; a known list is sent even when empty, so []
// clears them.
func TestModelToNetworkIPv6AliasesAndSettingPreference(t *testing.T) {
	ctx := context.Background()
	r := &networkResource{}
	ula := types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("fd7a:6c9e:4b28:4::1/64"),
	})

	for _, tc := range []struct {
		name        string
		pref        types.String
		aliases     types.List
		wantPref    *string
		wantAliases []string
	}{
		{"unset", types.StringUnknown(), types.ListUnknown(types.StringType), nil, nil},
		{"null aliases", types.StringNull(), types.ListNull(types.StringType), nil, nil},
		{
			"empty list clears",
			types.StringValue("auto"),
			types.ListValueMust(types.StringType, []attr.Value{}),
			strPtr("auto"),
			[]string{},
		},
		{"list is sent", types.StringValue("manual"), ula, strPtr("manual"), []string{"fd7a:6c9e:4b28:4::1/64"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model := networkResourceModel{
				Name:   types.StringValue("ula"),
				Subnet: cidrtypes.NewIPv4PrefixValue("10.69.9.1/24"),
				IPv6:   ipv6WithAliases(tc.pref, tc.aliases),
			}
			api, diags := r.modelToNetwork(ctx, &model)
			if diags.HasError() {
				t.Fatalf("modelToNetwork: %v", diags)
			}
			if !reflect.DeepEqual(api.IPV6SettingPreference, tc.wantPref) {
				t.Errorf(
					"IPV6SettingPreference = %v, want %v",
					api.IPV6SettingPreference,
					tc.wantPref,
				)
			}
			if (api.IPV6Aliases == nil) != (tc.wantAliases == nil) ||
				!reflect.DeepEqual(api.IPV6Aliases, tc.wantAliases) && len(tc.wantAliases) > 0 {
				t.Errorf("IPV6Aliases = %#v, want %#v", api.IPV6Aliases, tc.wantAliases)
			}
		})
	}
}

// #699: the controller's ipv6_aliases and ipv6_setting_preference are read
// back, so an adopted network shows them and drift is visible. An absent list
// reads as null, a cleared one ([]) as an empty list, matching a configured
// `aliases = []`.
func TestNetworkIPv6FromAPIAliasesAndSettingPreference(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name     string
		aliases  []string
		wantNull bool
		wantLen  int
	}{
		{"absent", nil, true, 0},
		{"cleared", []string{}, false, 0},
		{"set", []string{"fd7a:6c9e:4b28:4::1/64", "fd7a:6c9e:4b28:5::1/64"}, false, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			obj, diags := networkIPv6FromAPI(ctx, &unifi.Network{
				IPV6InterfaceType:     strPtr("static"),
				IPV6SettingPreference: strPtr("manual"),
				IPV6Aliases:           tc.aliases,
			})
			if diags.HasError() {
				t.Fatalf("networkIPv6FromAPI: %v", diags)
			}
			attrs := obj.Attributes()
			if got := attrs["setting_preference"]; !got.Equal(types.StringValue("manual")) {
				t.Errorf("setting_preference = %v, want manual", got)
			}
			aliases, ok := attrs["aliases"].(types.List)
			if !ok {
				t.Fatalf("aliases is %T", attrs["aliases"])
			}
			if aliases.IsNull() != tc.wantNull || len(aliases.Elements()) != tc.wantLen {
				t.Errorf(
					"aliases = %v, want null=%v with %d entries",
					aliases,
					tc.wantNull,
					tc.wantLen,
				)
			}
		})
	}
}

// #699: when setting_preference changes, the controller rewrites the IPv6
// leaves it computes, so those the configuration leaves unset plan unknown
// while configured ones keep their planned value.
func TestUnknownUnsetLeaves(t *testing.T) {
	ctx := context.Background()
	pd := types.ObjectValueMust(networkIPv6PDAttrTypes(), map[string]attr.Value{
		"interface":             types.StringNull(),
		"prefixid":              types.StringNull(),
		"start":                 types.StringValue("::10"),
		"stop":                  types.StringValue("::20"),
		"auto_prefixid_enabled": types.BoolValue(false),
	})
	config := map[string]attr.Value{
		"start": types.StringValue("::10"),
		"stop":  types.StringNull(),
	}
	got, diags := unknownUnsetLeaves(
		ctx,
		pd,
		config,
		[]string{"start", "stop", "auto_prefixid_enabled"},
	)
	if diags.HasError() {
		t.Fatalf("unknownUnsetLeaves: %v", diags)
	}
	attrs := got.Attributes()
	if !attrs["start"].Equal(types.StringValue("::10")) {
		t.Errorf("configured start = %v, want ::10", attrs["start"])
	}
	if !attrs["stop"].IsUnknown() || !attrs["auto_prefixid_enabled"].IsUnknown() {
		t.Errorf("unset leaves = stop %v, auto_prefixid_enabled %v, want unknown",
			attrs["stop"], attrs["auto_prefixid_enabled"])
	}
	if !attrs["interface"].IsNull() {
		t.Errorf("interface = %v, want null (not in names)", attrs["interface"])
	}

	got, diags = unknownUnsetLeaves(ctx, pd, nil, []string{"start"})
	if diags.HasError() || !got.Attributes()["start"].IsUnknown() {
		t.Errorf(
			"a null config object must leave start unknown, got %v (%v)",
			got.Attributes()["start"],
			diags,
		)
	}
}
