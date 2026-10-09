package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/cidrtypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
