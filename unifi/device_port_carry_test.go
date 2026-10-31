package unifi

import (
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// unexposedPortOverrideFields are the DevicePortOverrides keys the port_override
// schema does not expose, which carryUnwritableFields copies from the controller.
var unexposedPortOverrideFields = []string{
	"eee_enabled",
	"ld_mode",
	"ld_status",
	"link_debounce",
	"link_debounce_auto",
	"multicast_router_mode",
	"precision_time_protocol_enabled",
	"qos_profile",
	"routed_networkconf_id",
	"sd_wan_underlay_port",
	"sd_wan_underlay_port_networkconf_id",
	"stable_port_enabled",
	"stp_bpdu_guard_enabled",
	"stp_uplink",
	"trusted_port_mac",
	"unit_id",
}

// schemaKeyToJSON maps port_override attribute names that differ from the
// DevicePortOverrides JSON key. Nested attributes are named by their flattened
// path (see exposedPortOverrideKeys).
var schemaKeyToJSON = map[string]string{
	"index":                     "port_idx",
	"port_profile_id":           "portconf_id",
	"egress_rate_limit_enabled": "egress_rate_limit_kbps_enabled",
}

// exposedPortOverrideKeys returns the port_override schema attributes as flat
// keys. A nested feature-group object (dot1x, stormctrl, …) exposes its
// attributes under the group prefix, e.g. stormctrl.bcast.level is
// stormctrl_bcast_level, which is how DevicePortOverrides names the field.
func exposedPortOverrideKeys(prefix string, attrs map[string]attr.Type) []string {
	var keys []string
	for k, typ := range attrs {
		if obj, ok := typ.(types.ObjectType); ok {
			keys = append(keys, exposedPortOverrideKeys(prefix+k+"_", obj.AttrTypes)...)
			continue
		}
		keys = append(keys, prefix+k)
	}
	return keys
}

func portOverrideJSONFields() map[string]reflect.StructField {
	fields := map[string]reflect.StructField{}
	t := reflect.TypeOf(unifi.DevicePortOverrides{})
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		name := strings.Split(f.Tag.Get("json"), ",")[0]
		if name != "" && name != "-" {
			fields[name] = f
		}
	}
	return fields
}

// Every DevicePortOverrides field must either be configurable through the schema
// or be carried forward from the controller. A field that is neither is dropped
// from every port whenever any port on the device is declared (see
// carryUnwritableFields), so a go-unifi upgrade that adds one must update both.
func Test_carryUnwritableFields_coversEveryUnexposedField(t *testing.T) {
	exposed := map[string]bool{}
	for _, k := range exposedPortOverrideKeys("", portOverrideAttrTypes()) {
		if j, ok := schemaKeyToJSON[k]; ok {
			k = j
		}
		if _, ok := portOverrideJSONFields()[k]; !ok {
			t.Errorf("port_override attribute %q does not map to a DevicePortOverrides field", k)
		}
		exposed[k] = true
	}
	carried := map[string]bool{}
	for _, k := range unexposedPortOverrideFields {
		carried[k] = true
		if exposed[k] {
			t.Errorf("%q is exposed by the schema; it must not be carried from the controller", k)
		}
	}

	for name := range portOverrideJSONFields() {
		if !exposed[name] && !carried[name] {
			t.Errorf(
				"DevicePortOverrides.%s is neither in the port_override schema nor carried by carryUnwritableFields",
				name,
			)
		}
	}
	for k := range carried {
		if _, ok := portOverrideJSONFields()[k]; !ok {
			t.Errorf("carried field %q does not exist on DevicePortOverrides", k)
		}
	}
}

