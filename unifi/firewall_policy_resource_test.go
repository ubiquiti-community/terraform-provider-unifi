package unifi

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func ptrInt64(v int64) *int64 { return &v }

// TestFirewallPolicyEndpointSpecificPort is a unit round-trip for the SPECIFIC
// port match (#207): a `port` set on a firewall policy endpoint must reach the
// go-unifi source/destination struct. It guards the fix where the port value
// was previously unrepresentable and silently dropped.
//
// This is a unit test (model -> API conversion) rather than an acceptance test
// because exercising it end-to-end requires zone-based firewall and named
// firewall zones, which the dockerized acceptance controller does not provide.
func TestFirewallPolicyEndpointSpecificPort(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	m := firewallPolicyEndpointModel{
		ZoneID:           types.StringValue("zone-1"),
		MatchingTarget:   types.StringValue("ANY"),
		NetworkIDs:       types.ListNull(types.StringType),
		ClientMACs:       types.ListNull(types.StringType),
		IPs:              types.ListNull(types.StringType),
		Port:             types.StringValue("443"),
		PortGroupID:      types.StringNull(),
		PortMatchingType: types.StringValue("SPECIFIC"),
	}

	src := endpointModelToSource(ctx, m, &diags)
	if diags.HasError() {
		t.Fatalf("source conversion errored: %v", diags)
	}
	if src.Port != "443" {
		t.Errorf("source Port = %q, want 443", src.Port)
	}
	if src.PortMatchingType != "SPECIFIC" {
		t.Errorf("source PortMatchingType = %q, want SPECIFIC", src.PortMatchingType)
	}

	m.Port = types.StringValue("8080")
	dst := endpointModelToDestination(ctx, m, &diags)
	if diags.HasError() {
		t.Fatalf("destination conversion errored: %v", diags)
	}
	if dst.Port != "8080" {
		t.Errorf("destination Port = %q, want 8080", dst.Port)
	}
	if dst.PortMatchingType != "SPECIFIC" {
		t.Errorf("destination PortMatchingType = %q, want SPECIFIC", dst.PortMatchingType)
	}
}

// TestFirewallPolicyPortStringHandling guards #288 and #286. A portless endpoint
// must serialize no port at all (an empty Go string the go-unifi marshaler drops)
// — not "0", which freezes the gateway firewall config (#288). A comma-separated
// list must survive (#286). On read, the controller's "" and the legacy "0" both
// map back to a null port so plans stay clean.
func TestFirewallPolicyPortStringHandling(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	base := firewallPolicyEndpointModel{
		ZoneID:           types.StringValue("z1"),
		MatchingTarget:   types.StringValue("ANY"),
		NetworkIDs:       types.ListNull(types.StringType),
		ClientMACs:       types.ListNull(types.StringType),
		IPs:              types.ListNull(types.StringType),
		WebDomains:       types.ListNull(types.StringType),
		PortGroupID:      types.StringNull(),
		PortMatchingType: types.StringValue("ANY"),
	}

	// Portless endpoint: model -> API must produce an empty port (omitted, never "0").
	for _, port := range []types.String{types.StringNull(), types.StringValue("")} {
		m := base
		m.Port = port
		if got := endpointModelToSource(ctx, m, &diags).Port; got != "" {
			t.Errorf("portless source Port = %q, want empty", got)
		}
	}

	// Comma-separated list survives (#286).
	m := base
	m.Port = types.StringValue("80,443")
	m.PortMatchingType = types.StringValue("SPECIFIC")
	if got := endpointModelToDestination(ctx, m, &diags).Port; got != "80,443" {
		t.Errorf("multi-port destination Port = %q, want 80,443", got)
	}
	if diags.HasError() {
		t.Fatalf("conversion errored: %v", diags)
	}

	// API -> model: "" and the legacy "0" both become null; a real list survives.
	cases := map[string]bool{"": true, "0": true, "161": false, "1812,1813": false}
	for apiPort, wantNull := range cases {
		got := apiSourceToEndpointModel(
			ctx,
			&unifi.FirewallPolicySource{ZoneID: "z1", MatchingTarget: "IP", Port: apiPort},
			&diags,
		)
		if got.Port.IsNull() != wantNull {
			t.Errorf("read port %q: IsNull = %v, want %v", apiPort, got.Port.IsNull(), wantNull)
		}
		if !wantNull && got.Port.ValueString() != apiPort {
			t.Errorf("read port %q: ValueString = %q", apiPort, got.Port.ValueString())
		}
	}
}

