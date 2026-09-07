package unifi

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/hwtypes"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// TestDeviceUpgradeState_v2NestsPrefixedGroups guards the v2 -> v3 schema
// upgrade: flat led_*/stp_*/lcm_* attributes and the port_override feature
// groups move into nested objects, and every other attribute passes through.
func TestDeviceUpgradeState_v2NestsPrefixedGroups(t *testing.T) {
	ctx := context.Background()
	r := &deviceResource{}

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	prior := []byte(`{
		"id": "dev-1", "site": "default", "mac": "00:11:22:33:44:55", "name": "sw",
		"led_override": "on", "led_override_color": "#00ff00", "led_override_color_brightness": 20,
		"stp_version": "rstp", "stp_priority": 4096,
		"lcm_brightness": 40, "lcm_brightness_override": true,
		"lcm_idle_timeout": "10m0s", "lcm_idle_timeout_override": false,
		"lcm_night_mode_begins": "22:00", "lcm_night_mode_ends": "06:00",
		"port_override": [{
			"index": 3, "name": "cam",
			"dot1x_ctrl": "auto", "dot1x_idle_timeout": "5m0s",
			"egress_rate_limit_kbps": 1000, "egress_rate_limit_kbps_enabled": true,
			"lldpmed_enabled": true, "lldpmed_notify_enabled": false,
			"port_security_enabled": true, "port_security_mac_address": ["aa:bb:cc:dd:ee:ff"],
			"stormctrl_type": "level",
			"stormctrl_bcast_enabled": true, "stormctrl_bcast_level": 50, "stormctrl_bcast_rate": null,
			"stormctrl_mcast_enabled": false, "stormctrl_ucast_enabled": false
		}]
	}`)

	up, ok := r.UpgradeState(ctx)[2]
	if !ok {
		t.Fatal("no upgrader registered for schema version 2")
	}
	resp := &fwresource.UpgradeStateResponse{}
	up.StateUpgrader(ctx, fwresource.UpgradeStateRequest{
		RawState: &tfprotov6.RawState{JSON: prior},
	}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade failed: %v", resp.Diagnostics)
	}
	val, err := resp.DynamicValue.Unmarshal(schemaType)
	if err != nil {
		t.Fatalf("unmarshal upgraded value: %v", err)
	}

	var root map[string]tftypes.Value
	if err := val.As(&root); err != nil {
		t.Fatalf("as object: %v", err)
	}
	obj := func(v tftypes.Value, name string) map[string]tftypes.Value {
		t.Helper()
		var m map[string]tftypes.Value
		if err := v.As(&m); err != nil {
			t.Fatalf("%s: as object: %v (value %v)", name, err, v)
		}
		return m
	}
	str := func(v tftypes.Value, name, want string) {
		t.Helper()
		var s string
		if err := v.As(&s); err != nil || s != want {
			t.Errorf("%s = %v (%v), want %q", name, v, err, want)
		}
	}
	num := func(v tftypes.Value, name string, want int64) {
		t.Helper()
		var f big.Float
		if err := v.As(&f); err != nil {
			t.Errorf("%s = %v (%v), want %d", name, v, err, want)
			return
		}
		if n, _ := f.Int64(); n != want {
			t.Errorf("%s = %d, want %d", name, n, want)
		}
	}
	boolean := func(v tftypes.Value, name string, want bool) {
		t.Helper()
		var b bool
		if err := v.As(&b); err != nil || b != want {
			t.Errorf("%s = %v (%v), want %v", name, v, err, want)
		}
	}

	str(root["name"], "name", "sw")
	for _, flat := range []string{"led_override", "stp_version", "lcm_brightness"} {
		if _, exists := root[flat]; exists {
			t.Errorf("flat attribute %q survived the upgrade", flat)
		}
	}

	led := obj(root["led"], "led")
	str(led["override"], "led.override", "on")
	str(led["color"], "led.color", "#00ff00")
	num(led["brightness"], "led.brightness", 20)

	stp := obj(root["stp"], "stp")
	str(stp["version"], "stp.version", "rstp")
	num(stp["priority"], "stp.priority", 4096)

	lcm := obj(root["lcm"], "lcm")
	num(lcm["brightness"], "lcm.brightness", 40)
	boolean(lcm["brightness_override"], "lcm.brightness_override", true)
	str(lcm["idle_timeout"], "lcm.idle_timeout", "10m0s")
	nm := obj(lcm["night_mode"], "lcm.night_mode")
	str(nm["begins"], "lcm.night_mode.begins", "22:00")
	str(nm["ends"], "lcm.night_mode.ends", "06:00")

	var overrides []tftypes.Value
	if err := root["port_override"].As(&overrides); err != nil || len(overrides) != 1 {
		t.Fatalf("port_override = %v (%v), want one element", root["port_override"], err)
	}
	po := obj(overrides[0], "port_override[0]")
	str(po["name"], "port_override.name", "cam")
	dot1x := obj(po["dot1x"], "dot1x")
	str(dot1x["ctrl"], "dot1x.ctrl", "auto")
	str(dot1x["idle_timeout"], "dot1x.idle_timeout", "5m0s")
	erl := obj(po["egress_rate_limit"], "egress_rate_limit")
	boolean(erl["enabled"], "egress_rate_limit.enabled", true)
	num(erl["kbps"], "egress_rate_limit.kbps", 1000)
	lldp := obj(po["lldpmed"], "lldpmed")
	boolean(lldp["enabled"], "lldpmed.enabled", true)
	boolean(lldp["notify_enabled"], "lldpmed.notify_enabled", false)
	ps := obj(po["port_security"], "port_security")
	boolean(ps["enabled"], "port_security.enabled", true)
	var macs []tftypes.Value
	if err := ps["mac_address"].As(&macs); err != nil || len(macs) != 1 {
		t.Errorf("port_security.mac_address = %v (%v)", ps["mac_address"], err)
	}
	sc := obj(po["stormctrl"], "stormctrl")
	str(sc["type"], "stormctrl.type", "level")
	bcast := obj(sc["bcast"], "stormctrl.bcast")
	boolean(bcast["enabled"], "stormctrl.bcast.enabled", true)
	num(bcast["level"], "stormctrl.bcast.level", 50)
	if !bcast["rate"].IsNull() {
		t.Errorf("stormctrl.bcast.rate = %v, want null", bcast["rate"])
	}
	// A class with no prior keys at all stays a null object rather than an
	// object of nulls (ucast had only *_enabled; bcast/mcast are present).
	ucast := obj(sc["ucast"], "stormctrl.ucast")
	boolean(ucast["enabled"], "stormctrl.ucast.enabled", false)
}

