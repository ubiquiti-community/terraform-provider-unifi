package unifi

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// A LAG as the controller reports it: the lead port carries op_mode
// "aggregate", its members and the number the controller gave the group.
func lagPort(idx, lag int64, name string) unifi.DevicePortOverrides {
	return unifi.DevicePortOverrides{
		PortIDX:          ptrInt64(idx),
		Name:             name,
		OpMode:           "aggregate",
		Forward:          "customize",
		NATiveNetworkID:  "0a1b2c3d-1111-4222-8333-444455556666",
		AggregateMembers: []int64{idx, idx + 1},
		LagIDX:           ptrInt64(lag),
	}
}

// Renaming one port of a switch sends the whole port_overrides array. The LAG
// ports in it must go out exactly as the controller holds them, lag_idx
// included: the schema has no attribute for it, so the number can only come
// from the controller.
func TestMergePortOverridesByIndex_changeElsewhereKeepsLAG(t *testing.T) {
	r := &deviceResource{}
	current := []unifi.DevicePortOverrides{
		lagPort(1, 1, "node-a LACP"),
		lagPort(3, 2, "node-b LACP"),
		{
			PortIDX:         ptrInt64(7),
			Name:            "node-c",
			Forward:         "customize",
			NATiveNetworkID: "0a1b2c3d-1111-4222-8333-444455556666",
		},
	}

	block := func(idx int64, name string, lag bool) attr.Value {
		over := map[string]attr.Value{
			"index":                 types.Int64Value(idx),
			"name":                  types.StringValue(name),
			"forward":               types.StringValue("customize"),
			"native_networkconf_id": types.StringValue("0a1b2c3d-1111-4222-8333-444455556666"),
			"op_mode":               types.StringValue("switch"),
		}
		if lag {
			over["op_mode"] = types.StringValue("aggregate")
			over["aggregate_members"] = types.ListValueMust(types.Int64Type, []attr.Value{
				types.Int64Value(idx), types.Int64Value(idx + 1),
			})
		}
		return portOverrideSetWith(t, over).Elements()[0]
	}
	set := types.SetValueMust(
		types.ObjectType{AttrTypes: portOverrideAttrTypes()},
		[]attr.Value{
			block(1, "node-a LACP", true),
			block(3, "node-b LACP", true),
			block(7, "node-c renamed", false),
		},
	)
	declared, diags := r.frameworkToPortOverrides(context.Background(), set)
	if diags.HasError() {
		t.Fatalf("frameworkToPortOverrides errored: %v", diags)
	}

	got := mergePortOverridesByIndex(current, declared)
	if len(got) != 3 {
		t.Fatalf("merged %d ports, want 3", len(got))
	}
	for i := range 2 {
		if !reflect.DeepEqual(got[i], current[i]) {
			t.Errorf("LAG port %d = %+v, want the controller's entry %+v", i, got[i], current[i])
		}
	}
	if got[2].Name != "node-c renamed" {
		t.Errorf("renamed port = %q", got[2].Name)
	}
	if got[2].LagIDX != nil {
		t.Errorf("plain port gained lag_idx %d", *got[2].LagIDX)
	}
}
