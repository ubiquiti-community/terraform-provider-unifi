package unifi

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// The port settings exposed on port_override (BPDU Guard, STP uplink,
// multicast router mode, EEE, link debounce, stable port, PTP, QoS profile)
// are only managed when set: an unset one keeps the controller's value, a set
// one is written.
func Test_carryUnwritableFields_portSettingsManagedWhenSet(t *testing.T) {
	on, off := true, false
	ms500, ms900 := int64(500), int64(900)
	current := unifi.DevicePortOverrides{
		PortIDX:                      ptrInt64(5),
		StpBpduGuardEnabled:          &on,
		StpUplink:                    &on,
		MulticastRouterMode:          "ALL",
		EeeEnabled:                   &on,
		LinkDebounce:                 &ms500,
		LinkDebounceAuto:             &off,
		StablePortEnabled:            &on,
		PrecisionTimeProtocolEnabled: &on,
		QOSProfile:                   &unifi.DeviceQOSProfile{QOSProfileMode: "dante_audio"},
	}

	kept := carryUnwritableFields(current, unifi.DevicePortOverrides{PortIDX: ptrInt64(5)})
	if got, want := mustJSON(t, kept), mustJSON(t, current); got != want {
		t.Errorf("unset settings were not kept:\n got %s\nwant %s", got, want)
	}

	declared := unifi.DevicePortOverrides{
		PortIDX:                      ptrInt64(5),
		StpBpduGuardEnabled:          &off,
		StpUplink:                    &off,
		MulticastRouterMode:          "NONE",
		EeeEnabled:                   &off,
		LinkDebounce:                 &ms900,
		LinkDebounceAuto:             &on,
		StablePortEnabled:            &off,
		PrecisionTimeProtocolEnabled: &off,
		QOSProfile:                   &unifi.DeviceQOSProfile{},
	}
	written := carryUnwritableFields(current, declared)
	if got, want := mustJSON(t, written), mustJSON(t, declared); got != want {
		t.Errorf("set settings were not written:\n got %s\nwant %s", got, want)
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func Test_portOverride_portSettingsRoundTrip(t *testing.T) {
	ctx := context.Background()
	on, off := true, false
	ms := int64(1200)
	api := unifi.DevicePortOverrides{
		PortIDX:                      ptrInt64(7),
		StpBpduGuardEnabled:          &on,
		StpUplink:                    &off,
		MulticastRouterMode:          "CUSTOM",
		MulticastRouterNetworkIDs:    []string{"net-av"},
		EeeEnabled:                   &off,
		LinkDebounce:                 &ms,
		LinkDebounceAuto:             &off,
		StablePortEnabled:            &on,
		PrecisionTimeProtocolEnabled: &on,
		QOSProfile: &unifi.DeviceQOSProfile{
			QOSProfileMode: "custom",
			QOSPolicies: []unifi.DeviceQOSPolicies{{
				QOSMatching: &unifi.DeviceQOSMatching{Protocol: "udp", DstPort: ptrInt64(319)},
				QOSMarking:  &unifi.DeviceQOSMarking{Queue: ptrInt64(7), DscpCode: ptrInt64(46)},
			}},
		},
	}
	model, d := apiPortOverrideToModel(api)
	if d.HasError() {
		t.Fatalf("apiPortOverrideToModel: %v", d)
	}
	if model.MulticastRouterMode.ValueString() != "CUSTOM" ||
		!model.STPBPDUGuardEnabled.ValueBool() ||
		!model.PTPEnabled.ValueBool() ||
		model.EEEEnabled.ValueBool() {
		t.Errorf("read back: %+v", model)
	}
	back, d := portOverrideModelToAPI(ctx, model)
	if d.HasError() {
		t.Fatalf("portOverrideModelToAPI: %v", d)
	}
	for name, pair := range map[string][2]any{
		"stp_bpdu_guard_enabled":          {api.StpBpduGuardEnabled, back.StpBpduGuardEnabled},
		"stp_uplink":                      {api.StpUplink, back.StpUplink},
		"multicast_router_mode":           {api.MulticastRouterMode, back.MulticastRouterMode},
		"eee_enabled":                     {api.EeeEnabled, back.EeeEnabled},
		"link_debounce":                   {api.LinkDebounce, back.LinkDebounce},
		"link_debounce_auto":              {api.LinkDebounceAuto, back.LinkDebounceAuto},
		"stable_port_enabled":             {api.StablePortEnabled, back.StablePortEnabled},
		"precision_time_protocol_enabled": {api.PrecisionTimeProtocolEnabled, back.PrecisionTimeProtocolEnabled},
		"qos_profile":                     {api.QOSProfile, back.QOSProfile},
	} {
		if got, want := mustJSON(t, pair[1]), mustJSON(t, pair[0]); got != want {
			t.Errorf("%s: %s, want %s", name, got, want)
		}
	}

	// Unset settings are not written (nil/""), so carryUnwritableFields keeps them.
	unset, d := apiPortOverrideToModel(unifi.DevicePortOverrides{PortIDX: ptrInt64(7)})
	if d.HasError() {
		t.Fatalf("apiPortOverrideToModel: %v", d)
	}
	unset.STPBPDUGuardEnabled = types.BoolUnknown()
	po, d := portOverrideModelToAPI(ctx, unset)
	if d.HasError() {
		t.Fatalf("portOverrideModelToAPI: %v", d)
	}
	if po.StpBpduGuardEnabled != nil || po.MulticastRouterMode != "" || po.QOSProfile != nil ||
		po.LinkDebounce != nil || po.LinkDebounceAuto != nil {
		t.Errorf("unset settings were written: %s", mustJSON(t, po))
	}
}

// smpte_st2110 is a QoS preset for switch ports only.
func Test_qosProfileMode_smpteOnPortOverrideOnly(t *testing.T) {
	valid := func(a schema.SingleNestedAttribute, v string) bool {
		mode, ok := a.Attributes["mode"].(schema.StringAttribute)
		if !ok {
			t.Fatalf("qos_profile.mode is %T", a.Attributes["mode"])
		}
		for _, x := range mode.Validators {
			resp := &validator.StringResponse{}
			x.ValidateString(context.Background(), validator.StringRequest{
				Path: path.Root("mode"), ConfigValue: types.StringValue(v),
			}, resp)
			if resp.Diagnostics.HasError() {
				return false
			}
		}
		return true
	}
	if !valid(devicePortQOSProfileSchema(), "smpte_st2110") {
		t.Errorf("port_override qos_profile.mode rejects smpte_st2110")
	}
	if valid(portQOSProfileSchema("port profile", portQOSProfileModes), "smpte_st2110") {
		t.Errorf("port profile qos_profile.mode accepts smpte_st2110")
	}
}