// TestDevicePortOverride_nestedGroupsRoundTrip checks that the nested port
// feature groups convert API -> model -> API without loss, and that a group
// left unknown in the plan is resolved from the API object.
func TestDevicePortOverride_nestedGroupsRoundTrip(t *testing.T) {
	ctx := context.Background()
	r := &deviceResource{}

	api := unifi.DevicePortOverrides{
		PortIDX:                      ptrInt64(4),
		Dot1XCtrl:                    "mac_based",
		Dot1XIDleTimeout:             ptrInt64(300),
		EgressRateLimitKbps:          ptrInt64(2048),
		EgressRateLimitKbpsEnabled:   true,
		LldpmedEnabled:               true,
		LldpmedNotifyEnabled:         true,
		PortSecurityEnabled:          true,
		PortSecurityMACAddress:       []string{"aa:bb:cc:dd:ee:ff"},
		StormctrlType:                "rate",
		StormctrlBroadcastastEnabled: true,
		StormctrlBroadcastastRate:    ptrInt64(1000),
		StormctrlUcastEnabled:        true,
		StormctrlUcastLevel:          ptrInt64(10),
	}

	model, diags := apiPortOverrideToModel(api)
	if diags.HasError() {
		t.Fatalf("apiPortOverrideToModel: %v", diags)
	}
	sc := model.Stormctrl.Attributes()
	if got := attrAs[types.String](t, sc["type"]).ValueString(); got != "rate" {
		t.Errorf("stormctrl.type = %q, want rate", got)
	}
	bcast := attrAs[types.Object](t, sc["bcast"]).Attributes()
	if !attrAs[types.Bool](t, bcast["enabled"]).ValueBool() ||
		attrAs[types.Int64](t, bcast["rate"]).ValueInt64() != 1000 || !bcast["level"].IsNull() {
		t.Errorf("stormctrl.bcast = %v", bcast)
	}
	dot1x := model.Dot1X.Attributes()
	if attrAs[timetypes.GoDuration](t, dot1x["idle_timeout"]).ValueString() != "5m0s" {
		t.Errorf("dot1x.idle_timeout = %v, want 5m0s", dot1x["idle_timeout"])
	}

	objVal, d := types.ObjectValueFrom(ctx, model.AttributeTypes(), model)
	if d.HasError() {
		t.Fatalf("ObjectValueFrom: %v", d)
	}
	set, d := types.SetValue(
		types.ObjectType{AttrTypes: portOverrideAttrTypes()},
		[]attr.Value{objVal},
	)
	if d.HasError() {
		t.Fatalf("SetValue: %v", d)
	}
	back, d := r.frameworkToPortOverrides(ctx, set)
	if d.HasError() || len(back) != 1 {
		t.Fatalf("frameworkToPortOverrides: %v (%d)", d, len(back))
	}
	got := back[0]
	if got.Dot1XCtrl != "mac_based" || got.Dot1XIDleTimeout == nil || *got.Dot1XIDleTimeout != 300 {
		t.Errorf("dot1x round trip: ctrl=%q idle=%v", got.Dot1XCtrl, got.Dot1XIDleTimeout)
	}
	if !got.EgressRateLimitKbpsEnabled || got.EgressRateLimitKbps == nil ||
		*got.EgressRateLimitKbps != 2048 {
		t.Errorf(
			"egress_rate_limit round trip: %v %v",
			got.EgressRateLimitKbpsEnabled,
			got.EgressRateLimitKbps,
		)
	}
	if !got.LldpmedEnabled || !got.LldpmedNotifyEnabled {
		t.Errorf("lldpmed round trip: %v %v", got.LldpmedEnabled, got.LldpmedNotifyEnabled)
	}
	if !got.PortSecurityEnabled || len(got.PortSecurityMACAddress) != 1 {
		t.Errorf(
			"port_security round trip: %v %v",
			got.PortSecurityEnabled,
			got.PortSecurityMACAddress,
		)
	}
	if got.StormctrlType != "rate" || !got.StormctrlBroadcastastEnabled ||
		got.StormctrlBroadcastastRate == nil || *got.StormctrlBroadcastastRate != 1000 ||
		got.StormctrlBroadcastastLevel != nil ||
		!got.StormctrlUcastEnabled || got.StormctrlUcastLevel == nil || *got.StormctrlUcastLevel != 10 ||
		got.StormctrlMcastEnabled {
		t.Errorf("stormctrl round trip: %+v", got)
	}

	// A plan that declares only dot1x.ctrl leaves idle_timeout unknown and the
	// other groups wholly unknown; reconcile must resolve all of them.
	planned := model
	planned.Dot1X = types.ObjectValueMust(portDot1xAttrTypes(), map[string]attr.Value{
		"ctrl":         types.StringValue("mac_based"),
		"idle_timeout": timetypes.NewGoDurationUnknown(),
	})
	planned.Stormctrl = types.ObjectUnknown(portStormctrlAttrTypes())
	planned.Lldpmed = types.ObjectUnknown(portLldpmedAttrTypes())
	plannedObj, d := types.ObjectValueFrom(ctx, planned.AttributeTypes(), planned)
	if d.HasError() {
		t.Fatalf("planned ObjectValueFrom: %v", d)
	}
	prior, d := types.SetValue(
		types.ObjectType{AttrTypes: portOverrideAttrTypes()},
		[]attr.Value{plannedObj},
	)
	if d.HasError() {
		t.Fatalf("planned SetValue: %v", d)
	}
	reconciled, d := r.reconcilePortOverrides(ctx, prior, []unifi.DevicePortOverrides{api})
	if d.HasError() {
		t.Fatalf("reconcilePortOverrides: %v", d)
	}
	var out []portOverrideModel
	if d := reconciled.ElementsAs(ctx, &out, false); d.HasError() || len(out) != 1 {
		t.Fatalf("reading reconciled: %v (%d)", d, len(out))
	}
	if out[0].Stormctrl.IsUnknown() || out[0].Lldpmed.IsUnknown() {
		t.Fatal("wholly unknown groups were not resolved from the API")
	}
	if out[0].Dot1X.Attributes()["idle_timeout"].IsUnknown() {
		t.Fatal("dot1x.idle_timeout left unknown after reconcile")
	}
	if got := attrAs[types.String](
		t,
		out[0].Dot1X.Attributes()["ctrl"],
	).ValueString(); got != "mac_based" {
		t.Errorf("declared dot1x.ctrl overwritten: %q", got)
	}
}

