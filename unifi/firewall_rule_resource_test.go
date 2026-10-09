package unifi

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/hwtypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// firewallRuleTestObject returns base with overrides applied to its attributes.
func firewallRuleTestObject(
	t *testing.T,
	base types.Object,
	overrides map[string]attr.Value,
) types.Object {
	t.Helper()
	attrs := base.Attributes()
	for k, v := range overrides {
		attrs[k] = v
	}
	obj, d := types.ObjectValue(base.AttributeTypes(context.Background()), attrs)
	if d.HasError() {
		t.Fatalf("building test object: %v", d)
	}
	return obj
}

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
						"state.established",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"state.invalid",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"state.new",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"state.related",
						"false",
					),
				),
			},
			{
				ResourceName:      "unifi_firewall_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:    "unifi_firewall_rule.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
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
						"state.established",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"state.related",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"state.new",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"state.invalid",
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
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"destination.port",
						"443",
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
						"source.address",
						"10.0.0.0/8",
					),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"source.network_type",
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
						"destination.address",
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
						"source.firewall_group_ids.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"destination.firewall_group_ids.#",
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

func TestAccFirewallRule_withSrcMac(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallRuleConfig_withSrcMac(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_firewall_rule.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"source.mac",
						"00:11:22:33:44:55",
					),
					resource.TestCheckResourceAttr(
						"unifi_firewall_rule.test",
						"protocol_match_excepted",
						"true",
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

func testAccFirewallRuleConfig_withSrcMac() string {
	return `
resource "unifi_firewall_rule" "test" {
  name       = "tfacc-firewall-rule-src-mac"
  action     = "drop"
  ruleset    = "LAN_IN"
  rule_index = 2020

  protocol                = "tcp"
  protocol_match_excepted = true

  source = {
    mac = "00:11:22:33:44:55"
  }
}
`
}

func testAccFirewallRuleConfig_basic() string {
	return `
resource "unifi_firewall_rule" "test" {
  name       = "tfacc-firewall-rule"
  action     = "drop"
  ruleset    = "LAN_IN"
  rule_index = 2000

  logging = false
  state = {
    established = false
    invalid     = false
    new         = false
    related     = false
  }
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

  logging = false
  state = {
    established = false
    invalid     = false
    new         = false
    related     = false
  }
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

  logging = false
  state = {
    established = false
    invalid     = false
    new         = false
    related     = false
  }
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

  logging = false
  state = {
    established = false
    invalid     = false
    new         = false
    related     = false
  }
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

  state = {
    established = false
    invalid     = false
    new         = false
    related     = false
  }
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

  logging = false
  state = {
    established = true
    related     = true
    new         = false
    invalid     = false
  }
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

  destination = {
    port = "443"
  }

  logging = false
  state = {
    established = false
    invalid     = false
    new         = false
    related     = false
  }
}
`
}

func testAccFirewallRuleConfig_withSrcAddress() string {
	return `
resource "unifi_firewall_rule" "test" {
  name       = "tfacc-firewall-rule-src"
  action     = "drop"
  ruleset    = "LAN_IN"
  rule_index = 2070

  source = {
    address      = "10.0.0.0/8"
    network_type = "NETv4"
  }

  logging = false
  state = {
    established = false
    invalid     = false
    new         = false
    related     = false
  }
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

  destination = {
    address = "192.168.0.0/16"
  }

  logging = false
  state = {
    established = false
    invalid     = false
    new         = false
    related     = false
  }
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

  source = {
    firewall_group_ids = [unifi_firewall_group.src.id]
  }
  destination = {
    firewall_group_ids = [unifi_firewall_group.dst.id]
  }

  logging = false
  state = {
    established = false
    invalid     = false
    new         = false
    related     = false
  }
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

  logging = false
  state = {
    established = false
    invalid     = false
    new         = false
    related     = false
  }
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

  logging = false
  state = {
    established = false
    invalid     = false
    new         = false
    related     = false
  }
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

  logging = false
  state = {
    established = false
    invalid     = false
    new         = false
    related     = false
  }
}
`
}

func TestNewFirewallRuleResource(t *testing.T) {
	r := NewFirewallRuleResource()
	if r == nil {
		t.Fatal("NewFirewallRuleResource() returned nil")
	}
	if _, ok := r.(fwresource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ResourceWithImportState")
	}
	if _, ok := r.(fwresource.ResourceWithIdentity); !ok {
		t.Error("resource does not implement ResourceWithIdentity")
	}
}

func TestNewFirewallRuleListResource(t *testing.T) {
	r := NewFirewallRuleListResource()
	if r == nil {
		t.Fatal("NewFirewallRuleListResource() returned nil")
	}
}

func Test_firewallRuleResource_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.MetadataRequest
		resp *fwresource.MetadataResponse
	}
	tests := []struct {
		name string
		r    *firewallRuleResource
		args args
	}{
		{
			name: "type name is unifi_firewall_rule",
			r:    &firewallRuleResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.MetadataRequest{ProviderTypeName: "unifi"},
				resp: &fwresource.MetadataResponse{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Metadata(tt.args.ctx, tt.args.req, tt.args.resp)
			if tt.args.resp.TypeName != "unifi_firewall_rule" {
				t.Errorf("TypeName = %q, want %q", tt.args.resp.TypeName, "unifi_firewall_rule")
			}
		})
	}
}

