package unifi

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/cidrtypes"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

const networkTestID = "6650bb00000000000000a001"

// networkTestDocument is a default LAN with synthetic values. Like the
// controller's built-in network it is stored without enabled. go-unifi does
// not model external_id and routing_table_id.
const networkTestDocument = `{
	"_id": "6650bb00000000000000a001",
	"site_id": "6650aa00000000000000a000",
	"name": "Default",
	"purpose": "corporate",
	"networkgroup": "LAN",
	"attr_hidden_id": "LAN",
	"attr_no_delete": true,
	"ip_subnet": "198.51.100.1/24",
	"vlan_enabled": false,
	"domain_name": "localdomain",
	"is_nat": true,
	"internet_access_enabled": true,
	"network_isolation_enabled": false,
	"igmp_snooping": false,
	"mdns_enabled": true,
	"lte_lan_enabled": true,
	"auto_scale_enabled": true,
	"gateway_type": "default",
	"setting_preference": "auto",
	"dhcpguard_enabled": false,
	"dhcpd_enabled": true,
	"dhcpd_start": "198.51.100.6",
	"dhcpd_stop": "198.51.100.254",
	"dhcpd_leasetime": 86400,
	"dhcpd_dns_enabled": false,
	"dhcpd_gateway_enabled": false,
	"dhcpd_time_offset_enabled": false,
	"dhcpd_conflict_checking": true,
	"dhcpd_boot_enabled": true,
	"dhcpd_boot_server": "198.51.100.5",
	"dhcpd_boot_filename": "pxelinux.0",
	"ipv6_interface_type": "none",
	"ipv6_setting_preference": "auto",
	"external_id": "6650bb00-0000-0000-0000-00000000e001",
	"routing_table_id": 254
}`

func newNetworkHarness(t *testing.T, document string) *updateHarness {
	t.Helper()
	h := newUpdateHarness(t, "unifi_network", "networkconf", networkTestID, document,
		func(c *Client) harnessResource { return &networkResource{client: c} })
	h.importAndRead()
	return h
}

// networkBaseConfig matches networkTestDocument. subnet and dhcp_server are
// Optional-only, so leaving either out plans a change.
func networkBaseConfig() map[string]attr.Value {
	return map[string]attr.Value{
		"name":        types.StringValue("Default"),
		"subnet":      cidrtypes.NewIPv4PrefixValue("198.51.100.1/24"),
		"dhcp_server": networkDHCPServer(nil),
	}
}

// networkDHCPServer returns a dhcp_server value with the range of
// networkTestDocument and the given leaves set.
func networkDHCPServer(set map[string]attr.Value) attr.Value {
	vals := map[string]attr.Value{
		"boot":                types.ObjectNull(dhcpBootModel{}.AttributeTypes()),
		"enabled":             types.BoolNull(),
		"start":               types.StringValue("198.51.100.6"),
		"stop":                types.StringValue("198.51.100.254"),
		"gateway_enabled":     types.BoolNull(),
		"conflict_checking":   types.BoolNull(),
		"ntp":                 types.ObjectNull(dhcpServerOptionModel{}.AttributeTypes()),
		"time_offset_enabled": types.BoolNull(),
		"dns":                 types.ObjectNull(dhcpServerOptionModel{}.AttributeTypes()),
		"leasetime":           timetypes.NewGoDurationNull(),
		"wins":                types.ObjectNull(winsModel{}.AttributeTypes()),
		"wpad_url":            types.StringNull(),
		"tftp_server":         types.StringNull(),
		"unifi_controller":    types.StringNull(),
	}
	for k, v := range set {
		vals[k] = v
	}
	return types.ObjectValueMust(dhcpServerModel{}.AttributeTypes(), vals)
}

