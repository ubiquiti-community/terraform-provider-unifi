package unifi

import (
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

const portProfileTestID = "6650dd00000000000000b001"

// portProfileTestDocument is a profile made in the UniFi UI, with every field
// the schema does not expose turned on (#741).
const portProfileTestDocument = `{
	"_id": "6650dd00000000000000b001",
	"site_id": "6650aa00000000000000a000",
	"name": "Uplink",
	"forward": "all",
	"native_networkconf_id": "6650aa00000000000000a00f",
	"tagged_vlan_mgmt": "auto",
	"setting_preference": "manual",
	"autoneg": true,
	"op_mode": "switch",
	"poe_mode": "auto",
	"isolation": false,
	"port_keepalive_enabled": false,
	"port_security_enabled": false,
	"stp_port_mode": true,
	"lldpmed_enabled": true,
	"lldpmed_notify_enabled": false,
	"egress_rate_limit_kbps_enabled": false,
	"stormctrl_bcast_enabled": false,
	"stormctrl_mcast_enabled": false,
	"stormctrl_ucast_enabled": false,
	"dot1x_ctrl": "force_authorized",
	"dot1x_idle_timeout": 300,
	"eee_enabled": true,
	"flow_control_enabled": true,
	"ld_status": true,
	"link_debounce_auto": true,
	"precision_time_protocol_enabled": true,
	"stable_port_enabled": true,
	"stp_bpdu_guard_enabled": true,
	"stp_uplink": true
}`

// unexposedPortProfileFields are the PortProfile keys go-unifi always sends
// (no omitempty) that the schema does not expose, which carryUnexposedFields
// copies from the controller.
var unexposedPortProfileFields = []string{
	"eee_enabled",
	"flow_control_enabled",
	"ld_status",
	"link_debounce_auto",
	"precision_time_protocol_enabled",
	"stable_port_enabled",
	"stp_bpdu_guard_enabled",
	"stp_uplink",
}

// Renaming a profile must not switch off what only the UI can set (#741).
func Test_portProfileResource_Update_keepsUnexposedFields(t *testing.T) {
	h := newUpdateHarness(
		t, "unifi_port_profile", "portconf", portProfileTestID, portProfileTestDocument,
		func(c *Client) harnessResource { return &portProfileResource{client: c} },
	)
	h.importAndRead()

	puts := h.apply(h.config(map[string]attr.Value{
		"name":     types.StringValue("Uplink 2"),
		"poe_mode": types.StringValue("auto"),
	}))

	if len(puts) == 0 {
		t.Fatal("renaming the profile sent no PUT")
	}
	if got := h.fake.stored("name"); got != "Uplink 2" {
		t.Errorf("name = %v after update, want Uplink 2", got)
	}
	for _, key := range unexposedPortProfileFields {
		if got := h.fake.stored(key); got != true {
			t.Errorf("%s = %v after update, want the controller's true", key, got)
		}
	}
}

// Every PortProfile field go-unifi always sends must be written from the
// schema or carried from the controller; anything else is reset to its zero
// value on every update. A go-unifi upgrade that adds such a field fails here.
func Test_carryUnexposedPortProfileFields_coversEveryAlwaysSentField(t *testing.T) {
	fields := alwaysSentPortProfileFields()

	// A profile with every always-sent field set, read into the model and
	// written back: the fields that survive are the ones the schema writes.
	var current unifi.PortProfile
	for _, f := range fields {
		reflect.ValueOf(&current).Elem().FieldByIndex(f.Index).Set(nonZero(t, f.Type))
	}
	r := &portProfileResource{}
	var model portProfileResourceModel
	if diags := r.portProfileToModel(t.Context(), &current, &model, "default"); diags.HasError() {
		t.Fatalf("portProfileToModel: %v", diags)
	}
	body, diags := r.modelToAPIPortProfile(t.Context(), &model)
	if diags.HasError() {
		t.Fatalf("modelToAPIPortProfile: %v", diags)
	}
	carried := *body
	carryUnexposedPortProfileFields(&current, &carried)

	unexposed := map[string]bool{}
	for _, k := range unexposedPortProfileFields {
		unexposed[k] = true
	}
	for name, f := range fields {
		written := !reflect.ValueOf(*body).FieldByIndex(f.Index).IsZero()
		kept := !reflect.ValueOf(carried).FieldByIndex(f.Index).IsZero()
		switch {
		case written && unexposed[name]:
			t.Errorf("%s is written from the schema; it must not be carried", name)
		case !kept:
			t.Errorf(
				"PortProfile.%s is always sent but neither written from the schema nor carried",
				name,
			)
		}
		delete(unexposed, name)
	}
	for k := range unexposed {
		t.Errorf("carried field %q is not an always-sent PortProfile field", k)
	}
}

// alwaysSentPortProfileFields returns the PortProfile fields without
// omitempty, by JSON name.
func alwaysSentPortProfileFields() map[string]reflect.StructField {
	fields := map[string]reflect.StructField{}
	typ := reflect.TypeFor[unifi.PortProfile]()
	for i := range typ.NumField() {
		f := typ.Field(i)
		name, opts, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name != "" && name != "-" && !strings.Contains(opts, "omitempty") {
			fields[name] = f
		}
	}
	return fields
}
