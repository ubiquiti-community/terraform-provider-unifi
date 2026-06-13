package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/iptypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// TestSiteToSiteVPNModelRoundTrip validates the model <-> go-unifi Network
// conversion for the unifi_site_to_site_vpn resource (#78). It is a unit test
// rather than an acceptance test because the dockerized acceptance controller
// has no WAN/peer to establish an IPsec tunnel; the live round-trip is exercised
// against a real controller during development.
func TestSiteToSiteVPNModelRoundTrip(t *testing.T) {
	ctx := context.Background()
	r := &siteToSiteVPNResource{}

	subnets, d := types.ListValueFrom(
		ctx,
		types.StringType,
		[]string{"192.0.2.0/24", "198.51.100.0/24"},
	)
	if d.HasError() {
		t.Fatalf("building remote_subnets: %v", d)
	}
	model := &siteToSiteVPNResourceModel{
		Name:          types.StringValue("HQ-to-Branch"),
		Enabled:       types.BoolValue(true),
		Interface:     types.StringValue("wan"),
		PeerIP:        iptypes.NewIPv4AddressValue("203.0.113.9"),
		KeyExchange:   types.StringValue("ikev2"),
		PreSharedKey:  types.StringValue("s3cret-psk"),
		RemoteSubnets: subnets,
		Profile:       types.StringValue("customized"),
		IKEEncryption: types.StringValue("aes256"),
		IKEDhGroup:    types.Int64Value(14),
		PFS:           types.BoolValue(true),
	}

	network, diags := r.modelToNetwork(ctx, model)
	if diags.HasError() {
		t.Fatalf("modelToNetwork: %v", diags)
	}
	if network.Purpose != unifi.PurposeSiteVPN {
		t.Errorf("Purpose = %q, want %q", network.Purpose, unifi.PurposeSiteVPN)
	}
	if network.VPNType == nil || *network.VPNType != "ipsec-vpn" {
		t.Errorf("VPNType = %v, want ipsec-vpn", network.VPNType)
	}
	if network.IPSecPeerIP == nil || *network.IPSecPeerIP != "203.0.113.9" {
		t.Errorf("IPSecPeerIP = %v", network.IPSecPeerIP)
	}
	if network.IPSecPreSharedKey == nil || *network.IPSecPreSharedKey != "s3cret-psk" {
		t.Errorf("IPSecPreSharedKey not set")
	}
	if network.IPSecDhGroup == nil || *network.IPSecDhGroup != 14 {
		t.Errorf("IPSecDhGroup = %v, want 14", network.IPSecDhGroup)
	}
	if !network.IPSecPfs {
		t.Error("IPSecPfs = false, want true")
	}
	if len(network.RemoteVPNSubnets) != 2 {
		t.Errorf("RemoteVPNSubnets = %v, want 2 entries", network.RemoteVPNSubnets)
	}

	// API -> model: secret is preserved (not re-read), other fields map back.
	apiNetwork := &unifi.Network{
		ID:                "net-1",
		Name:              unifi.Ptr("HQ-to-Branch"),
		Purpose:           unifi.PurposeSiteVPN,
		Enabled:           true,
		VPNType:           unifi.Ptr("ipsec-vpn"),
		IPSecInterface:    unifi.Ptr("wan"),
		IPSecPeerIP:       unifi.Ptr("203.0.113.9"),
		IPSecKeyExchange:  unifi.Ptr("ikev2"),
		IPSecPreSharedKey: unifi.Ptr("echoed-by-controller"),
		IPSecPfs:          true,
		RemoteVPNSubnets:  []string{"192.0.2.0/24", "198.51.100.0/24"},
	}
	out := &siteToSiteVPNResourceModel{
		PreSharedKey: types.StringValue("s3cret-psk"), // prior state value
	}
	if diags := r.networkToModel(ctx, apiNetwork, out, "default"); diags.HasError() {
		t.Fatalf("networkToModel: %v", diags)
	}
	if out.ID.ValueString() != "net-1" {
		t.Errorf("ID = %q, want net-1", out.ID.ValueString())
	}
	if out.PeerIP.ValueString() != "203.0.113.9" {
		t.Errorf("PeerIP = %q", out.PeerIP.ValueString())
	}
	// The controller echoes the PSK on read, but networkToModel must preserve the
	// configured/state value to avoid perpetual diffs.
	if out.PreSharedKey.ValueString() != "s3cret-psk" {
		t.Errorf(
			"PreSharedKey = %q, want preserved s3cret-psk (not the API echo)",
			out.PreSharedKey.ValueString(),
		)
	}
	if l := len(out.RemoteSubnets.Elements()); l != 2 {
		t.Errorf("RemoteSubnets length = %d, want 2", l)
	}
}
