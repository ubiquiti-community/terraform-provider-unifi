package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// #543: group_rekey and wpa_enc were Optional+Computed *and* carried a static
// Default. The two contracts conflict and Default wins, so on an adopted SSID
// every undeclared attribute was planned at the constant rather than at the
// controller's value - measured on a live site, that rewrote group rekeying on
// five of six SSIDs and the encryption mode on two.
//
// An Optional+Computed attribute must take the controller's value when the
// configuration does not set one, which is what UseStateForUnknown gives.
func TestWLANNoStaticDefaultsOnComputedAttributes(t *testing.T) {
	var resp resource.SchemaResponse
	(&wlanFrameworkResource{}).Schema(
		context.Background(),
		resource.SchemaRequest{},
		&resp,
	)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}

	// group_rekey and wpa_enc came from #543. The rest were measured against a
	// live six-SSID site for #544: each one's Default disagreed with what the
	// controller actually stores on at least two SSIDs, and four of them are
	// security settings - importing a WPA3 SSID proposed turning WPA3 off
	// (wpa3_support, wpa3_transition), dropping 802.11r roaming
	// (bss_transition) and disabling management-frame protection (pmf_mode).
	for _, name := range []string{
		"group_rekey",
		"wpa.enc",
		"wpa3.support",
		"wpa3.transition",
		"pmf_mode",
		"bss_transition",
		"hide_ssid",
		"no2ghz_oui",
		"enhanced_iot",
		"wlan_band",
		"wlan_bands",
		// Third batch (#544): booleans go-unifi always sends, so an unset
		// value still creates the SSID with `false`, exactly as the Default
		// did; nas_identifier_type is absent on four of six live SSIDs, so the
		// controller does not need it on a stored WLAN.
		"wpa3.fast_roaming",
		"wpa3.enhanced_192",
		"is_guest",
		"vlan_enabled",
		"multicast_enhance",
		"mac_filter.enabled",
		"private_preshared_keys_enabled",
		"nas_identifier_type",
		"l2_isolation",
		"proxy_arp",
		"uapsd",
		"fast_roaming_enabled",
		"hotspot2conf_enabled",
		"mlo_enabled",
		// Create-only defaults (#544): planned on create, held on update.
		"enabled",
		"ap_group.mode",
		"mac_filter.policy",
		"minrate_setting_preference",
		"wpa.mode",
		"dtim_mode",
	} {
		attr, ok := lookupSchemaAttribute(resp.Schema.Attributes, name)
		if !ok {
			t.Errorf("attribute %q is missing from the schema", name)
			continue
		}

		var (
			def      any
			nModifs  int
			unmapped bool
		)
		switch a := attr.(type) {
		case schema.BoolAttribute:
			def, nModifs = a.Default, len(a.PlanModifiers)
		case schema.Int64Attribute:
			def, nModifs = a.Default, len(a.PlanModifiers)
		case schema.StringAttribute:
			def, nModifs = a.Default, len(a.PlanModifiers)
		case schema.SetAttribute:
			def, nModifs = a.Default, len(a.PlanModifiers)
		default:
			unmapped = true
		}
		if unmapped {
			t.Errorf("%s has unexpected type %T", name, attr)
			continue
		}

		if def != nil {
			t.Errorf(
				"%s must not carry a static Default: the framework applies "+
					"defaults before plan modifiers, so it overwrites the "+
					"controller's value on an adopted SSID (#543, #544)",
				name,
			)
		}
		if nModifs == 0 {
			t.Errorf("%s needs UseStateForUnknown so the prior value is held", name)
		}
	}
}

