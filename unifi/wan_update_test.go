package unifi

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

const wanTestID = "6650cc00000000000000a001"

// wanTestDocument is a PPPoE uplink with synthetic values, stored without
// enabled, wan_vlan, wan_dhcp_options and wan_ip_aliases. go-unifi does not
// model routing_table_id.
const wanTestDocument = `{
	"_id": "6650cc00000000000000a001",
	"site_id": "6650aa00000000000000a000",
	"name": "Internet 1",
	"purpose": "wan",
	"wan_networkgroup": "WAN",
	"attr_hidden_id": "WAN",
	"attr_no_delete": true,
	"wan_type": "pppoe",
	"wan_username": "isp-user",
	"x_wan_password": "isp-secret",
	"wan_type_v6": "disabled",
	"wan_vlan_enabled": false,
	"wan_dns_preference": "auto",
	"wan_load_balance_type": "failover-only",
	"wan_load_balance_weight": 50,
	"wan_failover_priority": 1,
	"wan_smartq_enabled": true,
	"wan_smartq_up_rate": 20000,
	"wan_smartq_down_rate": 90000,
	"igmp_proxy_upstream": false,
	"report_wan_event": false,
	"mac_override_enabled": false,
	"wan_provider_capabilities": {
		"download_kilobits_per_second": 100000,
		"upload_kilobits_per_second": 40000
	},
	"routing_table_id": 201
}`

func newWANHarness(t *testing.T) *updateHarness {
	t.Helper()
	return newWANHarnessFrom(t, wanTestDocument)
}

func newWANHarnessFrom(t *testing.T, document string) *updateHarness {
	t.Helper()
	h := newUpdateHarness(t, "unifi_wan", "networkconf", wanTestID, document,
		func(c *Client) harnessResource { return &wanResource{client: c} })
	h.importAndRead()
	return h
}

// wanBaseConfig matches wanTestDocument. username is Optional-only, and type
// defaults to dhcp.
func wanBaseConfig() map[string]attr.Value {
	return map[string]attr.Value{
		"name":     types.StringValue("Internet 1"),
		"type":     types.StringValue("pppoe"),
		"username": types.StringValue("isp-user"),
	}
}