func Test_networkResource_Update(t *testing.T) {
	t.Run("default LAN import plans nothing", func(t *testing.T) {
		h := newNetworkHarness(t, networkTestDocument)
		h.assertEqual("plan after import", h.state.Raw, h.plan(h.config(networkBaseConfig())))
	})

	t.Run("enabled reads a missing key as true and keeps false", func(t *testing.T) {
		for doc, want := range map[string]bool{
			networkTestDocument: true,
			strings.Replace(networkTestDocument, `"is_nat"`, `"enabled": true, "is_nat"`, 1):  true,
			strings.Replace(networkTestDocument, `"is_nat"`, `"enabled": false, "is_nat"`, 1): false,
		} {
			h := newNetworkHarness(t, doc)
			var enabled types.Bool
			h.state.GetAttribute(h.ctx, path.Root("enabled"), &enabled)
			if enabled.ValueBool() != want || enabled.IsNull() {
				t.Errorf("enabled = %v, want %t", enabled, want)
			}
			cfg := networkBaseConfig()
			cfg["enabled"] = types.BoolValue(want)
			h.assertEqual("plan after import", h.state.Raw, h.plan(h.config(cfg)))
		}
	})

	t.Run("an unrelated change sends exactly that key", func(t *testing.T) {
		h := newNetworkHarness(t, networkTestDocument)
		cfg := networkBaseConfig()
		cfg["igmp_snooping"] = types.BoolValue(true)
		assertBodies(t, h.apply(h.config(cfg)), `{"igmp_snooping":true}`)
		for _, key := range []string{
			"enabled",
			"ipv6_ra_enabled",
			"ipv6_pd_auto_prefixid_enabled",
			"dhcpdv6_enabled",
		} {
			if got := h.fake.stored(key); got != nil {
				t.Errorf("stored %s = %v, want absent", key, got)
			}
		}
		if got := h.fake.stored("routing_table_id"); got != json.Number("254") {
			t.Errorf("stored routing_table_id = %v, want 254", got)
		}
	})

	t.Run("nothing to change sends no PUT", func(t *testing.T) {
		h := newNetworkHarness(t, networkTestDocument)
		cfg := networkBaseConfig()
		cfg["timeouts"] = types.ObjectValueMust(
			timeoutsNullValue().AttributeTypes(h.ctx),
			map[string]attr.Value{
				"create": types.StringNull(),
				"delete": types.StringNull(),
				"read":   types.StringNull(),
				"update": types.StringValue("30m"),
			},
		)
		assertBodies(t, h.apply(h.config(cfg)))
	})

	t.Run("an unknown planned value takes the prior value, never a zero", func(t *testing.T) {
		h := newNetworkHarness(t, networkTestDocument)
		cfg := networkBaseConfig()
		cfg["igmp_snooping"] = types.BoolValue(true)
		config := h.config(cfg)
		planned := tfsdk.Plan{Schema: h.schema, Raw: h.plan(config)}
		planned.SetAttribute(h.ctx, path.Root("multicast_dns"), types.BoolUnknown())
		if diags := h.update(planned.Raw, config); diags.HasError() {
			t.Fatalf("Update: %v", diags)
		}
		assertBodies(t, h.fake.putsSince(0), `{"igmp_snooping":true}`)
	})

	t.Run("an explicit false is sent", func(t *testing.T) {
		h := newNetworkHarness(t, networkTestDocument)
		cfg := networkBaseConfig()
		cfg["enabled"] = types.BoolValue(false)
		assertBodies(t, h.apply(h.config(cfg)), `{"enabled":false}`)
	})

	t.Run("a boot filename change sends the boot group", func(t *testing.T) {
		h := newNetworkHarness(t, networkTestDocument)
		cfg := networkBaseConfig()
		cfg["dhcp_server"] = networkDHCPServer(map[string]attr.Value{
			"boot": types.ObjectValueMust(dhcpBootModel{}.AttributeTypes(), map[string]attr.Value{
				"enabled":  types.BoolValue(true),
				"server":   types.StringValue("198.51.100.5"),
				"filename": types.StringValue("ipxe.efi"),
			}),
		})
		assertBodies(t, h.apply(h.config(cfg)),
			`{"dhcpd_boot_enabled":true,"dhcpd_boot_filename":"ipxe.efi",`+
				`"dhcpd_boot_server":"198.51.100.5"}`)
	})

	t.Run("unmanaged DHCP blocks keep the controller's values", func(t *testing.T) {
		document := strings.Replace(networkTestDocument, `"dhcpguard_enabled": false,`,
			`"dhcpguard_enabled": true, "dhcpd_ip_1": "198.51.100.2",`+
				` "dhcpd_dns_1": "198.51.100.53",`, 1)
		document = strings.Replace(document, `"dhcpd_dns_enabled": false`,
			`"dhcpd_dns_enabled": true`, 1)
		h := newNetworkHarness(t, document)
		cfg := networkBaseConfig()
		delete(cfg, "dhcp_server")
		assertBodies(t, h.apply(h.config(cfg)))

		cfg["igmp_snooping"] = types.BoolValue(true)
		assertBodies(t, h.apply(h.config(cfg)), `{"igmp_snooping":true}`)
		for key, want := range map[string]any{
			"dhcpguard_enabled": true,
			"dhcpd_ip_1":        "198.51.100.2",
			"dhcpd_dns_enabled": true,
			"dhcpd_dns_1":       "198.51.100.53",
			"dhcpd_enabled":     true,
		} {
			if got := h.fake.stored(key); got != want {
				t.Errorf("stored %s = %v, want %v", key, got, want)
			}
		}
	})

	t.Run("adding dhcp_server without a range sends the block with the derived range",
		func(t *testing.T) {
			document := strings.Replace(networkTestDocument, `"dhcpd_enabled": true,
	"dhcpd_start": "198.51.100.6",
	"dhcpd_stop": "198.51.100.254",
	"dhcpd_leasetime": 86400,`, `"dhcpd_enabled": false, "dhcpd_leasetime": 3600,`, 1)
			document = strings.Replace(document, `"dhcpd_conflict_checking": true`,
				`"dhcpd_conflict_checking": false`, 1)
			h := newNetworkHarness(t, document)
			cfg := networkBaseConfig()
			delete(cfg, "dhcp_server")
			h.assertEqual("plan after import", h.state.Raw, h.plan(h.config(cfg)))

			cfg["dhcp_server"] = networkDHCPServer(map[string]attr.Value{
				"enabled": types.BoolValue(true),
				"start":   types.StringNull(),
				"stop":    types.StringNull(),
			})
			config := h.config(cfg)
			planned := tfsdk.Plan{Schema: h.schema, Raw: h.plan(config)}
			for _, leaf := range []string{"start", "stop", "boot"} {
				var v attr.Value
				planned.GetAttribute(h.ctx, path.Root("dhcp_server").AtName(leaf), &v)
				if !v.IsUnknown() {
					t.Errorf("dhcp_server.%s planned %v, want unknown", leaf, v)
				}
			}

			assertBodies(t, h.apply(config),
				`{"dhcpd_conflict_checking":true,"dhcpd_dns_1":"","dhcpd_dns_2":"",`+
					`"dhcpd_dns_3":"","dhcpd_dns_4":"","dhcpd_dns_enabled":false,`+
					`"dhcpd_enabled":true,"dhcpd_gateway_enabled":false,"dhcpd_leasetime":86400,`+
					`"dhcpd_ntp_1":"","dhcpd_ntp_2":"","dhcpd_ntp_enabled":false,`+
					`"dhcpd_start":"198.51.100.6","dhcpd_stop":"198.51.100.254",`+
					`"dhcpd_tftp_server":"","dhcpd_time_offset_enabled":false,`+
					`"dhcpd_unifi_controller":"","dhcpd_wins_1":"","dhcpd_wins_2":"",`+
					`"dhcpd_wins_enabled":false,"dhcpd_wpad_url":""}`)
			for key, want := range map[string]any{
				"dhcpd_boot_enabled":  true,
				"dhcpd_boot_server":   "198.51.100.5",
				"dhcpd_boot_filename": "pxelinux.0",
			} {
				if got := h.fake.stored(key); got != want {
					t.Errorf("stored %s = %v, want %v", key, got, want)
				}
			}
			var boot types.Object
			h.state.GetAttribute(h.ctx, path.Root("dhcp_server").AtName("boot"), &boot)
			if got := boot.Attributes()["filename"]; !got.Equal(types.StringValue("pxelinux.0")) {
				t.Errorf("state dhcp_server.boot.filename = %v, want the controller's", got)
			}
		})

	t.Run("enabling DHCP relay without dhcp_server turns the DHCP server off", func(t *testing.T) {
		h := newNetworkHarness(t, networkTestDocument)
		cfg := networkBaseConfig()
		delete(cfg, "dhcp_server")
		cfg["dhcp_relay"] = types.ObjectValueMust(dhcpRelayModel{}.AttributeTypes(),
			map[string]attr.Value{
				"enabled": types.BoolValue(true),
				"servers": types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("198.51.100.9"),
				}),
			})
		assertBodies(t, h.apply(h.config(cfg)),
			`{"dhcp_relay_enabled":true,"dhcp_relay_servers":["198.51.100.9"],`+
				`"dhcpd_boot_enabled":false,"dhcpd_conflict_checking":false,"dhcpd_enabled":false,`+
				`"dhcpd_start":"198.51.100.6","dhcpd_stop":"198.51.100.254",`+
				`"setting_preference":"manual"}`)
	})

	t.Run("removing vlan fails before any request", func(t *testing.T) {
		document := strings.Replace(networkTestDocument, `"vlan_enabled": false,`,
			`"vlan_enabled": true, "vlan": 42,`, 1)
		h := newNetworkHarness(t, document)
		cfg := networkBaseConfig()
		cfg["vlan"] = types.Int64Value(42)
		h.assertEqual("plan after import", h.state.Raw, h.plan(h.config(cfg)))

		delete(cfg, "vlan")
		config := h.config(cfg)
		diags := h.update(h.plan(config), config)
		detail := diagDetails(diags)
		if !diags.HasError() || !strings.Contains(detail, "vlan cannot be removed") ||
			!strings.Contains(detail, "api.err.VlanUsed") {
			t.Fatalf("diagnostics = %v, want an error naming vlan and the controller rule", diags)
		}
		if n := h.fake.putCount(); n != 0 {
			t.Errorf("%d PUTs sent, want none", n)
		}
	})

	t.Run("a controller error names the sent keys", func(t *testing.T) {
		h := newNetworkHarness(t, networkTestDocument)
		h.fake.reject = func(map[string]any) string { return "api.err.InvalidPayload" }
		cfg := networkBaseConfig()
		cfg["igmp_snooping"] = types.BoolValue(true)
		config := h.config(cfg)
		detail := diagDetails(h.update(h.plan(config), config))
		for _, want := range []string{
			"api.err.InvalidPayload",
			"The request set igmp_snooping.",
			"merged",
		} {
			if !strings.Contains(detail, want) {
				t.Errorf("diagnostic %q lacks %q", detail, want)
			}
		}
	})
}

