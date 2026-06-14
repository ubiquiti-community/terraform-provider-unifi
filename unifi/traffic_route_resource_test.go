package unifi

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
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
	tests := []struct {
		name string
		want fwresource.Resource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewTrafficRouteResource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTrafficRouteResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewTrafficRouteListResource(t *testing.T) {
	tests := []struct {
		name string
		want fwlist.ListResource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewTrafficRouteListResource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTrafficRouteListResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_destinationIPModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    destinationIPModel
		want map[string]attr.Type
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("destinationIPModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sourceNetworkModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    sourceNetworkModel
		want map[string]attr.Type
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sourceNetworkModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sourceClientModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    sourceClientModel
		want map[string]attr.Type
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sourceClientModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sourceModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    sourceModel
		want map[string]attr.Type
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sourceModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_destinationModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    destinationModel
		want map[string]attr.Type
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("destinationModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_trafficRouteResource_Metadata(t *testing.T) {
	type args struct {
		in0  context.Context
		req  fwresource.MetadataRequest
		resp *fwresource.MetadataResponse
	}
	tests := []struct {
		name string
		r    *trafficRouteResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Metadata(tt.args.in0, tt.args.req, tt.args.resp)
		})
	}
}

func Test_trafficRouteResource_IdentitySchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwresource.IdentitySchemaRequest
		resp *fwresource.IdentitySchemaResponse
	}
	tests := []struct {
		name string
		r    *trafficRouteResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.IdentitySchema(tt.args.in0, tt.args.in1, tt.args.resp)
		})
	}
}

func Test_trafficRouteResource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		in1  fwresource.SchemaRequest
		resp *fwresource.SchemaResponse
	}
	tests := []struct {
		name string
		r    *trafficRouteResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Schema(tt.args.ctx, tt.args.in1, tt.args.resp)
		})
	}
}

func Test_trafficRouteResource_Configure(t *testing.T) {
	type args struct {
		in0  context.Context
		req  fwresource.ConfigureRequest
		resp *fwresource.ConfigureResponse
	}
	tests := []struct {
		name string
		r    *trafficRouteResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Configure(tt.args.in0, tt.args.req, tt.args.resp)
		})
	}
}

func Test_trafficRouteResource_Create(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.CreateRequest
		resp *fwresource.CreateResponse
	}
	tests := []struct {
		name string
		r    *trafficRouteResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Create(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_trafficRouteResource_Read(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ReadRequest
		resp *fwresource.ReadResponse
	}
	tests := []struct {
		name string
		r    *trafficRouteResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Read(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_trafficRouteResource_Update(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.UpdateRequest
		resp *fwresource.UpdateResponse
	}
	tests := []struct {
		name string
		r    *trafficRouteResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Update(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_trafficRouteResource_Delete(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.DeleteRequest
		resp *fwresource.DeleteResponse
	}
	tests := []struct {
		name string
		r    *trafficRouteResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Delete(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_trafficRouteResource_ImportState(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ImportStateRequest
		resp *fwresource.ImportStateResponse
	}
	tests := []struct {
		name string
		r    *trafficRouteResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.ImportState(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_trafficRouteResource_modelToAPI(t *testing.T) {
	type args struct {
		ctx   context.Context
		model *trafficRouteResourceModel
		site  string
	}
	tests := []struct {
		name  string
		r     *trafficRouteResource
		args  args
		want  *unifi.TrafficRoute
		want1 diag.Diagnostics
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.r.modelToAPI(tt.args.ctx, tt.args.model, tt.args.site)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("trafficRouteResource.modelToAPI() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("trafficRouteResource.modelToAPI() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_trafficRouteResource_apiToModel(t *testing.T) {
	type args struct {
		ctx   context.Context
		route *unifi.TrafficRoute
		model *trafficRouteResourceModel
		site  string
	}
	tests := []struct {
		name string
		r    *trafficRouteResource
		args args
		want diag.Diagnostics
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.apiToModel(tt.args.ctx, tt.args.route, tt.args.model, tt.args.site); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("trafficRouteResource.apiToModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_trafficRouteResource_defaultWANNetworkID(t *testing.T) {
	type args struct {
		ctx  context.Context
		site string
	}
	tests := []struct {
		name    string
		r       *trafficRouteResource
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.defaultWANNetworkID(tt.args.ctx, tt.args.site)
			if (err != nil) != tt.wantErr {
				t.Errorf("trafficRouteResource.defaultWANNetworkID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("trafficRouteResource.defaultWANNetworkID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_trafficRouteResource_ListResourceConfigSchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwlist.ListResourceSchemaRequest
		resp *fwlist.ListResourceSchemaResponse
	}
	tests := []struct {
		name string
		r    *trafficRouteResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.ListResourceConfigSchema(tt.args.in0, tt.args.in1, tt.args.resp)
		})
	}
}

func Test_trafficRouteResource_List(t *testing.T) {
	type args struct {
		ctx    context.Context
		req    fwlist.ListRequest
		stream *fwlist.ListResultsStream
	}
	tests := []struct {
		name string
		r    *trafficRouteResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.List(tt.args.ctx, tt.args.req, tt.args.stream)
		})
	}
}