// TestDeviceModelToAPIDevice_nestedGroups checks the top-level led/stp/lcm
// objects reach the API struct, and that null/unknown groups stay off it.
func TestDeviceModelToAPIDevice_nestedGroups(t *testing.T) {
	ctx := context.Background()
	r := &deviceResource{}

	nightMode := types.ObjectValueMust(deviceLcmNightModeAttrTypes(), map[string]attr.Value{
		"begins": types.StringValue("22:00"),
		"ends":   types.StringValue("06:00"),
	})
	model := &deviceResourceModel{
		MAC:  hwtypes.NewMACAddressValue("00:11:22:33:44:55"),
		Name: types.StringValue("sw"),
		Led: types.ObjectValueMust(deviceLedAttrTypes(), map[string]attr.Value{
			"override":   types.StringValue("on"),
			"color":      types.StringValue("#00ff00"),
			"brightness": types.Int64Value(20),
		}),
		Stp: types.ObjectValueMust(deviceStpAttrTypes(), map[string]attr.Value{
			"version":  types.StringValue("rstp"),
			"priority": types.Int64Unknown(),
		}),
		Lcm: types.ObjectValueMust(deviceLcmAttrTypes(), map[string]attr.Value{
			"brightness":            types.Int64Value(40),
			"brightness_override":   types.BoolValue(true),
			"idle_timeout":          timetypes.NewGoDurationValue(10 * time.Minute),
			"idle_timeout_override": types.BoolValue(false),
			"night_mode":            nightMode,
		}),
		ConfigNetwork:   types.ObjectNull(configNetworkAttrTypes()),
		PortOverride:    types.SetNull(types.ObjectType{AttrTypes: portOverrideAttrTypes()}),
		RadioTable:      types.ListNull(types.ObjectType{AttrTypes: radioTableAttrTypes()}),
		OutletOverrides: types.ListNull(types.ObjectType{AttrTypes: outletOverrideAttrTypes()}),
	}

	dev, diags := r.modelToAPIDevice(ctx, model)
	if diags.HasError() {
		t.Fatalf("modelToAPIDevice: %v", diags)
	}
	if dev.LedOverride != "on" || dev.LedOverrideColor != "#00ff00" ||
		dev.LedOverrideColorBrightness == nil || *dev.LedOverrideColorBrightness != 20 {
		t.Errorf(
			"led not mapped: %q %q %v",
			dev.LedOverride,
			dev.LedOverrideColor,
			dev.LedOverrideColorBrightness,
		)
	}
	if dev.StpVersion != "rstp" || dev.StpPriority != nil {
		t.Errorf(
			"stp: version=%q priority=%v (unknown priority must stay off the wire)",
			dev.StpVersion,
			dev.StpPriority,
		)
	}
	if dev.LcmBrightness == nil || *dev.LcmBrightness != 40 || !dev.LcmBrightnessOverride ||
		dev.LcmIDleTimeout == nil || *dev.LcmIDleTimeout != 600 || dev.LcmIDleTimeoutOverride ||
		dev.LcmNightModeBegins != "22:00" || dev.LcmNightModeEnds != "06:00" {
		t.Errorf("lcm not mapped: %+v", dev)
	}

	// Unknown groups (the plan omitted them) contribute nothing.
	model.Led = types.ObjectUnknown(deviceLedAttrTypes())
	model.Stp = types.ObjectNull(deviceStpAttrTypes())
	model.Lcm = types.ObjectUnknown(deviceLcmAttrTypes())
	dev, diags = r.modelToAPIDevice(ctx, model)
	if diags.HasError() {
		t.Fatalf("modelToAPIDevice (unknown groups): %v", diags)
	}
	if dev.LedOverride != "" || dev.StpVersion != "" || dev.LcmBrightness != nil ||
		dev.LcmNightModeBegins != "" {
		t.Errorf("unknown/null groups leaked into the API struct: %+v", dev)
	}

	// setResourceData builds the objects back from the API and preserves a
	// configured LED value the controller did not echo.
	readDiags := diags
	back := &deviceResourceModel{
		Led: types.ObjectValueMust(deviceLedAttrTypes(), map[string]attr.Value{
			"override":   types.StringValue("on"),
			"color":      types.StringNull(),
			"brightness": types.Int64Unknown(),
		}),
	}
	r.setResourceData(ctx, &readDiags, &unifi.Device{
		ID: "dev-1", MAC: "00:11:22:33:44:55",
		StpVersion: "stp", StpPriority: ptrInt64(8192),
		LcmBrightness: ptrInt64(40), LcmNightModeBegins: "22:00",
	}, back, "default")
	if readDiags.HasError() {
		t.Fatalf("setResourceData: %v", readDiags)
	}
	led := back.Led.Attributes()
	if attrAs[types.String](t, led["override"]).ValueString() != "on" {
		t.Errorf("configured led.override not preserved: %v", led["override"])
	}
	if !led["brightness"].IsNull() {
		t.Errorf("unknown led.brightness should resolve to null: %v", led["brightness"])
	}
	if attrAs[types.Int64](t, back.Stp.Attributes()["priority"]).ValueInt64() != 8192 {
		t.Errorf("stp.priority not read: %v", back.Stp)
	}
	nm := attrAs[types.Object](t, back.Lcm.Attributes()["night_mode"]).Attributes()
	if attrAs[types.String](t, nm["begins"]).ValueString() != "22:00" || !nm["ends"].IsNull() {
		t.Errorf("lcm.night_mode not read: %v", nm)
	}
}

