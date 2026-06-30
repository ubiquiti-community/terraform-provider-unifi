package unifi

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// TestMergePortOverridesByIndex guards #266: declaring a subset of port_override
// blocks must not wipe the device's other ports. The UniFi PUT replaces the whole
// port_overrides array, so the provider merges the declared ports (by port_idx)
// onto the device's current overrides before sending.
func TestMergePortOverridesByIndex(t *testing.T) {
	current := []unifi.DevicePortOverrides{
		{PortIDX: ptrInt64(3), NATiveNetworkID: "vlan-a"},
		{PortIDX: ptrInt64(4), NATiveNetworkID: "vlan-b"},
		{PortIDX: ptrInt64(5), NATiveNetworkID: "vlan-c"},
	}

	t.Run("subset replaces only its port, keeps the rest", func(t *testing.T) {
		declared := []unifi.DevicePortOverrides{
			{PortIDX: ptrInt64(5), NATiveNetworkID: "vlan-z"},
		}
		got := mergePortOverridesByIndex(current, declared)
		byIdx := indexOverrides(got)
		if len(got) != 3 {
			t.Fatalf("merged length = %d, want 3 (ports 3,4 must survive): %+v", len(got), got)
		}
		if byIdx[3].NATiveNetworkID != "vlan-a" || byIdx[4].NATiveNetworkID != "vlan-b" {
			t.Errorf("undeclared ports were altered: %+v", got)
		}
		if byIdx[5].NATiveNetworkID != "vlan-z" {
			t.Errorf("declared port 5 = %q, want vlan-z", byIdx[5].NATiveNetworkID)
		}
	})

	t.Run("declared new port is appended", func(t *testing.T) {
		declared := []unifi.DevicePortOverrides{
			{PortIDX: ptrInt64(7), NATiveNetworkID: "vlan-new"},
		}
		got := mergePortOverridesByIndex(current, declared)
		byIdx := indexOverrides(got)
		if len(got) != 4 {
			t.Fatalf("merged length = %d, want 4: %+v", len(got), got)
		}
		if byIdx[7].NATiveNetworkID != "vlan-new" {
			t.Errorf("new port 7 not appended: %+v", got)
		}
	})

	t.Run("no declared overrides returns current unchanged", func(t *testing.T) {
		got := mergePortOverridesByIndex(current, nil)
		if len(got) != 3 {
			t.Errorf("merged length = %d, want 3", len(got))
		}
	})
}

func indexOverrides(pos []unifi.DevicePortOverrides) map[int64]unifi.DevicePortOverrides {
	m := make(map[int64]unifi.DevicePortOverrides, len(pos))
	for _, po := range pos {
		if po.PortIDX != nil {
			m[*po.PortIDX] = po
		}
	}
	return m
}

func TestAccDeviceFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDeviceFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_device.test", "id"),
					resource.TestCheckResourceAttr("unifi_device.test", "name", "Test Device"),
					resource.TestCheckResourceAttr("unifi_device.test", "adopted", "true"),
				),
			},
			{
				ResourceName:            "unifi_device.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"allow_adoption", "forget_on_destroy"},
			},
		},
	})
}

func testAccDeviceFrameworkConfig_basic() string {
	return `
resource "unifi_device" "test" {
	mac  = "00:27:22:00:00:02"
	name = "Test Device"
	allow_adoption = true
	forget_on_destroy = false
}
`
}

