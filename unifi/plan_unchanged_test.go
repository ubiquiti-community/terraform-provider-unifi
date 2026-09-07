package unifi

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/cidrtypes"
	"github.com/hashicorp/terraform-plugin-framework-nettypes/hwtypes"
	"github.com/hashicorp/terraform-plugin-framework-nettypes/iptypes"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestPlanUnchangedConfigIsEmpty drives the real provider server's plan step
// for each nested-group resource with a configuration that only sets Required
// attributes, and a prior state at defaults with every Computed attribute
// known. The planned state must equal the prior state.
//
// This is the only kind of test that catches the schema pitfall behind the
// nested groups: the framework applies attribute Defaults before its
// "did the plan change?" gate, so an object Default that can never equal the
// prior state (unknown or controller-computed leaves) trips the gate on every
// plan and re-plans the resource's other Computed attributes as unknown — a
// perpetual diff that a demo-controller acceptance test does not see.
func TestPlanUnchangedConfigIsEmpty(t *testing.T) {
	cases := []struct {
		typeName  string
		resource  resource.Resource
		overrides map[string]attr.Value
	}{
		{"unifi_network", &networkResource{}, map[string]attr.Value{
			// ModifyPlan asserts the flat default; everything else drifts freely.
			"ipv6.interface_type": types.StringValue("none"),
		}},
		{"unifi_wlan", &wlanFrameworkResource{}, nil},
		{"unifi_setting", &settingResource{}, nil},
		{"unifi_device", &deviceResource{}, nil},
		{"unifi_port_profile", &portProfileResource{}, nil},
		{"unifi_firewall_rule", &firewallRuleResource{}, nil},
		{"unifi_firewall_policy", &firewallPolicyResource{}, nil},
		{"unifi_site_to_site_vpn", &siteToSiteVPNResource{}, nil},
		{"unifi_radius_profile", &radiusProfileResource{}, nil},
		{"unifi_radius_user", &radiusUserResource{}, nil},
		{"unifi_client_qos_rate", &clientQosRateResource{}, nil},
		{"unifi_static_route", &staticRouteFrameworkResource{}, nil},
		{"unifi_vpn_server", &vpnServerResource{}, nil},
		{"unifi_wan", &wanResource{}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.typeName, func(t *testing.T) {
			ctx := context.Background()
			var schemaResp resource.SchemaResponse
			tc.resource.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			if schemaResp.Diagnostics.HasError() {
				t.Fatalf("schema: %v", schemaResp.Diagnostics)
			}
			f := planFixture{ctx: ctx, t: t, overrides: tc.overrides}
			prior, config := f.object(schemaResp.Schema, "")

			planned := planResourceChange(t, tc.typeName, schemaResp.Schema, prior, config)

			diffs, err := prior.Diff(planned)
			if err != nil {
				t.Fatalf("diff: %v", err)
			}
			if len(diffs) == 0 {
				return
			}
			lines := make([]string, 0, len(diffs))
			for _, d := range diffs {
				lines = append(lines, fmt.Sprintf("  %s: %v -> %v", d.Path, d.Value1, d.Value2))
			}
			sort.Strings(lines)
			t.Errorf("an unchanged configuration produced a non-empty plan:\n%s",
				strings.Join(lines, "\n"))
		})
	}
}

// planResourceChange calls PlanResourceChange on the real provider with the
// proposed new state equal to the prior state (what Terraform core sends for
// an unchanged configuration) and returns the planned state.
func planResourceChange(
	t *testing.T,
	typeName string,
	s schema.Schema,
	prior, config tftypes.Value,
) tftypes.Value {
	t.Helper()
	ctx := context.Background()
	schemaType := s.Type().TerraformType(ctx)

	srv, err := providerserver.NewProtocol6WithError(New())()
	if err != nil {
		t.Fatalf("provider server: %v", err)
	}
	dv := func(v tftypes.Value) *tfprotov6.DynamicValue {
		d, err := tfprotov6.NewDynamicValue(schemaType, v)
		if err != nil {
			t.Fatalf("dynamic value: %v", err)
		}
		return &d
	}
	resp, err := srv.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
		TypeName:         typeName,
		PriorState:       dv(prior),
		ProposedNewState: dv(prior),
		Config:           dv(config),
	})
	if err != nil {
		t.Fatalf("PlanResourceChange: %v", err)
	}
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("plan diagnostic: %s: %s", d.Summary, d.Detail)
		}
	}
	planned, err := resp.PlannedState.Unmarshal(schemaType)
	if err != nil {
		t.Fatalf("unmarshal planned state: %v", err)
	}
	return planned
}

