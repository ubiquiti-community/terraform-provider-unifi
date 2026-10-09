package unifi

import (
	"context"
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
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
						"qos_rate.max_down",
						"-1",
					),
					resource.TestCheckResourceAttr(
						"unifi_client_qos_rate.test",
						"qos_rate.max_up",
						"-1",
					),
				),
			},
			// Classic string import by bare controller ID.
			{
				ResourceName:      "unifi_client_qos_rate.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Classic string import using the documented "site:id" form.
			{
				ResourceName: "unifi_client_qos_rate.test",
				ImportState:  true,
				ImportStateIdFunc: testAccClientQosRateSitePrefixedID(
					"unifi_client_qos_rate.test",
				),
				ImportStateVerify: true,
			},
			// Identity-based import (import block with identity, Terraform 1.12+).
			{
				ResourceName:    "unifi_client_qos_rate.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

// testAccClientQosRateSitePrefixedID returns an ImportStateIdFunc yielding the
// "site:id" composite string import format.
func testAccClientQosRateSitePrefixedID(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found in state: %s", resourceName)
		}
		site := rs.Primary.Attributes["site"]
		if site == "" {
			return "", fmt.Errorf("resource %s has no site attribute", resourceName)
		}
		return site + ":" + rs.Primary.ID, nil
	}
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
						"qos_rate.max_down",
						"1000",
					),
					resource.TestCheckResourceAttr(
						"unifi_client_qos_rate.test",
						"qos_rate.max_up",
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
	name = "tfacc-qos-group"
	qos_rate = {
		max_down = 1000
		max_up   = 500
	}
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
						"qos_rate.max_down",
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
						"qos_rate.max_down",
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
	name = "tfacc-update-group"
	qos_rate = {
		max_down = 100
	}
}
`
}

func testAccClientQosRateConfig_update_after() string {
	return `
