package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func roamingAssistant(enabled bool, rssi *int64) types.Object {
	return types.ObjectValueMust(wlanRoamingAssistantAttrTypes(), map[string]attr.Value{
		"enabled": types.BoolValue(enabled),
		"rssi":    types.Int64PointerValue(rssi),
	})
}

// Update builds the request from state after applyPlanToState, so a planned
// roaming_assistant_* change must reach state; otherwise the old values are sent
// and the apply ends in "Provider produced inconsistent result".
func TestWLANApplyPlanToState_roamingAssistant(t *testing.T) {
	ctx := context.Background()
	r := &wlanFrameworkResource{}
	na, sixE := int64(-75), int64(-80)

	state := wlanFrameworkResourceModel{
		RoamingAssistantNa: roamingAssistant(false, nil),
		RoamingAssistant6E: roamingAssistant(false, nil),
	}
	plan := &wlanFrameworkResourceModel{
		RoamingAssistantNa: roamingAssistant(true, &na),
		RoamingAssistant6E: roamingAssistant(true, &sixE),
	}
	r.applyPlanToState(ctx, plan, &state)

	if !state.RoamingAssistantNa.Equal(plan.RoamingAssistantNa) {
		t.Errorf("planned roaming_assistant_na not applied: %v", state.RoamingAssistantNa)
	}
	if !state.RoamingAssistant6E.Equal(plan.RoamingAssistant6E) {
		t.Errorf("planned roaming_assistant_6e not applied: %v", state.RoamingAssistant6E)
	}

	wlan, d := r.planToWLAN(ctx, state)
	if d.HasError() {
		t.Fatalf("planToWLAN: %v", d)
	}
	if !wlan.RoamingAssistantNaEnabled || wlan.RoamingAssistantNaRssi == nil || *wlan.RoamingAssistantNaRssi != na {
		t.Errorf("payload 5 GHz = %v/%v, want true/%d", wlan.RoamingAssistantNaEnabled, wlan.RoamingAssistantNaRssi, na)
	}
	if !wlan.RoamingAssistant6EEnabled || wlan.RoamingAssistant6ERssi == nil || *wlan.RoamingAssistant6ERssi != sixE {
		t.Errorf("payload 6 GHz = %v/%v, want true/%d", wlan.RoamingAssistant6EEnabled, wlan.RoamingAssistant6ERssi, sixE)
	}

	// An unknown planned group keeps the controller's values.
	kept := state
	r.applyPlanToState(ctx, &wlanFrameworkResourceModel{
		RoamingAssistantNa: types.ObjectUnknown(wlanRoamingAssistantAttrTypes()),
		RoamingAssistant6E: types.ObjectUnknown(wlanRoamingAssistantAttrTypes()),
	}, &kept)
	if !kept.RoamingAssistantNa.Equal(state.RoamingAssistantNa) || !kept.RoamingAssistant6E.Equal(state.RoamingAssistant6E) {
		t.Errorf("unknown planned roaming assistant must keep state: na=%v 6e=%v", kept.RoamingAssistantNa, kept.RoamingAssistant6E)
	}
}
