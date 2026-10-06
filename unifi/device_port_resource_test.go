package unifi

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/hwtypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// Tests for unifi_device_port (device_port_resource.go). Scope mirrors
// device_resource_test.go's own convention for device-level code:
// pure-function and ImportState unit tests only, no TestAcc - a specific
// device/port can't be simulated by the demo controller used for
// acceptance tests.

func TestParseDevicePortImportID(t *testing.T) {
	tests := []struct {
		name      string
		importID  string
		wantMAC   string
		wantIndex int64
		wantErr   bool
	}{
		{
			name:      "colon-separated MAC, slash-separated index",
			importID:  "a8:9c:6c:08:ea:3b/2",
			wantMAC:   "a8:9c:6c:08:ea:3b",
			wantIndex: 2,
		},
		{
			name:      "dash-separated MAC is normalized via cleanMAC",
			importID:  "A8-9C-6C-08-EA-3B/9",
			wantMAC:   "a8:9c:6c:08:ea:3b",
			wantIndex: 9,
		},
		{
			name:     "no slash at all",
			importID: "a8:9c:6c:08:ea:3b2",
			wantErr:  true,
		},
		{
			name:     "trailing slash, empty index",
			importID: "a8:9c:6c:08:ea:3b/",
			wantErr:  true,
		},
		{
			name:     "leading slash, empty mac",
			importID: "/2",
			wantErr:  true,
		},
		{
			name:     "non-numeric index",
			importID: "a8:9c:6c:08:ea:3b/nine",
			wantErr:  true,
		},
		{
			name:     "colon instead of slash (old format) is rejected",
			importID: "a8:9c:6c:08:ea:3b:2",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mac, index, err := parseDevicePortImportID(tt.importID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseDevicePortImportID(%q) error = %v, wantErr %v",
					tt.importID, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if mac != tt.wantMAC {
				t.Errorf("mac = %q, want %q", mac, tt.wantMAC)
			}
			if index != tt.wantIndex {
				t.Errorf("index = %d, want %d", index, tt.wantIndex)
			}
		})
	}
}

// TestModelToAPIPortOverride mirrors a live override this resource was
// verified against (switch.tf's port 1: see conversation history / import test).
func TestModelToAPIPortOverride(t *testing.T) {
	ctx := context.Background()

	model := devicePortResourceModel{
		Index:           types.Int64Value(1),
		Name:            types.StringValue("Port 1"),
		NativeNetworkID: types.StringValue("681150205d9a574fb4d5ca60"),
		Forward:         types.StringValue("customize"),
		TaggedVLANMgmt:  types.StringValue("auto"),
		PoeMode:         types.StringValue("off"),
		Autoneg:         types.BoolValue(true),
		// OpMode left null (Go zero value): op_mode has no schema Default, so
		// this is what an omitted op_mode actually looks like at this layer,
		// and it's only ever written when explicitly non-default (see
		// modelToAPIPortOverride's comment - UDM gateways reject it on
		// update otherwise).
	}

	po, diags := modelToAPIPortOverride(ctx, model, unifi.DevicePortOverrides{})
	if diags.HasError() {
		t.Fatalf("modelToAPIPortOverride returned errors: %v", diags)
	}

	if po.PortIDX == nil || *po.PortIDX != 1 {
		t.Errorf("PortIDX = %v, want 1", po.PortIDX)
	}
	if po.Name != "Port 1" {
		t.Errorf("Name = %q, want %q", po.Name, "Port 1")
	}
	if po.NATiveNetworkID != "681150205d9a574fb4d5ca60" {
		t.Errorf("NATiveNetworkID = %q, want %q", po.NATiveNetworkID, "681150205d9a574fb4d5ca60")
	}
	if po.Forward != "customize" {
		t.Errorf("Forward = %q, want %q", po.Forward, "customize")
	}
	if po.TaggedVLANMgmt != "auto" {
		t.Errorf("TaggedVLANMgmt = %q, want %q", po.TaggedVLANMgmt, "auto")
	}
	if po.PoeMode != "off" {
		t.Errorf("PoeMode = %q, want %q", po.PoeMode, "off")
	}
	if !po.Autoneg {
		t.Errorf("Autoneg = false, want true")
	}
	if po.OpMode != "" {
		t.Errorf("OpMode = %q, want empty (default \"switch\" is never written)", po.OpMode)
	}
}