// attrAs is a checked type assertion for attr.Value in tests.
func attrAs[T attr.Value](t *testing.T, v attr.Value) T {
	t.Helper()
	out, ok := v.(T)
	if !ok {
		t.Fatalf("value %v is %T, want %T", v, v, out)
	}
	return out
}

// TestAccDeviceFramework_nestedGroups exercises the nested led/stp objects and
// the nested port_override feature groups against a simulated switch.
func TestAccDeviceFramework_nestedGroups(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "unifi_device" "groups" {
	mac               = "00:27:22:00:00:01"
	name              = "Test Device Groups"
	forget_on_destroy = false

	led = {
		override = "on"
	}
	stp = {
		version  = "rstp"
		priority = 4096
	}

	port_override {
		index = 1
		name  = "grouped"
		dot1x = {
			ctrl = "auto"
		}
		stormctrl = {
			type = "level"
			bcast = {
				enabled = true
				level   = 40
			}
		}
	}
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_device.groups", "id"),
					resource.TestCheckResourceAttr("unifi_device.groups", "led.override", "on"),
					resource.TestCheckResourceAttr("unifi_device.groups", "stp.version", "rstp"),
					resource.TestCheckResourceAttr("unifi_device.groups", "stp.priority", "4096"),
					resource.TestCheckResourceAttr("unifi_device.groups", "port_override.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(
						"unifi_device.groups",
						"port_override.*",
						map[string]string{
							"index":                   "1",
							"dot1x.ctrl":              "auto",
							"stormctrl.type":          "level",
							"stormctrl.bcast.enabled": "true",
							"stormctrl.bcast.level":   "40",
						},
					),
				),
			},
			{
				Config: `
resource "unifi_device" "groups" {
	mac               = "00:27:22:00:00:01"
	name              = "Test Device Groups"
	forget_on_destroy = false

	stp = {
		version  = "rstp"
		priority = 8192
	}
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_device.groups", "stp.priority", "8192"),
				),
			},
		},
	})
}

// TestDeviceUpgradeState_olderVersionsAlsoNest guards the ordering inside the
// shared upgrader: v0 state converts integer-second durations to strings on
// the flat keys and THEN nests, and v1 (already strings) nests too.
func TestDeviceUpgradeState_olderVersionsAlsoNest(t *testing.T) {
	ctx := context.Background()
	r := &deviceResource{}
	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	cases := map[string]struct {
		version int64
		prior   string
	}{
		"v0 integer seconds": {0, `{"id":"d","mac":"00:11:22:33:44:55","lcm_idle_timeout":600,
			"port_override":[{"index":1,"dot1x_idle_timeout":300,"tagged_networkconf_ids":["n1"]}]}`},
		"v1 duration strings": {1, `{"id":"d","mac":"00:11:22:33:44:55","lcm_idle_timeout":"10m0s",
			"port_override":[{"index":1,"dot1x_idle_timeout":"5m0s","tagged_networkconf_ids":["n1"]}]}`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			up, ok := r.UpgradeState(ctx)[tc.version]
			if !ok {
				t.Fatalf("no upgrader for version %d", tc.version)
			}
			resp := &fwresource.UpgradeStateResponse{}
			up.StateUpgrader(ctx, fwresource.UpgradeStateRequest{
				RawState: &tfprotov6.RawState{JSON: []byte(tc.prior)},
			}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("upgrade failed: %v", resp.Diagnostics)
			}
			val, err := resp.DynamicValue.Unmarshal(schemaType)
			if err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			var root, lcm, po, dot1x map[string]tftypes.Value
			var overrides []tftypes.Value
			if err := val.As(&root); err != nil {
				t.Fatalf("as root: %v", err)
			}
			if err := root["lcm"].As(&lcm); err != nil {
				t.Fatalf("as lcm: %v (%v)", err, root["lcm"])
			}
			var idle string
			if err := lcm["idle_timeout"].As(&idle); err != nil || idle != "10m0s" {
				t.Errorf("lcm.idle_timeout = %v (%v), want 10m0s", lcm["idle_timeout"], err)
			}
			if err := root["port_override"].As(&overrides); err != nil || len(overrides) != 1 {
				t.Fatalf("port_override = %v (%v)", root["port_override"], err)
			}
			if err := overrides[0].As(&po); err != nil {
				t.Fatalf("as override: %v", err)
			}
			if err := po["dot1x"].As(&dot1x); err != nil {
				t.Fatalf("as dot1x: %v (%v)", err, po["dot1x"])
			}
			if err := dot1x["idle_timeout"].As(&idle); err != nil || idle != "5m0s" {
				t.Errorf("dot1x.idle_timeout = %v (%v), want 5m0s", dot1x["idle_timeout"], err)
			}
			if po["tagged_networkconf_ids"].Type().Is(tftypes.List{}) {
				t.Error("tagged_networkconf_ids must reconcile to the Set schema type")
			}
		})
	}
}
