package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// #544: an Optional+Computed attribute that also carries a static Default is
// overwritten by that Default, because the framework applies defaults
// (TransformDefaults) before it runs plan modifiers (SchemaModifyPlan). On the
// VPN resources that re-enabled a tunnel disabled on the controller, moved it
// to another WAN or port, narrowed a v6 WireGuard server to v4, and on
// unifi_wireguard_peer planned `allowed_ips = []`, dropping the routes behind
// a peer.
//
// Each case adopts a resource whose controller values all differ from the old
// defaults, declares only what a minimal configuration must, and requires an
// empty plan.
func TestVPNAdoptedResourcesKeepControllerValues(t *testing.T) {
	cases := []struct {
		name     string
		typeName string
		resource resource.Resource
		declare  []string
		live     map[string]attr.Value
		config   map[string]attr.Value
	}{
		{
			name:     "vpn_server wireguard",
			typeName: "unifi_vpn_server",
			resource: &vpnServerResource{},
			declare:  []string{"wireguard"},
			live: map[string]attr.Value{
				"enabled": types.BoolValue(false),
				// A server created in the UI stores neither key.
				"wan.ip":               types.StringNull(),
				"wan.interface":        types.StringNull(),
				"wireguard.port":       types.Int64Value(51821),
				"wireguard.ip_version": types.StringValue("v6"),
			},
		},
		{
			name:     "vpn_server wan block declared for the interface only",
			typeName: "unifi_vpn_server",
			resource: &vpnServerResource{},
			declare:  []string{"wireguard", "wan"},
			live: map[string]attr.Value{
				"wan.ip":        types.StringNull(),
				"wan.interface": types.StringValue("wan2"),
			},
			config: map[string]attr.Value{
				"wan.interface": types.StringValue("wan2"),
			},
		},
		{
			name:     "vpn_server l2tp",
			typeName: "unifi_vpn_server",
			resource: &vpnServerResource{},
			declare:  []string{"l2tp"},
			live: map[string]attr.Value{
				"l2tp.allow_weak_ciphers": types.BoolValue(true),
			},
		},
		{
			name:     "vpn_server openvpn",
			typeName: "unifi_vpn_server",
			resource: &vpnServerResource{},
			declare:  []string{"openvpn"},
			live: map[string]attr.Value{
				"openvpn.port":              types.Int64Value(1195),
				"openvpn.mode":              types.StringValue("site-to-site"),
				"openvpn.encryption_cipher": types.StringValue("AES_256_CBC"),
			},
		},
		{
			name:     "vpn_server openvpn without mode or cipher on the controller",
			typeName: "unifi_vpn_server",
			resource: &vpnServerResource{},
			declare:  []string{"openvpn"},
			live: map[string]attr.Value{
				"openvpn.mode":              types.StringNull(),
				"openvpn.encryption_cipher": types.StringNull(),
			},
		},
		{
			name:     "vpn_client",
			typeName: "unifi_vpn_client",
			resource: &vpnClientResource{},
			live: map[string]attr.Value{
				"enabled":                         types.BoolValue(false),
				"default_route":                   types.BoolValue(true),
				"pull_dns":                        types.BoolValue(true),
				"wireguard.preshared_key_enabled": types.BoolValue(true),
				"wireguard.interface":             types.StringValue("wan2"),
			},
		},
		{
			name:     "vpn_client without an interface on the controller",
			typeName: "unifi_vpn_client",
			resource: &vpnClientResource{},
			live: map[string]attr.Value{
				"wireguard.interface": types.StringNull(),
			},
		},
		{
			name:     "site_to_site_vpn",
			typeName: "unifi_site_to_site_vpn",
			resource: &siteToSiteVPNResource{},
			live: map[string]attr.Value{
				"enabled":      types.BoolValue(false),
				"interface":    types.StringValue("wan2"),
				"key_exchange": types.StringValue("ikev1"),
			},
		},
		{
			name:     "site_to_site_vpn without interface or key exchange on the controller",
			typeName: "unifi_site_to_site_vpn",
			resource: &siteToSiteVPNResource{},
			live: map[string]attr.Value{
				"interface":    types.StringNull(),
				"key_exchange": types.StringNull(),
			},
		},
		{
			name:     "wireguard_peer",
			typeName: "unifi_wireguard_peer",
			resource: &wireguardPeerResource{},
			live: map[string]attr.Value{
				"allowed_ips": types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("192.0.2.0/24"),
				}),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prior, planned := adoptedPlan(
				t, tc.typeName, tc.resource, tc.declare, tc.live, tc.config,
			)
			assertNoPlannedChanges(t, prior, planned)
		})
	}
}