// TestApplyAPIPortOverrideToDevicePortModel is the reverse of
// TestModelToAPIPortOverride: it feeds in the shape actually observed from
// a live controller during testing (see conversation history) and checks
// the model fields it populates.
func TestApplyAPIPortOverrideToDevicePortModel(t *testing.T) {
	po := unifi.DevicePortOverrides{
		PortIDX:            ptrInt64(1),
		Name:               "Port 1",
		NATiveNetworkID:    "681150205d9a574fb4d5ca60",
		Forward:            "customize",
		TaggedVLANMgmt:     "auto",
		PoeMode:            "off",
		SettingPreference:  "auto",
		Autoneg:            true,
		FlowControlEnabled: true,
		StpPortMode:        true,
	}

	var model devicePortResourceModel
	diags := applyAPIPortOverrideToDevicePortModel(&model, po)
	if diags.HasError() {
		t.Fatalf("applyAPIPortOverrideToDevicePortModel returned errors: %v", diags)
	}

	if model.Index.ValueInt64() != 1 {
		t.Errorf("Index = %v, want 1", model.Index)
	}
	if model.Name.ValueString() != "Port 1" {
		t.Errorf("Name = %q, want %q", model.Name.ValueString(), "Port 1")
	}
	if model.NativeNetworkID.ValueString() != "681150205d9a574fb4d5ca60" {
		t.Errorf("NativeNetworkID = %q, want %q",
			model.NativeNetworkID.ValueString(), "681150205d9a574fb4d5ca60")
	}
	if model.Forward.ValueString() != "customize" {
		t.Errorf("Forward = %q, want %q", model.Forward.ValueString(), "customize")
	}
	if model.TaggedVLANMgmt.ValueString() != "auto" {
		t.Errorf("TaggedVLANMgmt = %q, want %q", model.TaggedVLANMgmt.ValueString(), "auto")
	}
	if model.PoeMode.ValueString() != "off" {
		t.Errorf("PoeMode = %q, want %q", model.PoeMode.ValueString(), "off")
	}
	if model.SettingPreference.ValueString() != "auto" {
		t.Errorf("SettingPreference = %q, want %q", model.SettingPreference.ValueString(), "auto")
	}
	if !model.Autoneg.ValueBool() {
		t.Errorf("Autoneg = false, want true")
	}
	// po.OpMode is "" (never set above): must read back as the literal
	// "switch", not null - see TestApplyAPIPortOverrideToDevicePortModel_
	// OpModeNeverNull for why.
	if model.OpMode.ValueString() != "switch" {
		t.Errorf("OpMode = %q, want %q", model.OpMode.ValueString(), "switch")
	}
}

// TestApplyAPIPortOverrideToDevicePortModel_OpModeNeverNull guards against a
// plan/apply consistency bug: apiPortOverrideToModel (device_resource.go)
// represents op_mode "" (the API's default/switch mode) as null, which is
// fine for the SetNestedBlock resource (reconcilePortOverrides only resolves
// fields still unknown, so an explicitly-configured value is never
// clobbered) but not for this resource - applyAPIPortOverrideToDevicePortModel
// overwrites model.OpMode unconditionally, and op_mode has no schema
// Default to reconcile a null result against. A user who explicitly writes
// `op_mode = "switch"` on a port that reports "" would otherwise see state
// go null while config says "switch" - Terraform's post-apply consistency
// check treats that as a provider bug and fails the apply.
func TestApplyAPIPortOverrideToDevicePortModel_OpModeNeverNull(t *testing.T) {
	tests := []struct {
		name       string
		apiOpMode  string
		wantOpMode string
	}{
		{name: "empty (default) reads back as literal switch", apiOpMode: "", wantOpMode: "switch"},
		{name: "mirror reads back unchanged", apiOpMode: "mirror", wantOpMode: "mirror"},
		{name: "aggregate reads back unchanged", apiOpMode: "aggregate", wantOpMode: "aggregate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			po := unifi.DevicePortOverrides{
				PortIDX: ptrInt64(1),
				OpMode:  tt.apiOpMode,
			}
			var model devicePortResourceModel
			diags := applyAPIPortOverrideToDevicePortModel(&model, po)
			if diags.HasError() {
				t.Fatalf("applyAPIPortOverrideToDevicePortModel returned errors: %v", diags)
			}
			if model.OpMode.IsNull() {
				t.Fatalf("OpMode is null, want %q", tt.wantOpMode)
			}
			if model.OpMode.ValueString() != tt.wantOpMode {
				t.Errorf("OpMode = %q, want %q", model.OpMode.ValueString(), tt.wantOpMode)
			}
		})
	}
}

