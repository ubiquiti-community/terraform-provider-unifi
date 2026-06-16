package unifi

import (
	"context"
	"reflect"
	"testing"

	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// TestFirewallZoneModelRoundTrip validates the model <-> go-unifi struct
// conversion for the unifi_firewall_zone resource (#214). It is a unit test
// rather than an acceptance test because zone-based firewall is not available
// in the dockerized acceptance controller.
func TestFirewallZoneModelRoundTrip(t *testing.T) {
	ctx := context.Background()
	r := &firewallZoneResource{}

	nids, d := types.ListValueFrom(ctx, types.StringType, []string{"net-a", "net-b"})
	if d.HasError() {
		t.Fatalf("building network_ids list: %v", d)
	}
	model := &firewallZoneResourceModel{
		Name:       types.StringValue("DMZ"),
		NetworkIDs: nids,
	}

	zone, diags := r.modelToFirewallZone(ctx, model)
	if diags.HasError() {
		t.Fatalf("modelToFirewallZone: %v", diags)
	}
	if zone.Name != "DMZ" {
		t.Errorf("Name = %q, want DMZ", zone.Name)
	}
	if len(zone.NetworkIDs) != 2 || zone.NetworkIDs[0] != "net-a" || zone.NetworkIDs[1] != "net-b" {
		t.Errorf("NetworkIDs = %v, want [net-a net-b]", zone.NetworkIDs)
	}

	apiZone := &unifi.FirewallZone{
		ID:          "z1",
		Name:        "DMZ",
		NetworkIDs:  []string{"net-a", "net-b"},
		ZoneKey:     "dmz",
		DefaultZone: false,
	}
	var out firewallZoneResourceModel
	if diags := r.firewallZoneToModel(ctx, apiZone, &out, "default"); diags.HasError() {
		t.Fatalf("firewallZoneToModel: %v", diags)
	}
	if out.ID.ValueString() != "z1" {
		t.Errorf("ID = %q, want z1", out.ID.ValueString())
	}
	if out.Name.ValueString() != "DMZ" {
		t.Errorf("Name = %q, want DMZ", out.Name.ValueString())
	}
	if out.Site.ValueString() != "default" {
		t.Errorf("Site = %q, want default", out.Site.ValueString())
	}
	if out.ZoneKey.ValueString() != "dmz" {
		t.Errorf("ZoneKey = %q, want dmz", out.ZoneKey.ValueString())
	}
	var gotNids []string
	if diags := out.NetworkIDs.ElementsAs(ctx, &gotNids, false); diags.HasError() {
		t.Fatalf("reading network_ids: %v", diags)
	}
	if len(gotNids) != 2 {
		t.Errorf("network_ids round-trip = %v, want 2 entries", gotNids)
	}
}

func TestNewFirewallZoneResource(t *testing.T) {
	got := NewFirewallZoneResource()
	if got == nil {
		t.Fatal("NewFirewallZoneResource() returned nil")
	}
	if _, ok := got.(fwresource.ResourceWithImportState); !ok {
		t.Errorf("NewFirewallZoneResource() does not implement resource.ResourceWithImportState")
	}
	if _, ok := got.(fwresource.ResourceWithIdentity); !ok {
		t.Errorf("NewFirewallZoneResource() does not implement resource.ResourceWithIdentity")
	}
}

func TestNewFirewallZoneListResource(t *testing.T) {
	got := NewFirewallZoneListResource()
	if got == nil {
		t.Fatal("NewFirewallZoneListResource() returned nil")
	}
	if _, ok := got.(fwlist.ListResourceWithConfigure); !ok {
		t.Errorf("NewFirewallZoneListResource() does not implement list.ListResourceWithConfigure")
	}
}

func Test_firewallZoneResource_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.MetadataRequest
		resp *fwresource.MetadataResponse
	}
	tests := []struct {
		name string
		r    *firewallZoneResource
		args args
	}{
		{
			name: "type_name",
			r:    &firewallZoneResource{},
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
			if tt.args.resp.TypeName != "unifi_firewall_zone" {
				t.Errorf("TypeName = %q, want unifi_firewall_zone", tt.args.resp.TypeName)
			}
		})
	}
}