// TestVPNServerCreateKeepsDefaults pins the create side: with the Defaults
// gone, a new server must still be created the way it always was.
func TestVPNServerCreateKeepsDefaults(t *testing.T) {
	t.Run("plan", func(t *testing.T) {
		attrs := createPlan(t, "unifi_vpn_server", &vpnServerResource{},
			[]string{"wireguard", "wan"})
		wg := plannedObject(t, attrs["wireguard"], "wireguard")
		assertPlanned(t, wg["ip_version"], "wireguard.ip_version", "v4")
		// Filled in by vpnServerCreateDefaults.
		assertPlannedUnknown(t, wg["port"], "wireguard.port")
		assertPlannedUnknown(t, attrs["enabled"], "enabled")
		wan := plannedObject(t, attrs["wan"], "wan")
		assertPlanned(t, wan["ip"], "wan.ip", "any")
		assertPlanned(t, wan["interface"], "wan.interface", "wan")

		attrs = createPlan(t, "unifi_vpn_server", &vpnServerResource{},
			[]string{"openvpn"})
		ovpn := plannedObject(t, attrs["openvpn"], "openvpn")
		assertPlanned(t, ovpn["mode"], "openvpn.mode", "server")
		assertPlanned(t, ovpn["encryption_cipher"], "openvpn.encryption_cipher", "AES_256_GCM")
		// An undeclared wan block still sends nothing, as before.
		assertPlannedUnknown(t, attrs["wan"], "wan")
	})

	t.Run("request", func(t *testing.T) {
		for vpnType, port := range map[string]int64{
			"wireguard-server": 51820,
			"openvpn-server":   1194,
		} {
			n := &unifi.Network{VPNType: new(vpnType)}
			vpnServerCreateDefaults(n)
			if n.Enabled == nil || !*n.Enabled {
				t.Errorf("%s: enabled = %v, want true", vpnType, n.Enabled)
			}
			if n.LocalPort == nil || *n.LocalPort != port {
				t.Errorf("%s: local_port = %v, want %d", vpnType, n.LocalPort, port)
			}
		}

		// Declared values win.
		n := &unifi.Network{
			VPNType:   new("wireguard-server"),
			Enabled:   new(false),
			LocalPort: new(int64(51999)),
		}
		vpnServerCreateDefaults(n)
		if *n.Enabled || *n.LocalPort != 51999 {
			t.Errorf("declared values overwritten: enabled=%v port=%v", *n.Enabled, *n.LocalPort)
		}
	})
}

// TestVPNTunnelCreatePlans pins the create side of the client and
// site-to-site resources: the WAN interface and key exchange are still planned
// for a new tunnel, and enabled is filled in by networkCreateEnabled.
func TestVPNTunnelCreatePlans(t *testing.T) {
	attrs := createPlan(t, "unifi_vpn_client", &vpnClientResource{}, nil)
	wg := plannedObject(t, attrs["wireguard"], "wireguard")
	assertPlanned(t, wg["interface"], "wireguard.interface", "wan")
	assertPlannedUnknown(t, attrs["enabled"], "enabled")

	attrs = createPlan(t, "unifi_site_to_site_vpn", &siteToSiteVPNResource{}, nil)
	assertPlanned(t, attrs["interface"], "interface", "wan")
	assertPlanned(t, attrs["key_exchange"], "key_exchange", "ikev2")
	assertPlannedUnknown(t, attrs["enabled"], "enabled")

	n := &unifi.Network{}
	networkCreateEnabled(n)
	if n.Enabled == nil || !*n.Enabled {
		t.Errorf("enabled = %v, want true", n.Enabled)
	}
	n = &unifi.Network{Enabled: new(false)}
	networkCreateEnabled(n)
	if *n.Enabled {
		t.Error("a declared enabled = false was overwritten")
	}
}
