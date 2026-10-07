package unifi

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
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

// TestNetworkAdoptedIPv6InterfaceTypeIsKept pins the ModifyPlan side of #550:
// the nested ipv6 group has no schema Default, but planIPv6Defaults used to
// re-assert interface_type = "none" on every plan whose configuration omitted
// it, which reintroduced the overwrite the v0.59.0 release removed. An adopted
// network holding "static" with no ipv6 block configured must plan unchanged,
// while a create with nothing configured still gets "none".
func TestNetworkAdoptedIPv6InterfaceTypeIsKept(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	(&networkResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}
	f := planFixture{ctx: ctx, t: t, overrides: map[string]attr.Value{
		"ipv6.interface_type": types.StringValue("static"),
	}}
	prior, config := f.object(schemaResp.Schema, "")

	planned := planResourceChange(t, "unifi_network", schemaResp.Schema, prior, prior, config)
	diffs, err := prior.Diff(planned)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	for _, d := range diffs {
		t.Errorf("adopted network planned a change: %s: %v -> %v", d.Path, d.Value1, d.Value2)
	}

	schemaType := schemaResp.Schema.Type().TerraformType(ctx)
	created := planResourceChange(t, "unifi_network", schemaResp.Schema,
		tftypes.NewValue(schemaType, nil), config, config)
	var attrs map[string]tftypes.Value
	if err := created.As(&attrs); err != nil {
		t.Fatalf("as object: %v", err)
	}
	ipv6 := plannedObject(t, attrs["ipv6"], "ipv6")
	var got string
	if err := ipv6["interface_type"].As(&got); err != nil || got != "none" {
		t.Errorf(
			"create planned ipv6.interface_type = %v (%v), want none",
			ipv6["interface_type"], err,
		)
	}
}