// TestDevicePortOverrideRoundTrip chains both conversions: an API override
// applied to a model, then converted back to an API override, must carry the
// same declared field values (the round trip that matters for import - what
// Read populates must be re-sendable by Create/Update without drift).
func TestDevicePortOverrideRoundTrip(t *testing.T) {
	ctx := context.Background()

	original := unifi.DevicePortOverrides{
		PortIDX:         ptrInt64(3),
		Name:            "Uplink",
		NATiveNetworkID: "net-abc",
		Forward:         "all",
		TaggedVLANMgmt:  "auto",
		PoeMode:         "auto",
		Autoneg:         true,
	}

	var model devicePortResourceModel
	if diags := applyAPIPortOverrideToDevicePortModel(&model, original); diags.HasError() {
		t.Fatalf("applyAPIPortOverrideToDevicePortModel returned errors: %v", diags)
	}

	roundTripped, diags := modelToAPIPortOverride(ctx, model, unifi.DevicePortOverrides{})
	if diags.HasError() {
		t.Fatalf("modelToAPIPortOverride returned errors: %v", diags)
	}

	if roundTripped.PortIDX == nil || *roundTripped.PortIDX != *original.PortIDX {
		t.Errorf("PortIDX = %v, want %v", roundTripped.PortIDX, original.PortIDX)
	}
	if roundTripped.Name != original.Name {
		t.Errorf("Name = %q, want %q", roundTripped.Name, original.Name)
	}
	if roundTripped.NATiveNetworkID != original.NATiveNetworkID {
		t.Errorf("NATiveNetworkID = %q, want %q",
			roundTripped.NATiveNetworkID, original.NATiveNetworkID)
	}
	if roundTripped.Forward != original.Forward {
		t.Errorf("Forward = %q, want %q", roundTripped.Forward, original.Forward)
	}
	if roundTripped.TaggedVLANMgmt != original.TaggedVLANMgmt {
		t.Errorf(
			"TaggedVLANMgmt = %q, want %q",
			roundTripped.TaggedVLANMgmt,
			original.TaggedVLANMgmt,
		)
	}
	if roundTripped.PoeMode != original.PoeMode {
		t.Errorf("PoeMode = %q, want %q", roundTripped.PoeMode, original.PoeMode)
	}
	if roundTripped.Autoneg != original.Autoneg {
		t.Errorf("Autoneg = %v, want %v", roundTripped.Autoneg, original.Autoneg)
	}
	// original.OpMode is "" (the switch/default mode): applyAPIPortOverrideToDevicePortModel
	// reads it back as the literal "switch" (not null), and since that's an
	// explicit, known, non-default-triggering value, modelToAPIPortOverride
	// must still round-trip it back to "" rather than sending "switch" or
	// treating it as a revert.
	if roundTripped.OpMode != original.OpMode {
		t.Errorf("OpMode = %q, want %q", roundTripped.OpMode, original.OpMode)
	}
}

