package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func TestAccTrafficRoute_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTrafficRouteConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"description",
						"tfacc-basic-route",
					),
					resource.TestCheckResourceAttr("unifi_traffic_route.test", "enabled", "true"),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip.0.address",
						"192.168.1.2",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"kill_switch_enabled",
						"false",
					),
				),
			},
			{
				ResourceName:    "unifi_traffic_route.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func testAccTrafficRouteConfig_basic() string {
	return `
data "unifi_network" "default" {
	name = "Default"
}

resource "unifi_traffic_route" "test" {
	description         = "tfacc-basic-route"
	enabled             = true
	next_hop				    = "192.168.1.1"
	network_id			    = data.unifi_network.default.id
	destination = {
		ip = [{ address = "192.168.1.2" }]
	}
	kill_switch_enabled = false
}
`
}

func TestAccTrafficRoute_ipAddresses(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTrafficRouteConfig_ipAddresses(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"description",
						"tfacc-ip-route",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip.#",
						"2",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip.0.address",
						"10.0.0.0/8",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip.0.ports.#",
						"2",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip.0.ports.0",
						"80",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip.0.ports.1",
						"443",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip.1.address",
						"192.168.1.0/24",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip.1.ports.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip.1.ports.0",
						"8080-8090",
					),
				),
			},
			{
				ResourceName:    "unifi_traffic_route.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func testAccTrafficRouteConfig_ipAddresses() string {
	return `
resource "unifi_traffic_route" "test" {
	description     = "tfacc-ip-route"
	enabled         = true

	destination = {
		ip = [
			{
				address = "10.0.0.0/8"
				ports   = ["80", "443"]
			},
			{
				address = "192.168.1.0/24"
				ports   = ["8080-8090"]
			},
		]
	}
}
`
}

func TestAccTrafficRoute_ipRanges(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTrafficRouteConfig_ipRanges(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"description",
						"tfacc-iprange-route",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip.0.address",
						"10.0.0.1-10.0.0.100",
					),
				),
			},
			{
				ResourceName:    "unifi_traffic_route.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func testAccTrafficRouteConfig_ipRanges() string {
	return `
resource "unifi_traffic_route" "test" {
	description     = "tfacc-iprange-route"
	enabled         = true

	destination = {
		ip = [{ address = "10.0.0.1-10.0.0.100" }]
	}
}
`
}

func TestAccTrafficRoute_sourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTrafficRouteConfig_sourceDefault(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"description",
						"tfacc-source-default-route",
					),
					resource.TestCheckNoResourceAttr(
						"unifi_traffic_route.test",
						"source.networks.#",
					),
					resource.TestCheckNoResourceAttr(
						"unifi_traffic_route.test",
						"source.clients.#",
					),
				),
			},
		},
	})
}

func testAccTrafficRouteConfig_sourceDefault() string {
	return `
resource "unifi_traffic_route" "test" {
	description     = "tfacc-source-default-route"
	enabled         = true
	destination = {
		domain = ["test.example.com"]
	}
}
`
}

func TestAccTrafficRoute_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Initial creation
			{
				Config: testAccTrafficRouteConfig_updateStep1(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"description",
						"tfacc-update-route",
					),
					resource.TestCheckResourceAttr("unifi_traffic_route.test", "enabled", "true"),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.domain.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.domain.0",
						"before.example.com",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"kill_switch_enabled",
						"false",
					),
				),
			},
			// Step 2: Update description, domains, and enable kill switch
			{
				Config: testAccTrafficRouteConfig_updateStep2(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"description",
						"tfacc-update-route-modified",
					),
					resource.TestCheckResourceAttr("unifi_traffic_route.test", "enabled", "true"),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.domain.#",
						"2",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.domain.0",
						"after1.example.com",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.domain.1",
						"after2.example.com",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"kill_switch_enabled",
						"true",
					),
				),
			},
			// Step 3: Disable the route
			{
				Config: testAccTrafficRouteConfig_updateStep3(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_traffic_route.test", "enabled", "false"),
				),
			},
		},
	})
}

func testAccTrafficRouteConfig_updateStep1() string {
	return `
resource "unifi_traffic_route" "test" {
	description     = "tfacc-update-route"
	enabled         = true
	destination = {
		domain = ["before.example.com"]
	}
}
`
}

func testAccTrafficRouteConfig_updateStep2() string {
	return `
resource "unifi_traffic_route" "test" {
	description        = "tfacc-update-route-modified"
	enabled            = true
	destination = {
		domain = ["after1.example.com", "after2.example.com"]
	}
	kill_switch_enabled = true
}
`
}

func testAccTrafficRouteConfig_updateStep3() string {
	return `
resource "unifi_traffic_route" "test" {
	description     = "tfacc-update-route-modified"
	enabled         = false
	destination = {
		domain = ["after1.example.com", "after2.example.com"]
	}
}
`
}