// TestFirewallPolicyPreservesFirmwareFields guards #220: the UCG Max firmware
// rejects a PUT that omits connection_state_type, icmp_typename, icmp_v6_typename
// or the source/destination matching_target_type. These fields are not
// user-settable, so the provider round-trips them through state. This test reads
// an API object into the model and converts it back, asserting nothing is dropped.
func TestFirewallPolicyPreservesFirmwareFields(t *testing.T) {
	ctx := context.Background()

	// A policy as the controller returns it, with all firmware-managed fields set.
	api := &unifi.FirewallPolicy{
		ID:                  "pol-1",
		Name:                "allow-vpn-to-nas-snmp",
		Action:              "ALLOW",
		Enabled:             true,
		Protocol:            "all",
		Version:             "BOTH",
		ConnectionStateType: "ALL",
		ICMPTypename:        "ANY",
		ICMPV6Typename:      "ANY",
		Source: &unifi.FirewallPolicySource{
			ZoneID:             "zone-vpn",
			MatchingTarget:     "IP",
			MatchingTargetType: "OBJECT",
		},
		Destination: &unifi.FirewallPolicyDestination{
			ZoneID:             "zone-internal",
			MatchingTarget:     "IP",
			MatchingTargetType: "OBJECT",
			PortMatchingType:   "SPECIFIC",
			Port:               "161",
		},
	}

	// Read API -> model (Read/Create response path).
	var model firewallPolicyModel
	if diags := firewallPolicyToModel(ctx, api, &model); diags.HasError() {
		t.Fatalf("firewallPolicyToModel errored: %v", diags)
	}
	if model.ConnectionStateType.ValueString() != "ALL" {
		t.Errorf("ConnectionStateType = %q, want ALL", model.ConnectionStateType.ValueString())
	}
	if model.ICMPTypename.ValueString() != "ANY" {
		t.Errorf("ICMPTypename = %q, want ANY", model.ICMPTypename.ValueString())
	}
	if model.ICMPV6Typename.ValueString() != "ANY" {
		t.Errorf("ICMPV6Typename = %q, want ANY", model.ICMPV6Typename.ValueString())
	}

	// Convert model -> API (Update PUT path) and assert the fields survive.
	out, diags := modelToFirewallPolicy(ctx, model)
	if diags.HasError() {
		t.Fatalf("modelToFirewallPolicy errored: %v", diags)
	}
	if out.ConnectionStateType != "ALL" {
		t.Errorf("PUT ConnectionStateType = %q, want ALL", out.ConnectionStateType)
	}
	if out.ICMPTypename != "ANY" {
		t.Errorf("PUT ICMPTypename = %q, want ANY", out.ICMPTypename)
	}
	if out.ICMPV6Typename != "ANY" {
		t.Errorf("PUT ICMPV6Typename = %q, want ANY", out.ICMPV6Typename)
	}
	if out.Source == nil || out.Source.MatchingTargetType != "OBJECT" {
		t.Errorf("PUT source MatchingTargetType not preserved: %+v", out.Source)
	}
	if out.Destination == nil || out.Destination.MatchingTargetType != "OBJECT" {
		t.Errorf("PUT destination MatchingTargetType not preserved: %+v", out.Destination)
	}
	if out.Destination == nil || out.Destination.Port != "161" {
		t.Errorf("PUT destination Port not preserved: %+v", out.Destination)
	}
}

// TestFirewallPolicyConnectionStatesRoundTrip guards #227: a policy whose
// connection_state_type is CUSTOM must round-trip its connection_states. The
// model->API conversion previously hard-coded an empty slice, so updates sent
// "connection_states": [] and the firmware rejected CUSTOM policies (HTTP 400).
func TestFirewallPolicyConnectionStatesRoundTrip(t *testing.T) {
	ctx := context.Background()
	fp := &unifi.FirewallPolicy{
		ID:                  "p1",
		Name:                "deny-vpn-to-lan",
		Action:              "BLOCK",
		Protocol:            "all",
		ConnectionStateType: "CUSTOM",
		ConnectionStates:    []string{"NEW", "ESTABLISHED"},
		Source: &unifi.FirewallPolicySource{
			ZoneID:           "z1",
			MatchingTarget:   "ANY",
			PortMatchingType: "ANY",
		},
		Destination: &unifi.FirewallPolicyDestination{
			ZoneID:           "z2",
			MatchingTarget:   "ANY",
			PortMatchingType: "ANY",
		},
	}

	var model firewallPolicyModel
	if d := firewallPolicyToModel(ctx, fp, &model); d.HasError() {
		t.Fatalf("firewallPolicyToModel: %v", d)
	}
	var states []string
	if d := model.ConnectionStates.ElementsAs(ctx, &states, false); d.HasError() {
		t.Fatalf("reading connection_states: %v", d)
	}
	if len(states) != 2 || states[0] != "NEW" || states[1] != "ESTABLISHED" {
		t.Errorf("read connection_states = %v, want [NEW ESTABLISHED]", states)
	}

	out, d := modelToFirewallPolicy(ctx, model)
	if d.HasError() {
		t.Fatalf("modelToFirewallPolicy: %v", d)
	}
	if len(out.ConnectionStates) != 2 || out.ConnectionStates[0] != "NEW" ||
		out.ConnectionStates[1] != "ESTABLISHED" {
		t.Errorf("PUT dropped connection_states: %v, want [NEW ESTABLISHED]", out.ConnectionStates)
	}
}

