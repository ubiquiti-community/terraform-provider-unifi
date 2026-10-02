package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// Region matching (#492) round-trips through both endpoints. The controller
// only accepts matching_target REGION on an external zone, and rejects a write
// that leaves the list empty with api.err.EmptyFirewallSourceRegions - so the
// codes must survive the model conversion in both directions or an update
// either fails or, worse, changes what the policy matches.
func TestFirewallPolicyRegionsRoundTrip(t *testing.T) {
	ctx := context.Background()

	regions, d := types.ListValue(types.StringType, []attr.Value{
		types.StringValue("DE"),
		types.StringValue("US"),
		types.StringValue("SE"),
	})
	if d.HasError() {
		t.Fatalf("building regions: %v", d)
	}

	model := func(r types.List) firewallPolicyEndpointModel {
		return firewallPolicyEndpointModel{
			ZoneID:                types.StringValue("zone-external"),
			MatchingTarget:        types.StringValue("REGION"),
			MatchingTargetType:    types.StringValue("ANY"),
			NetworkIDs:            types.ListNull(types.StringType),
			ClientMACs:            types.ListNull(types.StringType),
			IPs:                   types.ListNull(types.StringType),
			WebDomains:            types.ListNull(types.StringType),
			Regions:               r,
			Port:                  types.StringNull(),
			PortGroupID:           types.StringValue(""),
			IPGroupID:             types.StringValue(""),
			PortMatchingType:      types.StringValue("ANY"),
			MatchOppositeIPs:      types.BoolValue(false),
			MatchOppositeNetworks: types.BoolValue(false),
			MatchOppositePorts:    types.BoolValue(false),
			MatchMAC:              types.BoolValue(false),
		}
	}

	t.Run("source: configured regions reach the API", func(t *testing.T) {
		var d diag.Diagnostics
		diags := &d
		src := endpointModelToSource(ctx, model(regions), diags)
		if diags.HasError() {
			t.Fatalf("conversion: %v", diags)
		}
		if got := src.Regions; len(got) != 3 || got[0] != "DE" || got[2] != "SE" {
			t.Errorf("Regions = %v, want [DE US SE]", got)
		}
		if src.MatchingTarget != "REGION" {
			t.Errorf("MatchingTarget = %q, want REGION", src.MatchingTarget)
		}
	})

	t.Run("destination: configured regions reach the API", func(t *testing.T) {
		var d diag.Diagnostics
		diags := &d
		dst := endpointModelToDestination(ctx, model(regions), diags)
		if diags.HasError() {
			t.Fatalf("conversion: %v", diags)
		}
		if got := dst.Regions; len(got) != 3 {
			t.Errorf("Regions = %v, want 3 codes", got)
		}
	})

	t.Run("source: controller regions are read back", func(t *testing.T) {
		var d diag.Diagnostics
		diags := &d
		m := apiSourceToEndpointModel(ctx, &unifi.FirewallPolicySource{
			ZoneID:         "zone-external",
			MatchingTarget: "REGION",
			Regions:        []string{"KP", "RU"},
		}, diags)
		if diags.HasError() {
			t.Fatalf("conversion: %v", diags)
		}
		var got []string
		m.Regions.ElementsAs(ctx, &got, false)
		if len(got) != 2 || got[0] != "KP" || got[1] != "RU" {
			t.Errorf("regions = %v, want [KP RU]", got)
		}
	})

	t.Run("destination: controller regions are read back", func(t *testing.T) {
		var d diag.Diagnostics
		diags := &d
		m := apiDestinationToEndpointModel(ctx, &unifi.FirewallPolicyDestination{
			ZoneID:         "zone-external",
			MatchingTarget: "REGION",
			Regions:        []string{"CN"},
		}, diags)
		if diags.HasError() {
			t.Fatalf("conversion: %v", diags)
		}
		var got []string
		m.Regions.ElementsAs(ctx, &got, false)
		if len(got) != 1 || got[0] != "CN" {
			t.Errorf("regions = %v, want [CN]", got)
		}
	})

	t.Run("an endpoint without regions sends none", func(t *testing.T) {
		var d diag.Diagnostics
		diags := &d
		m := model(types.ListNull(types.StringType))
		m.MatchingTarget = types.StringValue("ANY")
		src := endpointModelToSource(ctx, m, diags)
		if diags.HasError() {
			t.Fatalf("conversion: %v", diags)
		}
		if len(src.Regions) != 0 {
			t.Errorf("Regions = %v, want empty for a non-REGION endpoint", src.Regions)
		}
	})
}