// encodedKeys returns the top-level keys of the SDK encoding of a network with
// the given purpose whose every field holds a non-zero value.
func encodedKeys(t *testing.T, purpose string) map[string]bool {
	t.Helper()
	var n unifi.Network
	fillNonZero(reflect.ValueOf(&n).Elem())
	n.Purpose = purpose
	b, err := json.Marshal(&n)
	if err != nil {
		t.Fatalf("encoding %s network: %v", purpose, err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("decoding %s network: %v", purpose, err)
	}
	keys := make(map[string]bool, len(m))
	for k := range m {
		keys[k] = true
	}
	return keys
}

func fillNonZero(v reflect.Value) {
	switch v.Kind() {
	case reflect.Struct:
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				fillNonZero(v.Field(i))
			}
		}
	case reflect.Pointer:
		v.Set(reflect.New(v.Type().Elem()))
		fillNonZero(v.Elem())
	case reflect.Slice:
		v.Set(reflect.MakeSlice(v.Type(), 1, 1))
		fillNonZero(v.Index(0))
	case reflect.String:
		v.SetString("x")
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int64:
		v.SetInt(1)
	case reflect.Float64:
		v.SetFloat(1)
	default:
		panic(fmt.Sprintf("fillNonZero: unhandled kind %s", v.Kind()))
	}
}