// nonZero returns a non-zero value of type t.
func nonZero(t *testing.T, typ reflect.Type) reflect.Value {
	switch typ.Kind() {
	case reflect.Bool:
		return reflect.ValueOf(true)
	case reflect.String:
		return reflect.ValueOf("x").Convert(typ)
	case reflect.Int64:
		return reflect.ValueOf(int64(7)).Convert(typ)
	case reflect.Ptr:
		v := reflect.New(typ.Elem())
		if typ.Elem().Kind() == reflect.Struct {
			v.Elem().Field(0).Set(nonZero(t, typ.Elem().Field(0).Type))
		} else {
			v.Elem().Set(nonZero(t, typ.Elem()))
		}
		return v
	case reflect.Slice:
		s := reflect.MakeSlice(typ, 1, 1)
		if typ.Elem().Kind() != reflect.Struct {
			s.Index(0).Set(nonZero(t, typ.Elem()))
		}
		return s
	}
	t.Fatalf("no non-zero value for %s", typ)
	return reflect.Value{}
}

func Test_carryUnwritableFields_carriesUnexposedFields(t *testing.T) {
	fields := portOverrideJSONFields()
	for _, name := range unexposedPortOverrideFields {
		t.Run(name, func(t *testing.T) {
			f := fields[name]
			current := unifi.DevicePortOverrides{PortIDX: ptrInt64(3)}
			reflect.ValueOf(&current).Elem().FieldByIndex(f.Index).Set(nonZero(t, f.Type))
			declared := unifi.DevicePortOverrides{PortIDX: ptrInt64(3), Name: "Port 3"}

			got := mergePortOverridesByIndex(
				[]unifi.DevicePortOverrides{current},
				[]unifi.DevicePortOverrides{declared},
			)

			want := reflect.ValueOf(current).FieldByIndex(f.Index).Interface()
			if have := reflect.ValueOf(got[0]).
				FieldByIndex(f.Index).
				Interface(); !reflect.DeepEqual(
				have,
				want,
			) {
				t.Errorf("%s = %v after merge, want the controller's %v", name, have, want)
			}
			if got[0].Name != "Port 3" {
				t.Errorf("declared name lost: %q", got[0].Name)
			}
		})
	}
}

// A switch port as UniFi Network 10.6 stores it on a USW Pro Max 16 PoE. Declaring
// it with its current values must merge back into an identical entry, so the
// update diff for port_overrides is empty and nothing is written. Before the fix
// the merged entry lacked stp_edge_state, multicast_router_mode,
// link_debounce_auto and qos_profile, which sent the whole array and deleted
// them from every port.
func Test_mergePortOverridesByIndex_unchangedPortIsIdentical(t *testing.T) {
	current := unifi.DevicePortOverrides{
		PortIDX:             ptrInt64(10),
		Name:                "Cottage",
		NATiveNetworkID:     "66ec38ab34dbcb19dce4a67f",
		Forward:             "all",
		OpMode:              "switch",
		PoeMode:             "off",
		SettingPreference:   "auto",
		TaggedVLANMgmt:      "auto",
		Autoneg:             boolPtr(true),
		StpPortMode:         boolPtr(true),
		LinkDebounceAuto:    boolPtr(true),
		MulticastRouterMode: "NONE",
		StpEdgeState:        "auto",
		QOSProfile:          &unifi.DeviceQOSProfile{QOSProfileMode: "custom"},
	}
	declared := unifi.DevicePortOverrides{
		PortIDX:           ptrInt64(10),
		Name:              "Cottage",
		NATiveNetworkID:   "66ec38ab34dbcb19dce4a67f",
		Forward:           "all",
		PoeMode:           "off",
		SettingPreference: "auto",
		TaggedVLANMgmt:    "auto",
		Autoneg:           boolPtr(true),
		StpPortMode:       boolPtr(true),
	}

	got := mergePortOverridesByIndex(
		[]unifi.DevicePortOverrides{current},
		[]unifi.DevicePortOverrides{declared},
	)
	if !reflect.DeepEqual(got, []unifi.DevicePortOverrides{current}) {
		t.Errorf("merged = %+v, want the controller's entry unchanged %+v", got[0], current)
	}
}
