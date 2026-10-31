package unifi

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// #544: a static schema Default is applied before plan modifiers run, so an
// attribute the configuration leaves out is planned at the constant instead of
// the controller's value. unifi_network and unifi_wan must carry none: every
// attribute either reads the controller (UseStateForUnknown) or defaults only
// when no prior value exists (boolDefaultOnCreate / StringDefaultOnCreate).
func TestNetworkSchemaCarriesNoStaticDefaults(t *testing.T) {
	assertNoStaticDefaults(t, "unifi_network", &networkResource{})
}

// assertNoStaticDefaults fails for every attribute of r, nested ones included,
// that carries a static schema Default.
func assertNoStaticDefaults(t *testing.T, name string, r resource.Resource) {
	t.Helper()
	ctx := context.Background()
	var resp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("%s schema: %v", name, resp.Diagnostics)
	}
	var walk func(prefix string, attrs map[string]schema.Attribute)
	walk = func(prefix string, attrs map[string]schema.Attribute) {
		for k, a := range attrs {
			if _, ok := attributeDefault(ctx, a); ok {
				t.Errorf("%s.%s%s carries a static Default (#544)", name, prefix, k)
			}
			if single, ok := a.(schema.SingleNestedAttribute); ok {
				walk(prefix+k+".", single.Attributes)
			}
		}
	}
	walk("", resp.Schema.Attributes)
}

// networkAdoptedDocument is networkTestDocument with every formerly defaulted
// attribute set away from the old schema default, as a network configured in
// the UI would be.
func networkAdoptedDocument() string {
	doc := networkTestDocument
	for old, repl := range map[string]string{
		`"is_nat"`:                           `"enabled": false, "is_nat"`,
		`"internet_access_enabled": true`:    `"internet_access_enabled": false`,
		`"network_isolation_enabled": false`: `"network_isolation_enabled": true`,
		`"igmp_snooping": false`:             `"igmp_snooping": true`,
		`"dhcpd_leasetime": 86400`:           `"dhcpd_leasetime": 3600`,
		`"dhcpd_dns_enabled": false`: `"dhcpd_dns_enabled": true, "dhcpd_dns_1": "198.51.100.53", ` +
			`"dhcpd_ntp_enabled": true, "dhcpd_ntp_1": "198.51.100.123", ` +
			`"dhcpd_wins_enabled": true, "dhcpd_wins_1": "198.51.100.137"`,
		`"dhcpd_gateway_enabled": false`:     `"dhcpd_gateway_enabled": true`,
		`"dhcpd_time_offset_enabled": false`: `"dhcpd_time_offset_enabled": true`,
		`"dhcpd_conflict_checking": true`:    `"dhcpd_conflict_checking": false`,
		`"dhcpguard_enabled": false`:         `"dhcpguard_enabled": true, "dhcpd_ip_1": "198.51.100.1"`,
		// Guarding on a corporate network needs manual; every live LAN is.
		`"setting_preference": "auto"`: `"setting_preference": "manual"`,
	} {
		if !strings.Contains(doc, old) {
			panic("networkTestDocument no longer contains " + old)
		}
		doc = strings.Replace(doc, old, repl, 1)
	}
	return doc
}

// An imported network whose configuration declares only the DHCP range (and
// the guarding block, without enabled) must plan no change, and an unrelated
// edit must send only that key. Before #544 was fixed here the plan proposed
// re-enabling the network and its internet access, disabling IGMP snooping,
// isolation, DHCP gateway/time offset/DNS/NTP/WINS and guarding, re-enabling
// conflict checking and resetting the lease to 24h.
func TestNetworkAdoptedKeepsControllerValues(t *testing.T) {
	h := newNetworkHarness(t, networkAdoptedDocument())
	cfg := networkBaseConfig()
	cfg["dhcp_guarding"] = types.ObjectValueMust(dhcpGuardingModel{}.AttributeTypes(),
		map[string]attr.Value{
			"enabled": types.BoolNull(),
			"servers": types.ListValueMust(types.StringType,
				[]attr.Value{types.StringValue("198.51.100.1")}),
		})
	h.assertEqual("plan after import", h.state.Raw, h.plan(h.config(cfg)))

	cfg["domain_name"] = types.StringValue("lan.example")
	assertBodies(t, h.apply(h.config(cfg)), `{"domain_name":"lan.example"}`)
}

// A new network still gets the values it was always created with: internet
// access, a DHCP server with conflict checking and a 24h lease, and the
// default gateway type.
func TestNetworkCreateKeepsCreateTimeDefaults(t *testing.T) {
	ctx := context.Background()
	h := newUpdateHarness(t, "unifi_network", "networkconf", networkTestID,
		networkTestDocument,
		func(c *Client) harnessResource { return &networkResource{client: c} })
	config := h.config(networkBaseConfig())
	planned := planResourceChange(t, "unifi_network", h.schema,
		tftypes.NewValue(h.schema.Type().TerraformType(ctx), nil), config, config)

	var attrs map[string]tftypes.Value
	if err := planned.As(&attrs); err != nil {
		t.Fatalf("as object: %v", err)
	}
	dhcp := plannedObject(t, attrs["dhcp_server"], "dhcp_server")
	for name, v := range map[string]tftypes.Value{
		"internet_access":               attrs["internet_access"],
		"dhcp_server.enabled":           dhcp["enabled"],
		"dhcp_server.conflict_checking": dhcp["conflict_checking"],
	} {
		var b bool
		if err := v.As(&b); err != nil || !b {
			t.Errorf("create planned %s = %v (%v), want true", name, v, err)
		}
	}
	for name, want := range map[string]struct {
		v    tftypes.Value
		want string
	}{
		"gateway_type":          {attrs["gateway_type"], "default"},
		"dhcp_server.leasetime": {dhcp["leasetime"], timetypes.NewGoDurationValueFromStringMust("24h0m0s").ValueString()},
	} {
		var s string
		if err := want.v.As(&s); err != nil || s != want.want {
			t.Errorf("create planned %s = %v (%v), want %q", name, want.v, err, want.want)
		}
	}
	// Attributes with no create-time default are left to the controller.
	for _, name := range []string{"enabled", "igmp_snooping", "network_isolation"} {
		if attrs[name].IsKnown() {
			t.Errorf(
				"create planned %s = %v, want unknown (from the controller)",
				name,
				attrs[name],
			)
		}
	}
}
