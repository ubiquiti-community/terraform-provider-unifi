package unifi

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/cidrtypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// An unset setting_preference is unknown on create (#550 dropped its Default).
// ValueStringPointer turns an unknown into a pointer to "", and the controller
// rejects an empty setting_preference with api.err.InvalidPayload, so the
// request must omit it and let the controller apply its own default.
func TestModelToNetworkOmitsUnknownSettingPreference(t *testing.T) {
	ctx := context.Background()
	r := &networkResource{}

	model := networkResourceModel{
		Name:              types.StringValue("Test VLAN"),
		Subnet:            cidrtypes.NewIPv4PrefixValue("192.168.10.1/24"),
		SettingPreference: types.StringUnknown(),
	}
	api, diags := r.modelToNetwork(ctx, &model)
	if diags.HasError() {
		t.Fatalf("modelToNetwork: %v", diags)
	}
	if api.SettingPreference != nil {
		t.Errorf(
			"unknown setting_preference serialized as %q, want omitted",
			*api.SettingPreference,
		)
	}

	model.SettingPreference = types.StringValue("manual")
	api, diags = r.modelToNetwork(ctx, &model)
	if diags.HasError() {
		t.Fatalf("modelToNetwork: %v", diags)
	}
	if api.SettingPreference == nil || *api.SettingPreference != "manual" {
		t.Errorf("known setting_preference = %v, want manual", api.SettingPreference)
	}
}

// An imported Default LAN can be stored without setting_preference, and the
// controller writes "manual" when the network is next saved. An update must
// leave the null prior unknown instead of planning it as a known null, which
// failed with "inconsistent result after apply" (#723).
func TestNetworkUpdatePlansUnknownForNullSettingPreference(t *testing.T) {
	doc := strings.Replace(networkTestDocument, `"setting_preference": "auto",`, "", 1)
	h := newNetworkHarness(t, doc)
	h.assertEqual("plan after import", h.state.Raw, h.plan(h.config(networkBaseConfig())))

	cfg := networkBaseConfig()
	cfg["igmp_snooping"] = types.BoolValue(true)
	var attrs map[string]tftypes.Value
	if err := h.plan(h.config(cfg)).As(&attrs); err != nil {
		t.Fatalf("planned state as object: %v", err)
	}
	if attrs["setting_preference"].IsKnown() {
		t.Errorf(
			"update planned setting_preference = %v, want unknown",
			attrs["setting_preference"],
		)
	}
}