// TestFirewallPolicyEndpointListFieldsRoundTrip guards #242 and the wiring of the
// list-typed match fields. web_domains (FQDN matching, matching_target=WEB) is new;
// network_ids and client_macs were declared in the schema but never mapped to/from
// the API (model->API dropped them, API->model forced them to null). This asserts
// all three survive both conversion directions.
func TestFirewallPolicyEndpointListFieldsRoundTrip(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	webDomains, _ := types.ListValueFrom(
		ctx,
		types.StringType,
		[]string{"example.com", "ads.example.net"},
	)
	networkIDs, _ := types.ListValueFrom(ctx, types.StringType, []string{"net-1", "net-2"})
	clientMACs, _ := types.ListValueFrom(ctx, types.StringType, []string{"00:11:22:33:44:55"})

	m := firewallPolicyEndpointModel{
		ZoneID:           types.StringValue("zone-1"),
		MatchingTarget:   types.StringValue("WEB"),
		NetworkIDs:       networkIDs,
		ClientMACs:       clientMACs,
		IPs:              types.ListNull(types.StringType),
		WebDomains:       webDomains,
		Port:             types.StringNull(),
		PortGroupID:      types.StringNull(),
		PortMatchingType: types.StringValue("ANY"),
	}

	// model -> API (PUT path)
	src := endpointModelToSource(ctx, m, &diags)
	if diags.HasError() {
		t.Fatalf("source conversion errored: %v", diags)
	}
	if len(src.WebDomains) != 2 || src.WebDomains[0] != "example.com" {
		t.Errorf("source WebDomains = %v, want [example.com ads.example.net]", src.WebDomains)
	}
	if len(src.NetworkIDs) != 2 || src.NetworkIDs[1] != "net-2" {
		t.Errorf("source NetworkIDs = %v, want [net-1 net-2]", src.NetworkIDs)
	}
	if len(src.ClientMACs) != 1 || src.ClientMACs[0] != "00:11:22:33:44:55" {
		t.Errorf("source ClientMACs = %v, want [00:11:22:33:44:55]", src.ClientMACs)
	}

	dst := endpointModelToDestination(ctx, m, &diags)
	if diags.HasError() {
		t.Fatalf("destination conversion errored: %v", diags)
	}
	if len(dst.WebDomains) != 2 || dst.WebDomains[1] != "ads.example.net" {
		t.Errorf("destination WebDomains = %v, want [example.com ads.example.net]", dst.WebDomains)
	}

	// API -> model (read path)
	apiSrc := &unifi.FirewallPolicySource{
		ZoneID:         "zone-1",
		MatchingTarget: "WEB",
		WebDomains:     []string{"example.com"},
		NetworkIDs:     []string{"net-9"},
		ClientMACs:     []string{"aa:bb:cc:dd:ee:ff"},
	}
	got := apiSourceToEndpointModel(ctx, apiSrc, &diags)
	if diags.HasError() {
		t.Fatalf("apiSourceToEndpointModel errored: %v", diags)
	}
	var wd, nids, macs []string
	got.WebDomains.ElementsAs(ctx, &wd, false)
	got.NetworkIDs.ElementsAs(ctx, &nids, false)
	got.ClientMACs.ElementsAs(ctx, &macs, false)
	if len(wd) != 1 || wd[0] != "example.com" {
		t.Errorf("read WebDomains = %v, want [example.com]", wd)
	}
	if len(nids) != 1 || nids[0] != "net-9" {
		t.Errorf("read NetworkIDs = %v, want [net-9]", nids)
	}
	if len(macs) != 1 || macs[0] != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("read ClientMACs = %v, want [aa:bb:cc:dd:ee:ff]", macs)
	}
}

