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
// (TransformDefaults) before it runs plan modifiers (SchemaModifyPlan). Here
// that re-enabled QoS rules, content filters and DNS records disabled on the
// controller, reset a rule's burst allowance, lifted the limits of a
// rate-limited client group, and planned to replace a dynamic DNS entry bound
// to `wan2`.
//
// Each case adopts a resource whose controller values all differ from the old
// defaults, declares only what a minimal configuration must, and requires an
// empty plan.
func TestQoSAndDNSAdoptedResourcesKeepControllerValues(t *testing.T) {
	cases := []struct {
		name     string
		typeName string
		resource resource.Resource
		declare  []string
		live     map[string]attr.Value
		config   map[string]attr.Value
	}{
		{
			name:     "qos_rule",
			typeName: "unifi_qos_rule",
			resource: &qosRuleResource{},
			live: map[string]attr.Value{
				"enabled":        types.BoolValue(false),
				"download_burst": types.StringValue("LONG"),
				"upload_burst":   types.StringValue("SHORT"),
			},
		},
		{
			name:     "client_qos_rate",
			typeName: "unifi_client_qos_rate",
			resource: &clientQosRateResource{},
			live: map[string]attr.Value{
				"qos_rate.max_down": types.Int64Value(5000),
				"qos_rate.max_up":   types.Int64Value(1000),
			},
		},
		{
			name:     "client_qos_rate qos_rate block declared for one limit",
			typeName: "unifi_client_qos_rate",
			resource: &clientQosRateResource{},
			declare:  []string{"qos_rate"},
			live: map[string]attr.Value{
				"qos_rate.max_down": types.Int64Value(5000),
				"qos_rate.max_up":   types.Int64Value(1000),
			},
			config: map[string]attr.Value{
				"qos_rate.max_down": types.Int64Value(5000),
			},
		},
		{
			name:     "content_filter",
			typeName: "unifi_content_filter",
			resource: &contentFilterResource{},
			live:     map[string]attr.Value{"enabled": types.BoolValue(false)},
		},
		{
			name:     "dns_record",
			typeName: "unifi_dns_record",
			resource: &dnsRecordFrameworkResource{},
			live:     map[string]attr.Value{"enabled": types.BoolValue(false)},
		},
		{
			name:     "dynamic_dns",
			typeName: "unifi_dynamic_dns",
			resource: &dynamicDNSResource{},
			live:     map[string]attr.Value{"interface": types.StringValue("wan2")},
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

// TestQoSAndDNSCreateKeepsDefaults pins the create side: with the Defaults
// gone, new objects are still created the way they always were.
func TestQoSAndDNSCreateKeepsDefaults(t *testing.T) {
	attrs := createPlan(t, "unifi_qos_rule", &qosRuleResource{}, nil)
	assertPlanned(t, attrs["download_burst"], "download_burst", "OFF")
	assertPlanned(t, attrs["upload_burst"], "upload_burst", "OFF")
	assertPlannedUnknown(t, attrs["enabled"], "enabled")

	attrs = createPlan(t, "unifi_dynamic_dns", &dynamicDNSResource{}, nil)
	assertPlanned(t, attrs["interface"], "interface", "wan")

	attrs = createPlan(t, "unifi_client_qos_rate", &clientQosRateResource{}, nil)
	assertPlannedUnknown(t, attrs["qos_rate"], "qos_rate")
	// A declared block leaves its undeclared limits to clientQosRateCreateDefaults.
	attrs = createPlan(t, "unifi_client_qos_rate", &clientQosRateResource{}, []string{"qos_rate"})
	rate := plannedObject(t, attrs["qos_rate"], "qos_rate")
	assertPlannedUnknown(t, rate["max_down"], "qos_rate.max_down")
	assertPlannedUnknown(t, rate["max_up"], "qos_rate.max_up")

	for _, typeName := range []string{"unifi_content_filter", "unifi_dns_record"} {
		var r resource.Resource = &contentFilterResource{}
		if typeName == "unifi_dns_record" {
			r = &dnsRecordFrameworkResource{}
		}
		attrs = createPlan(t, typeName, r, nil)
		assertPlannedUnknown(t, attrs["enabled"], typeName+".enabled")
	}

	g := &unifi.ClientGroup{}
	clientQosRateCreateDefaults(g)
	if g.QOSRateMaxDown == nil || *g.QOSRateMaxDown != -1 ||
		g.QOSRateMaxUp == nil || *g.QOSRateMaxUp != -1 {
		t.Errorf("create defaults: %v %v, want -1/-1", g.QOSRateMaxDown, g.QOSRateMaxUp)
	}
	g = &unifi.ClientGroup{QOSRateMaxDown: new(int64(5000))}
	clientQosRateCreateDefaults(g)
	if *g.QOSRateMaxDown != 5000 || g.QOSRateMaxUp == nil || *g.QOSRateMaxUp != -1 {
		t.Errorf("declared value overwritten: %v %v", *g.QOSRateMaxDown, g.QOSRateMaxUp)
	}
}
