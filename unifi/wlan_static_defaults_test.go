package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
		"wpa_enc",
		"wpa3_support",
		"wpa3_transition",
		"pmf_mode",
		"bss_transition",
		"hide_ssid",
		"no2ghz_oui",
		"enhanced_iot",
		"wlan_band",
		"wlan_bands",
	} {
		attr, ok := resp.Schema.Attributes[name]
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
