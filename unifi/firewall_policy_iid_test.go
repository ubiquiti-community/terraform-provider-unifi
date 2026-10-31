package unifi

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// The controller stores an IID endpoint exactly as sent (verified on Network
// 10.6.109), with the interface identifier as "<address>/<mask>".
const testFirewallPolicyIID = "::eaff:1eff:fed7:bf17/::ffff:ffff:ffff:ffff"

func firewallPolicyIIDEndpoint() firewallPolicyEndpointModel {
	return firewallPolicyEndpointModel{
		ZoneID:                types.StringValue("zone-internal"),
		MatchingTarget:        types.StringValue("IID"),
		MatchingTargetType:    types.StringValue("SPECIFIC"),
		NetworkIDs:            types.ListNull(types.StringType),
		ClientMACs:            types.ListNull(types.StringType),
		IPs:                   types.ListNull(types.StringType),
		WebDomains:            types.ListNull(types.StringType),
		Regions:               types.ListNull(types.StringType),
		IID:                   types.StringValue(testFirewallPolicyIID),
		Port:                  types.StringValue("443"),
		PortGroupID:           types.StringValue(""),
		IPGroupID:             types.StringValue(""),
		PortMatchingType:      types.StringValue("SPECIFIC"),
		MatchOppositeIPs:      types.BoolValue(false),
		MatchOppositeNetworks: types.BoolValue(false),
		MatchOppositePorts:    types.BoolValue(false),
		MatchOppositeIID:      types.BoolValue(true),
		MatchMAC:              types.BoolValue(false),
	}
}