// TestWLANApGroupHasNoObjectDefault: a Default on the ap_group object reset an
// adopted SSID to `mode = "all"` with no ids, which the sparse update turns
// into clearing ap_group_ids (#544).
func TestWLANApGroupHasNoObjectDefault(t *testing.T) {
	var resp resource.SchemaResponse
	(&wlanFrameworkResource{}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	a, ok := resp.Schema.Attributes["ap_group"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf(
			"ap_group is %T, want schema.SingleNestedAttribute",
			resp.Schema.Attributes["ap_group"],
		)
	}
	if a.Default != nil {
		t.Error("ap_group must not carry an object Default (#544)")
	}
	if len(a.PlanModifiers) == 0 {
		t.Error("ap_group needs UseStateForUnknown so the prior object is held")
	}
}

// TestWLANAdoptedSSIDKeepsControllerValues drives the real plan step for an
// adopted SSID whose controller values all differ from the former schema
// Defaults, with a configuration that sets only the required attributes. Every
// one of them must be held at the controller's value (#544).
func TestWLANAdoptedSSIDKeepsControllerValues(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	(&wlanFrameworkResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}

	f := planFixture{ctx: ctx, t: t, overrides: map[string]attr.Value{
		"enabled":                        types.BoolValue(false),
		"is_guest":                       types.BoolValue(true),
		"vlan_enabled":                   types.BoolValue(true),
		"multicast_enhance":              types.BoolValue(true),
		"private_preshared_keys_enabled": types.BoolValue(true),
		"l2_isolation":                   types.BoolValue(true),
		"proxy_arp":                      types.BoolValue(true),
		"uapsd":                          types.BoolValue(true),
		"fast_roaming_enabled":           types.BoolValue(true),
		"hotspot2conf_enabled":           types.BoolValue(true),
		"mlo_enabled":                    types.BoolValue(true),
		"nas_identifier_type":            types.StringValue("ap_name"),
		"minrate_setting_preference":     types.StringValue("manual"),
		"dtim_mode":                      types.StringValue("custom"),
		"wpa.mode":                       types.StringValue("auto"),
		"wpa3.fast_roaming":              types.BoolValue(true),
		"wpa3.enhanced_192":              types.BoolValue(true),
		"mac_filter.enabled":             types.BoolValue(true),
		"mac_filter.policy":              types.StringValue("allow"),
		"ap_group": types.ObjectValueMust(wlanApGroupAttrTypes(), map[string]attr.Value{
			"ids": types.SetValueMust(types.StringType, []attr.Value{
				types.StringValue("aaaaaaaaaaaaaaaaaaaaaaaa"),
			}),
			"mode": types.StringValue("groups"),
		}),
	}}
	prior, config := f.object(schemaResp.Schema, "")
	planned := planResourceChange(t, "unifi_wlan", schemaResp.Schema, prior, prior, config)

	diffs, err := prior.Diff(planned)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	for _, d := range diffs {
		t.Errorf("adopted SSID planned %s: %v -> %v", d.Path, d.Value1, d.Value2)
	}
}

// TestWLANCreateKeepsCreateTimeDefaults checks that a new SSID declared with
// only its required attributes still plans the create-time values the former
// Defaults gave, where the request needs them, and leaves the rest to the
// controller.
func TestWLANCreateKeepsCreateTimeDefaults(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	(&wlanFrameworkResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}
	f := planFixture{ctx: ctx, t: t}
	_, config := f.object(schemaResp.Schema, "")
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)
	created := planResourceChange(t, "unifi_wlan", schemaResp.Schema,
		tftypes.NewValue(schemaType, nil), config, config)

	var attrs map[string]tftypes.Value
	if err := created.As(&attrs); err != nil {
		t.Fatalf("as object: %v", err)
	}
	wantString := func(name string, v tftypes.Value, want string) {
		t.Helper()
		var got string
		if err := v.As(&got); err != nil || got != want {
			t.Errorf("%s = %v (%v), want %q", name, v, err, want)
		}
	}
	var enabled bool
	if err := attrs["enabled"].As(&enabled); err != nil || !enabled {
		t.Errorf(
			"enabled = %v (%v), want true: go-unifi always sends the key",
			attrs["enabled"],
			err,
		)
	}
	wantString("minrate_setting_preference", attrs["minrate_setting_preference"], "auto")
	wantString("dtim_mode", attrs["dtim_mode"], "default")
	wantString("wpa.mode", plannedObject(t, attrs["wpa"], "wpa")["mode"], "wpa2")
	apGroup := plannedObject(t, attrs["ap_group"], "ap_group")
	wantString("ap_group.mode", apGroup["mode"], "all")
	if !apGroup["ids"].IsKnown() || !apGroup["ids"].IsNull() {
		t.Errorf(
			"ap_group.ids = %v, want null (Create resolves the default AP group)",
			apGroup["ids"],
		)
	}
	for _, name := range []string{"is_guest", "nas_identifier_type", "l2_isolation", "mlo_enabled"} {
		if attrs[name].IsKnown() {
			t.Errorf("%s = %v, want unknown (taken from the controller)", name, attrs[name])
		}
	}
}
