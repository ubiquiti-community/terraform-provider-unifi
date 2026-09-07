package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// TestPortProfileUpgradeState_v1NestsPrefixedGroups guards the v1 -> v2 schema
// upgrade: flat dot1x_*/egress_*/lldpmed_*/port_security_*/stormctrl_*
// attributes move into nested objects.
func TestPortProfileUpgradeState_v1NestsPrefixedGroups(t *testing.T) {
	ctx := context.Background()
	r := &portProfileResource{}

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	if schemaResp.Schema.Version != 2 {
		t.Fatalf("port profile schema Version = %d, want 2", schemaResp.Schema.Version)
	}
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	prior := []byte(`{
		"id": "pp-1", "site": "default", "name": "trunk", "autoneg": true,
		"dot1x_ctrl": "mac_based", "dot1x_idle_timeout": "5m0s",
		"egress_rate_limit_kbps": 512, "egress_rate_limit_kbps_enabled": true,
		"lldpmed_enabled": true, "lldpmed_notify_enabled": null,
		"port_security_enabled": true, "port_security_mac_address": ["aa:bb:cc:dd:ee:ff"],
		"stormctrl_type": "rate",
		"stormctrl_bcast_enabled": true, "stormctrl_bcast_level": null, "stormctrl_bcast_rate": 5000,
		"stormctrl_mcast_enabled": false, "stormctrl_mcast_level": null, "stormctrl_mcast_rate": null,
		"stormctrl_ucast_enabled": false, "stormctrl_ucast_level": null, "stormctrl_ucast_rate": null
	}`)

	ups := r.UpgradeState(ctx)
	for _, v := range []int64{0, 1} {
		if _, ok := ups[v]; !ok {
			t.Fatalf("no upgrader registered for schema version %d", v)
		}
	}
	resp := &fwresource.UpgradeStateResponse{}
	ups[1].StateUpgrader(ctx, fwresource.UpgradeStateRequest{
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
	for _, flat := range []string{"dot1x_ctrl", "stormctrl_type", "port_security_enabled"} {
		if _, exists := root[flat]; exists {
			t.Errorf("flat attribute %q survived the upgrade", flat)
		}
	}
	dot1x := obj(root["dot1x"], "dot1x")
	str(dot1x["ctrl"], "dot1x.ctrl", "mac_based")
	str(dot1x["idle_timeout"], "dot1x.idle_timeout", "5m0s")
	sc := obj(root["stormctrl"], "stormctrl")
	str(sc["type"], "stormctrl.type", "rate")
	bcast := obj(sc["bcast"], "stormctrl.bcast")
	var enabled bool
	if err := bcast["enabled"].As(&enabled); err != nil || !enabled {
		t.Errorf("stormctrl.bcast.enabled = %v (%v), want true", bcast["enabled"], err)
	}
	ps := obj(root["port_security"], "port_security")
	var macs []tftypes.Value
	if err := ps["mac_address"].As(&macs); err != nil || len(macs) != 1 {
		t.Errorf("port_security.mac_address = %v (%v), want one entry", ps["mac_address"], err)
	}
	lldp := obj(root["lldpmed"], "lldpmed")
	if !lldp["notify_enabled"].IsNull() {
		t.Errorf("lldpmed.notify_enabled = %v, want null", lldp["notify_enabled"])
	}
}

// TestPortProfileNestedGroups_wireAndReadBack checks that the nested feature
// groups are written to and read from the API struct, including the storm
// control, egress and priority queue fields that were previously dropped.
func TestPortProfileNestedGroups_wireAndReadBack(t *testing.T) {
	ctx := context.Background()
	r := &portProfileResource{}

	macs, d := types.SetValue(
		types.StringType,
		[]attr.Value{types.StringValue("aa:bb:cc:dd:ee:ff")},
	)
	if d.HasError() {
		t.Fatalf("mac set: %v", d)
	}
	model := &portProfileResourceModel{
		Name:   types.StringValue("cams"),
		OpMode: types.StringValue("switch"),
		Dot1X: types.ObjectValueMust(portDot1xAttrTypes(), map[string]attr.Value{
			"ctrl":         types.StringValue("mac_based"),
			"idle_timeout": portProfileDot1xDefault().Attributes()["idle_timeout"],
		}),
		EgressRateLimit: types.ObjectValueMust(
			portEgressRateLimitAttrTypes(),
			map[string]attr.Value{
				"enabled": types.BoolValue(true),
				"kbps":    types.Int64Value(4096),
			},
		),
		Lldpmed: portProfileLldpmedDefault(),
		PortSecurity: types.ObjectValueMust(
			portProfilePortSecurityAttrTypes(),
			map[string]attr.Value{
				"enabled":     types.BoolValue(true),
				"mac_address": macs,
			},
		),
		Stormctrl: types.ObjectValueMust(portStormctrlAttrTypes(), map[string]attr.Value{
			"type": types.StringValue("level"),
			"bcast": types.ObjectValueMust(portStormctrlClassAttrTypes(), map[string]attr.Value{
				"enabled": types.BoolValue(true),
				"level":   types.Int64Value(30),
				"rate":    types.Int64Null(),
			}),
			"mcast": types.ObjectValueMust(portStormctrlClassAttrTypes(), map[string]attr.Value{
				"enabled": types.BoolValue(false),
				"level":   types.Int64Null(),
				"rate":    types.Int64Null(),
			}),
			"ucast": types.ObjectUnknown(portStormctrlClassAttrTypes()),
		}),
		PriorityQueue2Level:       types.Int64Value(20),
		TaggedNetworkConfIDs:      types.SetNull(types.StringType),
		ExcludedNetworkConfIDs:    types.SetNull(types.StringType),
		MulticastRouterNetworkIDs: types.SetNull(types.StringType),
	}

	api, diags := r.modelToAPIPortProfile(ctx, model)
	if diags.HasError() {
		t.Fatalf("modelToAPIPortProfile: %v", diags)
	}
	if api.Dot1XCtrl != "mac_based" || api.Dot1XIDleTimeout == nil || *api.Dot1XIDleTimeout != 300 {
		t.Errorf("dot1x: %q %v", api.Dot1XCtrl, api.Dot1XIDleTimeout)
	}
	if !api.EgressRateLimitKbpsEnabled || api.EgressRateLimitKbps == nil ||
		*api.EgressRateLimitKbps != 4096 {
		t.Errorf(
			"egress_rate_limit: %v %v",
			api.EgressRateLimitKbpsEnabled,
			api.EgressRateLimitKbps,
		)
	}
	if !api.LldpmedEnabled || api.LldpmedNotifyEnabled {
		t.Errorf("lldpmed: %v %v", api.LldpmedEnabled, api.LldpmedNotifyEnabled)
	}
	if !api.PortSecurityEnabled || len(api.PortSecurityMACAddress) != 1 {
		t.Errorf("port_security: %v %v", api.PortSecurityEnabled, api.PortSecurityMACAddress)
	}
	if api.StormctrlType != "level" || !api.StormctrlBroadcastastEnabled ||
		api.StormctrlBroadcastastLevel == nil || *api.StormctrlBroadcastastLevel != 30 ||
		api.StormctrlMcastEnabled || api.StormctrlUcastEnabled || api.StormctrlUcastLevel != nil {
		t.Errorf("stormctrl: %+v", api)
	}
	if api.PriorityQueue2Level == nil || *api.PriorityQueue2Level != 20 ||
		api.PriorityQueue1Level != nil {
		t.Errorf("priority queues: %v %v", api.PriorityQueue1Level, api.PriorityQueue2Level)
	}

	// Read back: every group is rebuilt from the API response.
	var back portProfileResourceModel
	if d := r.portProfileToModel(ctx, api, &back, "default"); d.HasError() {
		t.Fatalf("portProfileToModel: %v", d)
	}
	sc := back.Stormctrl.Attributes()
	if attrAs[types.String](t, sc["type"]).ValueString() != "level" {
		t.Errorf("stormctrl.type read back = %v", sc["type"])
	}
	bcast := attrAs[types.Object](t, sc["bcast"]).Attributes()
	if !attrAs[types.Bool](t, bcast["enabled"]).ValueBool() ||
		attrAs[types.Int64](t, bcast["level"]).ValueInt64() != 30 {
		t.Errorf("stormctrl.bcast read back = %v", bcast)
	}
	if attrAs[types.Int64](t, back.EgressRateLimit.Attributes()["kbps"]).ValueInt64() != 4096 {
		t.Errorf("egress_rate_limit read back = %v", back.EgressRateLimit)
	}
	if back.PriorityQueue2Level.ValueInt64() != 20 {
		t.Errorf("priority_queue2_level read back = %v", back.PriorityQueue2Level)
	}

	// A wholly-unknown plan group keeps the controller's values on update.
	plan := &portProfileResourceModel{
		Stormctrl: types.ObjectUnknown(portStormctrlAttrTypes()),
		Dot1X: types.ObjectValueMust(portDot1xAttrTypes(), map[string]attr.Value{
			"ctrl":         types.StringValue("auto"),
			"idle_timeout": portProfileDot1xDefault().Attributes()["idle_timeout"],
		}),
	}
	state := back
	r.applyPlanToState(ctx, plan, &state)
	if !state.Stormctrl.Equal(back.Stormctrl) {
		t.Errorf("unknown planned stormctrl must keep controller value: %v", state.Stormctrl)
	}
	if got := attrAs[types.String](
		t,
		state.Dot1X.Attributes()["ctrl"],
	).ValueString(); got != "auto" {
		t.Errorf("planned dot1x.ctrl not applied: %q", got)
	}

	// The controller only echoes a null lldpmed.notify_enabled as false; when
	// the practitioner never set it, it stays null.
	var fresh portProfileResourceModel
	if d := r.portProfileToModel(
		ctx,
		&unifi.PortProfile{ID: "x"},
		&fresh,
		"default",
	); d.HasError() {
		t.Fatalf("portProfileToModel (fresh): %v", d)
	}
	if !fresh.Lldpmed.Attributes()["notify_enabled"].IsNull() {
		t.Errorf("lldpmed.notify_enabled should stay null when never configured: %v", fresh.Lldpmed)
	}
}

// TestAccPortProfileFramework_nestedGroups exercises the nested feature
// groups end to end: create with partial groups (nested defaults fill the
// rest), import, update values, then drop the groups.
func TestAccPortProfileFramework_nestedGroups(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "unifi_port_profile" "groups" {
	name    = "Test Port Profile Groups"
	op_mode = "switch"

	dot1x = {
		ctrl = "auto"
	}
	egress_rate_limit = {
		enabled = true
		kbps    = 1000
	}
	port_security = {
		enabled     = true
		mac_address = ["aa:bb:cc:dd:ee:01"]
	}
	stormctrl = {
		type = "level"
		bcast = {
			enabled = true
			level   = 50
		}
	}
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_port_profile.groups", "id"),
					check("dot1x.ctrl", "auto"),
					check("dot1x.idle_timeout", "5m0s"),
					check("egress_rate_limit.enabled", "true"),
					check("egress_rate_limit.kbps", "1000"),
					check("port_security.enabled", "true"),
					check("port_security.mac_address.#", "1"),
					check("stormctrl.type", "level"),
					check("stormctrl.bcast.enabled", "true"),
					check("stormctrl.bcast.level", "50"),
					check("stormctrl.mcast.enabled", "false"),
					check("lldpmed.enabled", "true"),
				),
			},
			{
				ResourceName:      "unifi_port_profile.groups",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Disable the residual-value groups explicitly (their value leaves
			// are Computed, so the controller may keep the old kbps/MAC list),
			// switch storm control to rate mode and change dot1x.
			{
				Config: `
resource "unifi_port_profile" "groups" {
	name    = "Test Port Profile Groups"
	op_mode = "switch"

	dot1x = {
		ctrl         = "mac_based"
		idle_timeout = "10m0s"
	}
	egress_rate_limit = {
		enabled = false
	}
	port_security = {
		enabled = false
	}
	stormctrl = {
		type = "rate"
		bcast = {
			enabled = true
			rate    = 5000
		}
		ucast = {
			enabled = true
			rate    = 6000
		}
	}
}
`,
				Check: resource.ComposeTestCheckFunc(
					check("dot1x.ctrl", "mac_based"),
					check("dot1x.idle_timeout", "10m0s"),
					check("egress_rate_limit.enabled", "false"),
					check("port_security.enabled", "false"),
					check("stormctrl.type", "rate"),
					check("stormctrl.bcast.rate", "5000"),
					check("stormctrl.ucast.rate", "6000"),
				),
			},
			{
				ResourceName:      "unifi_port_profile.groups",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Removing the dot1x block resets it to its defaults (the group has an
			// object default); the Computed groups keep the controller's values.
			{
				Config: `
resource "unifi_port_profile" "groups" {
	name    = "Test Port Profile Groups"
	op_mode = "switch"
}
`,
				Check: resource.ComposeTestCheckFunc(
					check("dot1x.ctrl", "force_authorized"),
					check("dot1x.idle_timeout", "5m0s"),
					check("stormctrl.type", "rate"),
					check("stormctrl.bcast.enabled", "true"),
				),
			},
		},
	})
}

// check asserts an attribute of unifi_port_profile.groups.
func check(key, value string) resource.TestCheckFunc {
	return resource.TestCheckResourceAttr("unifi_port_profile.groups", key, value)
}