func Test_firewallZoneResource_IdentitySchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwresource.IdentitySchemaRequest
		resp *fwresource.IdentitySchemaResponse
	}
	tests := []struct {
		name string
		r    *firewallZoneResource
		args args
	}{
		{
			name: "has_id_attribute",
			r:    &firewallZoneResource{},
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

func Test_firewallZoneResource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.SchemaRequest
		resp *fwresource.SchemaResponse
	}
	tests := []struct {
		name string
		r    *firewallZoneResource
		args args
	}{
		{
			name: "key_attributes",
			r:    &firewallZoneResource{},
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
			attrs := tt.args.resp.Schema.Attributes

			if a, ok := attrs["id"]; !ok {
				t.Error("missing 'id' attribute")
			} else if !a.IsComputed() {
				t.Error("'id' should be Computed")
			}

			if a, ok := attrs["name"]; !ok {
				t.Error("missing 'name' attribute")
			} else if !a.IsRequired() {
				t.Error("'name' should be Required")
			}

			if a, ok := attrs["network_ids"]; !ok {
				t.Error("missing 'network_ids' attribute")
			} else if !a.IsOptional() || !a.IsComputed() {
				t.Error("'network_ids' should be Optional+Computed")
			}

			if a, ok := attrs["zone_key"]; !ok {
				t.Error("missing 'zone_key' attribute")
			} else if !a.IsComputed() {
				t.Error("'zone_key' should be Computed")
			}

			if a, ok := attrs["default_zone"]; !ok {
				t.Error("missing 'default_zone' attribute")
			} else if !a.IsComputed() {
				t.Error("'default_zone' should be Computed")
			}
		})
	}
}

func Test_firewallZoneResource_Configure(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ConfigureRequest
		resp *fwresource.ConfigureResponse
	}
	tests := []struct {
		name      string
		r         *firewallZoneResource
		args      args
		wantError bool
	}{
		{
			name: "nil_provider_data",
			r:    &firewallZoneResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.ConfigureRequest{},
				resp: &fwresource.ConfigureResponse{},
			},
			wantError: false,
		},
		{
			name: "wrong_type",
			r:    &firewallZoneResource{},
			args: args{
				ctx: context.Background(),
				req: fwresource.ConfigureRequest{
					ProviderData: "not-a-client",
				},
				resp: &fwresource.ConfigureResponse{},
			},
			wantError: true,
		},
		{
			name: "correct_client",
			r:    &firewallZoneResource{},
			args: args{
				ctx: context.Background(),
				req: fwresource.ConfigureRequest{
					ProviderData: &Client{},
				},
				resp: &fwresource.ConfigureResponse{},
			},
			wantError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Configure(tt.args.ctx, tt.args.req, tt.args.resp)
			if tt.wantError && !tt.args.resp.Diagnostics.HasError() {
				t.Error("expected error but got none")
			}
			if !tt.wantError && tt.args.resp.Diagnostics.HasError() {
				t.Errorf("unexpected error: %v", tt.args.resp.Diagnostics)
			}
		})
	}
}

