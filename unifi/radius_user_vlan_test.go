package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestResolveVLAN_DeterministicBranches covers the paths of resolveVLAN that do
// not touch the controller (#67): an explicit vlan is returned as-is, and with
// neither vlan nor network_id the result is nil (untagged fallback). The
// network_id-derivation branch calls GetNetwork and is exercised by acceptance
// tests against a real controller.
func TestResolveVLAN_DeterministicBranches(t *testing.T) {
	ctx := context.Background()
	r := &radiusUserResource{} // client is nil; these branches never use it

	t.Run("explicit vlan wins", func(t *testing.T) {
		model := &radiusUserResourceModel{
			VLAN:      types.Int64Value(100),
			NetworkID: types.StringValue("net-abc"), // ignored when vlan is set
		}
		vlan, diags := r.resolveVLAN(ctx, model, "default")
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if vlan == nil || *vlan != 100 {
			t.Fatalf("vlan = %v, want 100", vlan)
		}
	})

	t.Run("no vlan and no network_id yields nil", func(t *testing.T) {
		model := &radiusUserResourceModel{
			VLAN:      types.Int64Null(),
			NetworkID: types.StringNull(),
		}
		vlan, diags := r.resolveVLAN(ctx, model, "default")
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if vlan != nil {
			t.Fatalf("vlan = %v, want nil", *vlan)
		}
	})

	t.Run("empty network_id string yields nil", func(t *testing.T) {
		model := &radiusUserResourceModel{
			VLAN:      types.Int64Null(),
			NetworkID: types.StringValue(""),
		}
		vlan, diags := r.resolveVLAN(ctx, model, "default")
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if vlan != nil {
			t.Fatalf("vlan = %v, want nil", *vlan)
		}
	})
}
