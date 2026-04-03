package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFirewallRule_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallRuleConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_firewall_rule.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"name",
						"tfacc-firewall-rule",
					),
					resource.TestCheckResourceAttr("unifi_firewall_rule.test", "action", "drop"),
					resource.TestCheckResourceAttr("unifi_firewall_rule.test", "ruleset", "LAN_IN"),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"rule_index",
						"2000",
					),
					resource.TestCheckResourceAttr("unifi_firewall_rule.test", "enabled", "true"),
					resource.TestCheckResourceAttr("unifi_firewall_rule.test", "logging", "false"),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"state_established",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"state_invalid",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"state_new",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"state_related",
						"false",
					),
				),
			},
			{
				ResourceName:      "unifi_firewall_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccFirewallRule_accept(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallRuleConfig_accept(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_firewall_rule.test", "id"),
					resource.TestCheckResourceAttr("unifi_firewall_rule.test", "action", "accept"),
					resource.TestCheckResourceAttr("unifi_firewall_rule.test", "ruleset", "WAN_IN"),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"rule_index",
						"2010",
					),
				),
			},
			{
				ResourceName:      "unifi_firewall_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccFirewallRule_reject(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallRuleConfig_reject(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_firewall_rule.test", "id"),
					resource.TestCheckResourceAttr("unifi_firewall_rule.test", "action", "reject"),
					resource.TestCheckResourceAttr("unifi_firewall_rule.test", "ruleset", "LAN_IN"),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"rule_index",
						"2020",
					),
				),
			},
		},
	})
}

func TestAccFirewallRule_disabled(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallRuleConfig_disabled(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_firewall_rule.test", "id"),
					resource.TestCheckResourceAttr("unifi_firewall_rule.test", "enabled", "false"),
				),
			},
			{
				ResourceName:      "unifi_firewall_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccFirewallRule_withLogging(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallRuleConfig_withLogging(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_firewall_rule.test", "id"),
					resource.TestCheckResourceAttr("unifi_firewall_rule.test", "logging", "true"),
				),
			},
			{
				ResourceName:      "unifi_firewall_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccFirewallRule_withStateMatching(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallRuleConfig_withStateMatching(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_firewall_rule.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"state_established",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"state_related",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"state_new",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"state_invalid",
						"false",
					),
				),
			},
			{
				ResourceName:      "unifi_firewall_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccFirewallRule_withProtocol(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallRuleConfig_withProtocol(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_firewall_rule.test", "id"),
					resource.TestCheckResourceAttr("unifi_firewall_rule.test", "protocol", "tcp"),
					resource.TestCheckResourceAttr("unifi_firewall_rule.test", "dst_port", "443"),
				),
			},
			{
				ResourceName:      "unifi_firewall_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccFirewallRule_withSrcAddress(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallRuleConfig_withSrcAddress(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_firewall_rule.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"src_address",
						"10.0.0.0/8",
					),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"src_network_type",
						"NETv4",
					),
				),
			},
			{
				ResourceName:      "unifi_firewall_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccFirewallRule_withDstAddress(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallRuleConfig_withDstAddress(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_firewall_rule.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"dst_address",
						"192.168.0.0/16",
					),
				),
			},
			{
				ResourceName:      "unifi_firewall_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccFirewallRule_withFirewallGroups(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallRuleConfig_withFirewallGroups(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_firewall_rule.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"src_firewall_group_ids.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"dst_firewall_group_ids.#",
						"1",
					),
				),
			},
			{
				ResourceName:      "unifi_firewall_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccFirewallRule_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallRuleConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"name",
						"tfacc-firewall-rule",
					),
					resource.TestCheckResourceAttr("unifi_firewall_rule.test", "action", "drop"),
					resource.TestCheckResourceAttr("unifi_firewall_rule.test", "enabled", "true"),
				),
			},
			{
				Config: testAccFirewallRuleConfig_updated(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"name",
						"tfacc-firewall-rule-updated",
					),
					resource.TestCheckResourceAttr("unifi_firewall_rule.test", "action", "accept"),
					resource.TestCheckResourceAttr("unifi_firewall_rule.test", "enabled", "false"),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"rule_index",
						"2001",
					),
				),
			},
		},
	})
}

func TestAccFirewallRule_highRuleIndex(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallRuleConfig_highRuleIndex(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_firewall_rule.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"rule_index",
						"4000",
					),
				),
			},
			{
				ResourceName:      "unifi_firewall_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccFirewallRule_guestRuleset(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallRuleConfig_guestRuleset(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_firewall_rule.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"ruleset",
						"GUEST_IN",
					),
				),
			},
		},
	})
}

func testAccFirewallRuleConfig_basic() string {
	return `
resource "unifi_firewall_rule" "test" {
  name       = "tfacc-firewall-rule"
  action     = "drop"
  ruleset    = "LAN_IN"
  rule_index = 2000

  logging           = false
  state_established = false
  state_invalid     = false
  state_new         = false
  state_related     = false
}
`
}

