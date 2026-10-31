package unifi

import (
	"context"
	"fmt"
	"net/netip"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// firewallPolicyIIDTargetConflicts are the endpoint attributes that select
// what another matching_target matches. None of them can be combined with an
// IID match.
var firewallPolicyIIDTargetConflicts = []string{
	"ips", "ip_group_id", "network_ids", "client_macs", "web_domains", "regions",
}

// firewallPolicyIIDValidator checks the iid format the controller accepts:
// "<address>/<mask>", both in IPv6 notation. Anything else (a bare address, a
// prefix length such as "/64", an empty string) is rejected by the controller
// with api.err.InvalidInterfaceIdentifier, so it is caught at plan time.
type firewallPolicyIIDValidator struct{}

func (firewallPolicyIIDValidator) Description(context.Context) string {
	return "value must be an IPv6 interface identifier written as <address>/<mask>, " +
		"for example ::eaff:1eff:fed7:bf17/::ffff:ffff:ffff:ffff"
}

func (v firewallPolicyIIDValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v firewallPolicyIIDValidator) ValidateString(
	ctx context.Context,
	req validator.StringRequest,
	resp *validator.StringResponse,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if value == "" {
		// An empty iid is the unset value; the endpoint validator reports it
		// when matching_target is IID.
		return
	}
	if !validFirewallPolicyIID(value) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Interface Identifier",
			fmt.Sprintf("%q is not valid: %s.", value, v.Description(ctx)),
		)
	}
}

func validFirewallPolicyIID(value string) bool {
	addr, mask, ok := strings.Cut(value, "/")
	if !ok {
		return false
	}
	for _, part := range []string{addr, mask} {
		ip, err := netip.ParseAddr(part)
		if err != nil || !ip.Is6() || ip.Is4In6() || ip.Zone() != "" {
			return false
		}
	}
	return true
}

// firewallPolicyEndpointValidator enforces the iid / matching_target pairing
// on a source or destination: IID requires iid and excludes the attributes of
// the other targets, and iid / match_opposite_iid are only valid with IID.
type firewallPolicyEndpointValidator struct{}

func (firewallPolicyEndpointValidator) Description(context.Context) string {
	return "iid is required when matching_target is IID, and iid and " +
		"match_opposite_iid are only valid with matching_target IID"
}

func (v firewallPolicyEndpointValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (firewallPolicyEndpointValidator) ValidateObject(
	_ context.Context,
	req validator.ObjectRequest,
	resp *validator.ObjectResponse,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	attrs := req.ConfigValue.Attributes()

	target, ok := attrs["matching_target"].(basetypes.StringValue)
	if !ok || target.IsNull() || target.IsUnknown() {
		return
	}

	iid, _ := attrs["iid"].(basetypes.StringValue)
	iidSet := !iid.IsNull() && !iid.IsUnknown() && iid.ValueString() != ""

	if target.ValueString() == "IID" {
		if !iid.IsUnknown() && !iidSet {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName("iid"),
				"Missing Interface Identifier",
				`iid must be set when matching_target is "IID".`,
			)
		}
		for _, name := range firewallPolicyIIDTargetConflicts {
			if firewallPolicyAttrConfigured(attrs[name]) {
				resp.Diagnostics.AddAttributeError(
					req.Path.AtName(name),
					"Conflicting Matching Target Attribute",
					fmt.Sprintf(
						`%s cannot be set when matching_target is "IID"; the interface identifier is matched through iid.`,
						name,
					),
				)
			}
		}
		return
	}

	if iidSet {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("iid"),
			"Interface Identifier Without IID Target",
			fmt.Sprintf(
				`iid is only valid when matching_target is "IID", got %q.`,
				target.ValueString(),
			),
		)
	}
	if opposite, ok := attrs["match_opposite_iid"].(basetypes.BoolValue); ok &&
		!opposite.IsNull() && !opposite.IsUnknown() && opposite.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("match_opposite_iid"),
			"Interface Identifier Inversion Without IID Target",
			fmt.Sprintf(
				`match_opposite_iid is only valid when matching_target is "IID", got %q.`,
				target.ValueString(),
			),
		)
	}
}

// firewallPolicyAttrConfigured reports whether a configured attribute carries
// a value: a non-empty string or a non-empty list. Unknown values are not
// reported, since they cannot be checked until apply.
func firewallPolicyAttrConfigured(v attr.Value) bool {
	if v == nil || v.IsNull() || v.IsUnknown() {
		return false
	}
	switch tv := v.(type) {
	case basetypes.StringValue:
		return tv.ValueString() != ""
	case basetypes.ListValue:
		return len(tv.Elements()) > 0
	default:
		return true
	}
}