func TestAccTrafficRoute_regions(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTrafficRouteConfig_regions(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"description",
						"tfacc-region-route",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.region.#",
						"2",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.region.0",
						"US",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.region.1",
						"CA",
					),
				),
			},
			{
				ResourceName:    "unifi_traffic_route.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func testAccTrafficRouteConfig_regions() string {
	return `
resource "unifi_traffic_route" "test" {
	description     = "tfacc-region-route"
	enabled         = true
	destination = {
		region = ["US", "CA"]
	}
}
`
}

func TestAccTrafficRoute_fullConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTrafficRouteConfig_full(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"description",
						"tfacc-full-route",
					),
					resource.TestCheckResourceAttr("unifi_traffic_route.test", "enabled", "true"),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"kill_switch_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip.#",
						"2",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip.0.address",
						"172.16.0.0/12",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip.1.address",
						"192.168.0.1-192.168.0.50",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"source.clients.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"source.clients.0.mac",
						"aa:bb:cc:dd:ee:ff",
					),
				),
			},
			{
				ResourceName:    "unifi_traffic_route.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func testAccTrafficRouteConfig_full() string {
	return `
resource "unifi_traffic_route" "test" {
	description         = "tfacc-full-route"
	enabled             = true
	kill_switch_enabled = true

	destination = {
		ip = [
			{ address = "172.16.0.0/12" },
			{ address = "192.168.0.1-192.168.0.50" },
		]
	}

	source = { clients = [{ mac = "aa:bb:cc:dd:ee:ff" }] }
}
`
}

func TestNewTrafficRouteResource(t *testing.T) {
	r := NewTrafficRouteResource()
	if r == nil {
		t.Fatal("NewTrafficRouteResource() returned nil")
	}
	if _, ok := r.(fwresource.ResourceWithConfigure); !ok {
		t.Error("expected ResourceWithConfigure interface")
	}
	if _, ok := r.(fwresource.ResourceWithImportState); !ok {
		t.Error("expected ResourceWithImportState interface")
	}
}

func TestNewTrafficRouteListResource(t *testing.T) {
	r := NewTrafficRouteListResource()
	if r == nil {
		t.Fatal("NewTrafficRouteListResource() returned nil")
	}
	if _, ok := r.(fwlist.ListResource); !ok {
		t.Error("expected fwlist.ListResource interface")
	}
}

func Test_destinationIPModel_AttributeTypes(t *testing.T) {
	m := destinationIPModel{}
	got := m.AttributeTypes()
	for _, key := range []string{"address", "ports"} {
		if _, ok := got[key]; !ok {
			t.Errorf("AttributeTypes() missing key %q", key)
		}
	}
	if got["address"] != types.StringType {
		t.Errorf("address type = %v, want StringType", got["address"])
	}
}

func Test_sourceNetworkModel_AttributeTypes(t *testing.T) {
	m := sourceNetworkModel{}
	got := m.AttributeTypes()
	if _, ok := got["id"]; !ok {
		t.Error("AttributeTypes() missing key 'id'")
	}
	if got["id"] != types.StringType {
		t.Errorf("id type = %v, want StringType", got["id"])
	}
}

func Test_sourceClientModel_AttributeTypes(t *testing.T) {
	m := sourceClientModel{}
	got := m.AttributeTypes()
	if _, ok := got["mac"]; !ok {
		t.Error("AttributeTypes() missing key 'mac'")
	}
	if got["mac"] != types.StringType {
		t.Errorf("mac type = %v, want StringType", got["mac"])
	}
}

func Test_sourceModel_AttributeTypes(t *testing.T) {
	m := sourceModel{}
	got := m.AttributeTypes()
	for _, key := range []string{"networks", "clients"} {
		if _, ok := got[key]; !ok {
			t.Errorf("AttributeTypes() missing key %q", key)
		}
	}
}

func Test_destinationModel_AttributeTypes(t *testing.T) {
	m := destinationModel{}
	got := m.AttributeTypes()
	for _, key := range []string{"domain", "ip", "region"} {
		if _, ok := got[key]; !ok {
			t.Errorf("AttributeTypes() missing key %q", key)
		}
	}
}

func Test_trafficRouteResource_Metadata(t *testing.T) {
	tests := []struct {
		providerTypeName, wantTypeName string
	}{
		{"unifi", "unifi_traffic_route"},
		{"test", "test_traffic_route"},
	}
	for _, tt := range tests {
		t.Run(tt.providerTypeName, func(t *testing.T) {
			r := &trafficRouteResource{}
			resp := &fwresource.MetadataResponse{}
			r.Metadata(context.Background(), fwresource.MetadataRequest{ProviderTypeName: tt.providerTypeName}, resp)
			if resp.TypeName != tt.wantTypeName {
				t.Errorf("TypeName = %q, want %q", resp.TypeName, tt.wantTypeName)
			}
		})
	}
}