// TestModelToAPIPortOverride_UnsetBoolsFallBackToCurrent guards against
// re-introducing the bug where an Optional+Computed bool left unset in
// config (Null or, absent a UseStateForUnknown plan modifier, Unknown) was
// sent to the controller as false via ValueBool()'s zero-value fallback,
// silently disabling settings like autoneg/full_duplex on adoption of an
// already-customized port.
func TestModelToAPIPortOverride_UnsetBoolsFallBackToCurrent(t *testing.T) {
	ctx := context.Background()

	current := unifi.DevicePortOverrides{
		PortIDX:               ptrInt64(1),
		Autoneg:               true,
		FullDuplex:            true,
		FlowControlEnabled:    true,
		Isolation:             false,
		LldpmedEnabled:        true,
		StormctrlUcastEnabled: true,
		StpPortMode:           true,
	}

	tests := []struct {
		name  string
		model devicePortResourceModel
	}{
		{
			name: "null bools (never configured)",
			model: devicePortResourceModel{
				Index: types.Int64Value(1),
			},
		},
		{
			name: "unknown bools (no UseStateForUnknown plan modifier)",
			model: devicePortResourceModel{
				Index:                 types.Int64Value(1),
				Autoneg:               types.BoolUnknown(),
				FullDuplex:            types.BoolUnknown(),
				FlowControlEnabled:    types.BoolUnknown(),
				Isolation:             types.BoolUnknown(),
				LldpmedEnabled:        types.BoolUnknown(),
				StormctrlUcastEnabled: types.BoolUnknown(),
				StpPortMode:           types.BoolUnknown(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			po, diags := modelToAPIPortOverride(ctx, tt.model, current)
			if diags.HasError() {
				t.Fatalf("modelToAPIPortOverride returned errors: %v", diags)
			}

			if po.Autoneg != current.Autoneg {
				t.Errorf("Autoneg = %v, want current value %v", po.Autoneg, current.Autoneg)
			}
			if po.FullDuplex != current.FullDuplex {
				t.Errorf(
					"FullDuplex = %v, want current value %v",
					po.FullDuplex,
					current.FullDuplex,
				)
			}
			if po.FlowControlEnabled != current.FlowControlEnabled {
				t.Errorf("FlowControlEnabled = %v, want current value %v",
					po.FlowControlEnabled, current.FlowControlEnabled)
			}
			if po.LldpmedEnabled != current.LldpmedEnabled {
				t.Errorf("LldpmedEnabled = %v, want current value %v",
					po.LldpmedEnabled, current.LldpmedEnabled)
			}
			if po.StormctrlUcastEnabled != current.StormctrlUcastEnabled {
				t.Errorf("StormctrlUcastEnabled = %v, want current value %v",
					po.StormctrlUcastEnabled, current.StormctrlUcastEnabled)
			}
			if po.StpPortMode != current.StpPortMode {
				t.Errorf(
					"StpPortMode = %v, want current value %v",
					po.StpPortMode,
					current.StpPortMode,
				)
			}
		})
	}
}

// TestModelToAPIPortOverride_ExplicitBoolOverridesCurrent ensures a config
// value that is explicitly known (not null/unknown) always wins over the
// controller's current value, even when it disagrees.
func TestModelToAPIPortOverride_ExplicitBoolOverridesCurrent(t *testing.T) {
	ctx := context.Background()

	current := unifi.DevicePortOverrides{
		PortIDX: ptrInt64(1),
		Autoneg: true,
	}
	model := devicePortResourceModel{
		Index:   types.Int64Value(1),
		Autoneg: types.BoolValue(false),
	}

	po, diags := modelToAPIPortOverride(ctx, model, current)
	if diags.HasError() {
		t.Fatalf("modelToAPIPortOverride returned errors: %v", diags)
	}
	if po.Autoneg != false {
		t.Errorf("Autoneg = %v, want false (explicit config value)", po.Autoneg)
	}
}

// TestModelToAPIPortOverride_OpModeRevertToSwitch guards against two bugs:
//  1. declaring op_mode = "switch" to revert a port currently in
//     mirror/aggregate mode was silently dropped (encoded as "" like the
//     never-configured case), so carryUnwritableFields kept re-carrying the
//     controller's non-default op_mode forward forever.
//  2. op_mode having a schema Default made an omitted op_mode indistinguishable
//     from an explicit "switch", so applying an unrelated attribute change on
//     an imported mirror/aggregate port - without ever mentioning op_mode -
//     would hit the same revert branch and silently reset it. op_mode has no
//     Default (see the schema comment), so Null/Unknown here is what that
//     omission actually looks like at this layer.
func TestModelToAPIPortOverride_OpModeRevertToSwitch(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		opMode     types.String
		currentOM  string
		wantOpMode string
	}{
		{
			name:       "revert from mirror to switch is sent explicitly",
			opMode:     types.StringValue("switch"),
			currentOM:  "mirror",
			wantOpMode: "switch",
		},
		{
			name:       "revert from aggregate to switch is sent explicitly",
			opMode:     types.StringValue("switch"),
			currentOM:  "aggregate",
			wantOpMode: "switch",
		},
		{
			name:       "declaring switch when already switch stays omitted (UDM safety, #213)",
			opMode:     types.StringValue("switch"),
			currentOM:  "",
			wantOpMode: "",
		},
		{
			name:       "declaring mirror is always sent regardless of current",
			opMode:     types.StringValue("mirror"),
			currentOM:  "",
			wantOpMode: "mirror",
		},
		{
			name:       "omitted (null) op_mode does not revert a mirror port",
			opMode:     types.StringNull(),
			currentOM:  "mirror",
			wantOpMode: "",
		},
		{
			name:       "omitted (unknown) op_mode does not revert an aggregate port",
			opMode:     types.StringUnknown(),
			currentOM:  "aggregate",
			wantOpMode: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := devicePortResourceModel{
				Index:  types.Int64Value(1),
				OpMode: tt.opMode,
			}
			current := unifi.DevicePortOverrides{
				PortIDX: ptrInt64(1),
				OpMode:  tt.currentOM,
			}

			po, diags := modelToAPIPortOverride(ctx, model, current)
			if diags.HasError() {
				t.Fatalf("modelToAPIPortOverride returned errors: %v", diags)
			}
			if po.OpMode != tt.wantOpMode {
				t.Errorf("OpMode = %q, want %q", po.OpMode, tt.wantOpMode)
			}
		})
	}
}

