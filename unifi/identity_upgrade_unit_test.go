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
			if got := schemaResp.IdentitySchema.GetVersion(); got != 2 {
				t.Fatalf("identity schema version = %d, want 2", got)
			}
			if _, ok := r.UpgradeIdentity(ctx)[1]; !ok {
				t.Fatal("missing identity upgrader for version 1")
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

			// Terraform 1.12+ also upgrades a state written without any
			// identity; that must stay fully null so Read derives it.
			resp = fwresource.UpgradeIdentityResponse{
				Identity: &tfsdk.ResourceIdentity{Schema: schemaResp.IdentitySchema},
			}
			upgrader.IdentityUpgrader(ctx, fwresource.UpgradeIdentityRequest{
				RawIdentity: &tfprotov6.RawState{},
			}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
			if resp.Identity.Raw.IsNull() || !resp.Identity.Raw.IsFullyNull() {
				t.Errorf(
					"identity upgraded from none = %s, want every attribute null",
					resp.Identity.Raw,
				)
			}
		})
	}
}

func TestUpgradeSiteIdentityBranches(t *testing.T) {
	ctx := context.Background()

	upgrade := func(t *testing.T, r fwresource.ResourceWithIdentity, client *Client, raw *tfprotov6.RawState) (map[string]tftypes.Value, fwresource.UpgradeIdentityResponse) {
		t.Helper()
		var schemaResp fwresource.IdentitySchemaResponse
		r.IdentitySchema(ctx, fwresource.IdentitySchemaRequest{}, &schemaResp)
		resp := fwresource.UpgradeIdentityResponse{
			Identity: &tfsdk.ResourceIdentity{Schema: schemaResp.IdentitySchema},
		}
		upgradeSiteIdentity(
			ctx,
			client,
			fwresource.UpgradeIdentityRequest{RawIdentity: raw},
			&resp,
		)
		var got map[string]tftypes.Value
		if !resp.Diagnostics.HasError() {
			if err := resp.Identity.Raw.As(&got); err != nil {
				t.Fatalf("upgraded identity is not an object: %s", err)
			}
		}
		return got, resp
	}
	str := func(t *testing.T, v tftypes.Value) *string {
		t.Helper()
		if v.IsNull() {
			return nil
		}
		var s string
		if err := v.As(&s); err != nil {
			t.Fatalf("not a string: %s", err)
		}
		return &s
	}

	t.Run("invalid json", func(t *testing.T) {
		_, resp := upgrade(t, &firewallGroupResource{}, &Client{Site: "default"},
			&tfprotov6.RawState{JSON: []byte(`{`)})
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected an error for unparseable identity")
		}
	})

	t.Run("stored site is kept", func(t *testing.T) {
		got, resp := upgrade(t, &firewallGroupResource{}, &Client{Site: "default"},
			&tfprotov6.RawState{JSON: []byte(`{"id":"abc","site":"other"}`)})
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
		}
		if s := str(t, got["site"]); s == nil || *s != "other" {
			t.Errorf("site = %v, want \"other\"", s)
		}
	})

	t.Run("no client leaves site null", func(t *testing.T) {
		got, resp := upgrade(t, &firewallGroupResource{}, nil,
			&tfprotov6.RawState{JSON: []byte(`{"id":"abc"}`)})
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
		}
		if s := str(t, got["site"]); s != nil {
			t.Errorf("site = %q, want null", *s)
		}
		if s := str(t, got["id"]); s == nil || *s != "abc" {
			t.Errorf("id = %v, want \"abc\"", s)
		}
	})

	// A state written by Terraform < 1.12 has no identity, but Terraform
	// 1.12+ still requests the upgrade. Filling only site made the result a
	// stored identity with a null id, which Read then either rejected as an
	// identity change or kept. v0.60.0 stored that null-id identity as
	// version 1, so it has to upgrade to no identity as well.
	for name, raw := range map[string]*tfprotov6.RawState{
		"no stored identity":       nil,
		"empty stored identity":    {},
		"empty object":             {JSON: []byte(`{}`)},
		"null identity":            {JSON: []byte(`null`)},
		"identity with empty keys": {JSON: []byte(`{"id":"","site":""}`)},
		"null id from v0.60.0":     {JSON: []byte(`{"id":null,"site":"default"}`)},
	} {
		t.Run(name, func(t *testing.T) {
			got, resp := upgrade(t, &firewallGroupResource{}, &Client{Site: "default"}, raw)
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
			if resp.Identity.Raw.IsNull() {
				t.Fatal("upgraded identity is null, which the framework rejects")
			}
			for attr, v := range got {
				if s := str(t, v); s != nil {
					t.Errorf("%s = %q, want null", attr, *s)
				}
			}
		})
	}

	t.Run("wireguard network_id is carried over", func(t *testing.T) {
		got, resp := upgrade(t, &wireguardPeerResource{}, &Client{Site: "default"},
			&tfprotov6.RawState{JSON: []byte(`{"id":"abc","network_id":"net1"}`)})
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
		}
		if s := str(t, got["network_id"]); s == nil || *s != "net1" {
			t.Errorf("network_id = %v, want \"net1\"", s)
		}
	})
}