func TestFirewallPolicyIIDSchema(t *testing.T) {
	r := &firewallPolicyResource{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	for _, ep := range []string{"source", "destination"} {
		nested, ok := resp.Schema.Attributes[ep].(schema.SingleNestedAttribute)
		if !ok {
			t.Fatalf("%s is not a SingleNestedAttribute", ep)
		}
		if len(nested.Validators) == 0 {
			t.Errorf("%s has no endpoint validator", ep)
		}
		iid, ok := nested.Attributes["iid"].(schema.StringAttribute)
		if !ok || !iid.Optional || !iid.Computed || iid.Default != nil ||
			len(iid.PlanModifiers) == 0 {
			t.Errorf("%s.iid must be Optional+Computed with a create-only default, got %#v",
				ep, iid)
		}
		opp, ok := nested.Attributes["match_opposite_iid"].(schema.BoolAttribute)
		if !ok || !opp.Optional || !opp.Computed || opp.Default != nil ||
			len(opp.PlanModifiers) == 0 {
			t.Errorf(
				"%s.match_opposite_iid must be Optional+Computed with a create-only default, got %#v",
				ep, opp,
			)
		}
	}
}

// TestFirewallPolicyIIDRoundTrip covers model -> API -> model on both
// endpoints: the interface ID and its inversion must reach the controller, and
// a policy read back (including on import) must reproduce the configuration.
func TestFirewallPolicyIIDRoundTrip(t *testing.T) {
	ctx := context.Background()
	m := firewallPolicyIIDEndpoint()

	t.Run("source", func(t *testing.T) {
		diags := &diag.Diagnostics{}
		src := endpointModelToSource(ctx, m, diags)
		if diags.HasError() {
			t.Fatalf("diags: %v", diags)
		}
		if src.MatchingTarget != "IID" || src.Iid != testFirewallPolicyIID ||
			!src.MatchOppositeIid {
			t.Fatalf("source = %+v, want IID %s inverted", src, testFirewallPolicyIID)
		}
		if src.MatchingTargetType != "SPECIFIC" {
			t.Errorf("MatchingTargetType = %q, want SPECIFIC", src.MatchingTargetType)
		}
		if len(src.IPs) != 0 || src.IPGroupID != "" {
			t.Errorf("IID source carries an IP match: %+v", src)
		}
		back := apiSourceToEndpointModel(ctx, src, diags)
		if back.IID != m.IID || back.MatchOppositeIID != m.MatchOppositeIID ||
			back.MatchingTarget != m.MatchingTarget {
			t.Errorf(
				"read back %+v, want iid %v match_opposite_iid %v",
				back,
				m.IID,
				m.MatchOppositeIID,
			)
		}
	})

	t.Run("destination", func(t *testing.T) {
		diags := &diag.Diagnostics{}
		dst := endpointModelToDestination(ctx, m, diags)
		if diags.HasError() {
			t.Fatalf("diags: %v", diags)
		}
		if dst.MatchingTarget != "IID" || dst.Iid != testFirewallPolicyIID ||
			!dst.MatchOppositeIid {
			t.Fatalf("destination = %+v, want IID %s inverted", dst, testFirewallPolicyIID)
		}
		back := apiDestinationToEndpointModel(ctx, dst, diags)
		if back.IID != m.IID || back.MatchOppositeIID != m.MatchOppositeIID ||
			back.Port != m.Port {
			t.Errorf(
				"read back %+v, want iid %v match_opposite_iid %v",
				back,
				m.IID,
				m.MatchOppositeIID,
			)
		}
	})

	t.Run("controller document without iid reads the defaults", func(t *testing.T) {
		diags := &diag.Diagnostics{}
		got := apiDestinationToEndpointModel(ctx, &unifi.FirewallPolicyDestination{
			ZoneID: "zone", MatchingTarget: "ANY", PortMatchingType: "ANY",
		}, diags)
		// Matches the schema defaults, so a non-IID policy plans no diff.
		if got.IID != types.StringValue("") || got.MatchOppositeIID != types.BoolValue(false) {
			t.Errorf(
				"iid = %v, match_opposite_iid = %v, want \"\" and false",
				got.IID,
				got.MatchOppositeIID,
			)
		}
	})

	t.Run("non-IID endpoint sends no iid", func(t *testing.T) {
		n := firewallPolicyIIDEndpoint()
		n.MatchingTarget = types.StringValue("ANY")
		n.IID = types.StringValue("")
		n.MatchOppositeIID = types.BoolValue(false)
		diags := &diag.Diagnostics{}
		src := endpointModelToSource(ctx, n, diags)
		if src.Iid != "" || src.MatchOppositeIid {
			t.Errorf("source = %+v, want no iid", src)
		}
	})
}

func TestFirewallPolicyIIDValidator(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		value string
		valid bool
	}{
		{testFirewallPolicyIID, true},
		{"::eaff:1eff:fed7:bf17/::ffff:ffff:ffff:0", true},
		{"2001:db8::1/::ffff:ffff:ffff:ffff", true},
		{"", true}, // unset; the endpoint validator decides
		{"::eaff:1eff:fed7:bf17", false},
		{"::eaff:1eff:fed7:bf17/64", false},
		{"10.0.0.1/255.255.255.0", false},
		{"::ffff:10.0.0.1/::ffff:ffff:ffff:ffff", false},
		{"fe80::1%eth0/::ffff:ffff:ffff:ffff", false},
		{"garbage", false},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			resp := &validator.StringResponse{}
			firewallPolicyIIDValidator{}.ValidateString(ctx, validator.StringRequest{
				Path:        path.Root("destination").AtName("iid"),
				ConfigValue: types.StringValue(tt.value),
			}, resp)
			if got := !resp.Diagnostics.HasError(); got != tt.valid {
				t.Errorf("valid = %v, want %v (%v)", got, tt.valid, resp.Diagnostics)
			}
		})
	}
}

