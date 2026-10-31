package unifi

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// The target-bound endpoint attributes (match_mac/mac on an IP source,
// iid/match_opposite_iid on an IID endpoint) used to carry static Defaults, so
// an adopted IP/MAC binding planned `match_mac true -> false`, `mac X -> ""`
// and an adopted interface ID match planned its iid away (#544).
func firewallPolicyTargetBoundLive() map[string]attr.Value {
	return map[string]attr.Value{
		"source.matching_target":      types.StringValue("IP"),
		"source.match_mac":            types.BoolValue(true),
		"source.mac":                  types.StringValue("aa:bb:cc:dd:ee:ff"),
		"source.iid":                  types.StringValue(""),
		"source.match_opposite_iid":   types.BoolValue(false),
		"destination.matching_target": types.StringValue("IID"),
		"destination.iid": types.StringValue(
			"::eaff:1eff:fed7:bf17/::ffff:ffff:ffff:ffff",
		),
		"destination.match_opposite_iid":   types.BoolValue(true),
		"destination.match_mac":            types.BoolValue(false),
		"destination.mac":                  types.StringValue(""),
		"source.matching_target_type":      types.StringValue("SPECIFIC"),
		"destination.matching_target_type": types.StringValue("SPECIFIC"),
	}
}

func TestFirewallPolicyAdoptedTargetBoundAttributes(t *testing.T) {
	prior, planned := adoptedPlan(t, "unifi_firewall_policy", &firewallPolicyResource{},
		nil, firewallPolicyTargetBoundLive(), nil)
	assertNoPlannedChanges(t, prior, planned)
}

// A new policy is planned (and sent) exactly as with the old Defaults.
func TestFirewallPolicyCreateTargetBoundAttributes(t *testing.T) {
	attrs := createPlan(t, "unifi_firewall_policy", &firewallPolicyResource{}, nil)
	for _, endpoint := range []string{"source", "destination"} {
		ep := plannedObject(t, attrs[endpoint], endpoint)
		assertPlanned(t, ep["match_mac"], endpoint+".match_mac", false)
		assertPlanned(t, ep["mac"], endpoint+".mac", "")
		assertPlanned(t, ep["iid"], endpoint+".iid", "")
		assertPlanned(t, ep["match_opposite_iid"], endpoint+".match_opposite_iid", false)
	}
}

// Switching an endpoint's matching_target plans the create-time values for the
// attributes bound to the old target, as the Defaults did: carrying
// match_mac = true over to a CLIENT source would be dropped by the controller
// ("inconsistent result after apply"), and a stale iid would be sent with a
// target it does not belong to. The other endpoint keeps its values.
func TestFirewallPolicyTargetChangeResetsTargetBoundAttributes(t *testing.T) {
	r := &firewallPolicyResource{}
	s, prior, cfg := adoptedInputs(
		t,
		r,
		nil,
		firewallPolicyTargetBoundLive(),
		map[string]attr.Value{
			"source.matching_target":      types.StringValue("CLIENT"),
			"destination.matching_target": types.StringValue("IID"),
		},
	)
	// Terraform core proposes the configuration over the prior state. setAtPath
	// edits the nested value in place, so it works on a second copy of prior.
	_, proposed, _ := adoptedInputs(t, r, nil, firewallPolicyTargetBoundLive(), nil)
	proposed = setAtPath(t, proposed, strings.Split("source.matching_target", "."),
		tftypes.NewValue(tftypes.String, "CLIENT"))
	planned := planResourceChange(t, "unifi_firewall_policy", s, prior, proposed, cfg)

	var attrs map[string]tftypes.Value
	if err := planned.As(&attrs); err != nil {
		t.Fatalf("planned as object: %v", err)
	}
	src := plannedObject(t, attrs["source"], "source")
	assertPlanned(t, src["match_mac"], "source.match_mac", false)
	assertPlanned(t, src["mac"], "source.mac", "")

	dst := plannedObject(t, attrs["destination"], "destination")
	assertPlanned(t, dst["iid"], "destination.iid", "::eaff:1eff:fed7:bf17/::ffff:ffff:ffff:ffff")
	assertPlanned(t, dst["match_opposite_iid"], "destination.match_opposite_iid", true)
}