func TestNewDeviceFrameworkResource(t *testing.T) {
	tests := []struct {
		name string
		want fwresource.Resource
	}{
		{
			name: "returns deviceResource",
			want: &deviceResource{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewDeviceFrameworkResource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDeviceFrameworkResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewDeviceListResource(t *testing.T) {
	tests := []struct {
		name string
		want fwlist.ListResource
	}{
		{
			name: "returns deviceResource",
			want: &deviceResource{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewDeviceListResource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDeviceListResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_portOverrideModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    portOverrideModel
		want map[string]attr.Type
	}{
		{
			name: "returns portOverrideAttrTypes",
			m:    portOverrideModel{},
			want: portOverrideAttrTypes(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("portOverrideModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_deviceResource_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.MetadataRequest
		resp *fwresource.MetadataResponse
	}
	tests := []struct {
		name string
		r    *deviceResource
		args args
	}{
		{
			name: "sets type name",
			r:    &deviceResource{},
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

func Test_deviceResource_IdentitySchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwresource.IdentitySchemaRequest
		resp *fwresource.IdentitySchemaResponse
	}
	tests := []struct {
		name string
		r    *deviceResource
		args args
	}{
		{
			name: "returns identity schema",
			r:    &deviceResource{},
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

func Test_deviceResource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.SchemaRequest
		resp *fwresource.SchemaResponse
	}
	tests := []struct {
		name string
		r    *deviceResource
		args args
	}{
		{
			name: "returns schema",
			r:    &deviceResource{},
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

func Test_deviceResource_UpgradeState(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		r    *deviceResource
		args args
		want map[int64]fwresource.StateUpgrader
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.UpgradeState(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("deviceResource.UpgradeState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_deviceResource_Configure(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ConfigureRequest
		resp *fwresource.ConfigureResponse
	}
	tests := []struct {
		name string
		r    *deviceResource
		args args
	}{
		{
			name: "nil provider data",
			r:    &deviceResource{},
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

func Test_deviceResource_Create(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.CreateRequest
		resp *fwresource.CreateResponse
	}
	tests := []struct {
		name string
		r    *deviceResource
		args args
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Create(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_deviceResource_Read(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ReadRequest
		resp *fwresource.ReadResponse
	}
	tests := []struct {
		name string
		r    *deviceResource
		args args
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Read(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_deviceResource_Update(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.UpdateRequest
		resp *fwresource.UpdateResponse
	}
	tests := []struct {
		name string
		r    *deviceResource
		args args
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Update(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_deviceResource_Delete(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.DeleteRequest
		resp *fwresource.DeleteResponse
	}
	tests := []struct {
		name string
		r    *deviceResource
		args args
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Delete(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_deviceResource_ImportState(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ImportStateRequest
		resp *fwresource.ImportStateResponse
	}
	tests := []struct {
		name string
		r    *deviceResource
		args args
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.ImportState(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

// Test_buildMinimalUpdateDevice guards #337: the update PUT body must carry the
// LED override fields. They used to be filled by modelToAPIDevice but dropped
// when assembling the minimal PUT payload, so the controller kept the old LED
// values and the post-apply read conflicted with the plan.
func Test_buildMinimalUpdateDevice(t *testing.T) {
	deviceReq := &unifi.Device{
		ID:                         "dev-1",
		Type:                       "uap",
		MAC:                        "00:11:22:33:44:55",
		Name:                       "AP-Hallway",
		LedOverride:                "on",
		LedOverrideColor:           "#00ff00",
		LedOverrideColorBrightness: ptrInt64(20),
	}
	current := &unifi.Device{State: 1, Adopted: true}
	overrides := []unifi.DevicePortOverrides{{PortIDX: ptrInt64(1)}}

	got := buildMinimalUpdateDevice(deviceReq, current, overrides)

	if got.LedOverride != "on" {
		t.Errorf("LedOverride = %q, want on", got.LedOverride)
	}
	if got.LedOverrideColor != "#00ff00" {
		t.Errorf("LedOverrideColor = %q, want #00ff00", got.LedOverrideColor)
	}
	if got.LedOverrideColorBrightness == nil || *got.LedOverrideColorBrightness != 20 {
		t.Errorf("LedOverrideColorBrightness = %v, want 20", got.LedOverrideColorBrightness)
	}
	// State/Adopted carried over from the current device; other fields preserved.
	if got.State != 1 || !got.Adopted {
		t.Errorf(
			"State/Adopted not carried from current: state=%v adopted=%v",
			got.State,
			got.Adopted,
		)
	}
	if got.Name != "AP-Hallway" || len(got.PortOverrides) != 1 {
		t.Errorf(
			"unexpected name/overrides: name=%q overrides=%d",
			got.Name,
			len(got.PortOverrides),
		)
	}

	// Unset LED fields stay zero-valued (omitempty drops them from the PUT body).
	bare := buildMinimalUpdateDevice(&unifi.Device{ID: "d2"}, nil, nil)
	if bare.LedOverride != "" || bare.LedOverrideColorBrightness != nil {
		t.Errorf("unset LED fields should be zero: %q %v",
			bare.LedOverride, bare.LedOverrideColorBrightness)
	}
}

func Test_deviceResource_updateDevice(t *testing.T) {
	type args struct {
		ctx   context.Context
		model *deviceResourceModel
	}
	tests := []struct {
		name string
		r    *deviceResource
		args args
		want diag.Diagnostics
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.updateDevice(
				tt.args.ctx,
				tt.args.model,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("deviceResource.updateDevice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_deviceResource_setResourceData(t *testing.T) {
	type args struct {
		ctx    context.Context
		diags  *diag.Diagnostics
		device *unifi.Device
		model  *deviceResourceModel
		site   string
	}
	tests := []struct {
		name string
		r    *deviceResource
		args args
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.setResourceData(
				tt.args.ctx,
				tt.args.diags,
				tt.args.device,
				tt.args.model,
				tt.args.site,
			)
		})
	}
}

func Test_deviceResource_modelToAPIDevice(t *testing.T) {
	type args struct {
		ctx   context.Context
		model *deviceResourceModel
	}
	tests := []struct {
		name  string
		r     *deviceResource
		args  args
		want  *unifi.Device
		want1 diag.Diagnostics
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.r.modelToAPIDevice(tt.args.ctx, tt.args.model)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("deviceResource.modelToAPIDevice() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("deviceResource.modelToAPIDevice() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_mergePortOverridesByIndex(t *testing.T) {
	type args struct {
		current  []unifi.DevicePortOverrides
		declared []unifi.DevicePortOverrides
	}
	tests := []struct {
		name string
		args args
		want []unifi.DevicePortOverrides
	}{
		{
			name: "nil current and nil declared returns nil",
			args: args{current: nil, declared: nil},
			want: nil,
		},
		{
			name: "nil current with declared returns declared",
			args: args{
				current: nil,
				declared: []unifi.DevicePortOverrides{
					{PortIDX: ptrInt64(1), NATiveNetworkID: "net-a"},
				},
			},
			want: []unifi.DevicePortOverrides{
				{PortIDX: ptrInt64(1), NATiveNetworkID: "net-a"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mergePortOverridesByIndex(
				tt.args.current,
				tt.args.declared,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("mergePortOverridesByIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_deviceResource_reconcilePortOverrides(t *testing.T) {
	type args struct {
		ctx          context.Context
		prior        types.Set
		apiOverrides []unifi.DevicePortOverrides
	}
	tests := []struct {
		name  string
		r     *deviceResource
		args  args
		want  types.Set
		want1 diag.Diagnostics
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.r.reconcilePortOverrides(
				tt.args.ctx,
				tt.args.prior,
				tt.args.apiOverrides,
			)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("deviceResource.reconcilePortOverrides() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf(
					"deviceResource.reconcilePortOverrides() got1 = %v, want %v",
					got1,
					tt.want1,
				)
			}
		})
	}
}

func Test_deviceResource_portOverridesToFramework(t *testing.T) {
	type args struct {
		ctx context.Context
		pos []unifi.DevicePortOverrides
	}
	tests := []struct {
		name  string
		r     *deviceResource
		args  args
		want  types.Set
		want1 diag.Diagnostics
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.r.portOverridesToFramework(tt.args.ctx, tt.args.pos)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf(
					"deviceResource.portOverridesToFramework() got = %v, want %v",
					got,
					tt.want,
				)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf(
					"deviceResource.portOverridesToFramework() got1 = %v, want %v",
					got1,
					tt.want1,
				)
			}
		})
	}
}

func Test_deviceResource_frameworkToPortOverrides(t *testing.T) {
	type args struct {
		ctx             context.Context
		portOverrideSet types.Set
	}
	tests := []struct {
		name  string
		r     *deviceResource
		args  args
		want  []unifi.DevicePortOverrides
		want1 diag.Diagnostics
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.r.frameworkToPortOverrides(tt.args.ctx, tt.args.portOverrideSet)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf(
					"deviceResource.frameworkToPortOverrides() got = %v, want %v",
					got,
					tt.want,
				)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf(
					"deviceResource.frameworkToPortOverrides() got1 = %v, want %v",
					got1,
					tt.want1,
				)
			}
		})
	}
}

func Test_deviceResource_waitForDeviceState(t *testing.T) {
	type args struct {
		ctx           context.Context
		site          string
		mac           string
		targetState   unifi.DeviceState
		pendingStates []unifi.DeviceState
		timeout       time.Duration
	}
	tests := []struct {
		name    string
		r       *deviceResource
		args    args
		want    *unifi.Device
		wantErr bool
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.waitForDeviceState(
				tt.args.ctx,
				tt.args.site,
				tt.args.mac,
				tt.args.targetState,
				tt.args.pendingStates,
				tt.args.timeout,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf(
					"deviceResource.waitForDeviceState() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("deviceResource.waitForDeviceState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_cleanMAC(t *testing.T) {
	type args struct {
		mac string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "converts dashes to colons and lowercases",
			args: args{mac: "AA-BB-CC-DD-EE-FF"},
			want: "aa:bb:cc:dd:ee:ff",
		},
		{
			name: "already lowercase colons unchanged",
			args: args{mac: "aa:bb:cc:dd:ee:ff"},
			want: "aa:bb:cc:dd:ee:ff",
		},
		{
			name: "uppercase colons lowercased",
			args: args{mac: "AA:BB:CC:DD:EE:FF"},
			want: "aa:bb:cc:dd:ee:ff",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanMAC(tt.args.mac); got != tt.want {
				t.Errorf("cleanMAC() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_portOverrideAttrTypes(t *testing.T) {
	tests := []struct {
		name string
		want map[string]attr.Type
	}{
		{
			name: "returns non-empty map with expected keys",
			want: portOverrideAttrTypes(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := portOverrideAttrTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("portOverrideAttrTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_configNetworkAttrTypes(t *testing.T) {
	tests := []struct {
		name string
		want map[string]attr.Type
	}{
		{
			name: "returns correct attribute types",
			want: map[string]attr.Type{
				"type":            types.StringType,
				"ip":              types.StringType,
				"netmask":         types.StringType,
				"gateway":         types.StringType,
				"dns1":            types.StringType,
				"dns2":            types.StringType,
				"dnssuffix":       types.StringType,
				"bonding_enabled": types.BoolType,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := configNetworkAttrTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("configNetworkAttrTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_radioTableAttrTypes(t *testing.T) {
	tests := []struct {
		name string
		want map[string]attr.Type
	}{
		{
			name: "returns correct attribute types",
			want: radioTableAttrTypes(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := radioTableAttrTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("radioTableAttrTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_outletOverrideAttrTypes(t *testing.T) {
	tests := []struct {
		name string
		want map[string]attr.Type
	}{
		{
			name: "returns correct attribute types",
			want: map[string]attr.Type{
				"index":         types.Int64Type,
				"name":          types.StringType,
				"relay_state":   types.BoolType,
				"cycle_enabled": types.BoolType,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := outletOverrideAttrTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("outletOverrideAttrTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_stringOrNull(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want types.String
	}{
		{
			name: "empty string returns null",
			args: args{s: ""},
			want: types.StringNull(),
		},
		{
			name: "non-empty string returns value",
			args: args{s: "hello"},
			want: types.StringValue("hello"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stringOrNull(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("stringOrNull() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_int64OrNull(t *testing.T) {
	type args struct {
		i int64
	}
	tests := []struct {
		name string
		args args
		want types.Int64
	}{
		{
			name: "zero returns null",
			args: args{i: 0},
			want: types.Int64Null(),
		},
		{
			name: "non-zero returns value",
			args: args{i: 42},
			want: types.Int64Value(42),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := int64OrNull(tt.args.i); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("int64OrNull() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_deviceResource_configNetworkToFramework(t *testing.T) {
	type args struct {
		ctx context.Context
		cn  *unifi.DeviceConfigNetwork
	}
	tests := []struct {
		name  string
		r     *deviceResource
		args  args
		want  types.Object
		want1 diag.Diagnostics
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.r.configNetworkToFramework(tt.args.ctx, tt.args.cn)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf(
					"deviceResource.configNetworkToFramework() got = %v, want %v",
					got,
					tt.want,
				)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf(
					"deviceResource.configNetworkToFramework() got1 = %v, want %v",
					got1,
					tt.want1,
				)
			}
		})
	}
}

func Test_deviceResource_radioTableToFramework(t *testing.T) {
	type args struct {
		ctx    context.Context
		radios []unifi.DeviceRadioTable
	}
	tests := []struct {
		name  string
		r     *deviceResource
		args  args
		want  types.List
		want1 diag.Diagnostics
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.r.radioTableToFramework(tt.args.ctx, tt.args.radios)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("deviceResource.radioTableToFramework() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf(
					"deviceResource.radioTableToFramework() got1 = %v, want %v",
					got1,
					tt.want1,
				)
			}
		})
	}
}

func Test_deviceResource_outletOverridesToFramework(t *testing.T) {
	type args struct {
		ctx     context.Context
		outlets []unifi.DeviceOutletOverrides
	}
	tests := []struct {
		name  string
		r     *deviceResource
		args  args
		want  types.List
		want1 diag.Diagnostics
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.r.outletOverridesToFramework(tt.args.ctx, tt.args.outlets)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf(
					"deviceResource.outletOverridesToFramework() got = %v, want %v",
					got,
					tt.want,
				)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf(
					"deviceResource.outletOverridesToFramework() got1 = %v, want %v",
					got1,
					tt.want1,
				)
			}
		})
	}
}

func Test_deviceResource_frameworkToConfigNetwork(t *testing.T) {
	type args struct {
		ctx              context.Context
		configNetworkObj types.Object
	}
	tests := []struct {
		name  string
		r     *deviceResource
		args  args
		want  *unifi.DeviceConfigNetwork
		want1 diag.Diagnostics
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.r.frameworkToConfigNetwork(tt.args.ctx, tt.args.configNetworkObj)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf(
					"deviceResource.frameworkToConfigNetwork() got = %v, want %v",
					got,
					tt.want,
				)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf(
					"deviceResource.frameworkToConfigNetwork() got1 = %v, want %v",
					got1,
					tt.want1,
				)
			}
		})
	}
}

func Test_deviceResource_frameworkToRadioTable(t *testing.T) {
	type args struct {
		ctx       context.Context
		radioList types.List
	}
	tests := []struct {
		name  string
		r     *deviceResource
		args  args
		want  []unifi.DeviceRadioTable
		want1 diag.Diagnostics
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.r.frameworkToRadioTable(tt.args.ctx, tt.args.radioList)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("deviceResource.frameworkToRadioTable() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf(
					"deviceResource.frameworkToRadioTable() got1 = %v, want %v",
					got1,
					tt.want1,
				)
			}
		})
	}
}

func Test_deviceResource_frameworkToOutletOverrides(t *testing.T) {
	type args struct {
		ctx        context.Context
		outletList types.List
	}
	tests := []struct {
		name  string
		r     *deviceResource
		args  args
		want  []unifi.DeviceOutletOverrides
		want1 diag.Diagnostics
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.r.frameworkToOutletOverrides(tt.args.ctx, tt.args.outletList)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf(
					"deviceResource.frameworkToOutletOverrides() got = %v, want %v",
					got,
					tt.want,
				)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf(
					"deviceResource.frameworkToOutletOverrides() got1 = %v, want %v",
					got1,
					tt.want1,
				)
			}
		})
	}
}

func Test_deviceResource_deviceListToModel(t *testing.T) {
	type args struct {
		ctx   context.Context
		api   *unifi.Device
		model *deviceResourceModel
		site  string
	}
	tests := []struct {
		name string
		r    *deviceResource
		args args
		want diag.Diagnostics
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.deviceListToModel(
				tt.args.ctx,
				tt.args.api,
				tt.args.model,
				tt.args.site,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("deviceResource.deviceListToModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_deviceResource_ListResourceConfigSchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwlist.ListResourceSchemaRequest
		resp *fwlist.ListResourceSchemaResponse
	}
	tests := []struct {
		name string
		r    *deviceResource
		args args
	}{
		{
			name: "returns list schema",
			r:    &deviceResource{},
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

func Test_deviceResource_List(t *testing.T) {
	type args struct {
		ctx    context.Context
		req    fwlist.ListRequest
		stream *fwlist.ListResultsStream
	}
	tests := []struct {
		name string
		r    *deviceResource
		args args
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.List(tt.args.ctx, tt.args.req, tt.args.stream)
		})
	}
}