// TestFirewallPolicyICMPProtocolRoundTrip guards #259: zone-based firewall ICMP
// policies (protocol "icmp"/"icmpv6") were rejected by the schema's OneOf
// validator even though the controller accepts and returns them. This asserts
// the protocol survives both conversion directions once the validator allows it.
func TestFirewallPolicyICMPProtocolRoundTrip(t *testing.T) {
	ctx := context.Background()
	for _, proto := range []string{"icmp", "icmpv6"} {
		fp := &unifi.FirewallPolicy{
			ID:       "p-icmp",
			Name:     "allow-internal-ping",
			Action:   "ALLOW",
			Protocol: proto,
			Source: &unifi.FirewallPolicySource{
				ZoneID:           "z1",
				MatchingTarget:   "IP",
				PortMatchingType: "ANY",
			},
			Destination: &unifi.FirewallPolicyDestination{
				ZoneID:           "z2",
				MatchingTarget:   "IP",
				PortMatchingType: "ANY",
			},
		}

		var model firewallPolicyModel
		if d := firewallPolicyToModel(ctx, fp, &model); d.HasError() {
			t.Fatalf("[%s] firewallPolicyToModel: %v", proto, d)
		}
		if model.Protocol.ValueString() != proto {
			t.Errorf("[%s] read Protocol = %q, want %q", proto, model.Protocol.ValueString(), proto)
		}

		out, d := modelToFirewallPolicy(ctx, model)
		if d.HasError() {
			t.Fatalf("[%s] modelToFirewallPolicy: %v", proto, d)
		}
		if out.Protocol != proto {
			t.Errorf("[%s] PUT dropped Protocol = %q, want %q", proto, out.Protocol, proto)
		}
	}
}

func TestNewFirewallPolicyResource(t *testing.T) {
	got := NewFirewallPolicyResource()
	if got == nil {
		t.Fatal("NewFirewallPolicyResource() returned nil")
	}
	if _, ok := got.(fwresource.ResourceWithImportState); !ok {
		t.Errorf("NewFirewallPolicyResource() does not implement resource.ResourceWithImportState")
	}
	if _, ok := got.(fwresource.ResourceWithIdentity); !ok {
		t.Errorf("NewFirewallPolicyResource() does not implement resource.ResourceWithIdentity")
	}
}

func TestNewFirewallPolicyListResource(t *testing.T) {
	got := NewFirewallPolicyListResource()
	if got == nil {
		t.Fatal("NewFirewallPolicyListResource() returned nil")
	}
	if _, ok := got.(fwlist.ListResourceWithConfigure); !ok {
		t.Errorf(
			"NewFirewallPolicyListResource() does not implement list.ListResourceWithConfigure",
		)
	}
}

func Test_firewallPolicyEndpointModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    firewallPolicyEndpointModel
		want map[string]attr.Type
	}{
		{
			name: "returns expected attribute types",
			m:    firewallPolicyEndpointModel{},
			want: map[string]attr.Type{
				"zone_id":              types.StringType,
				"matching_target":      types.StringType,
				"network_ids":          types.ListType{ElemType: types.StringType},
				"client_macs":          types.ListType{ElemType: types.StringType},
				"ips":                  types.ListType{ElemType: types.StringType},
				"web_domains":          types.ListType{ElemType: types.StringType},
				"port":                 types.StringType,
				"port_group_id":        types.StringType,
				"port_matching_type":   types.StringType,
				"matching_target_type": types.StringType,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("firewallPolicyEndpointModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_firewallPolicyResource_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.MetadataRequest
		resp *fwresource.MetadataResponse
	}
	tests := []struct {
		name string
		r    *firewallPolicyResource
		args args
	}{
		{
			name: "type name is unifi_firewall_policy",
			r:    &firewallPolicyResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.MetadataRequest{ProviderTypeName: "unifi"},
				resp: &fwresource.MetadataResponse{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Metadata(tt.args.ctx, tt.args.req, tt.args.resp)
			if tt.args.resp.TypeName != "unifi_firewall_policy" {
				t.Errorf("TypeName = %q, want unifi_firewall_policy", tt.args.resp.TypeName)
			}
		})
	}
}

func Test_firewallPolicyResource_IdentitySchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwresource.IdentitySchemaRequest
		resp *fwresource.IdentitySchemaResponse
	}
	tests := []struct {
		name string
		r    *firewallPolicyResource
		args args
	}{
		{
			name: "has id attribute",
			r:    &firewallPolicyResource{},
			args: args{
				in0:  context.Background(),
				in1:  fwresource.IdentitySchemaRequest{},
				resp: &fwresource.IdentitySchemaResponse{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.IdentitySchema(tt.args.in0, tt.args.in1, tt.args.resp)
			if _, ok := tt.args.resp.IdentitySchema.Attributes["id"]; !ok {
				t.Error("IdentitySchema missing 'id' attribute")
			}
		})
	}
}

func Test_firewallPolicyResource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.SchemaRequest
		resp *fwresource.SchemaResponse
	}
	tests := []struct {
		name string
		r    *firewallPolicyResource
		args args
	}{
		{
			name: "schema has key attributes",
			r:    &firewallPolicyResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.SchemaRequest{},
				resp: &fwresource.SchemaResponse{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Schema(tt.args.ctx, tt.args.req, tt.args.resp)
			for _, key := range []string{"id", "name", "action", "source", "destination"} {
				if _, ok := tt.args.resp.Schema.Attributes[key]; !ok {
					t.Errorf("Schema missing %q attribute", key)
				}
			}
		})
	}
}

