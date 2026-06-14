package unifi

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/hwtypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
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

func TestNewPowerSupervisorResource(t *testing.T) {
	r := NewPowerSupervisorResource()
	if r == nil {
		t.Fatal("NewPowerSupervisorResource() returned nil")
	}
	if _, ok := r.(fwresource.ResourceWithConfigure); !ok {
		t.Error("expected ResourceWithConfigure interface")
	}
	if _, ok := r.(fwresource.ResourceWithImportState); !ok {
		t.Error("expected ResourceWithImportState interface")
	}
	if _, ok := r.(fwresource.ResourceWithIdentity); !ok {
		t.Error("expected ResourceWithIdentity interface")
	}
	if _, ok := r.(fwresource.ResourceWithUpgradeState); !ok {
		t.Error("expected ResourceWithUpgradeState interface")
	}
}

func TestNewPowerSupervisorListResource(t *testing.T) {
	r := NewPowerSupervisorListResource()
	if r == nil {
		t.Fatal("NewPowerSupervisorListResource() returned nil")
	}
	if _, ok := r.(fwlist.ListResource); !ok {
		t.Error("expected fwlist.ListResource interface")
	}
	if _, ok := r.(fwlist.ListResourceWithConfigure); !ok {
		t.Error("expected fwlist.ListResourceWithConfigure interface")
	}
}

func Test_powerSourceAttrTypes(t *testing.T) {
	got := powerSourceAttrTypes()
	want := map[string]attr.Type{
		"client_psu_index":   types.Int64Type,
		"power_source_index": types.Int64Type,
		"power_source_mac":   types.StringType,
		"power_source_type":  types.StringType,
	}
	for k, wantType := range want {
		if gotType, ok := got[k]; !ok {
			t.Errorf("powerSourceAttrTypes() missing key %q", k)
		} else if gotType != wantType {
			t.Errorf("powerSourceAttrTypes()[%q] = %v, want %v", k, gotType, wantType)
		}
	}
}

func Test_powerSupervisorResource_Metadata(t *testing.T) {
	for _, tt := range []struct{ p, w string }{
		{"unifi", "unifi_power_supervisor"},
		{"test", "test_power_supervisor"},
	} {
		t.Run(tt.p, func(t *testing.T) {
			r := &powerSupervisorResource{}
			resp := &fwresource.MetadataResponse{}
			r.Metadata(
				context.Background(),
				fwresource.MetadataRequest{ProviderTypeName: tt.p},
				resp,
			)
			if resp.TypeName != tt.w {
				t.Errorf("TypeName = %q, want %q", resp.TypeName, tt.w)
			}
		})
	}
}

func Test_powerSupervisorResource_IdentitySchema(t *testing.T) {
	r := &powerSupervisorResource{}
	resp := &fwresource.IdentitySchemaResponse{}
	r.IdentitySchema(context.Background(), fwresource.IdentitySchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("IdentitySchema() returned errors: %v", resp.Diagnostics)
	}
	if len(resp.IdentitySchema.Attributes) == 0 {
		t.Error("IdentitySchema() returned no attributes")
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Error("IdentitySchema() missing 'id' attribute")
	}
}

func Test_powerSupervisorResource_Schema(t *testing.T) {
	r := &powerSupervisorResource{}
	resp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() returned errors: %v", resp.Diagnostics)
	}
	for _, key := range []string{
		"id", "site", "device_mac", "enabled", "heartbeat_interval",
		"silence_threshold", "power_off_duration", "consecutive_failures", "power_sources", "timeouts",
	} {
		if _, ok := resp.Schema.Attributes[key]; !ok {
			t.Errorf("Schema() missing attribute %q", key)
		}
	}
}

func Test_powerSupervisorResource_UpgradeState(t *testing.T) {
	r := &powerSupervisorResource{}
	got := r.UpgradeState(context.Background())
	if got == nil {
		t.Fatal("UpgradeState() returned nil")
	}
	if _, ok := got[0]; !ok {
		t.Error("UpgradeState() missing version 0 upgrader")
	}
}

func Test_powerSupervisorResource_Configure(t *testing.T) {
	for _, tt := range []struct {
		name string
		data any
		err  bool
	}{
		{"nil", nil, false},
		{"wrong", "wrong", true},
		{"ok", &Client{}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := &powerSupervisorResource{}
			resp := &fwresource.ConfigureResponse{}
			r.Configure(
				context.Background(),
				fwresource.ConfigureRequest{ProviderData: tt.data},
				resp,
			)
			if tt.err && !resp.Diagnostics.HasError() {
				t.Error("expected error in diagnostics")
			}
			if !tt.err && resp.Diagnostics.HasError() {
				t.Errorf("unexpected error: %v", resp.Diagnostics)
			}
		})
	}
}