func Test_trafficRouteResource_IdentitySchema(t *testing.T) {
	r := &trafficRouteResource{}
	resp := &fwresource.IdentitySchemaResponse{}
	r.IdentitySchema(context.Background(), fwresource.IdentitySchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("IdentitySchema() produced errors: %v", resp.Diagnostics)
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Error("IdentitySchema missing 'id' attribute")
	}
}

func Test_trafficRouteResource_Schema(t *testing.T) {
	r := &trafficRouteResource{}
	resp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() produced errors: %v", resp.Diagnostics)
	}
	for _, attr := range []string{"id", "site", "description", "destination", "enabled", "kill_switch_enabled", "network_id", "next_hop", "source"} {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("missing attribute %q", attr)
		}
	}
}

func Test_trafficRouteResource_Configure(t *testing.T) {
	tests := []struct {
		name      string
		data      any
		wantError bool
	}{
		{"nil", nil, false},
		{"wrong type", "wrong", true},
		{"correct client", &Client{Site: "default"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &trafficRouteResource{}
			resp := &fwresource.ConfigureResponse{}
			r.Configure(context.Background(), fwresource.ConfigureRequest{ProviderData: tt.data}, resp)
			if tt.wantError && !resp.Diagnostics.HasError() {
				t.Error("expected error")
			}
			if !tt.wantError && resp.Diagnostics.HasError() {
				t.Errorf("unexpected error: %v", resp.Diagnostics)
			}
		})
	}
}

func Test_trafficRouteResource_ImportState(t *testing.T) {
	t.Skip("ImportState delegates to ImportStatePassthroughWithIdentity which requires full state schema setup")
}

func Test_trafficRouteResource_modelToAPI(t *testing.T) {
	ctx := context.Background()

	t.Run("nil client causes error on network lookup", func(t *testing.T) {
		// modelToAPI with an empty NetworkID will try to call defaultWANNetworkID,
		// which requires a live client. Test that the non-network-lookup path works
		// by pre-populating NetworkID.
		r := &trafficRouteResource{}
		model := &trafficRouteResourceModel{
			Description:       types.StringValue("test-route"),
			Enabled:           types.BoolValue(true),
			KillSwitchEnabled: types.BoolValue(false),
			NetworkID:         types.StringValue("some-network-id"),
			Destination:       types.ObjectNull(destinationModel{}.AttributeTypes()),
			Source:            types.ObjectNull(sourceModel{}.AttributeTypes()),
		}
		got, diags := r.modelToAPI(ctx, model, "default")
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		if got.Description != "test-route" {
			t.Errorf("Description = %q, want test-route", got.Description)
		}
		if !got.Enabled {
			t.Error("Enabled should be true")
		}
		if got.NetworkID != "some-network-id" {
			t.Errorf("NetworkID = %q, want some-network-id", got.NetworkID)
		}
	})

	t.Run("domain destination sets MatchingTarget", func(t *testing.T) {
		r := &trafficRouteResource{}
		domainList, d := types.ListValueFrom(ctx, types.StringType, []string{"example.com"})
		if d.HasError() {
			t.Fatalf("building domain list: %v", d)
		}
		dest := destinationModel{
			Domain: domainList,
			IP:     types.ListNull(types.ObjectType{AttrTypes: destinationIPModel{}.AttributeTypes()}),
			Region: types.ListNull(types.StringType),
		}
		destObj, d := types.ObjectValueFrom(ctx, destinationModel{}.AttributeTypes(), dest)
		if d.HasError() {
			t.Fatalf("building destination object: %v", d)
		}
		model := &trafficRouteResourceModel{
			Enabled:           types.BoolValue(true),
			KillSwitchEnabled: types.BoolValue(false),
			NetworkID:         types.StringValue("net-1"),
			Destination:       destObj,
			Source:            types.ObjectNull(sourceModel{}.AttributeTypes()),
		}
		got, diags := r.modelToAPI(ctx, model, "default")
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if got.MatchingTarget != "DOMAIN" {
			t.Errorf("MatchingTarget = %q, want DOMAIN", got.MatchingTarget)
		}
		if len(got.Domains) != 1 || got.Domains[0].Domain != "example.com" {
			t.Errorf("Domains = %v, want [{example.com}]", got.Domains)
		}
	})
}

