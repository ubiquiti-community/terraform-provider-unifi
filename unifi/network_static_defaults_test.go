package unifi

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// #544: setting_preference and ipv6_interface_type were Optional+Computed and
// also carried a static Default. The framework applies defaults
// (TransformDefaults) before it runs plan modifiers (SchemaModifyPlan), so the
// Default beats the controller's value for an attribute the configuration does
// not set.
//
// Measured on a live twelve-network site: 11 of 12 networks store
// setting_preference = "manual" against an "auto" default, and 6 of the 9 that
// report it store ipv6_interface_type = "static" against a "none" default. So
// importing a network proposed cutting its static IPv6 and switching it to
// controller-managed - which also resets DHCP guarding.
//
// The read path already normalizes an omitted field to "none" / "default"
// (#414), so the schema Default served only to override live values.
// lookupSchemaAttribute resolves a dotted path (e.g. "ipv6.interface_type")
// through nested attributes.
func lookupSchemaAttribute(
	attrs map[string]schema.Attribute,
	path string,
) (schema.Attribute, bool) {
	head, rest, nested := strings.Cut(path, ".")
	attr, ok := attrs[head]
	if !ok || !nested {
		return attr, ok
	}
	sn, ok := attr.(schema.SingleNestedAttribute)
	if !ok {
		return nil, false
	}
	return lookupSchemaAttribute(sn.Attributes, rest)
}

func TestNetworkNoStaticDefaultsOnAdoptedAttributes(t *testing.T) {
	var resp resource.SchemaResponse
	(&networkResource{}).Schema(
		context.Background(),
		resource.SchemaRequest{},
		&resp,
	)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}

	for _, name := range []string{"setting_preference", "ipv6.interface_type"} {
		attr, ok := lookupSchemaAttribute(resp.Schema.Attributes, name)
		if !ok {
			t.Errorf("attribute %q is missing from the schema", name)
			continue
		}
		a, ok := attr.(schema.StringAttribute)
		if !ok {
			t.Errorf("%s is %T, want schema.StringAttribute", name, attr)
			continue
		}

		if a.Default != nil {
			t.Errorf(
				"%s must not carry a static Default: it is applied before plan "+
					"modifiers and overwrites the controller's value on an "+
					"adopted network (#544)",
				name,
			)
		}
		if len(a.PlanModifiers) == 0 {
			t.Errorf("%s needs UseStateForUnknown so the prior value is held", name)
		}
	}
}
