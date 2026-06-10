package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// TestWLANPrivatePresharedKeys_roundTrip exercises the private pre-shared key
// (PPSK) mapping added for issue #47: a plan carrying PPSK entries must be
// translated to the go-unifi WLAN struct (planToWLAN) and back into the
// resource model (wlanToModel) without losing the per-key network binding or
// password.
func TestWLANPrivatePresharedKeys_roundTrip(t *testing.T) {
	ctx := context.Background()
	r := &wlanFrameworkResource{}

	ppskType := types.ObjectType{AttrTypes: wlanPrivatePresharedKeyModel{}.AttributeTypes()}
	ppskList, d := types.ListValueFrom(ctx, ppskType, []wlanPrivatePresharedKeyModel{
		{NetworkID: types.StringValue("net-a"), Password: types.StringValue("secretpass1")},
		{NetworkID: types.StringValue(""), Password: types.StringValue("secretpass2")},
	})
	if d.HasError() {
		t.Fatalf("building PPSK list: %v", d)
	}

	plan := wlanFrameworkResourceModel{
		Name:                        types.StringValue("ppsk-wlan"),
		Security:                    types.StringValue("wpapsk"),
		PrivatePresharedKeysEnabled: types.BoolValue(true),
		PrivatePresharedKeys:        ppskList,
	}

	// plan -> API
	wlan, diags := r.planToWLAN(ctx, plan)
	if diags.HasError() {
		t.Fatalf("planToWLAN: %v", diags)
	}
	if !wlan.PrivatePresharedKeysEnabled {
		t.Errorf("PrivatePresharedKeysEnabled = false, want true")
	}
	if got := len(wlan.PrivatePresharedKeys); got != 2 {
		t.Fatalf("PrivatePresharedKeys len = %d, want 2", got)
	}
	if wlan.PrivatePresharedKeys[0].NetworkID != "net-a" ||
		wlan.PrivatePresharedKeys[0].Password != "secretpass1" {
		t.Errorf("PPSK[0] = %+v, want {net-a secretpass1}", wlan.PrivatePresharedKeys[0])
	}
	if wlan.PrivatePresharedKeys[1].NetworkID != "" ||
		wlan.PrivatePresharedKeys[1].Password != "secretpass2" {
		t.Errorf("PPSK[1] = %+v, want { secretpass2}", wlan.PrivatePresharedKeys[1])
	}

	// API -> model
	var model wlanFrameworkResourceModel
	if diags := r.wlanToModel(ctx, wlan, &model, "default"); diags.HasError() {
		t.Fatalf("wlanToModel: %v", diags)
	}
	if !model.PrivatePresharedKeysEnabled.ValueBool() {
		t.Errorf("model.PrivatePresharedKeysEnabled = false, want true")
	}
	if model.PrivatePresharedKeys.IsNull() {
		t.Fatalf("model.PrivatePresharedKeys is null, want 2 entries")
	}
	var got []wlanPrivatePresharedKeyModel
	if diags := model.PrivatePresharedKeys.ElementsAs(ctx, &got, false); diags.HasError() {
		t.Fatalf("decoding model PPSK: %v", diags)
	}
	if len(got) != 2 {
		t.Fatalf("model PPSK len = %d, want 2", len(got))
	}
	if got[0].NetworkID.ValueString() != "net-a" ||
		got[0].Password.ValueString() != "secretpass1" {
		t.Errorf("model PPSK[0] = %+v, want {net-a secretpass1}", got[0])
	}
}

// TestWLANPrivatePresharedKeys_emptyIsNull verifies that a WLAN without PPSK
// entries reads back as a null list (not an empty list), avoiding spurious
// plan drift for WLANs that don't use private pre-shared keys.
func TestWLANPrivatePresharedKeys_emptyIsNull(t *testing.T) {
	ctx := context.Background()
	r := &wlanFrameworkResource{}

	var model wlanFrameworkResourceModel
	if diags := r.wlanToModel(ctx, &unifi.WLAN{}, &model, "default"); diags.HasError() {
		t.Fatalf("wlanToModel: %v", diags)
	}
	if model.PrivatePresharedKeysEnabled.ValueBool() {
		t.Errorf("PrivatePresharedKeysEnabled = true, want false")
	}
	if !model.PrivatePresharedKeys.IsNull() {
		t.Errorf("PrivatePresharedKeys = %v, want null", model.PrivatePresharedKeys)
	}
}