func Test_firewallPolicyResource_Configure(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ConfigureRequest
		resp *fwresource.ConfigureResponse
	}
	tests := []struct {
		name string
		r    *firewallPolicyResource
		args args
	}{
		{
			name: "nil provider data",
			r:    &firewallPolicyResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.ConfigureRequest{ProviderData: nil},
				resp: &fwresource.ConfigureResponse{},
			},
		},
		{
			name: "wrong type",
			r:    &firewallPolicyResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.ConfigureRequest{ProviderData: "wrong"},
				resp: &fwresource.ConfigureResponse{},
			},
		},
		{
			name: "correct client",
			r:    &firewallPolicyResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.ConfigureRequest{ProviderData: &Client{}},
				resp: &fwresource.ConfigureResponse{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Configure(tt.args.ctx, tt.args.req, tt.args.resp)
			switch tt.name {
			case "nil provider data":
				if tt.args.resp.Diagnostics.HasError() {
					t.Error("nil ProviderData should not error")
				}
			case "wrong type":
				if !tt.args.resp.Diagnostics.HasError() {
					t.Error("wrong type should produce an error")
				}
			case "correct client":
				if tt.args.resp.Diagnostics.HasError() {
					t.Errorf("correct client should not error: %v", tt.args.resp.Diagnostics)
				}
				if tt.r.client == nil {
					t.Error("client should be set after Configure")
				}
			}
		})
	}
}