func Test_firewallRuleResource_IdentitySchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwresource.IdentitySchemaRequest
		resp *fwresource.IdentitySchemaResponse
	}
	tests := []struct {
		name string
		r    *firewallRuleResource
		args args
	}{
		{
			name: "id attribute exists",
			r:    &firewallRuleResource{},
			args: args{
				in0:  context.Background(),
				in1:  fwresource.IdentitySchemaRequest{},
				resp: &fwresource.IdentitySchemaResponse{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.IdentitySchema(tt.args.in0, tt.args.in1, tt.args.resp)
			if _, ok := tt.args.resp.IdentitySchema.Attributes["id"]; !ok {
				t.Error("IdentitySchema missing 'id' attribute")
			}
			if _, ok := tt.args.resp.IdentitySchema.Attributes["site"]; !ok {
				t.Error("IdentitySchema missing 'site' attribute")
			}
		})
	}
}

func Test_firewallRuleResource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.SchemaRequest
		resp *fwresource.SchemaResponse
	}
	tests := []struct {
		name string
		r    *firewallRuleResource
		args args
	}{
		{
			name: "key attributes exist with correct configurability",
			r:    &firewallRuleResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.SchemaRequest{},
				resp: &fwresource.SchemaResponse{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Schema(tt.args.ctx, tt.args.req, tt.args.resp)
			s := tt.args.resp.Schema

			if s.Version != 1 {
				t.Errorf("Version = %d, want 1", s.Version)
			}

			checks := []struct {
				attr     string
				required bool
				optional bool
				computed bool
			}{
				{"id", false, false, true},
				{"name", true, false, false},
				{"action", true, false, false},
				{"ruleset", true, false, false},
				{"rule_index", true, false, false},
				{"enabled", false, true, true},
				{"icmp", false, true, false},
				{"source", false, true, true},
				{"destination", false, true, true},
				{"state", false, true, true},
			}
			for _, c := range checks {
				a, ok := s.Attributes[c.attr]
				if !ok {
					t.Errorf("missing attribute %q", c.attr)
					continue
				}
				if a.IsRequired() != c.required {
					t.Errorf("%s: Required = %v, want %v", c.attr, a.IsRequired(), c.required)
				}
				if a.IsOptional() != c.optional {
					t.Errorf("%s: Optional = %v, want %v", c.attr, a.IsOptional(), c.optional)
				}
				if a.IsComputed() != c.computed {
					t.Errorf("%s: Computed = %v, want %v", c.attr, a.IsComputed(), c.computed)
				}
			}

			// The flat attributes are gone and their leaves live in the
			// nested objects (mac keeps its custom type).
			for _, flat := range []string{
				"src_address", "src_mac", "dst_port", "icmp_typename", "state_established",
			} {
				if _, exists := s.Attributes[flat]; exists {
					t.Errorf("flat attribute %q still present", flat)
				}
			}
			nested := map[string][]string{
				"icmp": {"typename", "v6_typename"},
				"source": {
					"network_id",
					"network_type",
					"firewall_group_ids",
					"address",
					"address_ipv6",
					"port",
					"mac",
				},
				"destination": {
					"network_id",
					"network_type",
					"firewall_group_ids",
					"address",
					"address_ipv6",
					"port",
				},
				"state": {"established", "invalid", "new", "related"},
			}
			for name, leaves := range nested {
				obj, ok := s.Attributes[name].(schema.SingleNestedAttribute)
				if !ok {
					t.Errorf("%s: %T, want schema.SingleNestedAttribute", name, s.Attributes[name])
					continue
				}
				for _, leaf := range leaves {
					if _, ok := obj.Attributes[leaf]; !ok {
						t.Errorf("%s: missing leaf %q", name, leaf)
					}
				}
			}
			src, ok := s.Attributes["source"].(schema.SingleNestedAttribute)
			if !ok {
				t.Fatalf("source: %T, want schema.SingleNestedAttribute", s.Attributes["source"])
			}
			if got := src.Attributes["mac"].GetType(); !got.Equal(hwtypes.MACAddressType{}) {
				t.Errorf("source.mac type = %v, want hwtypes.MACAddressType", got)
			}
		})
	}
}