// planFixture synthesizes a prior state and a configuration from a resource
// schema such that an unchanged configuration must plan empty:
//
//   - Required attributes carry the same synthetic value in both;
//   - attributes with a Default carry that default in prior (null in config);
//   - other Computed attributes carry a synthetic known value in prior;
//   - nested objects are built leaf by leaf under the same rules, so their
//     computed leaves are known (as after an apply) even when the object
//     itself has a Default;
//   - Optional-only attributes are null in both;
//   - blocks are empty collections in both.
//
// overrides pins prior values by dotted attribute path.
type planFixture struct {
	ctx       context.Context
	t         *testing.T
	overrides map[string]attr.Value
}

type nestedAttributes interface {
	GetNestedObject() schema.NestedAttributeObject
}

func (f planFixture) object(s schema.Schema, prefix string) (prior, config tftypes.Value) {
	priorVals, configVals := f.attributes(s.Attributes, prefix)
	for name, block := range s.Blocks {
		v := f.block(block)
		priorVals[name] = v
		configVals[name] = v
	}
	objType, ok := s.Type().TerraformType(f.ctx).(tftypes.Object)
	if !ok {
		f.t.Fatalf("schema type is %T, want object", s.Type().TerraformType(f.ctx))
	}
	return tftypes.NewValue(objType, priorVals), tftypes.NewValue(objType, configVals)
}

func (f planFixture) attributes(
	attrs map[string]schema.Attribute,
	prefix string,
) (prior, config map[string]tftypes.Value) {
	prior = make(map[string]tftypes.Value, len(attrs))
	config = make(map[string]tftypes.Value, len(attrs))
	for name, a := range attrs {
		path := prefix + name
		tfType := a.GetType().TerraformType(f.ctx)
		null := tftypes.NewValue(tfType, nil)

		var value tftypes.Value
		switch {
		case f.overrides[path] != nil:
			value = f.toTerraform(f.overrides[path])
		case a.IsRequired() || a.IsComputed():
			// Nested objects are always built leaf by leaf: a post-apply
			// state holds controller values in the computed leaves, never the
			// object Default verbatim, and that is exactly the state an
			// unsound object Default fails to match.
			_, single := a.(schema.SingleNestedAttribute)
			if def, ok := attributeDefault(f.ctx, a); ok && !single {
				value = f.toTerraform(def)
			} else {
				value = f.synthesize(a, path)
			}
		default:
			value = null
		}
		prior[name] = value
		if a.IsRequired() {
			config[name] = value
		} else {
			config[name] = null
		}
	}
	return prior, config
}

func (f planFixture) block(b schema.Block) tftypes.Value {
	tfType := b.Type().TerraformType(f.ctx)
	switch tfType.(type) {
	case tftypes.Set, tftypes.List:
		return tftypes.NewValue(tfType, []tftypes.Value{})
	default:
		return tftypes.NewValue(tfType, nil)
	}
}

// synthesize returns a plausible known value for an attribute without a
// default: nested single objects recurse, collections are empty, scalars get
// a value their custom type (if any) can parse.
func (f planFixture) synthesize(a schema.Attribute, path string) tftypes.Value {
	tfType := a.GetType().TerraformType(f.ctx)
	if single, ok := a.(schema.SingleNestedAttribute); ok {
		vals, _ := f.attributes(single.Attributes, path+".")
		return tftypes.NewValue(tfType, vals)
	}
	if _, ok := a.(nestedAttributes); ok {
		return emptyCollection(tfType)
	}
	switch a.GetType().(type) {
	case timetypes.GoDurationType:
		return tftypes.NewValue(tftypes.String, "1m0s")
	case hwtypes.MACAddressType:
		return tftypes.NewValue(tftypes.String, "aa:bb:cc:dd:ee:ff")
	case iptypes.IPv4AddressType, iptypes.IPAddressType:
		return tftypes.NewValue(tftypes.String, "10.0.0.1")
	case iptypes.IPv6AddressType:
		return tftypes.NewValue(tftypes.String, "fd00::1")
	case cidrtypes.IPv4PrefixType:
		return tftypes.NewValue(tftypes.String, "10.0.0.0/24")
	case cidrtypes.IPv6PrefixType:
		return tftypes.NewValue(tftypes.String, "fd00::/64")
	}
	switch {
	case tfType.Is(tftypes.String):
		return tftypes.NewValue(tftypes.String, "x")
	case tfType.Is(tftypes.Number):
		return tftypes.NewValue(tftypes.Number, big.NewFloat(1))
	case tfType.Is(tftypes.Bool):
		return tftypes.NewValue(tftypes.Bool, false)
	}
	return emptyCollection(tfType)
}