// TestModelToAPIPortOverride_QOSProfileCarriedForward guards against the bug
// where qos_profile - not modeled by this resource's schema at all - was
// unconditionally dropped on every upsert (desiredPO always built with a nil
// QOSProfile, and mergePortOverridesByIndex fully replaces the matched entry
// with it), silently clearing a port's assigned QoS Profile on first apply.
func TestModelToAPIPortOverride_QOSProfileCarriedForward(t *testing.T) {
	ctx := context.Background()

	current := unifi.DevicePortOverrides{
		PortIDX:    ptrInt64(1),
		QOSProfile: &unifi.DeviceQOSProfile{QOSProfileMode: "custom"},
	}
	model := devicePortResourceModel{
		Index: types.Int64Value(1),
		Name:  types.StringValue("Port 1"),
	}

	po, diags := modelToAPIPortOverride(ctx, model, current)
	if diags.HasError() {
		t.Fatalf("modelToAPIPortOverride returned errors: %v", diags)
	}
	if po.QOSProfile != current.QOSProfile {
		t.Errorf(
			"QOSProfile = %v, want the current controller value %v",
			po.QOSProfile,
			current.QOSProfile,
		)
	}
}

// TestModelToAPIPortOverride_TaggedNetworkIDsSent guards against a field
// that was populated on Read (applyAPIPortOverrideToDevicePortModel) but
// never sent on write: modelToAPIPortOverride had no handling for
// tagged_networkconf_ids at all, so a user-declared value was silently
// never applied to the controller, and the resource would show a
// perpetual diff (state never able to match a non-empty config value).
func TestModelToAPIPortOverride_TaggedNetworkIDsSent(t *testing.T) {
	ctx := context.Background()

	taggedSet, diags := types.SetValue(
		types.StringType,
		[]attr.Value{types.StringValue("net-a"), types.StringValue("net-b")},
	)
	if diags.HasError() {
		t.Fatalf("building tagged set: %v", diags)
	}
	model := devicePortResourceModel{
		Index:            types.Int64Value(1),
		TaggedNetworkIDs: taggedSet,
	}

	po, diags := modelToAPIPortOverride(ctx, model, unifi.DevicePortOverrides{})
	if diags.HasError() {
		t.Fatalf("modelToAPIPortOverride returned errors: %v", diags)
	}

	got := make(map[string]bool, len(po.TaggedNetworkIDs))
	for _, id := range po.TaggedNetworkIDs {
		got[id] = true
	}
	if !got["net-a"] || !got["net-b"] || len(got) != 2 {
		t.Errorf("TaggedNetworkIDs = %v, want [net-a net-b]", po.TaggedNetworkIDs)
	}
}