func Test_firewallRuleResource_Configure(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ConfigureRequest
		resp *fwresource.ConfigureResponse
	}
	tests := []struct {
		name      string
		r         *firewallRuleResource
		args      args
		wantError bool
	}{
		{
			name: "nil provider data",
			r:    &firewallRuleResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.ConfigureRequest{},
				resp: &fwresource.ConfigureResponse{Diagnostics: diag.Diagnostics{}},
			},
			wantError: false,
		},
		{
			name: "wrong type",
			r:    &firewallRuleResource{},
			args: args{
				ctx: context.Background(),
				req: fwresource.ConfigureRequest{
					ProviderData: "not-a-client",
				},
				resp: &fwresource.ConfigureResponse{Diagnostics: diag.Diagnostics{}},
			},
			wantError: true,
		},
		{
			name: "correct client type",
			r:    &firewallRuleResource{},
			args: args{
				ctx: context.Background(),
				req: fwresource.ConfigureRequest{
					ProviderData: &Client{},
				},
				resp: &fwresource.ConfigureResponse{Diagnostics: diag.Diagnostics{}},
			},
			wantError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Configure(tt.args.ctx, tt.args.req, tt.args.resp)
			if tt.wantError && !tt.args.resp.Diagnostics.HasError() {
				t.Error("expected error but got none")
			}
			if !tt.wantError && tt.args.resp.Diagnostics.HasError() {
				t.Errorf("unexpected error: %s", tt.args.resp.Diagnostics.Errors())
			}
		})
	}
}

