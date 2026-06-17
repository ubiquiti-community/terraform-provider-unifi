package unifi

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// TestClientToModel_DefaultsWhenAPIOmitsFields proves the fix for the spurious
// in-place diff on every create/import: when the controller omits blocked / groups /
// qos_rate (as UniFi OS 5.x / Network App 10.x does for fixed-IP-only clients),
// Read must store the documented default (blocked=false) rather than null, and leave
// groups / qos_rate null so the UseStateForUnknown plan modifiers can keep the plan
// clean. For this minimal client clientToModel makes no API calls, so no live
// controller (and no mock) is needed.
func TestClientToModel_DefaultsWhenAPIOmitsFields(t *testing.T) {
	r := &clientResource{}

	client := &unifi.Client{
		ID:                     "61d1...",
		MAC:                    "02:00:00:de:ad:01",
		Name:                   "tf-test",
		FixedIP:                "192.168.40.251",
		Blocked:                nil, // controller omitted "blocked"
		UserGroupID:            "",  // no qos_rate / usergroup
		NetworkMembersGroupIDs: nil, // no groups
	}

	var model clientResourceModel
	diags := r.clientToModel(context.Background(), client, &model, "default")
	if diags.HasError() {
		t.Fatalf("clientToModel returned errors: %v", diags)
	}

	if model.Blocked.IsNull() || model.Blocked.IsUnknown() {
		t.Errorf("blocked: want concrete value, got null/unknown (%#v)", model.Blocked)
	}
	if model.Blocked.ValueBool() != false {
		t.Errorf("blocked: want false, got %v", model.Blocked.ValueBool())
	}
	if !model.Groups.IsNull() {
		t.Errorf("groups: want null, got %#v", model.Groups)
	}
	if !model.QOSRate.IsNull() {
		t.Errorf("qos_rate: want null, got %#v", model.QOSRate)
	}
}

// TestClientToModel_PreservesBlockedTrue ensures a blocked client still round-trips.
func TestClientToModel_PreservesBlockedTrue(t *testing.T) {
	r := &clientResource{}
	blocked := true
	client := &unifi.Client{MAC: "02:00:00:de:ad:02", Blocked: &blocked}

	var model clientResourceModel
	if diags := r.clientToModel(context.Background(), client, &model, "default"); diags.HasError() {
		t.Fatalf("clientToModel returned errors: %v", diags)
	}
	if model.Blocked.ValueBool() != true {
		t.Errorf("blocked: want true, got %v", model.Blocked.ValueBool())
	}
}

func TestAccClientFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_client.test", "name", "tfacc-client"),
					resource.TestCheckResourceAttr("unifi_client.test", "mac", "01:23:45:67:89:ab"),
					resource.TestCheckResourceAttr("unifi_client.test", "blocked", "false"),
				),
			},
			{
				ResourceName:    "unifi_client.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func testAccClientFrameworkConfig_basic() string {
	return `
resource "unifi_client" "test" {
	name = "tfacc-client"
	mac  = "01:23:45:67:89:ab"
}
`
}

func TestAccClientFramework_blocked(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientFrameworkConfig_blocked(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_client.test",
						"name",
						"tfacc-blocked-client",
					),
					resource.TestCheckResourceAttr("unifi_client.test", "blocked", "true"),
					resource.TestCheckResourceAttr(
						"unifi_client.test",
						"note",
						"Blocked for testing",
					),
				),
			},
		},
	})
}

func testAccClientFrameworkConfig_blocked() string {
	return `
resource "unifi_client" "test" {
	name    = "tfacc-blocked-client"
	mac     = "01:23:45:67:89:ac"
	blocked = true
	note    = "Blocked for testing"
}
`
}

func TestAccClientFramework_fixedIP(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientFrameworkConfig_fixedIP(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_client.test",
						"name",
						"tfacc-fixed-ip-client",
					),
					resource.TestCheckResourceAttr(
						"unifi_client.test",
						"fixed_ip",
						"192.168.2.100",
					),
				),
			},
		},
	})
}