// Every group key and every dhcp_server key must be a key of the go-unifi
// corporate encoding, or Update silently ignores it.
func TestNetworkDiffOptionsKeysExist(t *testing.T) {
	keys := encodedKeys(t, unifi.PurposeCorporate)
	for _, group := range slices.Concat(networkDiffOptions.Groups, [][]string{dhcpServerKeys, dhcpBootKeys}) {
		for _, k := range group {
			if !keys[k] {
				t.Errorf("%q is not a key of the corporate network encoding", k)
			}
		}
	}
}

// TestAccNetworkFramework_sparseUpdate creates a network, changes one attribute
// and imports it, and expects an empty plan after each apply.
func TestAccNetworkFramework_sparseUpdate(t *testing.T) {
	emptyPlan := resource.ConfigPlanChecks{
		PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:           testAccNetworkFrameworkConfig_sparseUpdate(false),
				ConfigPlanChecks: emptyPlan,
			},
			{
				Config:           testAccNetworkFrameworkConfig_sparseUpdate(true),
				ConfigPlanChecks: emptyPlan,
				Check: resource.TestCheckResourceAttr(
					"unifi_network.test_sparse", "igmp_snooping", "true"),
			},
			{
				ResourceName:            "unifi_network.test_sparse",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"dhcp_server", "dhcp_relay"},
			},
		},
	})
}

func testAccNetworkFrameworkConfig_sparseUpdate(igmpSnooping bool) string {
	return fmt.Sprintf(`
resource "unifi_network" "test_sparse" {
	name          = "tfacc-network-sparse"
	subnet        = "192.168.13.1/24"
	vlan          = 13
	igmp_snooping = %t
}
`, igmpSnooping)
}
