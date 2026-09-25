package unifi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// siteIdentityUpgraders upgrades version 0 identities of resources whose
// identity gained attributes in v0.56.0 (site, and network_id on
// wireguard_peer) without a version bump. Terraform decodes a stored v0.55.0
// identity such as {"id": ...} with the missing attribute as null, but
// OpenTofu requires every attribute of the current schema, so under OpenTofu
// every plan failed with `attribute "site" is required`. Bumping the version
// routes those identities through an upgrader on both.
//
// Stored values are carried over and a missing site is filled from the
// provider's configured site, which is what Read derives for a resource that
// does not set site itself. The result must be complete: Read passes a stored
// identity through unchanged, and the framework rejects a null one.
func siteIdentityUpgraders(client func() *Client) map[int64]resource.IdentityUpgrader {
	return map[int64]resource.IdentityUpgrader{
		0: {
			IdentityUpgrader: func(
				ctx context.Context,
				req resource.UpgradeIdentityRequest,
				resp *resource.UpgradeIdentityResponse,
			) {
				upgradeSiteIdentityV0(ctx, client(), req, resp)
			},
		},
	}
}

func upgradeSiteIdentityV0(
	ctx context.Context,
	client *Client,
	req resource.UpgradeIdentityRequest,
	resp *resource.UpgradeIdentityResponse,
) {
	prior := map[string]any{}
	if req.RawIdentity != nil && len(req.RawIdentity.JSON) > 0 {
		if err := json.Unmarshal(req.RawIdentity.JSON, &prior); err != nil {
			resp.Diagnostics.AddError(
				"Unable to Upgrade Resource Identity",
				fmt.Sprintf("Could not parse the stored version 0 identity: %s", err),
			)
			return
		}
	}

	obj, ok := resp.Identity.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		resp.Diagnostics.AddError(
			"Unable to Upgrade Resource Identity",
			"The identity schema is not an object type.",
		)
		return
	}
	attrs := make(map[string]tftypes.Value, len(obj.AttributeTypes))
	for name, t := range obj.AttributeTypes {
		var v any
		if s, ok := prior[name].(string); ok && s != "" {
			v = s
		} else if name == "site" && client != nil && client.Site != "" {
			v = client.Site
		}
		attrs[name] = tftypes.NewValue(t, v)
	}
	resp.Identity.Raw = tftypes.NewValue(obj, attrs)
}
