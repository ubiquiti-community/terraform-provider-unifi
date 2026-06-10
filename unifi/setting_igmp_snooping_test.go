package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi/settings"
)

// TestIgmpSnoopingModelMerge guards #164: the site-level igmp_snooping setting
// exposes only enabled + network_ids, and the model->setting conversion must
// overlay those onto the current remote setting so advanced querier/flood
// fields configured in the UI are preserved across an update.
func TestIgmpSnoopingModelMerge(t *testing.T) {
	ctx := context.Background()
	r := &settingResource{}
	var diags diag.Diagnostics

	// Current remote setting with advanced fields that must survive.
	base := &settings.IgmpSnooping{
		Enabled:             false,
		QuerierMode:         "CUSTOM",
		QuerierSwitches:     []string{"aa:bb:cc:dd:ee:ff"},
		FloodKnownProtocols: true,
	}
	nids, d := types.ListValueFrom(ctx, types.StringType, []string{"net-1", "net-2"})
	if d.HasError() {
		t.Fatalf("building network_ids: %v", d)
	}
	model := &settingIgmpSnoopingModel{
		Enabled:    types.BoolValue(true),
		NetworkIDs: nids,
	}

	out := r.igmpSnoopingModelToSetting(ctx, model, base, &diags)
	if diags.HasError() {
		t.Fatalf("igmpSnoopingModelToSetting: %v", diags)
	}
	if !out.Enabled {
		t.Error("Enabled not applied from model")
	}
	if len(out.NetworkIDs) != 2 || out.NetworkIDs[0] != "net-1" {
		t.Errorf("NetworkIDs = %v, want [net-1 net-2]", out.NetworkIDs)
	}
	// Advanced fields must be preserved from base (not dropped).
	if out.QuerierMode != "CUSTOM" || len(out.QuerierSwitches) != 1 || !out.FloodKnownProtocols {
		t.Errorf("advanced fields not preserved: querier_mode=%q querier_switches=%v flood=%v",
			out.QuerierMode, out.QuerierSwitches, out.FloodKnownProtocols)
	}

	// Read-back conversion.
	m := r.igmpSnoopingSettingToModel(ctx, out, &diags)
	if diags.HasError() {
		t.Fatalf("igmpSnoopingSettingToModel: %v", diags)
	}
	if !m.Enabled.ValueBool() {
		t.Error("model Enabled = false, want true")
	}
	var ids []string
	if d := m.NetworkIDs.ElementsAs(ctx, &ids, false); d.HasError() {
		t.Fatalf("reading model network_ids: %v", d)
	}
	if len(ids) != 2 {
		t.Errorf("model network_ids = %v, want 2", ids)
	}
}