func testAccFirewallRuleConfig_accept() string {
	return `
resource "unifi_firewall_rule" "test" {
  name       = "tfacc-firewall-rule-accept"
  action     = "accept"
  ruleset    = "WAN_IN"
  rule_index = 2010

  logging           = false
  state_established = false
  state_invalid     = false
  state_new         = false
  state_related     = false
}
`
}

func testAccFirewallRuleConfig_reject() string {
	return `
resource "unifi_firewall_rule" "test" {
  name       = "tfacc-firewall-rule-reject"
  action     = "reject"
  ruleset    = "LAN_IN"
  rule_index = 2020

  logging           = false
  state_established = false
  state_invalid     = false
  state_new         = false
  state_related     = false
}
`
}

func testAccFirewallRuleConfig_disabled() string {
	return `
resource "unifi_firewall_rule" "test" {
  name       = "tfacc-firewall-rule-disabled"
  action     = "drop"
  ruleset    = "LAN_IN"
  rule_index = 2030
  enabled    = false

  logging           = false
  state_established = false
  state_invalid     = false
  state_new         = false
  state_related     = false
}
`
}

func testAccFirewallRuleConfig_withLogging() string {
	return `
resource "unifi_firewall_rule" "test" {
  name       = "tfacc-firewall-rule-logging"
  action     = "drop"
  ruleset    = "LAN_IN"
  rule_index = 2040
  logging    = true

  state_established = false
  state_invalid     = false
  state_new         = false
  state_related     = false
}
`
}

func testAccFirewallRuleConfig_withStateMatching() string {
	return `
resource "unifi_firewall_rule" "test" {
  name       = "tfacc-firewall-rule-state"
  action     = "accept"
  ruleset    = "WAN_IN"
  rule_index = 2050

  logging           = false
  state_established = true
  state_related     = true
  state_new         = false
  state_invalid     = false
}
`
}

func testAccFirewallRuleConfig_withProtocol() string {
	return `
resource "unifi_firewall_rule" "test" {
  name       = "tfacc-firewall-rule-proto"
  action     = "accept"
  ruleset    = "WAN_IN"
  rule_index = 2060
  protocol   = "tcp"
  dst_port   = "443"

  logging           = false
  state_established = false
  state_invalid     = false
  state_new         = false
  state_related     = false
}
`
}

func testAccFirewallRuleConfig_withSrcAddress() string {
	return `
resource "unifi_firewall_rule" "test" {
  name            = "tfacc-firewall-rule-src"
  action          = "drop"
  ruleset         = "LAN_IN"
  rule_index      = 2070
  src_address     = "10.0.0.0/8"
  src_network_type = "NETv4"

  logging           = false
  state_established = false
  state_invalid     = false
  state_new         = false
  state_related     = false
}
`
}

func testAccFirewallRuleConfig_withDstAddress() string {
	return `
resource "unifi_firewall_rule" "test" {
  name       = "tfacc-firewall-rule-dst"
  action     = "drop"
  ruleset    = "LAN_IN"
  rule_index = 2080
  dst_address = "192.168.0.0/16"

  logging           = false
  state_established = false
  state_invalid     = false
  state_new         = false
  state_related     = false
}
`
}

func testAccFirewallRuleConfig_withFirewallGroups() string {
	return `
resource "unifi_firewall_group" "src" {
  name    = "tfacc-fwrule-src-group"
  type    = "address-group"
  members = ["10.0.0.1"]
}

resource "unifi_firewall_group" "dst" {
  name    = "tfacc-fwrule-dst-group"
  type    = "address-group"
  members = ["192.168.1.1"]
}

resource "unifi_firewall_rule" "test" {
  name       = "tfacc-firewall-rule-groups"
  action     = "drop"
  ruleset    = "LAN_IN"
  rule_index = 2090

  src_firewall_group_ids = [unifi_firewall_group.src.id]
  dst_firewall_group_ids = [unifi_firewall_group.dst.id]

  logging           = false
  state_established = false
  state_invalid     = false
  state_new         = false
  state_related     = false
}
`
}

func testAccFirewallRuleConfig_updated() string {
	return `
resource "unifi_firewall_rule" "test" {
  name       = "tfacc-firewall-rule-updated"
  action     = "accept"
  ruleset    = "LAN_IN"
  rule_index = 2001
  enabled    = false

  logging           = false
  state_established = false
  state_invalid     = false
  state_new         = false
  state_related     = false
}
`
}

func testAccFirewallRuleConfig_highRuleIndex() string {
	return `
resource "unifi_firewall_rule" "test" {
  name       = "tfacc-firewall-rule-high-idx"
  action     = "drop"
  ruleset    = "LAN_IN"
  rule_index = 4000

  logging           = false
  state_established = false
  state_invalid     = false
  state_new         = false
  state_related     = false
}
`
}

func testAccFirewallRuleConfig_guestRuleset() string {
	return `
resource "unifi_firewall_rule" "test" {
  name       = "tfacc-firewall-rule-guest"
  action     = "drop"
  ruleset    = "GUEST_IN"
  rule_index = 2000

  logging           = false
  state_established = false
  state_invalid     = false
  state_new         = false
  state_related     = false
}
`
}