func Test_firewallZoneResource_modelToFirewallZone(t *testing.T) {
	ctx := context.Background()
	r := &firewallZoneResource{}

	nids, d := types.ListValueFrom(ctx, types.StringType, []string{"net-a", "net-b"})
	if d.HasError() {
		t.Fatalf("building network_ids: %v", d)
	}

	tests := []struct {
		name       string
		model      *firewallZoneResourceModel
		wantName   string
		wantNetIDs []string
	}{
		{
			name: "basic",
			model: &firewallZoneResourceModel{
				Name:       types.StringValue("DMZ"),
				NetworkIDs: nids,
			},
			wantName:   "DMZ",
			wantNetIDs: []string{"net-a", "net-b"},
		},
		{
			name: "null_network_ids",
			model: &firewallZoneResourceModel{
				Name:       types.StringValue("Test"),
				NetworkIDs: types.ListNull(types.StringType),
			},
			wantName:   "Test",
			wantNetIDs: []string{},
		},
		{
			name: "no_networks",
			model: &firewallZoneResourceModel{
				Name:       types.StringValue("Empty"),
				NetworkIDs: types.ListNull(types.StringType),
			},
			wantName:   "Empty",
			wantNetIDs: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, diags := r.modelToFirewallZone(ctx, tt.model)
			if diags.HasError() {
				t.Fatalf("unexpected error: %v", diags)
			}
			if got.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", got.Name, tt.wantName)
			}
			if !reflect.DeepEqual(got.NetworkIDs, tt.wantNetIDs) {
				t.Errorf("NetworkIDs = %v, want %v", got.NetworkIDs, tt.wantNetIDs)
			}
		})
	}
}

func Test_firewallZoneResource_firewallZoneToModel(t *testing.T) {
	ctx := context.Background()
	r := &firewallZoneResource{}

	tests := []struct {
		name            string
		zone            *unifi.FirewallZone
		site            string
		wantID          string
		wantName        string
		wantZoneKey     string
		wantDefaultZone bool
		wantNetCount    int
	}{
		{
			name: "basic",
			zone: &unifi.FirewallZone{
				ID:          "z1",
				Name:        "LAN",
				ZoneKey:     "lan",
				DefaultZone: true,
				NetworkIDs:  []string{"n1"},
			},
			site:            "default",
			wantID:          "z1",
			wantName:        "LAN",
			wantZoneKey:     "lan",
			wantDefaultZone: true,
			wantNetCount:    1,
		},
		{
			name: "empty_network_ids",
			zone: &unifi.FirewallZone{
				ID:          "z2",
				Name:        "DMZ",
				ZoneKey:     "dmz",
				DefaultZone: false,
				NetworkIDs:  []string{},
			},
			site:            "site1",
			wantID:          "z2",
			wantName:        "DMZ",
			wantZoneKey:     "dmz",
			wantDefaultZone: false,
			wantNetCount:    0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var model firewallZoneResourceModel
			diags := r.firewallZoneToModel(ctx, tt.zone, &model, tt.site)
			if diags.HasError() {
				t.Fatalf("unexpected error: %v", diags)
			}
			if model.ID.ValueString() != tt.wantID {
				t.Errorf("ID = %q, want %q", model.ID.ValueString(), tt.wantID)
			}
			if model.Name.ValueString() != tt.wantName {
				t.Errorf("Name = %q, want %q", model.Name.ValueString(), tt.wantName)
			}
			if model.ZoneKey.ValueString() != tt.wantZoneKey {
				t.Errorf("ZoneKey = %q, want %q", model.ZoneKey.ValueString(), tt.wantZoneKey)
			}
			if model.DefaultZone.ValueBool() != tt.wantDefaultZone {
				t.Errorf(
					"DefaultZone = %v, want %v",
					model.DefaultZone.ValueBool(),
					tt.wantDefaultZone,
				)
			}
			var netIDs []string
			model.NetworkIDs.ElementsAs(ctx, &netIDs, false)
			if len(netIDs) != tt.wantNetCount {
				t.Errorf("NetworkIDs count = %d, want %d", len(netIDs), tt.wantNetCount)
			}
		})
	}
}

func Test_firewallZoneResource_ListResourceConfigSchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwlist.ListResourceSchemaRequest
		resp *fwlist.ListResourceSchemaResponse
	}
	tests := []struct {
		name string
		r    *firewallZoneResource
		args args
	}{
		{
			name: "has_site_attribute",
			r:    &firewallZoneResource{},
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
