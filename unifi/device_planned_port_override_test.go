package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
)

// What the framework plans for a block that sets op_mode, on a create and once
// anything on the device changes: every unset Optional+Computed attribute
// unknown. Stored like that, the apply fails after the write.
func TestResolvePlannedPortOverrides(t *testing.T) {
	ctx := context.Background()

	lldpmed := types.ObjectValueMust(portLldpmedAttrTypes(), map[string]attr.Value{
		"enabled":        types.BoolValue(false),
		"notify_enabled": types.BoolUnknown(),
	})
	planned := portOverrideSetWith(t, map[string]attr.Value{
		"index":     types.Int64Value(1),
		"op_mode":   types.StringValue("aggregate"),
		"autoneg":   types.BoolUnknown(),
		"isolation": types.BoolUnknown(),
		"dot1x":     types.ObjectUnknown(portDot1xAttrTypes()),
		"lldpmed":   lldpmed,
	})

	var diags diag.Diagnostics
	got := resolvePlannedPortOverrides(ctx, &diags, planned)
	if diags.HasError() {
		t.Fatalf("resolvePlannedPortOverrides errored: %v", diags)
	}
	raw, err := got.ToTerraformValue(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !raw.IsFullyKnown() {
		t.Fatalf("resolved set still holds unknown values: %v", raw)
	}

	var models []portOverrideModel
	if d := got.ElementsAs(ctx, &models, false); d.HasError() {
		t.Fatalf("reading set: %v", d)
	}
	pm := models[0]
	if !pm.Autoneg.IsNull() || !pm.Isolation.IsNull() || !pm.Dot1X.IsNull() {
		t.Errorf("unknown attributes resolved to autoneg=%v isolation=%v dot1x=%v, want null",
			pm.Autoneg, pm.Isolation, pm.Dot1X)
	}
	if pm.Index.ValueInt64() != 1 || pm.OpMode.ValueString() != "aggregate" {
		t.Errorf("planned values changed: index=%v op_mode=%v", pm.Index, pm.OpMode)
	}
	group, _, d := util.ObjectAs[portLldpmedModel](ctx, pm.Lldpmed)
	if d.HasError() {
		t.Fatalf("lldpmed: %v", d)
	}
	if group.Enabled.ValueBool() || group.Enabled.IsNull() || !group.NotifyEnabled.IsNull() {
		t.Errorf("lldpmed = %+v, want enabled=false and notify_enabled null", group)
	}

	// Nothing unknown, nothing to do.
	known := portOverrideSetWith(t, map[string]attr.Value{"index": types.Int64Value(7)})
	if same := resolvePlannedPortOverrides(ctx, &diags, known); !same.Equal(known) {
		t.Errorf("a fully known set changed: %v", same)
	}
}