func testAccClientFrameworkConfig_fixedIP() string {
	return `
resource "unifi_network" "test" {
	name    = "Test"
	subnet  = "192.168.2.1/24"
	vlan    = 2

	dhcp_server = {
		enabled    = true
		start = "192.168.2.6"
		stop  = "192.168.2.254"
	}
}

resource "unifi_client" "test" {
	name       = "tfacc-fixed-ip-client"
	mac        = "01:23:45:67:89:ad"
	fixed_ip   = "192.168.2.100"
	network_id = unifi_network.test.id
}
`
}

func TestAccClientFramework_groups(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create client with one group
			{
				Config: testAccClientFrameworkConfig_groups_one(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_client.test",
						"name",
						"tfacc-groups-client",
					),
					resource.TestCheckResourceAttr("unifi_client.test", "mac", "01:23:45:67:89:ae"),
					resource.TestCheckResourceAttr("unifi_client.test", "groups.#", "1"),
					resource.TestCheckResourceAttr(
						"unifi_client.test",
						"groups.0",
						"tfacc-group-a",
					),
				),
			},
			// Step 2: Import the client and verify groups survive
			{
				ResourceName:            "unifi_client.test",
				ImportState:             true,
				ImportStateKind:         resource.ImportBlockWithResourceIdentity,
				ImportStateVerifyIgnore: []string{"allow_existing", "skip_forget_on_destroy"},
			},
			// Step 3: Add another group
			{
				Config: testAccClientFrameworkConfig_groups_two(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_client.test",
						"name",
						"tfacc-groups-client",
					),
					resource.TestCheckResourceAttr("unifi_client.test", "groups.#", "2"),
					resource.TestCheckResourceAttr(
						"unifi_client.test",
						"groups.0",
						"tfacc-group-a",
					),
					resource.TestCheckResourceAttr(
						"unifi_client.test",
						"groups.1",
						"tfacc-group-b",
					),
				),
			},
		},
	})
}

func testAccClientFrameworkConfig_groups_one() string {
	return `
resource "unifi_client" "test" {
	name   = "tfacc-groups-client"
	mac    = "01:23:45:67:89:ae"
	groups = ["tfacc-group-a"]
}
`
}

func testAccClientFrameworkConfig_groups_two() string {
	return `
resource "unifi_client" "test" {
	name   = "tfacc-groups-client"
	mac    = "01:23:45:67:89:ae"
	groups = ["tfacc-group-a", "tfacc-group-b"]
}
`
}

