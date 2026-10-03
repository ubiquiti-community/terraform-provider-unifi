package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// #484 part 1: the controller stores whatever the UI accepted in
// ipsec_peer_ip, including a dynamic-DNS hostname. peer_ip used to be an
// IPv4-only type, so importing such a tunnel failed during refresh - before any
// configuration was even compared.
func TestSiteToSiteVPNPeerHostname(t *testing.T) {
	ctx := context.Background()
	r := &siteToSiteVPNResource{}

	for _, peer := range []string{"203.0.113.9", "peer.dyndns.example"} {
		t.Run(peer, func(t *testing.T) {
			model := &siteToSiteVPNResourceModel{}
			network := &unifi.Network{
				ID:          "net-1",
				Purpose:     unifi.PurposeSiteVPN,
				IPSecPeerIP: &peer,
			}
			if diags := r.networkToModel(ctx, network, model, "default"); diags.HasError() {
				t.Fatalf("networkToModel: %v", diags)
			}
			if model.PeerIP.ValueString() != peer {
				t.Errorf("peer_ip = %v, want %q verbatim", model.PeerIP, peer)
			}
		})
	}

	t.Run("an empty peer reads back null", func(t *testing.T) {
		empty := ""
		model := &siteToSiteVPNResourceModel{}
		if diags := r.networkToModel(ctx, &unifi.Network{
			ID: "net-1", Purpose: unifi.PurposeSiteVPN, IPSecPeerIP: &empty,
		}, model, "default"); diags.HasError() {
			t.Fatalf("networkToModel: %v", diags)
		}
		if !model.PeerIP.IsNull() {
			t.Errorf("peer_ip = %v, want null", model.PeerIP)
		}
	})
}

// #484 part 2: the IKE identifiers were absent from the resource entirely, and
// from go-unifi's site-vpn payload. Now that they travel, an unmanaged
// identifier must stay off the wire - sending an empty value with its flag
// false would strip authentication from a live tunnel.
func TestSiteToSiteVPNIKEIdentifiers(t *testing.T) {
	ctx := context.Background()
	r := &siteToSiteVPNResource{}

	base := func() *siteToSiteVPNResourceModel {
		return &siteToSiteVPNResourceModel{
			Name:                    types.StringValue("HQ-to-Branch"),
			Enabled:                 types.BoolValue(true),
			Interface:               types.StringValue("wan"),
			PeerIP:                  types.StringValue("203.0.113.9"),
			KeyExchange:             types.StringValue("ikev2"),
			RemoteSubnets:           types.ListNull(types.StringType),
			LocalIdentifier:         types.StringNull(),
			LocalIdentifierEnabled:  types.BoolNull(),
			RemoteIdentifier:        types.StringNull(),
			RemoteIdentifierEnabled: types.BoolNull(),
		}
	}

	t.Run("unmanaged identifiers stay off the wire", func(t *testing.T) {
		got, diags := r.modelToNetwork(ctx, base())
		if diags.HasError() {
			t.Fatalf("modelToNetwork: %v", diags)
		}
		if got.IPSecLocalIDentifier != nil {
			t.Errorf("local identifier = %q, want nil", *got.IPSecLocalIDentifier)
		}
		if got.IPSecRemoteIDentifier != nil {
			t.Errorf("remote identifier = %q, want nil", *got.IPSecRemoteIDentifier)
		}
		// With nothing configured the flags must be false, which omitempty then
		// keeps off the wire entirely.
		if got.IPSecLocalIDentifierEnabled || got.IPSecRemoteIDentifierEnabled {
			t.Error("the enabled flags must be false when nothing is configured")
		}
	})

	t.Run("setting an identifier enables it", func(t *testing.T) {
		model := base()
		model.LocalIdentifier = types.StringValue("local.example.com")
		model.RemoteIdentifier = types.StringValue("remote.example.com")

		got, diags := r.modelToNetwork(ctx, model)
		if diags.HasError() {
			t.Fatalf("modelToNetwork: %v", diags)
		}
		if got.IPSecLocalIDentifier == nil || *got.IPSecLocalIDentifier != "local.example.com" {
			t.Errorf("local identifier = %v, want local.example.com", got.IPSecLocalIDentifier)
		}
		if !got.IPSecLocalIDentifierEnabled || !got.IPSecRemoteIDentifierEnabled {
			t.Error("configuring an identifier must enable it, as the UI does")
		}
	})

	t.Run("an explicit flag wins over the implied one", func(t *testing.T) {
		model := base()
		model.LocalIdentifier = types.StringValue("local.example.com")
		model.LocalIdentifierEnabled = types.BoolValue(false)

		got, diags := r.modelToNetwork(ctx, model)
		if diags.HasError() {
			t.Fatalf("modelToNetwork: %v", diags)
		}
		if got.IPSecLocalIDentifierEnabled {
			t.Error("an explicit false must win over the value-implied true")
		}
	})

	t.Run("controller identifiers are read back", func(t *testing.T) {
		local, remote := "local.example.com", "remote.example.com"
		model := &siteToSiteVPNResourceModel{}
		if diags := r.networkToModel(ctx, &unifi.Network{
			ID:                           "net-1",
			Purpose:                      unifi.PurposeSiteVPN,
			IPSecLocalIDentifier:         &local,
			IPSecLocalIDentifierEnabled:  true,
			IPSecRemoteIDentifier:        &remote,
			IPSecRemoteIDentifierEnabled: false,
		}, model, "default"); diags.HasError() {
			t.Fatalf("networkToModel: %v", diags)
		}
		if model.LocalIdentifier.ValueString() != local {
			t.Errorf("local_identifier = %v, want %q", model.LocalIdentifier, local)
		}
		if !model.LocalIdentifierEnabled.ValueBool() {
			t.Error("local_identifier_enabled should be true")
		}
		if model.RemoteIdentifierEnabled.ValueBool() {
			t.Error("remote_identifier_enabled should be false")
		}
	})
}
