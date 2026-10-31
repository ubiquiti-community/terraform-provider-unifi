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
// routing resources that re-enabled a static or traffic route disabled on the
// controller, moved a switch route to the gateway, switched a traffic route's
// kill switch off (letting traffic leak past the VPN when it is down), and
// turned off an enabled BGP configuration.
//
// Each case adopts a resource whose controller values all differ from the old
// defaults, declares only what a minimal configuration must, and requires an
// empty plan.
func TestRoutingAdoptedResourcesKeepControllerValues(t *testing.T) {
	cases := []struct {
		name     string
		typeName string
		resource resource.Resource
		declare  []string
		live     map[string]attr.Value
		config   map[string]attr.Value
	}{
		{
			name:     "static_route",
			typeName: "unifi_static_route",
			resource: &staticRouteFrameworkResource{},
			live: map[string]attr.Value{
				"enabled":        types.BoolValue(false),
				"gateway.type":   types.StringValue("switch"),
				"gateway.device": types.StringValue("aa:bb:cc:dd:ee:ff"),
			},
		},
		{
			name:     "static_route gateway block declared for the device only",
			typeName: "unifi_static_route",
			resource: &staticRouteFrameworkResource{},
			declare:  []string{"gateway"},
			live: map[string]attr.Value{
				"gateway.type":   types.StringValue("switch"),
				"gateway.device": types.StringValue("aa:bb:cc:dd:ee:ff"),
			},
			config: map[string]attr.Value{
				"gateway.device": types.StringValue("aa:bb:cc:dd:ee:ff"),
			},
		},
		{
			name:     "traffic_route",
			typeName: "unifi_traffic_route",
			resource: &trafficRouteResource{},
			live: map[string]attr.Value{
				"enabled":             types.BoolValue(false),
				"kill_switch_enabled": types.BoolValue(true),
			},
		},
		{
			name:     "bgp",
			typeName: "unifi_bgp",
			resource: &bgpResource{},
			live: map[string]attr.Value{
				"enabled":          types.BoolValue(true),
				"upload_file_name": types.StringValue("edge.conf"),
				"description":      types.StringValue("Edge peering"),
			},
		},
		{
			name:     "bgp without a file name or description on the controller",
			typeName: "unifi_bgp",
			resource: &bgpResource{},
			live: map[string]attr.Value{
				"upload_file_name": types.StringNull(),
				"description":      types.StringNull(),
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

// TestRoutingCreateKeepsDefaults pins the create side: with the Defaults
// gone, a new route or BGP configuration is still created the way it always
// was.
func TestRoutingCreateKeepsDefaults(t *testing.T) {
	attrs := createPlan(t, "unifi_bgp", &bgpResource{}, nil)
	assertPlanned(t, attrs["upload_file_name"], "upload_file_name", "frr.conf")
	assertPlanned(t, attrs["description"], "description", "BGP Configuration")

	attrs = createPlan(t, "unifi_static_route", &staticRouteFrameworkResource{}, nil)
	assertPlanned(t, attrs["enabled"], "enabled", true)
	assertPlannedUnknown(t, attrs["gateway"], "gateway")

	r := &unifi.Routing{}
	staticRouteCreateDefaults(r, types.BoolUnknown())
	if !r.Enabled || r.GatewayType != "default" {
		t.Errorf("create defaults: enabled=%t gateway_type=%q, want true/default",
			r.Enabled, r.GatewayType)
	}
	r = &unifi.Routing{Enabled: false, GatewayType: "switch"}
	staticRouteCreateDefaults(r, types.BoolValue(false))
	if r.Enabled || r.GatewayType != "switch" {
		t.Errorf("declared values overwritten: enabled=%t gateway_type=%q",
			r.Enabled, r.GatewayType)
	}
}