func Test_firewallRuleResource_applyPlanToState(t *testing.T) {
	type args struct {
		in0   context.Context
		plan  *firewallRuleResourceModel
		state *firewallRuleResourceModel
	}
	tests := []struct {
		name string
		r    *firewallRuleResource
		args args
	}{
		{
			name: "non-null plan fields overwrite state, null fields preserve state",
			r:    &firewallRuleResource{},
			args: args{
				in0: context.Background(),
				plan: &firewallRuleResourceModel{
					Name:     types.StringValue("new"),
					Protocol: types.StringNull(),
				},
				state: &firewallRuleResourceModel{
					Name:     types.StringValue("old"),
					Protocol: types.StringValue("tcp"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.applyPlanToState(tt.args.in0, tt.args.plan, tt.args.state)
			if tt.args.state.Name.ValueString() != "new" {
				t.Errorf("Name = %q, want %q", tt.args.state.Name.ValueString(), "new")
			}
			if tt.args.state.Protocol.ValueString() != "tcp" {
				t.Errorf(
					"Protocol = %q, want %q (should be preserved)",
					tt.args.state.Protocol.ValueString(),
					"tcp",
				)
			}
		})
	}
}

func Test_firewallRuleResource_modelToFirewallRule(t *testing.T) {
	type args struct {
		ctx   context.Context
		model *firewallRuleResourceModel
	}
	ruleIndex2000 := int64(2000)
	ruleIndex3000 := int64(3000)
	// Null objects contribute nothing, exactly as unset flat attributes did.
	nullICMP := types.ObjectNull(firewallRuleICMPAttrTypes())
	nullSource := types.ObjectNull(firewallRuleSourceAttrTypes())
	nullDestination := types.ObjectNull(firewallRuleDestinationAttrTypes())
	nullState := types.ObjectNull(firewallRuleStateAttrTypes())
	tests := []struct {
		name string
		r    *firewallRuleResource
		args args
		want *unifi.FirewallRule
	}{
		{
			name: "basic fields",
			r:    &firewallRuleResource{},
			args: args{
				ctx: context.Background(),
				model: &firewallRuleResourceModel{
					Name:                types.StringValue("drop-rule"),
					Action:              types.StringValue("drop"),
					Ruleset:             types.StringValue("LAN_IN"),
					RuleIndex:           types.Int64Value(2000),
					Enabled:             types.BoolValue(true),
					Protocol:            types.StringNull(),
					ProtocolV6:          types.StringNull(),
					ICMP:                nullICMP,
					Source:              nullSource,
					Destination:         nullDestination,
					Logging:             types.BoolNull(),
					State:               nullState,
					IPSec:               types.StringNull(),
					SettingPreference:   types.StringNull(),
					ProtocolMatchExcept: types.BoolValue(false),
				},
			},
			want: &unifi.FirewallRule{
				Name:      "drop-rule",
				Action:    "drop",
				Ruleset:   "LAN_IN",
				RuleIndex: &ruleIndex2000,
				Enabled:   true,
			},
		},
		{
			name: "with protocol src dst",
			r:    &firewallRuleResource{},
			args: args{
				ctx: context.Background(),
				model: &firewallRuleResourceModel{
					Name:       types.StringValue("allow-https"),
					Action:     types.StringValue("accept"),
					Ruleset:    types.StringValue("WAN_IN"),
					RuleIndex:  types.Int64Value(3000),
					Enabled:    types.BoolValue(true),
					Protocol:   types.StringValue("tcp"),
					ProtocolV6: types.StringNull(),
					ICMP:       nullICMP,
					// Null leaves inside a known object are skipped too.
					Source: firewallRuleTestObject(
						t,
						firewallRuleSourceDefault(),
						map[string]attr.Value{
							"network_type": types.StringNull(),
							"address":      types.StringValue("10.0.0.1"),
						},
					),
					Destination: firewallRuleTestObject(
						t,
						firewallRuleDestinationDefault(),
						map[string]attr.Value{
							"network_type": types.StringNull(),
							"port":         types.StringValue("443"),
						},
					),
					Logging:             types.BoolNull(),
					State:               nullState,
					IPSec:               types.StringNull(),
					SettingPreference:   types.StringNull(),
					ProtocolMatchExcept: types.BoolValue(false),
				},
			},
			want: &unifi.FirewallRule{
				Name:       "allow-https",
				Action:     "accept",
				Ruleset:    "WAN_IN",
				RuleIndex:  &ruleIndex3000,
				Enabled:    true,
				Protocol:   "tcp",
				SrcAddress: "10.0.0.1",
				DstPort:    "443",
			},
		},
		{
			name: "minimal required fields only",
			r:    &firewallRuleResource{},
			args: args{
				ctx: context.Background(),
				model: &firewallRuleResourceModel{
					Name:                types.StringValue("min"),
					Action:              types.StringValue("drop"),
					Ruleset:             types.StringValue("LAN_IN"),
					RuleIndex:           types.Int64Value(2000),
					Enabled:             types.BoolValue(false),
					Protocol:            types.StringNull(),
					ProtocolV6:          types.StringNull(),
					ICMP:                nullICMP,
					Source:              nullSource,
					Destination:         nullDestination,
					Logging:             types.BoolNull(),
					State:               nullState,
					IPSec:               types.StringNull(),
					SettingPreference:   types.StringNull(),
					ProtocolMatchExcept: types.BoolNull(),
				},
			},
			want: &unifi.FirewallRule{
				Name:      "min",
				Action:    "drop",
				Ruleset:   "LAN_IN",
				RuleIndex: &ruleIndex2000,
				Enabled:   false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, diags := tt.r.modelToFirewallRule(tt.args.ctx, tt.args.model)
			if diags.HasError() {
				t.Fatalf("modelToFirewallRule: %v", diags)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("firewallRuleResource.modelToFirewallRule() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_firewallRuleResource_firewallRuleToModel(t *testing.T) {
	type args struct {
		ctx          context.Context
		firewallRule *unifi.FirewallRule
		model        *firewallRuleResourceModel
		site         string
	}
	ruleIndex3000 := int64(3000)
	ruleIndex2000 := int64(2000)
	tests := []struct {
		name      string
		r         *firewallRuleResource
		args      args
		checkFunc func(t *testing.T, model *firewallRuleResourceModel)
	}{
		{
			name: "basic API struct to model",
			r:    &firewallRuleResource{},
			args: args{
				ctx: context.Background(),
				firewallRule: &unifi.FirewallRule{
					ID:        "r1",
					Name:      "test",
					Action:    "accept",
					Ruleset:   "WAN_IN",
					RuleIndex: &ruleIndex3000,
					Enabled:   true,
				},
				model: &firewallRuleResourceModel{},
				site:  "default",
			},
			checkFunc: func(t *testing.T, m *firewallRuleResourceModel) {
				if m.ID.ValueString() != "r1" {
					t.Errorf("ID = %q, want %q", m.ID.ValueString(), "r1")
				}
				if m.Name.ValueString() != "test" {
					t.Errorf("Name = %q, want %q", m.Name.ValueString(), "test")
				}
				if m.Action.ValueString() != "accept" {
					t.Errorf("Action = %q, want %q", m.Action.ValueString(), "accept")
				}
				if m.Ruleset.ValueString() != "WAN_IN" {
					t.Errorf("Ruleset = %q, want %q", m.Ruleset.ValueString(), "WAN_IN")
				}
				if m.RuleIndex.ValueInt64() != 3000 {
					t.Errorf("RuleIndex = %d, want %d", m.RuleIndex.ValueInt64(), 3000)
				}
				if m.Enabled.ValueBool() != true {
					t.Error("Enabled should be true")
				}
				if m.Site.ValueString() != "default" {
					t.Errorf("Site = %q, want %q", m.Site.ValueString(), "default")
				}
			},
		},
		{
			name: "empty optional fields become null",
			r:    &firewallRuleResource{},
			args: args{
				ctx: context.Background(),
				firewallRule: &unifi.FirewallRule{
					ID:        "r2",
					Name:      "empty-opts",
					Action:    "drop",
					Ruleset:   "LAN_IN",
					RuleIndex: &ruleIndex2000,
				},
				model: &firewallRuleResourceModel{},
				site:  "default",
			},
			checkFunc: func(t *testing.T, m *firewallRuleResourceModel) {
				if !m.Protocol.IsNull() {
					t.Error("Protocol should be null")
				}
				if !m.ICMP.IsNull() {
					t.Errorf("icmp = %v, want null when neither ICMP type is set", m.ICMP)
				}
				src := m.Source.Attributes()
				if !src["address"].IsNull() {
					t.Error("source.address should be null")
				}
				if !src["firewall_group_ids"].IsNull() {
					t.Error("source.firewall_group_ids should be null")
				}
				if !src["mac"].IsNull() {
					t.Error("source.mac should be null")
				}
				if !m.Destination.Attributes()["port"].IsNull() {
					t.Error("destination.port should be null")
				}
				if !m.IPSec.IsNull() {
					t.Error("IPSec should be null")
				}
				if !m.State.Equal(firewallRuleStateDefault()) {
					t.Errorf("state = %v, want all-false default", m.State)
				}
			},
		},
		{
			name: "source.network_type defaults to NETv4 when empty",
			r:    &firewallRuleResource{},
			args: args{
				ctx: context.Background(),
				firewallRule: &unifi.FirewallRule{
					ID:             "r3",
					Name:           "netv4-default",
					Action:         "drop",
					Ruleset:        "LAN_IN",
					RuleIndex:      &ruleIndex2000,
					SrcNetworkType: "",
				},
				model: &firewallRuleResourceModel{},
				site:  "default",
			},
			checkFunc: func(t *testing.T, m *firewallRuleResourceModel) {
				got := attrAs[types.String](t, m.Source.Attributes()["network_type"])
				if got.ValueString() != "NETv4" {
					t.Errorf("source.network_type = %q, want %q", got.ValueString(), "NETv4")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := tt.r.firewallRuleToModel(
				tt.args.ctx,
				tt.args.firewallRule,
				tt.args.model,
				tt.args.site,
			)
			if diags.HasError() {
				t.Fatalf("firewallRuleToModel: %v", diags)
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, tt.args.model)
			}
		})
	}
}

func Test_firewallRuleResource_ListResourceConfigSchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwlist.ListResourceSchemaRequest
		resp *fwlist.ListResourceSchemaResponse
	}
	tests := []struct {
		name string
		r    *firewallRuleResource
		args args
	}{
		{
			name: "schema has site attribute",
			r:    &firewallRuleResource{},
			args: args{
				in0:  context.Background(),
				in1:  fwlist.ListResourceSchemaRequest{},
				resp: &fwlist.ListResourceSchemaResponse{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.ListResourceConfigSchema(tt.args.in0, tt.args.in1, tt.args.resp)
			if _, ok := tt.args.resp.Schema.Attributes["site"]; !ok {
				t.Error("ListResourceConfigSchema missing 'site' attribute")
			}
		})
	}
}

func TestAccFirewallRuleList_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallRuleConfig_basic(),
			},
			{
				Query: true,
				Config: `
					provider "unifi" {}
					list "unifi_firewall_rule" "test" {
						provider = unifi
						config {
							filter {
								name  = "name"
								value = "tfacc-firewall-rule"
						  }
					  }
					}
				`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLengthAtLeast("unifi_firewall_rule.test", 1),
				},
			},
		},
	})
}