resource "unifi_client_qos_rate" "test" {
	name = "tfacc-update-group-renamed"
	qos_rate = {
		max_down = 200
	}
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
	}{}
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
	}{}
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
	}{}
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
	}{}
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
	}{}
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
	}{}
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
					ID:   types.StringValue("group-id"),
					Name: types.StringValue("test-group"),
					QOSRate: types.ObjectValueMust(
						clientQosRateRateAttrTypes(),
						map[string]attr.Value{
							"max_down": types.Int64Value(1000),
							"max_up":   types.Int64Value(500),
						},
					),
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
					ID:      types.StringNull(),
					Name:    types.StringValue("minimal-group"),
					QOSRate: types.ObjectNull(clientQosRateRateAttrTypes()),
				},
			},
			want: &unifi.ClientGroup{
				Name: "minimal-group",
			},
			want1: nil,
		},
		{
			name: "converts model with null nested leaves",
			r:    &clientQosRateResource{},
			args: args{
				in0: context.Background(),
				plan: clientQosRateResourceModel{
					ID:   types.StringNull(),
					Name: types.StringValue("minimal-group"),
					QOSRate: types.ObjectValueMust(
						clientQosRateRateAttrTypes(),
						map[string]attr.Value{
							"max_down": types.Int64Null(),
							"max_up":   types.Int64Null(),
						},
					),
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
				t.Errorf(
					"clientQosRateResource.planToClientQosRate() got = %v, want %v",
					got,
					tt.want,
				)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf(
					"clientQosRateResource.planToClientQosRate() got1 = %v, want %v",
					got1,
					tt.want1,
				)
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
			if got := tt.r.clientQosRateToModel(
				tt.args.in0,
				tt.args.clientGroup,
				tt.args.model,
				tt.args.site,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("clientQosRateResource.clientQosRateToModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestClientQosRateUpgradeState_v0NestsQosRate guards the v0 -> v1 schema
// upgrade: the flat qos_rate_max_down/qos_rate_max_up attributes move into
// the nested `qos_rate` object and every other attribute passes through.
func TestClientQosRateUpgradeState_v0NestsQosRate(t *testing.T) {
	ctx := context.Background()
	r := &clientQosRateResource{}

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	if schemaResp.Schema.Version != 1 {
		t.Fatalf("client QOS rate schema Version = %d, want 1", schemaResp.Schema.Version)
	}
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	up, ok := r.UpgradeState(ctx)[0]
	if !ok {
		t.Fatal("no upgrader registered for schema version 0")
	}
	upgrade := func(prior string) map[string]tftypes.Value {
		t.Helper()
		resp := &fwresource.UpgradeStateResponse{}
		up.StateUpgrader(ctx, fwresource.UpgradeStateRequest{
			RawState: &tfprotov6.RawState{JSON: []byte(prior)},
		}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("upgrade failed: %v", resp.Diagnostics)
		}
		val, err := resp.DynamicValue.Unmarshal(schemaType)
		if err != nil {
			t.Fatalf("unmarshal upgraded value: %v", err)
		}
		var root map[string]tftypes.Value
		if err := val.As(&root); err != nil {
			t.Fatalf("as object: %v", err)
		}
		for _, flat := range []string{"qos_rate_max_down", "qos_rate_max_up"} {
			if _, exists := root[flat]; exists {
				t.Errorf("flat attribute %q survived the upgrade", flat)
			}
		}
		return root
	}
	num := func(v tftypes.Value, name string, want int64) {
		t.Helper()
		var f big.Float
		if err := v.As(&f); err != nil {
			t.Errorf("%s = %v (%v), want %d", name, v, err, want)
			return
		}
		if n, _ := f.Int64(); n != want {
			t.Errorf("%s = %d, want %d", name, n, want)
		}
	}

	root := upgrade(`{
		"id": "cg-1", "site": "default", "name": "wifi",
		"qos_rate_max_down": 2000, "qos_rate_max_up": -1
	}`)
	var name string
	if err := root["name"].As(&name); err != nil || name != "wifi" {
		t.Errorf("name = %v (%v), want wifi", root["name"], err)
	}
	var rate map[string]tftypes.Value
	if err := root["qos_rate"].As(&rate); err != nil {
		t.Fatalf("qos_rate: as object: %v (value %v)", err, root["qos_rate"])
	}
	num(rate["max_down"], "qos_rate.max_down", 2000)
	num(rate["max_up"], "qos_rate.max_up", -1)

	// State written before the rate attributes existed yields a null object,
	// not an object of nulls.
	root = upgrade(`{"id": "cg-2", "site": "default", "name": "bare"}`)
	if !root["qos_rate"].IsNull() {
		t.Errorf("qos_rate = %v, want null when no prior keys exist", root["qos_rate"])
	}
}

// TestClientQosRate_qosRateRoundTrip checks the nested qos_rate object
// converts API -> model -> API without loss, that the object default
// reproduces the old flat defaults, and that a null/unknown object stays off
// the wire.
func TestClientQosRate_qosRateRoundTrip(t *testing.T) {
	ctx := context.Background()
	r := &clientQosRateResource{}

	api := &unifi.ClientGroup{
		ID:             "cg-1",
		Name:           "wifi",
		QOSRateMaxDown: ptrInt64(2000),
		QOSRateMaxUp:   ptrInt64(10),
	}
	model := clientQosRateResourceModel{ID: types.StringValue("cg-1")}
	if d := r.clientQosRateToModel(ctx, api, &model, "default"); d.HasError() {
		t.Fatalf("clientQosRateToModel: %v", d)
	}
	rate := model.QOSRate.Attributes()
	if attrAs[types.Int64](t, rate["max_down"]).ValueInt64() != 2000 ||
		attrAs[types.Int64](t, rate["max_up"]).ValueInt64() != 10 {
		t.Errorf("qos_rate read back = %v", model.QOSRate)
	}

	back, d := r.planToClientQosRate(ctx, model)
	if d.HasError() {
		t.Fatalf("planToClientQosRate: %v", d)
	}
	if back.QOSRateMaxDown == nil || *back.QOSRateMaxDown != 2000 ||
		back.QOSRateMaxUp == nil || *back.QOSRateMaxUp != 10 {
		t.Errorf("qos_rate round trip: %v %v", back.QOSRateMaxDown, back.QOSRateMaxUp)
	}

	// The object default reproduces the -1/-1 the flat attributes defaulted to.
	model.QOSRate = clientQosRateRateDefault()
	back, d = r.planToClientQosRate(ctx, model)
	if d.HasError() {
		t.Fatalf("planToClientQosRate (default): %v", d)
	}
	if back.QOSRateMaxDown == nil || *back.QOSRateMaxDown != -1 ||
		back.QOSRateMaxUp == nil || *back.QOSRateMaxUp != -1 {
		t.Errorf("object default on the wire: %v %v", back.QOSRateMaxDown, back.QOSRateMaxUp)
	}

	// A null or unknown object (the plan omitted it) contributes nothing.
	for _, obj := range []types.Object{
		types.ObjectNull(clientQosRateRateAttrTypes()),
		types.ObjectUnknown(clientQosRateRateAttrTypes()),
	} {
		model.QOSRate = obj
		back, d = r.planToClientQosRate(ctx, model)
		if d.HasError() {
			t.Fatalf("planToClientQosRate (%v): %v", obj, d)
		}
		if back.QOSRateMaxDown != nil || back.QOSRateMaxUp != nil {
			t.Errorf("null/unknown qos_rate leaked into the API struct: %+v", back)
		}
	}

	// applyPlanToState: a known planned leaf replaces the state value while an
	// unknown planned leaf keeps it.
	state := clientQosRateResourceModel{
		QOSRate: types.ObjectValueMust(clientQosRateRateAttrTypes(), map[string]attr.Value{
			"max_down": types.Int64Value(2000),
			"max_up":   types.Int64Value(10),
		}),
	}
	plan := &clientQosRateResourceModel{
		QOSRate: types.ObjectValueMust(clientQosRateRateAttrTypes(), map[string]attr.Value{
			"max_down": types.Int64Value(500),
			"max_up":   types.Int64Unknown(),
		}),
	}
	r.applyPlanToState(ctx, plan, &state)
	rate = state.QOSRate.Attributes()
	if attrAs[types.Int64](t, rate["max_down"]).ValueInt64() != 500 ||
		attrAs[types.Int64](t, rate["max_up"]).ValueInt64() != 10 {
		t.Errorf("applyPlanToState qos_rate = %v, want max_down=500 max_up=10", state.QOSRate)
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
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.List(tt.args.ctx, tt.args.req, tt.args.stream)
		})
	}
}

func TestAccClientQosRateList_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccClientQosRateConfig_basic(),
			},
			{
				Query: true,
				Config: `
					provider "unifi" {}
					list "unifi_client_qos_rate" "test" {
						provider = unifi
						config {
							filter {
								name  = "name"
								value = "tfacc-group"
						  }
					  }
					}
				`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLengthAtLeast("unifi_client_qos_rate.test", 1),
				},
			},
		},
	})
}