func Test_powerSupervisorResource_modelToPowerSupervisor(t *testing.T) {
	r := &powerSupervisorResource{}

	t.Run("maps all settings fields", func(t *testing.T) {
		model := &powerSupervisorResourceModel{
			DeviceMAC:         hwtypes.NewMACAddressValue("aa:bb:cc:dd:ee:ff"),
			Enabled:           types.BoolValue(true),
			HeartbeatInterval: util.DurationValue(60, time.Second),
			SilenceThreshold:  util.DurationValue(300, time.Second),
			PowerOffDuration:  util.DurationValue(120, time.Second),
		}
		got := r.modelToPowerSupervisor(model)
		if got == nil {
			t.Fatal("modelToPowerSupervisor() returned nil")
		}
		if got.ClientMAC != "aa:bb:cc:dd:ee:ff" {
			t.Errorf("ClientMAC = %q, want aa:bb:cc:dd:ee:ff", got.ClientMAC)
		}
		if !got.Enabled {
			t.Error("Enabled should be true")
		}
		if got.Settings.HeartbeatInterval != 60 {
			t.Errorf("HeartbeatInterval = %d, want 60", got.Settings.HeartbeatInterval)
		}
		if got.Settings.SilenceThreshold != 300 {
			t.Errorf("SilenceThreshold = %d, want 300", got.Settings.SilenceThreshold)
		}
		if got.Settings.PowerOffDuration != 120 {
			t.Errorf("PowerOffDuration = %d, want 120", got.Settings.PowerOffDuration)
		}
		if got.PowerSources == nil {
			t.Error("PowerSources should be non-nil empty slice")
		}
	})

	t.Run("disabled supervisor", func(t *testing.T) {
		model := &powerSupervisorResourceModel{
			DeviceMAC:         hwtypes.NewMACAddressValue("11:22:33:44:55:66"),
			Enabled:           types.BoolValue(false),
			HeartbeatInterval: util.DurationValue(30, time.Second),
			SilenceThreshold:  util.DurationValue(900, time.Second),
			PowerOffDuration:  util.DurationValue(60, time.Second),
		}
		got := r.modelToPowerSupervisor(model)
		if got.Enabled {
			t.Error("Enabled should be false")
		}
		if got.ClientMAC != "11:22:33:44:55:66" {
			t.Errorf("ClientMAC = %q", got.ClientMAC)
		}
	})
}

func Test_powerSupervisorResource_powerSupervisorToModel(t *testing.T) {
	r := &powerSupervisorResource{}

	t.Run("populates all fields", func(t *testing.T) {
		supervisor := &unifi.PowerSupervisor{
			ID:                  "sup-123",
			ClientMAC:           "aa:bb:cc:dd:ee:ff",
			Enabled:             true,
			ConsecutiveFailures: 3,
			Settings: unifi.PowerSupervisorSettings{
				HeartbeatInterval: 60,
				SilenceThreshold:  900,
				PowerOffDuration:  120,
			},
			PowerSources: []unifi.PowerSupervisorSource{
				{
					ClientPsuIndex:   0,
					PowerSourceIndex: 2,
					PowerSourceMAC:   "de:ad:be:ef:00:01",
					PowerSourceType:  "poe_port",
				},
			},
		}
		var model powerSupervisorResourceModel
		diags := r.powerSupervisorToModel(supervisor, &model, "site1")
		if diags.HasError() {
			t.Fatalf("powerSupervisorToModel() errors: %v", diags)
		}
		if model.ID.ValueString() != "sup-123" {
			t.Errorf("ID = %q, want sup-123", model.ID.ValueString())
		}
		if model.Site.ValueString() != "site1" {
			t.Errorf("Site = %q, want site1", model.Site.ValueString())
		}
		if model.DeviceMAC.ValueString() != "aa:bb:cc:dd:ee:ff" {
			t.Errorf("DeviceMAC = %q", model.DeviceMAC.ValueString())
		}
		if !model.Enabled.ValueBool() {
			t.Error("Enabled should be true")
		}
		if model.ConsecutiveFailures.ValueInt64() != 3 {
			t.Errorf("ConsecutiveFailures = %d, want 3", model.ConsecutiveFailures.ValueInt64())
		}
		if model.PowerSources.IsNull() || model.PowerSources.IsUnknown() {
			t.Error("PowerSources should be a non-null list")
		}
		if len(model.PowerSources.Elements()) != 1 {
			t.Errorf("PowerSources len = %d, want 1", len(model.PowerSources.Elements()))
		}
	})

	t.Run("empty power sources", func(t *testing.T) {
		supervisor := &unifi.PowerSupervisor{
			ID:           "sup-456",
			ClientMAC:    "11:22:33:44:55:66",
			Enabled:      false,
			PowerSources: []unifi.PowerSupervisorSource{},
		}
		var model powerSupervisorResourceModel
		diags := r.powerSupervisorToModel(supervisor, &model, "default")
		if diags.HasError() {
			t.Fatalf("powerSupervisorToModel() errors: %v", diags)
		}
		if len(model.PowerSources.Elements()) != 0 {
			t.Errorf(
				"PowerSources should be empty, got %d elements",
				len(model.PowerSources.Elements()),
			)
		}
	})
}

func Test_powerSupervisorResource_ListResourceConfigSchema(t *testing.T) {
	r := &powerSupervisorResource{}
	resp := &fwlist.ListResourceSchemaResponse{}
	r.ListResourceConfigSchema(context.Background(), fwlist.ListResourceSchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ListResourceConfigSchema() returned errors: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["site"]; !ok {
		t.Error("ListResourceConfigSchema() missing 'site' attribute")
	}
}
