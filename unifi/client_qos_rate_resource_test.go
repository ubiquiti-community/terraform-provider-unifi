package unifi

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func TestAccClientQosRate_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientQosRateConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_client_qos_rate.test",
						"name",
						"tfacc-group",
					),
					resource.TestCheckResourceAttr(
						"unifi_client_qos_rate.test",
						"qos_rate_max_down",
						"-1",
					),
					resource.TestCheckResourceAttr(
						"unifi_client_qos_rate.test",
						"qos_rate_max_up",
						"-1",
					),
				),
			},
			{
				ResourceName:      "unifi_client_qos_rate.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccClientQosRateConfig_basic() string {
	return `
resource "unifi_client_qos_rate" "test" {
	name = "tfacc-group"
}
`
}

func TestAccClientQosRate_qos(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientQosRateConfig_qos(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_client_qos_rate.test",
						"name",
						"tfacc-qos-group",
					),
					resource.TestCheckResourceAttr(
						"unifi_client_qos_rate.test",
						"qos_rate_max_down",
						"1000",
					),
					resource.TestCheckResourceAttr(
						"unifi_client_qos_rate.test",
						"qos_rate_max_up",
						"500",
					),
				),
			},
		},
	})
}

func testAccClientQosRateConfig_qos() string {
	return `
resource "unifi_client_qos_rate" "test" {
	name               = "tfacc-qos-group"
	qos_rate_max_down  = 1000
	qos_rate_max_up    = 500
}
`
}

func TestAccClientQosRate_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientQosRateConfig_update_before(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_client_qos_rate.test",
						"name",
						"tfacc-update-group",
					),
					resource.TestCheckResourceAttr(
						"unifi_client_qos_rate.test",
						"qos_rate_max_down",
						"100",
					),
				),
			},
			{
				Config: testAccClientQosRateConfig_update_after(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_client_qos_rate.test",
						"name",
						"tfacc-update-group-renamed",
					),
					resource.TestCheckResourceAttr(
						"unifi_client_qos_rate.test",
						"qos_rate_max_down",
						"200",
					),
				),
			},
		},
	})
}

func testAccClientQosRateConfig_update_before() string {
	return `
resource "unifi_client_qos_rate" "test" {
	name               = "tfacc-update-group"
	qos_rate_max_down  = 100
}
`
}

func testAccClientQosRateConfig_update_after() string {
	return `
resource "unifi_client_qos_rate" "test" {
	name               = "tfacc-update-group-renamed"
	qos_rate_max_down  = 200
}
`
}

func TestNewClientQosRateResource(t *testing.T) {
	tests := []struct {
		name string
		want fwresource.Resource
	}{
		{
			name: "returns clientQosRateResource",
			want: &clientQosRateResource{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewClientQosRateResource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewClientQosRateResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewClientQosRateListResource(t *testing.T) {
	tests := []struct {
		name string
		want fwlist.ListResource
	}{
		{
			name: "returns clientQosRateResource",
			want: &clientQosRateResource{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewClientQosRateListResource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewClientQosRateListResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_clientQosRateResource_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.MetadataRequest
		resp *fwresource.MetadataResponse
	}
	tests := []struct {
		name string
		r    *clientQosRateResource
		args args
	}{
		{
			name: "sets type name",
			r:    &clientQosRateResource{},
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
		})
	}
}

func Test_clientQosRateResource_IdentitySchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwresource.IdentitySchemaRequest
		resp *fwresource.IdentitySchemaResponse
	}
	tests := []struct {
		name string
		r    *clientQosRateResource
		args args
	}{
		{
			name: "returns identity schema",
			r:    &clientQosRateResource{},
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
		})
	}
}

func Test_clientQosRateResource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.SchemaRequest
		resp *fwresource.SchemaResponse
	}
	tests := []struct {
		name string
		r    *clientQosRateResource
		args args
	}{
		{
			name: "returns schema",
			r:    &clientQosRateResource{},
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
		})
	}
}

