package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// #544: an Optional+Computed attribute that also carries a static Default is
// overwritten by that Default, because the framework applies defaults
// (TransformDefaults) before it runs plan modifiers (SchemaModifyPlan).
//
// On unifi_port_forward this is the sharpest instance of the whole class,
// because the wrong value opens a hole rather than closing one. Measured
// against a live site with eight rules:
//
//	enabled   default true     vs 3 of 8 disabled on the controller
//	protocol  default tcp_udp  vs tcp on 5 of 8
//
// Verified live on a rule that is disabled and forwards to SSH: importing it
// without declaring `enabled` planned `false -> true`, which would have
// reopened the port to the internet, and `tcp -> tcp_udp` alongside it.
//
// `enabled` is deprecated in favour of removing the rule, but deprecation does
// not protect anyone: the attribute still works and its Default still wins.
func TestPortForwardNoStaticDefaultsOnAdoptedAttributes(t *testing.T) {
	var resp resource.SchemaResponse
	(&portForwardResource{}).Schema(
		context.Background(),
		resource.SchemaRequest{},
		&resp,
	)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}

	enabled, ok := resp.Schema.Attributes["enabled"].(schema.BoolAttribute)
	if !ok {
		t.Fatalf("enabled is %T, want schema.BoolAttribute", resp.Schema.Attributes["enabled"])
	}
	if enabled.Default != nil {
		t.Error(
			"enabled must not carry a static Default: it is applied before plan " +
				"modifiers, so a rule disabled on the controller is planned back " +
				"to true and the apply reopens the port (#544)",
		)
	}
	if len(enabled.PlanModifiers) == 0 {
		t.Error("enabled needs UseStateForUnknown so the controller's value is held")
	}

	protocol, ok := resp.Schema.Attributes["protocol"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("protocol is %T, want schema.StringAttribute", resp.Schema.Attributes["protocol"])
	}
	if protocol.Default != nil {
		t.Error(
			"protocol must not carry a static Default: a tcp-only rule is " +
				"planned back to tcp_udp, widening the forward to UDP (#544)",
		)
	}
	if len(protocol.PlanModifiers) == 0 {
		t.Error("protocol needs UseStateForUnknown so the controller's value is held")
	}
}

// Companion to the schema guard above, covering the CREATE path, which has no
// controller value to hold: a rule created without declaring `enabled` must
// still be written disabled, so omitting the attribute never opens a port.
//
// This holds because unifi.PortForward.Enabled is a plain bool with no
// omitempty, and ValueBool() on an unknown value yields false - so the create
// body carries "enabled": false. Measured on a live controller by creating a
// probe rule with the attribute undeclared: the controller stored
// enabled=false. The test pins the serialization rather than the measurement.
func TestPortForwardCreateWithoutEnabledIsDisabled(t *testing.T) {
	r := &portForwardResource{}

	model := &portForwardResourceModel{
		Name: types.StringValue("probe"),
		// Enabled left unknown, as an undeclared Optional+Computed attribute
		// is during Create.
		Enabled:        types.BoolUnknown(),
		Protocol:       types.StringUnknown(),
		Logging:        types.BoolUnknown(),
		Wan:            types.ObjectNull(portForwardWanModel{}.AttributeTypes()),
		Forward:        types.ObjectNull(portForwardForwardModel{}.AttributeTypes()),
		SourceLimiting: types.ObjectNull(portForwardSourceLimitingModel{}.AttributeTypes()),
		DestinationIPs: types.ListNull(
			types.ObjectType{AttrTypes: portForwardDestinationIPModel{}.AttributeTypes()},
		),
	}

	pf, diags := r.modelToPortForward(context.Background(), model)
	if diags.HasError() {
		t.Fatalf("modelToPortForward: %v", diags)
	}
	if pf.Enabled {
		t.Error(
			"a rule created without declaring enabled must be written disabled: " +
				"omitting the attribute must never open a WAN port (#544)",
		)
	}
}