func Test_trafficRouteResource_apiToModel(t *testing.T) {
	ctx := context.Background()

	t.Run("basic fields populated", func(t *testing.T) {
		r := &trafficRouteResource{}
		route := &unifi.TrafficRoute{
			ID:                "route-123",
			Description:       "my-route",
			Enabled:           true,
			KillSwitchEnabled: false,
			NetworkID:         "net-abc",
			MatchingTarget:    "INTERNET",
			TargetDevices:     []unifi.TrafficRouteTargetDevices{{Type: "ALL_CLIENTS"}},
		}
		var model trafficRouteResourceModel
		diags := r.apiToModel(ctx, route, &model, "default")
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if model.ID.ValueString() != "route-123" {
			t.Errorf("ID = %q, want route-123", model.ID.ValueString())
		}
		if model.Description.ValueString() != "my-route" {
			t.Errorf("Description = %q, want my-route", model.Description.ValueString())
		}
		if !model.Enabled.ValueBool() {
			t.Error("Enabled should be true")
		}
		if model.Site.ValueString() != "default" {
			t.Errorf("Site = %q, want default", model.Site.ValueString())
		}
	})

	t.Run("domain route sets destination", func(t *testing.T) {
		r := &trafficRouteResource{}
		route := &unifi.TrafficRoute{
			ID:             "route-456",
			Enabled:        true,
			MatchingTarget: "DOMAIN",
			Domains: []unifi.TrafficRouteDomains{
				{Domain: "example.com"},
				{Domain: "test.com"},
			},
			TargetDevices: []unifi.TrafficRouteTargetDevices{{Type: "ALL_CLIENTS"}},
		}
		var model trafficRouteResourceModel
		diags := r.apiToModel(ctx, route, &model, "site1")
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if model.Destination.IsNull() {
			t.Fatal("Destination should not be null for a domain route")
		}
		var dest destinationModel
		if d := model.Destination.As(ctx, &dest, struct{ UnhandledNullAsEmpty, UnhandledUnknownAsEmpty bool }{}); d.HasError() {
			t.Fatalf("reading destination: %v", d)
		}
		var domains []string
		if d := dest.Domain.ElementsAs(ctx, &domains, false); d.HasError() {
			t.Fatalf("reading domains: %v", d)
		}
		if len(domains) != 2 {
			t.Errorf("domains len = %d, want 2", len(domains))
		}
	})
}

func Test_trafficRouteResource_ListResourceConfigSchema(t *testing.T) {
	r := &trafficRouteResource{}
	resp := &fwlist.ListResourceSchemaResponse{}
	r.ListResourceConfigSchema(context.Background(), fwlist.ListResourceSchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("ListResourceConfigSchema() produced errors: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["site"]; !ok {
		t.Error("ListResourceConfigSchema missing 'site' attribute")
	}
}

// Test_trafficRouteResource_modelToAPI_ipRange verifies IP range addresses are
// converted to TrafficRouteIPRanges (not IPAddresses) in the API struct.
func Test_trafficRouteResource_modelToAPI_ipRange(t *testing.T) {
	ctx := context.Background()
	r := &trafficRouteResource{}

	ipEntry := destinationIPModel{
		Address: types.StringValue("10.0.0.1-10.0.0.100"),
		Ports:   types.ListNull(types.StringType),
	}
	ipObj, d := types.ObjectValueFrom(ctx, destinationIPModel{}.AttributeTypes(), ipEntry)
	if d.HasError() {
		t.Fatalf("building ip object: %v", d)
	}
	ipList, d := types.ListValue(
		types.ObjectType{AttrTypes: destinationIPModel{}.AttributeTypes()},
		[]attr.Value{ipObj},
	)
	if d.HasError() {
		t.Fatalf("building ip list: %v", d)
	}
	dest := destinationModel{
		Domain: types.ListNull(types.StringType),
		IP:     ipList,
		Region: types.ListNull(types.StringType),
	}
	destObj, d := types.ObjectValueFrom(ctx, destinationModel{}.AttributeTypes(), dest)
	if d.HasError() {
		t.Fatalf("building destination object: %v", d)
	}

	model := &trafficRouteResourceModel{
		Enabled:           types.BoolValue(true),
		KillSwitchEnabled: types.BoolValue(false),
		NetworkID:         types.StringValue("net-1"),
		Destination:       destObj,
		Source:            types.ObjectNull(sourceModel{}.AttributeTypes()),
	}

	got, diags := r.modelToAPI(ctx, model, "default")
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if len(got.IPRanges) != 1 {
		t.Fatalf("IPRanges len = %d, want 1", len(got.IPRanges))
	}
	if got.IPRanges[0].Start != "10.0.0.1" || got.IPRanges[0].Stop != "10.0.0.100" {
		t.Errorf("IPRange = {%s-%s}, want {10.0.0.1-10.0.0.100}",
			got.IPRanges[0].Start, got.IPRanges[0].Stop)
	}
	if len(got.IPAddresses) != 0 {
		t.Errorf("IPAddresses should be empty for a range, got %v", got.IPAddresses)
	}
}