func TestNewClientResource(t *testing.T) {
	tests := []struct {
		name string
		want fwresource.Resource
	}{
		{
			name: "returns clientResource",
			want: &clientResource{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewClientResource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewClientResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewClientListResource(t *testing.T) {
	tests := []struct {
		name string
		want fwlist.ListResource
	}{
		{
			name: "returns clientResource",
			want: &clientResource{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewClientListResource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewClientListResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_qosRateModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    qosRateModel
		want map[string]attr.Type
	}{
		{
			name: "returns correct attribute types",
			m:    qosRateModel{},
			want: map[string]attr.Type{
				"id":       types.StringType,
				"name":     types.StringType,
				"max_up":   types.Int64Type,
				"max_down": types.Int64Type,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("qosRateModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_clientResource_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.MetadataRequest
		resp *fwresource.MetadataResponse
	}
	tests := []struct {
		name string
		r    *clientResource
		args args
	}{
		{
			name: "sets type name",
			r:    &clientResource{},
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

func Test_clientResource_IdentitySchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwresource.IdentitySchemaRequest
		resp *fwresource.IdentitySchemaResponse
	}
	tests := []struct {
		name string
		r    *clientResource
		args args
	}{
		{
			name: "returns identity schema",
			r:    &clientResource{},
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

func Test_clientResource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.SchemaRequest
		resp *fwresource.SchemaResponse
	}
	tests := []struct {
		name string
		r    *clientResource
		args args
	}{
		{
			name: "returns schema",
			r:    &clientResource{},
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

func Test_clientResource_Configure(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ConfigureRequest
		resp *fwresource.ConfigureResponse
	}
	tests := []struct {
		name string
		r    *clientResource
		args args
	}{
		{
			name: "nil provider data",
			r:    &clientResource{},
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

func Test_clientResource_Create(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.CreateRequest
		resp *fwresource.CreateResponse
	}
	tests := []struct {
		name string
		r    *clientResource
		args args
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Create(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_clientResource_Read(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ReadRequest
		resp *fwresource.ReadResponse
	}
	tests := []struct {
		name string
		r    *clientResource
		args args
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Read(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_clientResource_Update(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.UpdateRequest
		resp *fwresource.UpdateResponse
	}
	tests := []struct {
		name string
		r    *clientResource
		args args
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Update(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_clientResource_applyPlanToState(t *testing.T) {
	type args struct {
		in0   context.Context
		plan  *clientResourceModel
		state *clientResourceModel
	}
	tests := []struct {
		name string
		r    *clientResource
		args args
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.applyPlanToState(tt.args.in0, tt.args.plan, tt.args.state)
		})
	}
}

func Test_clientResource_Delete(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.DeleteRequest
		resp *fwresource.DeleteResponse
	}
	tests := []struct {
		name string
		r    *clientResource
		args args
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Delete(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_clientResource_ImportState(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ImportStateRequest
		resp *fwresource.ImportStateResponse
	}
	tests := []struct {
		name string
		r    *clientResource
		args args
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.ImportState(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_clientResource_planToClient(t *testing.T) {
	type args struct {
		ctx  context.Context
		site string
		plan clientResourceModel
	}
	tests := []struct {
		name  string
		r     *clientResource
		args  args
		want  *unifi.Client
		want1 diag.Diagnostics
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.r.planToClient(tt.args.ctx, tt.args.site, tt.args.plan)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("clientResource.planToClient() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("clientResource.planToClient() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_clientResource_reconcileCreatedClient(t *testing.T) {
	type args struct {
		ctx           context.Context
		site          string
		currentClient *unifi.Client
		plannedClient *unifi.Client
	}
	tests := []struct {
		name  string
		r     *clientResource
		args  args
		want  *unifi.Client
		want1 diag.Diagnostics
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.r.reconcileCreatedClient(
				tt.args.ctx,
				tt.args.site,
				tt.args.currentClient,
				tt.args.plannedClient,
			)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("clientResource.reconcileCreatedClient() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf(
					"clientResource.reconcileCreatedClient() got1 = %v, want %v",
					got1,
					tt.want1,
				)
			}
		})
	}
}

func Test_clientResource_clientToModel(t *testing.T) {
	type args struct {
		ctx    context.Context
		client *unifi.Client
		model  *clientResourceModel
		site   string
	}
	tests := []struct {
		name string
		r    *clientResource
		args args
		want diag.Diagnostics
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.clientToModel(
				tt.args.ctx,
				tt.args.client,
				tt.args.model,
				tt.args.site,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("clientResource.clientToModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_clientResource_mergeClient(t *testing.T) {
	type args struct {
		existing *unifi.Client
		planned  *unifi.Client
	}
	tests := []struct {
		name string
		r    *clientResource
		args args
		want *unifi.Client
	}{
		{
			name: "planned values override existing",
			r:    &clientResource{},
			args: args{
				existing: &unifi.Client{
					ID:   "existing-id",
					MAC:  "aa:bb:cc:dd:ee:ff",
					Name: "old-name",
				},
				planned: &unifi.Client{
					Name:    "new-name",
					FixedIP: "192.168.1.100",
				},
			},
			want: &unifi.Client{
				ID:         "existing-id",
				MAC:        "aa:bb:cc:dd:ee:ff",
				Name:       "new-name",
				FixedIP:    "192.168.1.100",
				UseFixedIP: true,
			},
		},
		{
			name: "empty fixed_ip clears UseFixedIP",
			r:    &clientResource{},
			args: args{
				existing: &unifi.Client{
					ID:         "id1",
					FixedIP:    "10.0.0.1",
					UseFixedIP: true,
				},
				planned: &unifi.Client{
					FixedIP: "",
				},
			},
			want: &unifi.Client{
				ID:         "id1",
				UseFixedIP: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.mergeClient(
				tt.args.existing,
				tt.args.planned,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("clientResource.mergeClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_clientResource_ListResourceConfigSchema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwlist.ListResourceSchemaRequest
		resp *fwlist.ListResourceSchemaResponse
	}
	tests := []struct {
		name string
		r    *clientResource
		args args
	}{
		{
			name: "returns list schema",
			r:    &clientResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwlist.ListResourceSchemaRequest{},
				resp: &fwlist.ListResourceSchemaResponse{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.ListResourceConfigSchema(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_clientResource_resolveGroupNames(t *testing.T) {
	type args struct {
		ctx  context.Context
		site string
		ids  []string
	}
	tests := []struct {
		name    string
		r       *clientResource
		args    args
		want    []string
		wantErr bool
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.resolveGroupNames(tt.args.ctx, tt.args.site, tt.args.ids)
			if (err != nil) != tt.wantErr {
				t.Errorf(
					"clientResource.resolveGroupNames() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("clientResource.resolveGroupNames() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_clientResource_resolveGroupID(t *testing.T) {
	type args struct {
		ctx       context.Context
		site      string
		groupName string
	}
	tests := []struct {
		name    string
		r       *clientResource
		args    args
		want    string
		wantErr bool
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.resolveGroupID(tt.args.ctx, tt.args.site, tt.args.groupName)
			if (err != nil) != tt.wantErr {
				t.Errorf("clientResource.resolveGroupID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("clientResource.resolveGroupID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_clientResource_resolveClientGroup(t *testing.T) {
	type args struct {
		ctx  context.Context
		site string
		qos  qosRateModel
	}
	tests := []struct {
		name  string
		r     *clientResource
		args  args
		want  string
		want1 diag.Diagnostics
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.r.resolveClientGroup(tt.args.ctx, tt.args.site, tt.args.qos)
			if got != tt.want {
				t.Errorf("clientResource.resolveClientGroup() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("clientResource.resolveClientGroup() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_clientResource_List(t *testing.T) {
	type args struct {
		ctx    context.Context
		req    fwlist.ListRequest
		stream *fwlist.ListResultsStream
	}
	tests := []struct {
		name string
		r    *clientResource
		args args
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.List(tt.args.ctx, tt.args.req, tt.args.stream)
		})
	}
}

// nullPortOverrideAttrValues returns every port-override attribute set to its
// typed null, so a test only has to override the few fields it cares about.
func nullPortOverrideAttrValues() map[string]attr.Value {
	attrs := portOverrideAttrTypes()
	vals := make(map[string]attr.Value, len(attrs))
	for name, t := range attrs {
		switch tt := t.(type) {
		case basetypes.StringType:
			vals[name] = types.StringNull()
		case basetypes.Int64Type:
			vals[name] = types.Int64Null()
		case basetypes.BoolType:
			vals[name] = types.BoolNull()
		case basetypes.ListType:
			vals[name] = types.ListNull(tt.ElemType)
		case timetypes.GoDurationType:
			vals[name] = timetypes.NewGoDurationNull()
		}
		// Any unhandled attr type is intentionally left out so ObjectValue fails
		// loudly (signalling the helper needs updating) rather than silently.
	}
	return vals
}

func portOverrideSetWith(t *testing.T, overrides map[string]attr.Value) types.Set {
	t.Helper()
	attrs := nullPortOverrideAttrValues()
	for k, v := range overrides {
		attrs[k] = v
	}
	obj, d := types.ObjectValue(portOverrideAttrTypes(), attrs)
	if d.HasError() {
		t.Fatalf("building port override object: %v", d)
	}
	set, d := types.SetValue(
		types.ObjectType{AttrTypes: portOverrideAttrTypes()},
		[]attr.Value{obj},
	)
	if d.HasError() {
		t.Fatalf("building port override set: %v", d)
	}
	return set
}

// TestFrameworkToPortOverrides_AggregateOpMode guards #177: to form an SFP+ link
// aggregation the port's op_mode must be written as "aggregate" alongside the
// aggregate_members. op_mode is otherwise skipped (default "switch") so gateway
// devices that reject op_mode on PUT keep working (#213).
func TestFrameworkToPortOverrides_AggregateOpMode(t *testing.T) {
	ctx := context.Background()
	r := &deviceResource{}

	members, d := types.ListValue(types.Int64Type, []attr.Value{
		types.Int64Value(9),
		types.Int64Value(10),
	})
	if d.HasError() {
		t.Fatalf("building members list: %v", d)
	}

	set := portOverrideSetWith(t, map[string]attr.Value{
		"index":             types.Int64Value(9),
		"op_mode":           types.StringValue("aggregate"),
		"aggregate_members": members,
	})

	pos, diags := r.frameworkToPortOverrides(ctx, set)
	if diags.HasError() {
		t.Fatalf("frameworkToPortOverrides errored: %v", diags)
	}
	if len(pos) != 1 {
		t.Fatalf("got %d port overrides, want 1", len(pos))
	}
	po := pos[0]
	if po.OpMode != "aggregate" {
		t.Errorf("OpMode = %q, want aggregate (LAG would not engage)", po.OpMode)
	}
	if len(po.AggregateMembers) != 2 || po.AggregateMembers[0] != 9 ||
		po.AggregateMembers[1] != 10 {
		t.Errorf("AggregateMembers = %v, want [9 10]", po.AggregateMembers)
	}
}

// TestFrameworkToPortOverrides_SwitchOpModeOmitted ensures the default "switch"
// op_mode is not sent on the wire (it has omitempty), preserving the gateway
// write fix (#213).
func TestFrameworkToPortOverrides_SwitchOpModeOmitted(t *testing.T) {
	ctx := context.Background()
	r := &deviceResource{}

	set := portOverrideSetWith(t, map[string]attr.Value{
		"index":   types.Int64Value(1),
		"op_mode": types.StringValue("switch"),
	})

	pos, diags := r.frameworkToPortOverrides(ctx, set)
	if diags.HasError() {
		t.Fatalf("frameworkToPortOverrides errored: %v", diags)
	}
	if len(pos) != 1 {
		t.Fatalf("got %d port overrides, want 1", len(pos))
	}
	if pos[0].OpMode != "" {
		t.Errorf("OpMode = %q, want empty (omitted) for the switch default", pos[0].OpMode)
	}
}

// TestPortOverridesToFramework_TaggedNetworkIDsTypedNull is a regression test
// for #235. portOverridesToFramework must initialize the tagged_networkconf_ids
// model field to a typed null list. Previously it was left as an untyped
// zero-value types.List, which made types.ObjectValueFrom fail with a
// "types.ListType[!!! MISSING TYPE !!!]" Value Conversion Error during the
// Read/refresh (and import) of any unifi_device that has port overrides.
func TestPortOverridesToFramework_TaggedNetworkIDsTypedNull(t *testing.T) {
	r := &deviceResource{}

	set, diags := r.portOverridesToFramework(context.Background(), []unifi.DevicePortOverrides{
		{Name: "Port 1"},
	})

	if diags.HasError() {
		t.Fatalf(
			"portOverridesToFramework returned diagnostics (regression #235): %v",
			diags.Errors(),
		)
	}
	if set.IsNull() {
		t.Fatal("expected a non-null port_override set for a single override")
	}

	elems := set.Elements()
	if len(elems) != 1 {
		t.Fatalf("expected 1 port_override element, got %d", len(elems))
	}

	obj, ok := elems[0].(types.Object)
	if !ok {
		t.Fatalf("expected port_override element to be types.Object, got %T", elems[0])
	}

	taggedAttr, ok := obj.Attributes()["tagged_networkconf_ids"]
	if !ok {
		t.Fatal("port_override is missing the tagged_networkconf_ids attribute")
	}

	list, ok := taggedAttr.(types.List)
	if !ok {
		t.Fatalf("expected tagged_networkconf_ids to be types.List, got %T", taggedAttr)
	}
	if !list.IsNull() {
		t.Errorf("expected tagged_networkconf_ids to be a null list, got %v", list)
	}
	if et := list.ElementType(context.Background()); !et.Equal(types.StringType) {
		t.Errorf("expected tagged_networkconf_ids element type to be string, got %v", et)
	}
}

func TestAccClientList_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccClientFrameworkConfig_basic(),
			},
			{
				Query: true,
				Config: `
					provider "unifi" {}
					list "unifi_client" "test" {
						provider = unifi
						config {
							filter {
								name  = "name"
								value = "tfacc-client"
						  }
					  }
					}
				`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLengthAtLeast("unifi_client.test", 1),
					querycheck.ExpectIdentity("unifi_client.test", map[string]knownvalue.Check{
						"mac": knownvalue.StringExact("01:23:45:67:89:ab"),
					}),
				},
			},
		},
	})
}
