package unifi

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ValidateConfig rejects match_mac = true wherever the controller does not
// store it (#575). Verified against UniFi Network 10.6.x:
//
//   - source, matching_target = "IP": the flag is stored (the IP/MAC binding,
//     which also needs the controller's `mac` field).
//   - source, any other target (CLIENT, ANY, ...): the flag is accepted and
//     silently dropped, so it reads back as false and Terraform reports
//     "Provider produced inconsistent result after apply".
//   - destination, any target: the flag is never stored.
//
// A CLIENT target already matches the clients in client_macs by MAC, so the
// flag has no meaning there; a plan-time error is clearer than a perpetual
// diff.
func (r *firewallPolicyResource) ValidateConfig(
	ctx context.Context,
	req resource.ValidateConfigRequest,
	resp *resource.ValidateConfigResponse,
) {
	var srcMatchMAC, dstMatchMAC types.Bool
	var srcTarget types.String
	resp.Diagnostics.Append(
		req.Config.GetAttribute(ctx, path.Root("source").AtName("match_mac"), &srcMatchMAC)...)
	resp.Diagnostics.Append(
		req.Config.GetAttribute(ctx, path.Root("source").AtName("matching_target"), &srcTarget)...)
	resp.Diagnostics.Append(
		req.Config.GetAttribute(ctx, path.Root("destination").AtName("match_mac"), &dstMatchMAC)...)
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
}
