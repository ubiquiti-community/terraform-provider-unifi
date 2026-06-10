package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// TestFirewallZoneModelRoundTrip validates the model <-> go-unifi struct
// conversion for the unifi_firewall_zone resource (#214). It is a unit test
// rather than an acceptance test because zone-based firewall is not available
// in the dockerized acceptance controller.
func TestFirewallZoneModelRoundTrip(t *testing.T) {
	ctx := context.Background()
	r := &firewallZoneResource{}

	nids, d := types.ListValueFrom(ctx, types.StringType, []string{"net-a", "net-b"})
	if d.HasError() {
		t.Fatalf("building network_ids list: %v", d)
	}
	model := &firewallZoneResourceModel{
		Name:       types.StringValue("DMZ"),
		NetworkIDs: nids,
	}

	zone, diags := r.modelToFirewallZone(ctx, model)
	if diags.HasError() {
		t.Fatalf("modelToFirewallZone: %v", diags)
	}
	if zone.Name != "DMZ" {
		t.Errorf("Name = %q, want DMZ", zone.Name)
	}
	if len(zone.NetworkIDs) != 2 || zone.NetworkIDs[0] != "net-a" || zone.NetworkIDs[1] != "net-b" {
		t.Errorf("NetworkIDs = %v, want [net-a net-b]", zone.NetworkIDs)
	}

	apiZone := &unifi.FirewallZone{
		ID:          "z1",
		Name:        "DMZ",
		NetworkIDs:  []string{"net-a", "net-b"},
		ZoneKey:     "dmz",
		DefaultZone: false,
	}
	var out firewallZoneResourceModel
	if diags := r.firewallZoneToModel(ctx, apiZone, &out, "default"); diags.HasError() {
		t.Fatalf("firewallZoneToModel: %v", diags)
	}
	if out.ID.ValueString() != "z1" {
		t.Errorf("ID = %q, want z1", out.ID.ValueString())
	}
	if out.Name.ValueString() != "DMZ" {
		t.Errorf("Name = %q, want DMZ", out.Name.ValueString())
	}
	if out.Site.ValueString() != "default" {
		t.Errorf("Site = %q, want default", out.Site.ValueString())
	}
	if out.ZoneKey.ValueString() != "dmz" {
		t.Errorf("ZoneKey = %q, want dmz", out.ZoneKey.ValueString())
	}
	var gotNids []string
	if diags := out.NetworkIDs.ElementsAs(ctx, &gotNids, false); diags.HasError() {
		t.Fatalf("reading network_ids: %v", diags)
	}
	if len(gotNids) != 2 {
		t.Errorf("network_ids round-trip = %v, want 2 entries", gotNids)
	}
}