func Test_clientQosRateResource_Configure(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ConfigureRequest
		resp *fwresource.ConfigureResponse
	}
	tests := []struct {
		name string
		r    *clientQosRateResource
		args args
	}{
		{
			name: "nil provider data",
			r:    &clientQosRateResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.ConfigureRequest{},
				resp: &fwresource.ConfigureResponse{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Configure(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_clientQosRateResource_Create(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.CreateRequest
		resp *fwresource.CreateResponse
	}
	tests := []struct {
		name string
		r    *clientQosRateResource
		args args
	}{
		
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Create(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_clientQosRateResource_Read(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ReadRequest
		resp *fwresource.ReadResponse
	}
	tests := []struct {
		name string
		r    *clientQosRateResource
		args args
	}{
		
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Read(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_clientQosRateResource_Update(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.UpdateRequest
		resp *fwresource.UpdateResponse
	}
	tests := []struct {
		name string
		r    *clientQosRateResource
		args args
	}{
		
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Update(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_clientQosRateResource_applyPlanToState(t *testing.T) {
	type args struct {
		in0   context.Context
		plan  *clientQosRateResourceModel
		state *clientQosRateResourceModel
	}
	tests := []struct {
		name string
		r    *clientQosRateResource
		args args
	}{
		
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.applyPlanToState(tt.args.in0, tt.args.plan, tt.args.state)
		})
	}
}

func Test_clientQosRateResource_Delete(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.DeleteRequest
		resp *fwresource.DeleteResponse
	}
	tests := []struct {
		name string
		r    *clientQosRateResource
		args args
	}{
		
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Delete(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_clientQosRateResource_ImportState(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ImportStateRequest
		resp *fwresource.ImportStateResponse
	}
	tests := []struct {
		name string
		r    *clientQosRateResource
		args args
	}{
		
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.ImportState(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_clientQosRateResource_planToClientQosRate(t *testing.T) {
	type args struct {
		in0  context.Context
		plan clientQosRateResourceModel
	}
	tests := []struct {
		name  string
		r     *clientQosRateResource
		args  args
		want  *unifi.ClientGroup
		want1 diag.Diagnostics
	}{
		{
			name: "converts model with all fields",
			r:    &clientQosRateResource{},
			args: args{
				in0: context.Background(),
				plan: clientQosRateResourceModel{
					ID:             types.StringValue("group-id"),
					Name:           types.StringValue("test-group"),
					QOSRateMaxDown: types.Int64Value(1000),
					QOSRateMaxUp:   types.Int64Value(500),
				},
			},
			want: &unifi.ClientGroup{
				ID:             "group-id",
				Name:           "test-group",
				QOSRateMaxDown: ptrInt64(1000),
				QOSRateMaxUp:   ptrInt64(500),
			},
			want1: nil,
		},
		{
			name: "converts model with null optional fields",
			r:    &clientQosRateResource{},
			args: args{
				in0: context.Background(),
				plan: clientQosRateResourceModel{
					ID:             types.StringNull(),
					Name:           types.StringValue("minimal-group"),
					QOSRateMaxDown: types.Int64Null(),
					QOSRateMaxUp:   types.Int64Null(),
				},
			},
			want: &unifi.ClientGroup{
				Name: "minimal-group",
			},
			want1: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.r.planToClientQosRate(tt.args.in0, tt.args.plan)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("clientQosRateResource.planToClientQosRate() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("clientQosRateResource.planToClientQosRate() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_clientQosRateResource_clientQosRateToModel(t *testing.T) {
	type args struct {
		in0         context.Context
		clientGroup *unifi.ClientGroup
		model       *clientQosRateResourceModel
		site        string
	}
	tests := []struct {
		name string
		r    *clientQosRateResource
		args args
		want diag.Diagnostics
	}{
		{
			name: "converts client group to model",
			r:    &clientQosRateResource{},
			args: args{
				in0: context.Background(),
				clientGroup: &unifi.ClientGroup{
					ID:             "cg-123",
					Name:           "test-group",
					QOSRateMaxDown: ptrInt64(2000),
					QOSRateMaxUp:   ptrInt64(1000),
				},
				model: &clientQosRateResourceModel{
					ID:   types.StringValue("cg-123"),
					Name: types.StringValue("test-group"),
				},
				site: "default",
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.clientQosRateToModel(tt.args.in0, tt.args.clientGroup, tt.args.model, tt.args.site); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("clientQosRateResource.clientQosRateToModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_clientQosRateResource_ListResourceConfigSchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwlist.ListResourceSchemaRequest
		resp *fwlist.ListResourceSchemaResponse
	}
	tests := []struct {
		name string
		r    *clientQosRateResource
		args args
	}{
		{
			name: "returns list schema",
			r:    &clientQosRateResource{},
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
		})
	}
}

func Test_clientQosRateResource_List(t *testing.T) {
	type args struct {
		ctx    context.Context
		req    fwlist.ListRequest
		stream *fwlist.ListResultsStream
	}
	tests := []struct {
		name string
		r    *clientQosRateResource
		args args
	}{
		
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.List(tt.args.ctx, tt.args.req, tt.args.stream)
		})
	}
}
