package unifi

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// See TestNetworkSchemaCarriesNoStaticDefaults (#544).
func TestWANSchemaCarriesNoStaticDefaults(t *testing.T) {
	assertNoStaticDefaults(t, "unifi_wan", &wanResource{})
}

// An imported WAN whose type, VLAN, egress QoS, Smart Queue and enabled flag
// differ from the old schema defaults keeps them when the configuration
// declares only part of each block.
func TestWANAdoptedKeepsControllerValues(t *testing.T) {
	doc := strings.Replace(wanTestDocument, `"wan_vlan_enabled": false,`,
		`"wan_vlan_enabled": true, "wan_vlan": 10, "wan_egress_qos_enabled": true, `+
			`"wan_egress_qos": "3", "enabled": false,`, 1)
	if doc == wanTestDocument {
		t.Fatal("wanTestDocument no longer contains wan_vlan_enabled")
	}
	h := newWANHarnessFrom(t, doc)
	cfg := wanBaseConfig()
	delete(cfg, "type")
	cfg["vlan"] = types.ObjectValueMust(vlanModel{}.AttributeTypes(), map[string]attr.Value{
		"enabled": types.BoolValue(true),
		"id":      types.Int64Null(),
	})
	cfg["egress_qos"] = types.ObjectValueMust(egressQosModel{}.AttributeTypes(),
		map[string]attr.Value{
			"enabled":  types.BoolNull(),
			"priority": types.Int64Value(3),
		})
	cfg["smartq"] = types.ObjectValueMust(smartqModel{}.AttributeTypes(), map[string]attr.Value{
		"enabled":   types.BoolNull(),
		"up_rate":   types.Int64Value(20000),
		"down_rate": types.Int64Value(90000),
	})
	h.assertEqual("plan after import", h.state.Raw, h.plan(h.config(cfg)))

	cfg["report_wan_event"] = types.BoolValue(true)
	assertBodies(t, h.apply(h.config(cfg)), `{"report_wan_event":true}`)
}
