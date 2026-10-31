package unifi

import (
	"context"
	"net"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ValidateConfig rejects match_mac = true wherever the controller does not
// store it (#575). Verified against UniFi Network 10.6.x:
//
//   - source, matching_target = "IP": the flag is stored (the IP/MAC binding).
//     The controller then requires the bound address in `mac` and rejects the
//     write without it (api.err.MissingFirewallPolicySourceIpMac).
//   - source, any other target (CLIENT, ANY, ...): the flag is accepted and
//     silently dropped, so it reads back as false and Terraform reports
//     "Provider produced inconsistent result after apply".
//   - destination, any target: the flag is never stored.
//
// A CLIENT target already matches the clients in client_macs by MAC, so the
// flag has no meaning there; a plan-time error is clearer than a perpetual
// diff. `mac` is only meaningful together with match_mac on an IP source, so
// it is rejected anywhere else.
func (r *firewallPolicyResource) ValidateConfig(
	ctx context.Context,
	req resource.ValidateConfigRequest,
	resp *resource.ValidateConfigResponse,
) {
	var srcMatchMAC, dstMatchMAC types.Bool
	var srcTarget, srcMAC, dstMAC types.String
	resp.Diagnostics.Append(
		req.Config.GetAttribute(ctx, path.Root("source").AtName("match_mac"), &srcMatchMAC)...)
	resp.Diagnostics.Append(
		req.Config.GetAttribute(ctx, path.Root("source").AtName("matching_target"), &srcTarget)...)
	resp.Diagnostics.Append(
		req.Config.GetAttribute(ctx, path.Root("destination").AtName("match_mac"), &dstMatchMAC)...)
	resp.Diagnostics.Append(
		req.Config.GetAttribute(ctx, path.Root("source").AtName("mac"), &srcMAC)...)
	resp.Diagnostics.Append(
		req.Config.GetAttribute(ctx, path.Root("destination").AtName("mac"), &dstMAC)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if srcMatchMAC.ValueBool() && !srcTarget.IsUnknown() && !srcTarget.IsNull() &&
		srcTarget.ValueString() != "IP" {
		detail := "The UniFi controller only stores match_mac on a source with matching_target = \"IP\"; " +
			"on matching_target = \"" + srcTarget.ValueString() + "\" it drops the flag, so it cannot be applied."
		if srcTarget.ValueString() == "CLIENT" {
			detail += " A CLIENT target already matches the clients in client_macs by MAC address: " +
				"remove match_mac or set it to false."
		}
		resp.Diagnostics.AddAttributeError(
			path.Root("source").AtName("match_mac"),
			"match_mac not supported for this matching_target",
			detail,
		)
	}

	if dstMatchMAC.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			path.Root("destination").AtName("match_mac"),
			"match_mac not supported on the destination",
			"The UniFi controller never stores match_mac on a firewall policy destination, "+
				"so it cannot be applied. Remove match_mac from destination or set it to false.",
		)
	}

	validateFirewallPolicySourceMAC(srcMatchMAC, srcTarget, srcMAC, resp)

	if dstMAC.ValueString() != "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("destination").AtName("mac"),
			"mac not supported on the destination",
			"The UniFi controller only binds a MAC address on a firewall policy source "+
				"(matching_target = \"IP\", match_mac = true). Remove mac from destination.",
		)
	}
}

// validateFirewallPolicySourceMAC checks the source IP/MAC binding: match_mac
// on an IP source requires a valid mac, and mac requires both.
func validateFirewallPolicySourceMAC(
	matchMAC types.Bool,
	target, mac types.String,
	resp *resource.ValidateConfigResponse,
) {
	macPath := path.Root("source").AtName("mac")
	if mac.IsUnknown() || matchMAC.IsUnknown() {
		return
	}

	if mac.ValueString() == "" {
		if matchMAC.ValueBool() && target.ValueString() == "IP" {
			resp.Diagnostics.AddAttributeError(
				macPath,
				"mac required with match_mac",
				"With match_mac = true on an IP source, the UniFi controller requires the bound "+
					"MAC address (api.err.MissingFirewallPolicySourceIpMac). Set source.mac.",
			)
		}
		return
	}

	if hw, err := net.ParseMAC(mac.ValueString()); err != nil || len(hw) != 6 {
		resp.Diagnostics.AddAttributeError(
			macPath,
			"Invalid MAC address",
			"source.mac must be a MAC address such as \"aa:bb:cc:dd:ee:ff\", got \""+
				mac.ValueString()+"\".",
		)
	}
	if !matchMAC.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			macPath,
			"mac requires match_mac",
			"source.mac is the address bound by match_mac; the UniFi controller ignores it "+
				"otherwise. Set match_mac = true or remove mac.",
		)
	}
	if !target.IsUnknown() && target.ValueString() != "IP" {
		resp.Diagnostics.AddAttributeError(
			macPath,
			"mac not supported for this matching_target",
			"source.mac is only valid with matching_target = \"IP\".",
		)
	}
}
