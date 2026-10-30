package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestSettingSectionChanged guards #580: a section whose only planned
// differences are unknown Optional+Computed leaves is not rewritten, since
// rewriting mgmt makes the controller regenerate its API token.
func TestSettingSectionChanged(t *testing.T) {
	sshType := map[string]attr.Type{"enabled": types.BoolType}
	mgmtType := map[string]attr.Type{
		"wifiman_enabled": types.BoolType,
		"ssh":             types.ObjectType{AttrTypes: sshType},
		"hour":            types.Int64Type,
	}
	obj := func(wifiman, hour attr.Value, ssh attr.Value) types.Object {
		return types.ObjectValueMust(mgmtType, map[string]attr.Value{
			"wifiman_enabled": wifiman, "ssh": ssh, "hour": hour,
		})
	}
	sshNull := types.ObjectNull(sshType)
	state := obj(types.BoolNull(), types.Int64Value(4), sshNull)

	cases := []struct {
		name string
		plan types.Object
		want bool
	}{
		{"identical", obj(types.BoolNull(), types.Int64Value(4), sshNull), false},
		{
			"only unknown leaves",
			obj(types.BoolUnknown(), types.Int64Value(4), types.ObjectUnknown(sshType)),
			false,
		},
		{"known leaf changed", obj(types.BoolUnknown(), types.Int64Value(5), sshNull), true},
		{
			"nested leaf set",
			obj(types.BoolNull(), types.Int64Value(4),
				types.ObjectValueMust(sshType, map[string]attr.Value{"enabled": types.BoolValue(true)})),
			true,
		},
	}
	for _, c := range cases {
		if got := settingSectionChanged(c.plan, state); got != c.want {
			t.Errorf("%s: settingSectionChanged = %v, want %v", c.name, got, c.want)
		}
	}
	if !settingSectionChanged(state, types.ObjectNull(mgmtType)) {
		t.Error("a section absent from state must be written")
	}
}