// TestClientLockDevice_SerializesSameMAC verifies lockDevice actually
// excludes concurrent holders for the same key, and TestClientLockDevice_
// DoesNotSerializeDifferentMACs verifies it doesn't over-serialize unrelated
// devices. Regression coverage for the concurrent-upsert lost-update bug:
// two unifi_device_port resources for the same device applied in parallel
// could both read the device before either wrote back, so whichever
// UpdateDevice landed second silently discarded the other's change.
func TestClientLockDevice_SerializesSameMAC(t *testing.T) {
	c := &Client{}

	unlock := c.lockDevice("aa:bb:cc:dd:ee:ff")

	acquired := make(chan struct{})
	go func() {
		unlock2 := c.lockDevice("aa:bb:cc:dd:ee:ff")
		close(acquired)
		unlock2()
	}()

	select {
	case <-acquired:
		t.Fatal("second lockDevice for the same MAC acquired while the first still held it")
	case <-time.After(50 * time.Millisecond):
	}

	unlock()

	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("second lockDevice never acquired after the first released")
	}
}

func TestClientLockDevice_DoesNotSerializeDifferentMACs(t *testing.T) {
	c := &Client{}

	unlock := c.lockDevice("aa:bb:cc:dd:ee:ff")
	defer unlock()

	acquired := make(chan struct{})
	go func() {
		unlock2 := c.lockDevice("11:22:33:44:55:66")
		close(acquired)
		unlock2()
	}()

	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("lockDevice for a different MAC blocked on an unrelated device's lock")
	}
}

// --- ImportState (framework-harness) tests, mirroring
// firewall_zone_resource_test.go's newFirewallZoneImportResponse pattern. ---

func newDevicePortImportResponse(
	ctx context.Context,
	r *devicePortResource,
) *fwresource.ImportStateResponse {
	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)

	return &fwresource.ImportStateResponse{
		State: tfsdk.State{
			Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
			Schema: schemaResp.Schema,
		},
	}
}

func TestDevicePortImportStateValid(t *testing.T) {
	ctx := context.Background()
	r := &devicePortResource{}
	resp := newDevicePortImportResponse(ctx, r)

	r.ImportState(ctx, fwresource.ImportStateRequest{ID: "a8:9c:6c:08:ea:3b/1"}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState returned diagnostics: %v", resp.Diagnostics)
	}

	// device_mac uses the custom hwtypes.MACAddressType, not a plain string.
	var mac hwtypes.MACAddress
	var id types.String
	var index types.Int64
	if diags := resp.State.GetAttribute(ctx, path.Root("device_mac"), &mac); diags.HasError() {
		t.Fatalf("reading device_mac: %v", diags)
	}
	if diags := resp.State.GetAttribute(ctx, path.Root("index"), &index); diags.HasError() {
		t.Fatalf("reading index: %v", diags)
	}
	if diags := resp.State.GetAttribute(ctx, path.Root("id"), &id); diags.HasError() {
		t.Fatalf("reading id: %v", diags)
	}

	if mac.ValueString() != "a8:9c:6c:08:ea:3b" {
		t.Errorf("device_mac = %q, want %q", mac.ValueString(), "a8:9c:6c:08:ea:3b")
	}
	if index.ValueInt64() != 1 {
		t.Errorf("index = %v, want 1", index.ValueInt64())
	}
	if id.ValueString() != "a8:9c:6c:08:ea:3b/1" {
		t.Errorf("id = %q, want %q", id.ValueString(), "a8:9c:6c:08:ea:3b/1")
	}
}

func TestDevicePortImportStateInvalid(t *testing.T) {
	ctx := context.Background()
	r := &devicePortResource{}
	resp := newDevicePortImportResponse(ctx, r)

	r.ImportState(ctx, fwresource.ImportStateRequest{ID: "not-a-valid-id"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("ImportState with an invalid ID: want diagnostics, got none")
	}
}