func Test_wanResource_Update(t *testing.T) {
	t.Run("import plans nothing", func(t *testing.T) {
		h := newWANHarness(t)
		h.assertEqual("plan after import", h.state.Raw, h.plan(h.config(wanBaseConfig())))
	})

	t.Run("an unrelated change sends exactly that key", func(t *testing.T) {
		h := newWANHarness(t)
		cfg := wanBaseConfig()
		cfg["report_wan_event"] = types.BoolValue(true)
		assertBodies(t, h.apply(h.config(cfg)), `{"report_wan_event":true}`)
		for _, key := range []string{"enabled", "wan_vlan", "wan_dhcp_options", "wan_ip_aliases"} {
			if got := h.fake.stored(key); got != nil {
				t.Errorf("stored %s = %v, want absent", key, got)
			}
		}
		for key, want := range map[string]any{
			"routing_table_id": json.Number("201"),
			"x_wan_password":   "isp-secret",
		} {
			if got := h.fake.stored(key); got != want {
				t.Errorf("stored %s = %v, want %v", key, got, want)
			}
		}
	})

	t.Run("nothing to change sends no PUT", func(t *testing.T) {
		h := newWANHarness(t)
		cfg := wanBaseConfig()
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
		h := newWANHarness(t)
		cfg := wanBaseConfig()
		cfg["report_wan_event"] = types.BoolValue(true)
		config := h.config(cfg)
		planned := tfsdk.Plan{Schema: h.schema, Raw: h.plan(config)}
		planned.SetAttribute(h.ctx, path.Root("enabled"), types.BoolUnknown())
		if diags := h.update(planned.Raw, config); diags.HasError() {
			t.Fatalf("Update: %v", diags)
		}
		assertBodies(t, h.fake.putsSince(0), `{"report_wan_event":true}`)
	})

	t.Run("an explicit false is sent", func(t *testing.T) {
		h := newWANHarness(t)
		cfg := wanBaseConfig()
		cfg["enabled"] = types.BoolValue(false)
		assertBodies(t, h.apply(h.config(cfg)), `{"enabled":false}`)
	})

	t.Run("username, password and networkgroup changes reach the controller", func(t *testing.T) {
		h := newWANHarness(t)
		cfg := wanBaseConfig()
		cfg["username"] = types.StringValue("isp-user-2")
		assertBodies(t, h.apply(h.config(cfg)), `{"wan_username":"isp-user-2"}`)

		cfg["password"] = types.StringValue("isp-secret-2")
		assertBodies(t, h.apply(h.config(cfg)), `{"x_wan_password":"isp-secret-2"}`)

		delete(cfg, "password")
		assertBodies(t, h.apply(h.config(cfg)))
		if got := h.fake.stored("x_wan_password"); got != "isp-secret-2" {
			t.Errorf("stored x_wan_password = %v, want isp-secret-2 kept", got)
		}

		cfg["networkgroup"] = types.StringValue("WAN2")
		assertBodies(t, h.apply(h.config(cfg)),
			`{"attr_hidden_id":"WAN2","wan_networkgroup":"WAN2"}`)
	})

	t.Run("password_wo is sent with every update while set", func(t *testing.T) {
		h := newWANHarness(t)
		cfg := wanBaseConfig()
		cfg["password_wo"] = types.StringValue("isp-secret-wo")
		cfg["report_wan_event"] = types.BoolValue(true)
		assertBodies(t, h.apply(h.config(cfg)),
			`{"report_wan_event":true,"x_wan_password":"isp-secret-wo"}`)
		var password types.String
		h.state.GetAttribute(h.ctx, path.Root("password_wo"), &password)
		if !password.IsNull() {
			t.Errorf("state password_wo = %v, want null", password)
		}
	})

	t.Run("a nested value is sent whole", func(t *testing.T) {
		h := newWANHarness(t)
		cfg := wanBaseConfig()
		cfg["provider_capabilities"] = types.ObjectValueMust(
			providerCapabilitiesModel{}.AttributeTypes(),
			map[string]attr.Value{
				"download_kilobits_per_second": types.Int64Value(250000),
				"upload_kilobits_per_second":   types.Int64Null(),
			},
		)
		assertBodies(t, h.apply(h.config(cfg)),
			`{"wan_provider_capabilities":{"download_kilobits_per_second":250000,`+
				`"upload_kilobits_per_second":40000}}`)
	})

	t.Run("a SmartQ rate change sends the SmartQ group", func(t *testing.T) {
		h := newWANHarness(t)
		cfg := wanBaseConfig()
		cfg["smartq"] = types.ObjectValueMust(smartqModel{}.AttributeTypes(),
			map[string]attr.Value{
				"enabled":   types.BoolValue(true),
				"up_rate":   types.Int64Value(20000),
				"down_rate": types.Int64Value(100000),
			})
		assertBodies(t, h.apply(h.config(cfg)),
			`{"wan_smartq_down_rate":100000,"wan_smartq_enabled":true,"wan_smartq_up_rate":20000}`)
	})

	// go-unifi does not encode the DS-Lite keys for a WAN, so the controller
	// keeps its value and only state follows the plan (#281).
	t.Run("planned DS-Lite values stay in state", func(t *testing.T) {
		h := newWANHarness(t)
		cfg := wanBaseConfig()
		cfg["dslite"] = types.ObjectValueMust(wanDsliteModel{}.AttributeTypes(),
			map[string]attr.Value{
				"remote_host":      types.StringValue("aftr.example.net"),
				"remote_host_auto": types.BoolValue(false),
			})
		config := h.config(cfg)
		planned := h.plan(config)
		if diags := h.update(planned, config); diags.HasError() {
			t.Fatalf("Update: %v", diags)
		}
		h.assertEqual("state after Update differs from the plan", planned, h.state.Raw)
	})

	t.Run("adding upnp, load_balance and dns sends only their known values", func(t *testing.T) {
		document := wanTestDocument
		for _, line := range []string{
			`"wan_dns_preference": "auto",`,
			`"wan_load_balance_type": "failover-only",`,
			`"wan_load_balance_weight": 50,`,
			`"wan_failover_priority": 1,`,
		} {
			document = strings.Replace(document, line, "", 1)
		}
		h := newWANHarnessFrom(t, document)
		for _, name := range []string{"upnp", "load_balance", "dns"} {
			var v types.Object
			h.state.GetAttribute(h.ctx, path.Root(name), &v)
			if !v.IsNull() {
				t.Fatalf("%s after import = %v, want null", name, v)
			}
		}
		cfg := wanBaseConfig()

		cfg["upnp"] = types.ObjectValueMust(upnpModel{}.AttributeTypes(), map[string]attr.Value{
			"enabled":         types.BoolValue(true),
			"wan_interface":   types.StringNull(),
			"nat_pmp_enabled": types.BoolNull(),
			"secure_mode":     types.BoolNull(),
		})
		assertBodies(t, h.apply(h.config(cfg)), `{"upnp_enabled":true}`)

		cfg["load_balance"] = types.ObjectValueMust(loadBalanceModel{}.AttributeTypes(),
			map[string]attr.Value{
				"type":              types.StringValue("failover-only"),
				"weight":            types.Int64Null(),
				"failover_priority": types.Int64Value(2),
			})
		assertBodies(t, h.apply(h.config(cfg)),
			`{"wan_failover_priority":2,"wan_load_balance_type":"failover-only"}`)

		cfg["dns"] = types.ObjectValueMust(dnsModel{}.AttributeTypes(), map[string]attr.Value{
			"primary":    types.StringValue("198.51.100.53"),
			"secondary":  types.StringNull(),
			"preference": types.StringNull(),
			"ipv6":       types.ObjectNull(dnsIPv6Model{}.AttributeTypes()),
		})
		config := h.config(cfg)
		var ipv6 types.Object
		planned := tfsdk.Plan{Schema: h.schema, Raw: h.plan(config)}
		planned.GetAttribute(h.ctx, path.Root("dns").AtName("ipv6"), &ipv6)
		if !ipv6.IsUnknown() {
			t.Fatalf("dns.ipv6 planned %v, want unknown", ipv6)
		}
		assertBodies(t, h.apply(config), `{"wan_dns1":"198.51.100.53"}`)
		for _, key := range []string{
			"upnp_wan_interface",
			"upnp_nat_pmp_enabled",
			"upnp_secure_mode",
			"wan_load_balance_weight",
			"wan_dns2",
			"wan_dns_preference",
			"wan_ipv6_dns1",
			"wan_ipv6_dns2",
			"wan_ipv6_dns_preference",
		} {
			if got := h.fake.stored(key); got != nil {
				t.Errorf("stored %s = %v, want absent", key, got)
			}
		}
	})

	t.Run("a controller error names the sent keys, not their values", func(t *testing.T) {
		h := newWANHarness(t)
		h.fake.reject = func(map[string]any) string { return "api.err.InvalidPayload" }
		cfg := wanBaseConfig()
		cfg["password"] = types.StringValue("isp-secret-2")
		config := h.config(cfg)
		detail := diagDetails(h.update(h.plan(config), config))
		for _, want := range []string{
			"api.err.InvalidPayload",
			"The request set x_wan_password.",
			"merged",
		} {
			if !strings.Contains(detail, want) {
				t.Errorf("diagnostic %q lacks %q", detail, want)
			}
		}
		if strings.Contains(detail, "isp-secret-2") {
			t.Errorf("diagnostic leaks the password: %q", detail)
		}
	})
}