func TestFirewallPolicyEndpointValidator(t *testing.T) {
	ctx := context.Background()
	strList := func(v ...string) types.List {
		l, _ := types.ListValueFrom(ctx, types.StringType, v)
		return l
	}
	tests := []struct {
		name    string
		mutate  func(m *firewallPolicyEndpointModel)
		wantErr string
	}{
		{name: "IID with iid", mutate: func(*firewallPolicyEndpointModel) {}},
		{
			name:    "IID without iid",
			mutate:  func(m *firewallPolicyEndpointModel) { m.IID = types.StringNull() },
			wantErr: "Missing Interface Identifier",
		},
		{
			name:    "IID with empty iid",
			mutate:  func(m *firewallPolicyEndpointModel) { m.IID = types.StringValue("") },
			wantErr: "Missing Interface Identifier",
		},
		{
			name:   "IID with unknown iid",
			mutate: func(m *firewallPolicyEndpointModel) { m.IID = types.StringUnknown() },
		},
		{
			name:    "IID with ips",
			mutate:  func(m *firewallPolicyEndpointModel) { m.IPs = strList("2001:db8::1") },
			wantErr: "Conflicting Matching Target Attribute",
		},
		{
			name:    "IID with ip_group_id",
			mutate:  func(m *firewallPolicyEndpointModel) { m.IPGroupID = types.StringValue("grp") },
			wantErr: "Conflicting Matching Target Attribute",
		},
		{
			name:    "IID with network_ids",
			mutate:  func(m *firewallPolicyEndpointModel) { m.NetworkIDs = strList("net") },
			wantErr: "Conflicting Matching Target Attribute",
		},
		{
			name:    "IID with client_macs",
			mutate:  func(m *firewallPolicyEndpointModel) { m.ClientMACs = strList("aa:bb:cc:dd:ee:ff") },
			wantErr: "Conflicting Matching Target Attribute",
		},
		{
			name:    "IID with web_domains",
			mutate:  func(m *firewallPolicyEndpointModel) { m.WebDomains = strList("example.com") },
			wantErr: "Conflicting Matching Target Attribute",
		},
		{
			name:    "IID with regions",
			mutate:  func(m *firewallPolicyEndpointModel) { m.Regions = strList("DE") },
			wantErr: "Conflicting Matching Target Attribute",
		},
		{
			name: "IP with iid",
			mutate: func(m *firewallPolicyEndpointModel) {
				m.MatchingTarget = types.StringValue("IP")
				m.MatchOppositeIID = types.BoolNull()
			},
			wantErr: "Interface Identifier Without IID Target",
		},
		{
			name: "ANY with match_opposite_iid",
			mutate: func(m *firewallPolicyEndpointModel) {
				m.MatchingTarget = types.StringValue("ANY")
				m.IID = types.StringNull()
			},
			wantErr: "Interface Identifier Inversion Without IID Target",
		},
		{
			name: "IP without iid",
			mutate: func(m *firewallPolicyEndpointModel) {
				m.MatchingTarget = types.StringValue("IP")
				m.IPs = strList("10.0.0.1")
				m.IID = types.StringNull()
				m.MatchOppositeIID = types.BoolNull()
			},
		},
		{
			name: "unknown matching_target",
			mutate: func(m *firewallPolicyEndpointModel) {
				m.MatchingTarget = types.StringUnknown()
				m.IPs = strList("10.0.0.1")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := firewallPolicyIIDEndpoint()
			tt.mutate(&m)
			obj, d := types.ObjectValueFrom(ctx, firewallPolicyEndpointModel{}.AttributeTypes(), m)
			if d.HasError() {
				t.Fatalf("building object: %v", d)
			}
			resp := &validator.ObjectResponse{}
			firewallPolicyEndpointValidator{}.ValidateObject(ctx, validator.ObjectRequest{
				Path:        path.Root("destination"),
				ConfigValue: obj,
			}, resp)
			if tt.wantErr == "" {
				if resp.Diagnostics.HasError() {
					t.Errorf("unexpected error: %v", resp.Diagnostics)
				}
				return
			}
			if !resp.Diagnostics.HasError() {
				t.Fatalf("expected %q, got no error", tt.wantErr)
			}
			if got := resp.Diagnostics.Errors()[0].Summary(); !strings.Contains(got, tt.wantErr) {
				t.Errorf("error = %q, want %q", got, tt.wantErr)
			}
		})
	}
}