func emptyCollection(tfType tftypes.Type) tftypes.Value {
	switch tfType.(type) {
	case tftypes.List, tftypes.Set, tftypes.Tuple:
		return tftypes.NewValue(tfType, []tftypes.Value{})
	case tftypes.Map, tftypes.Object:
		return tftypes.NewValue(tfType, map[string]tftypes.Value{})
	}
	return tftypes.NewValue(tfType, nil)
}

func (f planFixture) toTerraform(v attr.Value) tftypes.Value {
	tv, err := v.ToTerraformValue(f.ctx)
	if err != nil {
		f.t.Fatalf("to terraform value: %v", err)
	}
	return tv
}

// attributeDefault returns the attribute's static Default value, if any.
func attributeDefault(ctx context.Context, a schema.Attribute) (attr.Value, bool) {
	switch a := a.(type) {
	case schema.StringAttribute:
		if a.Default != nil {
			var resp defaults.StringResponse
			a.Default.DefaultString(ctx, defaults.StringRequest{}, &resp)
			return resp.PlanValue, true
		}
	case schema.BoolAttribute:
		if a.Default != nil {
			var resp defaults.BoolResponse
			a.Default.DefaultBool(ctx, defaults.BoolRequest{}, &resp)
			return resp.PlanValue, true
		}
	case schema.Int64Attribute:
		if a.Default != nil {
			var resp defaults.Int64Response
			a.Default.DefaultInt64(ctx, defaults.Int64Request{}, &resp)
			return resp.PlanValue, true
		}
	case schema.Float64Attribute:
		if a.Default != nil {
			var resp defaults.Float64Response
			a.Default.DefaultFloat64(ctx, defaults.Float64Request{}, &resp)
			return resp.PlanValue, true
		}
	case schema.NumberAttribute:
		if a.Default != nil {
			var resp defaults.NumberResponse
			a.Default.DefaultNumber(ctx, defaults.NumberRequest{}, &resp)
			return resp.PlanValue, true
		}
	case schema.ListAttribute:
		if a.Default != nil {
			var resp defaults.ListResponse
			a.Default.DefaultList(ctx, defaults.ListRequest{}, &resp)
			return resp.PlanValue, true
		}
	case schema.SetAttribute:
		if a.Default != nil {
			var resp defaults.SetResponse
			a.Default.DefaultSet(ctx, defaults.SetRequest{}, &resp)
			return resp.PlanValue, true
		}
	case schema.MapAttribute:
		if a.Default != nil {
			var resp defaults.MapResponse
			a.Default.DefaultMap(ctx, defaults.MapRequest{}, &resp)
			return resp.PlanValue, true
		}
	case schema.SingleNestedAttribute:
		if a.Default != nil {
			var resp defaults.ObjectResponse
			a.Default.DefaultObject(ctx, defaults.ObjectRequest{}, &resp)
			return resp.PlanValue, true
		}
	case schema.ListNestedAttribute:
		if a.Default != nil {
			var resp defaults.ListResponse
			a.Default.DefaultList(ctx, defaults.ListRequest{}, &resp)
			return resp.PlanValue, true
		}
	case schema.SetNestedAttribute:
		if a.Default != nil {
			var resp defaults.SetResponse
			a.Default.DefaultSet(ctx, defaults.SetRequest{}, &resp)
			return resp.PlanValue, true
		}
	case schema.MapNestedAttribute:
		if a.Default != nil {
			var resp defaults.MapResponse
			a.Default.DefaultMap(ctx, defaults.MapRequest{}, &resp)
			return resp.PlanValue, true
		}
	}
	return nil, false
}