// Every group key must be a key of the go-unifi WAN encoding, or Diff silently
// ignores it.
func TestWANDiffOptionsKeysExist(t *testing.T) {
	keys := encodedKeys(t, unifi.PurposeWAN)
	for _, group := range wanDiffOptions.Groups {
		for _, k := range group {
			if !keys[k] {
				t.Errorf("%q is not a key of the WAN network encoding", k)
			}
		}
	}
}

// TestAccWANFramework_sparseUpdate adopts the WAN, changes one attribute and
// imports it, and expects an empty plan after each apply.
func TestAccWANFramework_sparseUpdate(t *testing.T) {
	emptyPlan := resource.ConfigPlanChecks{
		PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:           testAccWANFrameworkConfig_sparseUpdate(false),
				ConfigPlanChecks: emptyPlan,
			},
			{
				Config:           testAccWANFrameworkConfig_sparseUpdate(true),
				ConfigPlanChecks: emptyPlan,
				Check: resource.TestCheckResourceAttr(
					"unifi_wan.test_sparse", "report_wan_event", "true"),
			},
			{
				ResourceName:      "unifi_wan.test_sparse",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccWANFrameworkConfig_sparseUpdate(reportWANEvent bool) string {
	return fmt.Sprintf(`
resource "unifi_wan" "test_sparse" {
	name             = "test-wan-sparse"
	type             = "dhcp"
	report_wan_event = %t
}
`, reportWANEvent)
}
