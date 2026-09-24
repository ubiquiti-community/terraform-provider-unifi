package unifi

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type identityUpgradeResource interface {
	fwresource.ResourceWithIdentity
	fwresource.ResourceWithUpgradeIdentity
}

// TestSiteIdentityUpgradeV0 guards the v0.56.0 regression where these
// resources gained identity attributes without a version bump, so every
// {"id"} identity written by v0.55.0 failed to decode.
func TestSiteIdentityUpgradeV0(t *testing.T) {
	ctx := context.Background()
	client := &Client{Site: "default"}

	resources := map[string]identityUpgradeResource{
		"ap_group":         &apGroupResource{client: client},
		"client_qos_rate":  &clientQosRateResource{client: client},
		"dns_record":       &dnsRecordFrameworkResource{client: client},
		"firewall_group":   &firewallGroupResource{client: client},
		"firewall_policy":  &firewallPolicyResource{client: client},
		"firewall_rule":    &firewallRuleResource{client: client},
		"firewall_zone":    &firewallZoneResource{client: client},
		"network":          &networkResource{client: client},
		"port_forward":     &portForwardResource{client: client},
		"port_profile":     &portProfileResource{client: client},
		"power_supervisor": &powerSupervisorResource{client: client},
		"radius_profile":   &radiusProfileResource{client: client},
		"radius_user":      &radiusUserResource{client: client},
		"site_to_site_vpn": &siteToSiteVPNResource{client: client},
		"static_route":     &staticRouteFrameworkResource{client: client},
		"traffic_route":    &trafficRouteResource{client: client},
		"vpn_client":       &vpnClientResource{client: client},
		"vpn_server":       &vpnServerResource{client: client},
		"wan":              &wanResource{client: client},
		"wireguard_peer":   &wireguardPeerResource{client: client},
		"wlan":             &wlanFrameworkResource{client: client},
	}

	for name, r := range resources {
		t.Run(name, func(t *testing.T) {
			var schemaResp fwresource.IdentitySchemaResponse
			r.IdentitySchema(ctx, fwresource.IdentitySchemaRequest{}, &schemaResp)
			if got := schemaResp.IdentitySchema.GetVersion(); got != 1 {
				t.Fatalf("identity schema version = %d, want 1", got)
			}
			upgrader, ok := r.UpgradeIdentity(ctx)[0]
			if !ok {
				t.Fatal("missing identity upgrader for version 0")
			}

			resp := fwresource.UpgradeIdentityResponse{
				Identity: &tfsdk.ResourceIdentity{Schema: schemaResp.IdentitySchema},
			}
			upgrader.IdentityUpgrader(ctx, fwresource.UpgradeIdentityRequest{
				RawIdentity: &tfprotov6.RawState{JSON: []byte(`{"id":"6a50ca48706390bd9c2259c9"}`)},
			}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}

			var got map[string]tftypes.Value
			if err := resp.Identity.Raw.As(&got); err != nil {
				t.Fatalf("upgraded identity is not an object: %s", err)
			}
			want := map[string]string{"id": "6a50ca48706390bd9c2259c9", "site": "default"}
			for attr, w := range want {
				var s string
				if err := got[attr].As(&s); err != nil || s != w {
					t.Errorf("%s = %v, want %q", attr, got[attr], w)
				}
			}
		})
	}
}
