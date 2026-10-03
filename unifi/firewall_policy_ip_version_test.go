package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// #544: ip_version was Optional+Computed and also carried a static Default of
// IPV4. The framework applies defaults before it runs plan modifiers, so the
// Default wins over UseStateForUnknown and an undeclared ip_version is planned
// at IPV4 whatever the controller holds - measured on a live site, that
// proposed stripping the IPv6 scope from 310 of 372 policies.
//
// The controller requires the field on both POST and PUT, so the default
// cannot simply be dropped; it has to be applied on create only, which is what
// planmodifiers.StringDefaultOnCreate gives.
func TestFirewallPolicyIPVersionHasNoStaticDefault(t *testing.T) {
	var resp resource.SchemaResponse
	(&firewallPolicyResource{}).Schema(
		context.Background(),
		resource.SchemaRequest{},
		&resp,
	)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}

	attr, ok := resp.Schema.Attributes["ip_version"]
	if !ok {
		t.Fatal("attribute ip_version is missing from the schema")
	}
	a, ok := attr.(schema.StringAttribute)
	if !ok {
		t.Fatalf("ip_version is %T, want schema.StringAttribute", attr)
	}

	if a.Default != nil {
		t.Error(
			"ip_version must not carry a static Default: it is applied before " +
				"plan modifiers and narrows an adopted BOTH/IPV6 policy to IPV4 (#544)",
		)
	}
	if len(a.PlanModifiers) == 0 {
		t.Error(
			"ip_version needs a create-only default plan modifier: the " +
				"controller rejects a create without the field (#544)",
		)
	}
}