// TestFirewallRuleUpgradeState_v0NestsPrefixedGroups guards the v0 -> v1
// schema upgrade: flat src_*/dst_*/icmp_*/state_* attributes move into the
// nested source/destination/icmp/state objects.
func TestFirewallRuleUpgradeState_v0NestsPrefixedGroups(t *testing.T) {
	ctx := context.Background()
	r := &firewallRuleResource{}

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	if schemaResp.Schema.Version != 1 {
		t.Fatalf("firewall rule schema Version = %d, want 1", schemaResp.Schema.Version)
	}
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	ups := r.UpgradeState(ctx)
	up, ok := ups[0]
	if !ok {
		t.Fatal("no upgrader registered for schema version 0")
	}
	upgrade := func(prior []byte) map[string]tftypes.Value {
		t.Helper()
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
		return root
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
	boolean := func(v tftypes.Value, name string, want bool) {
		t.Helper()
		var b bool
		if err := v.As(&b); err != nil || b != want {
			t.Errorf("%s = %v (%v), want %v", name, v, err, want)
		}
	}

	root := upgrade([]byte(`{
		"id": "fr-1", "site": "default", "name": "allow-https", "action": "accept",
		"ruleset": "WAN_IN", "rule_index": 3000, "enabled": true,
		"protocol": "tcp", "protocol_v6": null,
		"icmp_typename": null, "icmp_v6_typename": null,
		"src_network_id": null, "src_network_type": "ADDRv4",
		"src_firewall_group_ids": ["fg-1"], "src_address": "10.0.0.1",
		"src_address_ipv6": null, "src_port": null, "src_mac": "00:11:22:33:44:55",
		"dst_network_id": "net-1", "dst_network_type": "NETv4",
		"dst_firewall_group_ids": null, "dst_address": null, "dst_address_ipv6": null,
		"dst_port": "443",
		"logging": true,
		"state_established": true, "state_invalid": false,
		"state_new": false, "state_related": true,
		"ip_sec": null, "setting_preference": null, "protocol_match_excepted": false,
		"timeouts": null
	}`))

	for _, flat := range []string{
		"src_address", "src_mac", "dst_port", "icmp_typename", "state_established",
	} {
		if _, exists := root[flat]; exists {
			t.Errorf("flat attribute %q survived the upgrade", flat)
		}
	}
	str(root["name"], "name", "allow-https")
	src := obj(root["source"], "source")
	str(src["network_type"], "source.network_type", "ADDRv4")
	str(src["address"], "source.address", "10.0.0.1")
	str(src["mac"], "source.mac", "00:11:22:33:44:55")
	var groups []tftypes.Value
	if err := src["firewall_group_ids"].As(&groups); err != nil || len(groups) != 1 {
		t.Errorf(
			"source.firewall_group_ids = %v (%v), want one entry",
			src["firewall_group_ids"],
			err,
		)
	}
	if !src["port"].IsNull() {
		t.Errorf("source.port = %v, want null", src["port"])
	}
	dst := obj(root["destination"], "destination")
	str(dst["network_id"], "destination.network_id", "net-1")
	str(dst["network_type"], "destination.network_type", "NETv4")
	str(dst["port"], "destination.port", "443")
	if !dst["firewall_group_ids"].IsNull() {
		t.Errorf("destination.firewall_group_ids = %v, want null", dst["firewall_group_ids"])
	}
	st := obj(root["state"], "state")
	boolean(st["established"], "state.established", true)
	boolean(st["invalid"], "state.invalid", false)
	boolean(st["new"], "state.new", false)
	boolean(st["related"], "state.related", true)
	// Neither ICMP type was set: the Optional-only group is a null object,
	// not an object of nulls, so the next plan is empty.
	if !root["icmp"].IsNull() {
		t.Errorf("icmp = %v, want null when neither ICMP type was set", root["icmp"])
	}

	// A rule that did match on an ICMP type keeps it under icmp. Attributes
	// missing from this older state are filled with null.
	root = upgrade([]byte(`{
		"id": "fr-2", "name": "ping", "action": "accept", "ruleset": "LAN_IN",
		"rule_index": 2000, "enabled": true, "protocol": "icmp",
		"icmp_typename": "echo-request", "icmp_v6_typename": null,
		"src_network_type": "NETv4", "dst_network_type": "NETv4",
		"state_established": false, "state_invalid": false,
		"state_new": false, "state_related": false
	}`))
	icmp := obj(root["icmp"], "icmp")
	str(icmp["typename"], "icmp.typename", "echo-request")
	if !icmp["v6_typename"].IsNull() {
		t.Errorf("icmp.v6_typename = %v, want null", icmp["v6_typename"])
	}
	src = obj(root["source"], "source")
	str(src["network_type"], "source.network_type", "NETv4")
	if !src["mac"].IsNull() {
		t.Errorf("source.mac = %v, want null when absent from prior state", src["mac"])
	}
}

// TestFirewallRuleNestedGroups_wireAndReadBack checks that the nested
// source/destination/icmp/state groups are written to and read back from the
// API struct, that omitted groups send exactly what the flat defaults sent,
// and that applyPlanToState overlays only the leaves the plan knows.
func TestFirewallRuleNestedGroups_wireAndReadBack(t *testing.T) {
	ctx := context.Background()
	r := &firewallRuleResource{}

	groups, d := types.SetValue(types.StringType, []attr.Value{types.StringValue("fg-1")})
	if d.HasError() {
		t.Fatalf("group set: %v", d)
	}
	model := &firewallRuleResourceModel{
		Name:       types.StringValue("allow-https"),
		Action:     types.StringValue("accept"),
		Ruleset:    types.StringValue("WAN_IN"),
		RuleIndex:  types.Int64Value(3000),
		Enabled:    types.BoolValue(true),
		Protocol:   types.StringValue("tcp"),
		ProtocolV6: types.StringNull(),
		ICMP: types.ObjectValueMust(firewallRuleICMPAttrTypes(), map[string]attr.Value{
			"typename":    types.StringValue("echo-request"),
			"v6_typename": types.StringNull(),
		}),
		Source: types.ObjectValueMust(firewallRuleSourceAttrTypes(), map[string]attr.Value{
			"network_id":         types.StringNull(),
			"network_type":       types.StringValue("ADDRv4"),
			"firewall_group_ids": groups,
			"address":            types.StringValue("10.0.0.1"),
			"address_ipv6":       types.StringNull(),
			"port":               types.StringNull(),
			"mac":                hwtypes.NewMACAddressValue("00:11:22:33:44:55"),
		}),
		Destination: firewallRuleTestObject(
			t,
			firewallRuleDestinationDefault(),
			map[string]attr.Value{
				"network_id": types.StringValue("net-1"),
				"port":       types.StringValue("443"),
			},
		),
		Logging: types.BoolValue(true),
		State: firewallRuleTestObject(t, firewallRuleStateDefault(), map[string]attr.Value{
			"established": types.BoolValue(true),
			"related":     types.BoolValue(true),
		}),
		IPSec:               types.StringNull(),
		SettingPreference:   types.StringNull(),
		ProtocolMatchExcept: types.BoolValue(false),
	}

	api, diags := r.modelToFirewallRule(ctx, model)
	if diags.HasError() {
		t.Fatalf("modelToFirewallRule: %v", diags)
	}
	if api.ICMPTypename != "echo-request" || api.ICMPv6Typename != "" {
		t.Errorf("icmp: %q %q", api.ICMPTypename, api.ICMPv6Typename)
	}
	if api.SrcNetworkType != "ADDRv4" || api.SrcAddress != "10.0.0.1" ||
		api.SrcMACAddress != "00:11:22:33:44:55" || len(api.SrcFirewallGroupIDs) != 1 ||
		api.SrcFirewallGroupIDs[0] != "fg-1" || api.SrcNetworkID != "" || api.SrcPort != "" {
		t.Errorf("source: %+v", api)
	}
	if api.DstNetworkType != "NETv4" || api.DstNetworkID != "net-1" || api.DstPort != "443" ||
		api.DstFirewallGroupIDs != nil || api.DstAddress != "" {
		t.Errorf("destination: %+v", api)
	}
	if !api.StateEstablished || api.StateInvalid || api.StateNew || !api.StateRelated {
		t.Errorf("state: %+v", api)
	}

	// Read back: every group is rebuilt from the API response.
	var back firewallRuleResourceModel
	if d := r.firewallRuleToModel(ctx, api, &back, "default"); d.HasError() {
		t.Fatalf("firewallRuleToModel: %v", d)
	}
	for name, got := range map[string][2]types.Object{
		"icmp":        {back.ICMP, model.ICMP},
		"source":      {back.Source, model.Source},
		"destination": {back.Destination, model.Destination},
		"state":       {back.State, model.State},
	} {
		if !got[0].Equal(got[1]) {
			t.Errorf("%s read back = %v, want %v", name, got[0], got[1])
		}
	}

	// Omitted groups (schema defaults) send exactly what the flat defaults
	// sent: network_type NETv4 on both ends, every state flag false.
	defaults := &firewallRuleResourceModel{
		Name:        types.StringValue("min"),
		Action:      types.StringValue("drop"),
		Ruleset:     types.StringValue("LAN_IN"),
		RuleIndex:   types.Int64Value(2000),
		Enabled:     types.BoolValue(true),
		ICMP:        types.ObjectNull(firewallRuleICMPAttrTypes()),
		Source:      firewallRuleSourceDefault(),
		Destination: firewallRuleDestinationDefault(),
		State:       firewallRuleStateDefault(),
	}
	api, diags = r.modelToFirewallRule(ctx, defaults)
	if diags.HasError() {
		t.Fatalf("modelToFirewallRule (defaults): %v", diags)
	}
	if api.SrcNetworkType != "NETv4" || api.DstNetworkType != "NETv4" ||
		api.SrcAddress != "" || api.SrcMACAddress != "" || api.DstPort != "" ||
		api.SrcFirewallGroupIDs != nil || api.DstFirewallGroupIDs != nil ||
		api.ICMPTypename != "" || api.StateEstablished || api.StateInvalid ||
		api.StateNew || api.StateRelated {
		t.Errorf("defaults: %+v", api)
	}

	// Unknown groups contribute nothing, like null ones.
	unknown := &firewallRuleResourceModel{
		Name:        types.StringValue("unknown"),
		Source:      types.ObjectUnknown(firewallRuleSourceAttrTypes()),
		Destination: types.ObjectUnknown(firewallRuleDestinationAttrTypes()),
		State:       types.ObjectUnknown(firewallRuleStateAttrTypes()),
		ICMP:        types.ObjectUnknown(firewallRuleICMPAttrTypes()),
	}
	api, diags = r.modelToFirewallRule(ctx, unknown)
	if diags.HasError() {
		t.Fatalf("modelToFirewallRule (unknown): %v", diags)
	}
	if api.SrcNetworkType != "" || api.DstNetworkType != "" || api.StateEstablished {
		t.Errorf("unknown groups must contribute nothing: %+v", api)
	}

	// applyPlanToState: an unknown planned group keeps the state's value; a
	// known group re-asserts only its non-null leaves.
	plan := &firewallRuleResourceModel{
		Source: types.ObjectUnknown(firewallRuleSourceAttrTypes()),
		State: types.ObjectValueMust(firewallRuleStateAttrTypes(), map[string]attr.Value{
			"established": types.BoolNull(),
			"invalid":     types.BoolNull(),
			"new":         types.BoolValue(true),
			"related":     types.BoolNull(),
		}),
	}
	state := back
	r.applyPlanToState(ctx, plan, &state)
	if !state.Source.Equal(back.Source) {
		t.Errorf("unknown planned source must keep state value: %v", state.Source)
	}
	if !state.Destination.Equal(back.Destination) {
		t.Errorf("null planned destination must keep state value: %v", state.Destination)
	}
	st := state.State.Attributes()
	if !attrAs[types.Bool](t, st["new"]).ValueBool() {
		t.Errorf("planned state.new not applied: %v", st)
	}
	if !attrAs[types.Bool](t, st["established"]).ValueBool() ||
		!attrAs[types.Bool](t, st["related"]).ValueBool() {
		t.Errorf("null planned state leaves must keep state values: %v", st)
	}

	// A rule with neither ICMP type reads back as a null icmp (the group is
	// Optional-only), except when the practitioner configured an empty block.
	var fresh firewallRuleResourceModel
	if d := r.firewallRuleToModel(
		ctx,
		&unifi.FirewallRule{ID: "x"},
		&fresh,
		"default",
	); d.HasError() {
		t.Fatalf("firewallRuleToModel (fresh): %v", d)
	}
	if !fresh.ICMP.IsNull() {
		t.Errorf("icmp should be null when never configured: %v", fresh.ICMP)
	}
	emptyBlock := types.ObjectValueMust(firewallRuleICMPAttrTypes(), map[string]attr.Value{
		"typename":    types.StringNull(),
		"v6_typename": types.StringNull(),
	})
	configured := firewallRuleResourceModel{ICMP: emptyBlock}
	if d := r.firewallRuleToModel(
		ctx,
		&unifi.FirewallRule{ID: "x"},
		&configured,
		"default",
	); d.HasError() {
		t.Fatalf("firewallRuleToModel (empty block): %v", d)
	}
	if !configured.ICMP.Equal(emptyBlock) {
		t.Errorf("icmp = {} must be kept as configured: %v", configured.ICMP)
	}
}
