package unifi

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/hwtypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
)

// TestPowerSupervisorModelRoundTrip covers the model ⇄ go-unifi conversion for
// the Device Supervisor resource (#244): settings are sent as the user set them,
// power_sources are not sent (the controller resolves them) but are read back,
// and the computed consecutive_failures / id / power_sources land in the model.
func TestPowerSupervisorModelRoundTrip(t *testing.T) {
	ctx := context.Background()
	r := &powerSupervisorResource{}

	model := powerSupervisorResourceModel{
		DeviceMAC:         hwtypes.NewMACAddressValue("94:2a:6f:d6:ce:fd"),
		Enabled:           types.BoolValue(true),
		HeartbeatInterval: util.DurationValue(30, time.Second),
		SilenceThreshold:  util.DurationValue(600, time.Second),
		PowerOffDuration:  util.DurationValue(90, time.Second),
	}

	api := r.modelToPowerSupervisor(&model)
	if api.ClientMAC != "94:2a:6f:d6:ce:fd" {
		t.Errorf("ClientMAC = %q, want the device MAC", api.ClientMAC)
	}
	if !api.Enabled {
		t.Errorf("Enabled = false, want true")
	}
	if api.Settings.HeartbeatInterval != 30 || api.Settings.SilenceThreshold != 600 ||
		api.Settings.PowerOffDuration != 90 {
		t.Errorf("settings not mapped: %+v", api.Settings)
	}
	if api.PowerSources == nil {
		t.Errorf("PowerSources should be non-nil (sent empty), got nil")
	}

	// Simulate the controller's resting response and read it back.
	resp := &unifi.PowerSupervisor{
		ID:                  "000000000000000000000001",
		ClientMAC:           "94:2a:6f:d6:ce:fd",
		Enabled:             true,
		ConsecutiveFailures: 2,
		Settings: unifi.PowerSupervisorSettings{
			HeartbeatInterval: 30, SilenceThreshold: 600, PowerOffDuration: 90,
		},
		PowerSources: []unifi.PowerSupervisorSource{
			{
				ClientPsuIndex:   1,
				PowerSourceIndex: 4,
				PowerSourceMAC:   "f4:e2:c6:ad:4f:82",
				PowerSourceType:  "poe_port",
			},
		},
	}

	var out powerSupervisorResourceModel
	if d := r.powerSupervisorToModel(resp, &out, "default"); d.HasError() {
		t.Fatalf("powerSupervisorToModel: %v", d)
	}
	if out.ID.ValueString() != "000000000000000000000001" {
		t.Errorf("ID = %q", out.ID.ValueString())
	}
	if out.ConsecutiveFailures.ValueInt64() != 2 {
		t.Errorf("ConsecutiveFailures = %d, want 2", out.ConsecutiveFailures.ValueInt64())
	}
	if out.Site.ValueString() != "default" {
		t.Errorf("Site = %q, want default", out.Site.ValueString())
	}

	var sources []struct {
		ClientPsuIndex   int64  `tfsdk:"client_psu_index"`
		PowerSourceIndex int64  `tfsdk:"power_source_index"`
		PowerSourceMAC   string `tfsdk:"power_source_mac"`
		PowerSourceType  string `tfsdk:"power_source_type"`
	}
	if d := out.PowerSources.ElementsAs(ctx, &sources, false); d.HasError() {
		t.Fatalf("reading power_sources: %v", d)
	}
	if len(sources) != 1 || sources[0].PowerSourceMAC != "f4:e2:c6:ad:4f:82" ||
		sources[0].PowerSourceType != "poe_port" || sources[0].PowerSourceIndex != 4 {
		t.Errorf("power_sources not read back correctly: %+v", sources)
	}
}
