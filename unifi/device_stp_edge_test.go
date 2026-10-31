package unifi

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func TestFrameworkToPortOverrides_StpEdgeState(t *testing.T) {
	r := &deviceResource{}

	set := portOverrideSetWith(t, map[string]attr.Value{
		"index":          types.Int64Value(8),
		"stp_edge_state": types.StringValue("enabled"),
	})
	pos, diags := r.frameworkToPortOverrides(context.Background(), set)
	if diags.HasError() {
		t.Fatalf("frameworkToPortOverrides errored: %v", diags)
	}
	if pos[0].StpEdgeState != "enabled" {
		t.Errorf("StpEdgeState = %q, want enabled", pos[0].StpEdgeState)
	}

	unset := portOverrideSetWith(t, map[string]attr.Value{"index": types.Int64Value(8)})
	pos, diags = r.frameworkToPortOverrides(context.Background(), unset)
	if diags.HasError() {
		t.Fatalf("frameworkToPortOverrides errored: %v", diags)
	}
	b, err := json.Marshal(pos[0])
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(b, &wire); err != nil {
		t.Fatal(err)
	}
	if _, ok := wire["stp_edge_state"]; ok {
		t.Errorf("unset: stp_edge_state must not be sent: %s", b)
	}
}

func TestMergePortOverridesByIndex_StpEdgeState(t *testing.T) {
	current := unifi.DevicePortOverrides{PortIDX: ptrInt64(8), Name: "nas", StpEdgeState: "auto"}

	tests := []struct {
		name     string
		declared string
		want     string
	}{
		// Not configured: keep the controller's edge mode rather than deleting
		// it by sending the port without the field.
		{name: "unset carries controller value", declared: "", want: "auto"},
		{name: "configured value is sent", declared: "enabled", want: "enabled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			declared := unifi.DevicePortOverrides{
				PortIDX:      ptrInt64(8),
				Name:         "nas",
				StpEdgeState: tt.declared,
			}
			got := mergePortOverridesByIndex(
				[]unifi.DevicePortOverrides{current},
				[]unifi.DevicePortOverrides{declared},
			)
			if got[0].StpEdgeState != tt.want {
				t.Errorf("StpEdgeState = %q, want %q", got[0].StpEdgeState, tt.want)
			}
		})
	}
}

func reconcileEdge(t *testing.T, prior types.String, api string) types.String {
	t.Helper()
	r := &deviceResource{}
	priorSet := portOverrideSetWith(t, map[string]attr.Value{
		"index":          types.Int64Value(8),
		"stp_edge_state": prior,
	})
	got, diags := r.reconcilePortOverrides(
		context.Background(),
		priorSet,
		[]unifi.DevicePortOverrides{{PortIDX: ptrInt64(8), StpEdgeState: api}},
	)
	if diags.HasError() {
		t.Fatalf("reconcilePortOverrides errored: %v", diags)
	}
	var models []portOverrideModel
	if d := got.ElementsAs(context.Background(), &models, false); d.HasError() {
		t.Fatalf("reading set: %v", d)
	}
	return models[0].StpEdgeState
}

func TestReconcilePortOverrides_StpEdgeState(t *testing.T) {
	if got := reconcileEdge(
		t,
		types.StringValue("enabled"),
		"enabled",
	); got.ValueString() != "enabled" {
		t.Errorf("configured value read back as %v, want enabled", got)
	}
	// The controller not applying the configured mode shows up as drift.
	if got := reconcileEdge(t, types.StringValue("enabled"), "auto"); got.ValueString() != "auto" {
		t.Errorf("drift not surfaced: got %v, want auto", got)
	}
	// Unset in config stays null whatever the controller holds, so an
	// unmanaged edge mode does not plan a removal.
	if got := reconcileEdge(t, types.StringNull(), "auto"); !got.IsNull() {
		t.Errorf("unset read back as %v, want null", got)
	}
}