func Test_modelToFirewallPolicy(t *testing.T) {
	type args struct {
		ctx   context.Context
		model firewallPolicyModel
	}
	tests := []struct {
		name  string
		args  args
		want  *unifi.FirewallPolicy
		want1 diag.Diagnostics
	}{
		{
			name: "basic allow-lan policy",
			args: args{
				ctx: context.Background(),
				model: func() firewallPolicyModel {
					ctx := context.Background()
					srcEndpoint := firewallPolicyEndpointModel{
						ZoneID:             types.StringValue("z1"),
						MatchingTarget:     types.StringValue("ANY"),
						MatchingTargetType: types.StringNull(),
						NetworkIDs:         types.ListNull(types.StringType),
						ClientMACs:         types.ListNull(types.StringType),
						IPs:                types.ListNull(types.StringType),
						WebDomains:         types.ListNull(types.StringType),
						Port:               types.StringNull(),
						PortGroupID:        types.StringNull(),
						PortMatchingType:   types.StringValue("ANY"),
					}
					srcObj, _ := types.ObjectValueFrom(
						ctx,
						firewallPolicyEndpointModel{}.AttributeTypes(),
						srcEndpoint,
					)
					dstEndpoint := firewallPolicyEndpointModel{
						ZoneID:             types.StringValue("z2"),
						MatchingTarget:     types.StringValue("ANY"),
						MatchingTargetType: types.StringNull(),
						NetworkIDs:         types.ListNull(types.StringType),
						ClientMACs:         types.ListNull(types.StringType),
						IPs:                types.ListNull(types.StringType),
						WebDomains:         types.ListNull(types.StringType),
						Port:               types.StringNull(),
						PortGroupID:        types.StringNull(),
						PortMatchingType:   types.StringValue("ANY"),
					}
					dstObj, _ := types.ObjectValueFrom(
						ctx,
						firewallPolicyEndpointModel{}.AttributeTypes(),
						dstEndpoint,
					)
					return firewallPolicyModel{
						Name:                types.StringValue("allow-lan"),
						Action:              types.StringValue("ALLOW"),
						Enabled:             types.BoolValue(true),
						Protocol:            types.StringValue("all"),
						Description:         types.StringNull(),
						Logging:             types.BoolValue(false),
						Index:               types.Int64Null(),
						CreateAllowRespond:  types.BoolValue(false),
						IPVersion:           types.StringNull(),
						ConnectionStateType: types.StringNull(),
						ConnectionStates:    types.ListNull(types.StringType),
						ICMPTypename:        types.StringNull(),
						ICMPV6Typename:      types.StringNull(),
						Source:              srcObj,
						Destination:         dstObj,
						ID:                  types.StringNull(),
						Site:                types.StringNull(),
					}
				}(),
			},
			want: &unifi.FirewallPolicy{
				Name:             "allow-lan",
				Action:           "ALLOW",
				Enabled:          true,
				Protocol:         "all",
				ConnectionStates: []string{},
				Schedule:         &unifi.FirewallPolicySchedule{Mode: "ALWAYS"},
				Source: &unifi.FirewallPolicySource{
					ZoneID:           "z1",
					MatchingTarget:   "ANY",
					PortMatchingType: "ANY",
				},
				Destination: &unifi.FirewallPolicyDestination{
					ZoneID:           "z2",
					MatchingTarget:   "ANY",
					PortMatchingType: "ANY",
				},
			},
			want1: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := modelToFirewallPolicy(tt.args.ctx, tt.args.model)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("modelToFirewallPolicy() got = %+v, want %+v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("modelToFirewallPolicy() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_endpointModelToSource(t *testing.T) {
	type args struct {
		ctx   context.Context
		m     firewallPolicyEndpointModel
		diags *diag.Diagnostics
	}
	tests := []struct {
		name string
		args args
		want *unifi.FirewallPolicySource
	}{
		{
			name: "source with IP matching",
			args: args{
				ctx:   context.Background(),
				diags: &diag.Diagnostics{},
				m: func() firewallPolicyEndpointModel {
					ctx := context.Background()
					ips, _ := types.ListValueFrom(ctx, types.StringType, []string{"10.0.0.1"})
					return firewallPolicyEndpointModel{
						ZoneID:             types.StringValue("z1"),
						MatchingTarget:     types.StringValue("IP"),
						MatchingTargetType: types.StringValue("OBJECT"),
						NetworkIDs:         types.ListNull(types.StringType),
						ClientMACs:         types.ListNull(types.StringType),
						IPs:                ips,
						WebDomains:         types.ListNull(types.StringType),
						Port:               types.StringNull(),
						PortGroupID:        types.StringNull(),
						PortMatchingType:   types.StringValue("ANY"),
					}
				}(),
			},
			want: &unifi.FirewallPolicySource{
				ZoneID:             "z1",
				MatchingTarget:     "IP",
				MatchingTargetType: "OBJECT",
				PortMatchingType:   "ANY",
				IPs:                []string{"10.0.0.1"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := endpointModelToSource(
				tt.args.ctx,
				tt.args.m,
				tt.args.diags,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("endpointModelToSource() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_endpointModelToDestination(t *testing.T) {
	type args struct {
		ctx   context.Context
		m     firewallPolicyEndpointModel
		diags *diag.Diagnostics
	}
	tests := []struct {
		name string
		args args
		want *unifi.FirewallPolicyDestination
	}{
		{
			name: "destination with IP matching",
			args: args{
				ctx:   context.Background(),
				diags: &diag.Diagnostics{},
				m: func() firewallPolicyEndpointModel {
					ctx := context.Background()
					ips, _ := types.ListValueFrom(ctx, types.StringType, []string{"192.168.1.1"})
					return firewallPolicyEndpointModel{
						ZoneID:             types.StringValue("z2"),
						MatchingTarget:     types.StringValue("IP"),
						MatchingTargetType: types.StringValue("OBJECT"),
						NetworkIDs:         types.ListNull(types.StringType),
						ClientMACs:         types.ListNull(types.StringType),
						IPs:                ips,
						WebDomains:         types.ListNull(types.StringType),
						Port:               types.StringValue("80"),
						PortGroupID:        types.StringNull(),
						PortMatchingType:   types.StringValue("SPECIFIC"),
					}
				}(),
			},
			want: &unifi.FirewallPolicyDestination{
				ZoneID:             "z2",
				MatchingTarget:     "IP",
				MatchingTargetType: "OBJECT",
				Port:               "80",
				PortMatchingType:   "SPECIFIC",
				IPs:                []string{"192.168.1.1"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := endpointModelToDestination(
				tt.args.ctx,
				tt.args.m,
				tt.args.diags,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("endpointModelToDestination() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_firewallPolicyToModel(t *testing.T) {
	type args struct {
		ctx   context.Context
		fp    *unifi.FirewallPolicy
		model *firewallPolicyModel
	}
	tests := []struct {
		name string
		args args
		want diag.Diagnostics
	}{
		{
			name: "basic policy to model",
			args: args{
				ctx: context.Background(),
				fp: &unifi.FirewallPolicy{
					ID:       "pol-1",
					Name:     "test-policy",
					Action:   "ALLOW",
					Enabled:  true,
					Protocol: "all",
					Source: &unifi.FirewallPolicySource{
						ZoneID:           "z1",
						MatchingTarget:   "ANY",
						PortMatchingType: "ANY",
					},
					Destination: &unifi.FirewallPolicyDestination{
						ZoneID:           "z2",
						MatchingTarget:   "ANY",
						PortMatchingType: "ANY",
					},
				},
				model: &firewallPolicyModel{},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := firewallPolicyToModel(
				tt.args.ctx,
				tt.args.fp,
				tt.args.model,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("firewallPolicyToModel() = %v, want %v", got, tt.want)
			}
			if tt.args.model.Name.ValueString() != tt.args.fp.Name {
				t.Errorf(
					"model.Name = %q, want %q",
					tt.args.model.Name.ValueString(),
					tt.args.fp.Name,
				)
			}
			if tt.args.model.ID.ValueString() != tt.args.fp.ID {
				t.Errorf("model.ID = %q, want %q", tt.args.model.ID.ValueString(), tt.args.fp.ID)
			}
		})
	}
}

func Test_apiSourceToEndpointModel(t *testing.T) {
	type args struct {
		ctx   context.Context
		src   *unifi.FirewallPolicySource
		diags *diag.Diagnostics
	}
	tests := []struct {
		name string
		args args
		want firewallPolicyEndpointModel
	}{
		{
			name: "source with IP and port",
			args: args{
				ctx:   context.Background(),
				diags: &diag.Diagnostics{},
				src: &unifi.FirewallPolicySource{
					ZoneID:             "z1",
					MatchingTarget:     "IP",
					MatchingTargetType: "OBJECT",
					IPs:                []string{"10.0.0.1"},
					PortMatchingType:   "SPECIFIC",
					Port:               "443",
				},
			},
			want: func() firewallPolicyEndpointModel {
				ctx := context.Background()
				ips, _ := types.ListValueFrom(ctx, types.StringType, []string{"10.0.0.1"})
				networkIDs, _ := types.ListValueFrom(ctx, types.StringType, ([]string)(nil))
				clientMACs, _ := types.ListValueFrom(ctx, types.StringType, ([]string)(nil))
				webDomains, _ := types.ListValueFrom(ctx, types.StringType, ([]string)(nil))
				return firewallPolicyEndpointModel{
					ZoneID:             types.StringValue("z1"),
					MatchingTarget:     types.StringValue("IP"),
					MatchingTargetType: types.StringValue("OBJECT"),
					IPs:                ips,
					NetworkIDs:         networkIDs,
					ClientMACs:         clientMACs,
					WebDomains:         webDomains,
					Port:               types.StringValue("443"),
					PortGroupID:        types.StringValue(""),
					PortMatchingType:   types.StringValue("SPECIFIC"),
				}
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := apiSourceToEndpointModel(
				tt.args.ctx,
				tt.args.src,
				tt.args.diags,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("apiSourceToEndpointModel() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_apiDestinationToEndpointModel(t *testing.T) {
	type args struct {
		ctx   context.Context
		dst   *unifi.FirewallPolicyDestination
		diags *diag.Diagnostics
	}
	tests := []struct {
		name string
		args args
		want firewallPolicyEndpointModel
	}{
		{
			name: "destination with port",
			args: args{
				ctx:   context.Background(),
				diags: &diag.Diagnostics{},
				dst: &unifi.FirewallPolicyDestination{
					ZoneID:             "z2",
					MatchingTarget:     "ANY",
					MatchingTargetType: "OBJECT",
					PortMatchingType:   "SPECIFIC",
					Port:               "8080",
				},
			},
			want: func() firewallPolicyEndpointModel {
				ctx := context.Background()
				ips, _ := types.ListValueFrom(ctx, types.StringType, ([]string)(nil))
				networkIDs, _ := types.ListValueFrom(ctx, types.StringType, ([]string)(nil))
				clientMACs, _ := types.ListValueFrom(ctx, types.StringType, ([]string)(nil))
				webDomains, _ := types.ListValueFrom(ctx, types.StringType, ([]string)(nil))
				return firewallPolicyEndpointModel{
					ZoneID:             types.StringValue("z2"),
					MatchingTarget:     types.StringValue("ANY"),
					MatchingTargetType: types.StringValue("OBJECT"),
					IPs:                ips,
					NetworkIDs:         networkIDs,
					ClientMACs:         clientMACs,
					WebDomains:         webDomains,
					Port:               types.StringValue("8080"),
					PortGroupID:        types.StringValue(""),
					PortMatchingType:   types.StringValue("SPECIFIC"),
				}
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := apiDestinationToEndpointModel(
				tt.args.ctx,
				tt.args.dst,
				tt.args.diags,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("apiDestinationToEndpointModel() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_firewallPolicyResource_firewallPolicyListToModel(t *testing.T) {
	type args struct {
		ctx   context.Context
		api   *unifi.FirewallPolicy
		model *firewallPolicyModel
		site  string
	}
	tests := []struct {
		name string
		r    *firewallPolicyResource
		args args
		want diag.Diagnostics
	}{
		{
			name: "sets site and populates model",
			r:    &firewallPolicyResource{},
			args: args{
				ctx: context.Background(),
				api: &unifi.FirewallPolicy{
					ID:       "pol-1",
					Name:     "list-test",
					Action:   "BLOCK",
					Protocol: "all",
					Source: &unifi.FirewallPolicySource{
						ZoneID:           "z1",
						MatchingTarget:   "ANY",
						PortMatchingType: "ANY",
					},
					Destination: &unifi.FirewallPolicyDestination{
						ZoneID:           "z2",
						MatchingTarget:   "ANY",
						PortMatchingType: "ANY",
					},
				},
				model: &firewallPolicyModel{},
				site:  "mysite",
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.firewallPolicyListToModel(
				tt.args.ctx,
				tt.args.api,
				tt.args.model,
				tt.args.site,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf(
					"firewallPolicyResource.firewallPolicyListToModel() = %v, want %v",
					got,
					tt.want,
				)
			}
			if tt.args.model.Site.ValueString() != tt.args.site {
				t.Errorf("model.Site = %q, want %q", tt.args.model.Site.ValueString(), tt.args.site)
			}
		})
	}
}

func Test_firewallPolicyResource_ListResourceConfigSchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwlist.ListResourceSchemaRequest
		resp *fwlist.ListResourceSchemaResponse
	}
	tests := []struct {
		name string
		r    *firewallPolicyResource
		args args
	}{
		{
			name: "has site attribute",
			r:    &firewallPolicyResource{},
			args: args{
				in0:  context.Background(),
				in1:  fwlist.ListResourceSchemaRequest{},
				resp: &fwlist.ListResourceSchemaResponse{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.ListResourceConfigSchema(tt.args.in0, tt.args.in1, tt.args.resp)
			if _, ok := tt.args.resp.Schema.Attributes["site"]; !ok {
				t.Error("ListResourceConfigSchema missing 'site' attribute")
			}
		})
	}
}

// TestFirewallPolicyMatchingTargetType guards #293: a specific (non-ANY) match
// must carry a concrete matching_target_type — the controller rejects an empty
// one (api.err.MissingFirewallPolicySourceMatchingTargetType) when a source is
// switched from ANY to e.g. IP. A controller-assigned type is preserved.
func TestFirewallPolicyMatchingTargetType(t *testing.T) {
	cases := []struct {
		matchingTarget, current, want string
	}{
		{"IP", "", "SPECIFIC"},         // ANY -> IP, type was dropped
		{"IP", "ANY", "SPECIFIC"},      // ANY -> IP, stale "ANY" left over
		{"IP", "SPECIFIC", "SPECIFIC"}, // already correct
		{"IP", "OBJECT", "OBJECT"},     // controller-assigned object/group preserved
		{"NETWORK", "", "SPECIFIC"},
		{"ANY", "", ""}, // ANY source untouched
		{"ANY", "ANY", "ANY"},
	}
	for _, c := range cases {
		if got := firewallPolicyMatchingTargetType(c.matchingTarget, c.current); got != c.want {
			t.Errorf("matchingTargetType(%q,%q) = %q, want %q",
				c.matchingTarget, c.current, got, c.want)
		}
	}

	// End-to-end: an IP source whose type was lost serializes SPECIFIC.
	ctx := context.Background()
	var diags diag.Diagnostics
	ips, _ := types.ListValueFrom(ctx, types.StringType, []string{"10.0.40.138"})
	m := firewallPolicyEndpointModel{
		ZoneID:             types.StringValue("z1"),
		MatchingTarget:     types.StringValue("IP"),
		IPs:                ips,
		NetworkIDs:         types.ListNull(types.StringType),
		ClientMACs:         types.ListNull(types.StringType),
		WebDomains:         types.ListNull(types.StringType),
		Port:               types.StringNull(),
		PortGroupID:        types.StringNull(),
		PortMatchingType:   types.StringValue("ANY"),
		MatchingTargetType: types.StringValue("ANY"),
	}
	if got := endpointModelToSource(ctx, m, &diags).MatchingTargetType; got != "SPECIFIC" {
		t.Errorf("source MatchingTargetType = %q, want SPECIFIC", got)
	}
}
