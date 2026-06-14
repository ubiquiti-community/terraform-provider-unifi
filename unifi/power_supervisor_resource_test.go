package unifi

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/hwtypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
)

// TestPowerSupervisorModelRoundTrip covers the model ⇄ go-unifi conversion for
// the Device Supervisor resource (#244): settings are sent as the user set them,
// power_sources are not sent (the controller resolves them) but are read back,
// and the computed consecutive_failures / id / power_sources land in the model.
func TestPowerSupervisorModelRoundTrip(t *testing.T) {
	ctx := context.Background()
	r := &powerSupervisorResource{}

	model := powerSupervisorResourceModel{
		DeviceMAC:         hwtypes.NewMACAddressValue("94:2a:6f:d6:ce:fd"),
		Enabled:           types.BoolValue(true),
		HeartbeatInterval: util.DurationValue(30, time.Second),
		SilenceThreshold:  util.DurationValue(600, time.Second),
		PowerOffDuration:  util.DurationValue(90, time.Second),
	}

	api := r.modelToPowerSupervisor(&model)
	if api.ClientMAC != "94:2a:6f:d6:ce:fd" {
		t.Errorf("ClientMAC = %q, want the device MAC", api.ClientMAC)
	}
	if !api.Enabled {
		t.Errorf("Enabled = false, want true")
	}
	if api.Settings.HeartbeatInterval != 30 || api.Settings.SilenceThreshold != 600 ||
		api.Settings.PowerOffDuration != 90 {
		t.Errorf("settings not mapped: %+v", api.Settings)
	}
	if api.PowerSources == nil {
		t.Errorf("PowerSources should be non-nil (sent empty), got nil")
	}

	// Simulate the controller's resting response and read it back.
	resp := &unifi.PowerSupervisor{
		ID:                  "000000000000000000000001",
		ClientMAC:           "94:2a:6f:d6:ce:fd",
		Enabled:             true,
		ConsecutiveFailures: 2,
		Settings: unifi.PowerSupervisorSettings{
			HeartbeatInterval: 30, SilenceThreshold: 600, PowerOffDuration: 90,
		},
		PowerSources: []unifi.PowerSupervisorSource{
			{
				ClientPsuIndex:   1,
				PowerSourceIndex: 4,
				PowerSourceMAC:   "f4:e2:c6:ad:4f:82",
				PowerSourceType:  "poe_port",
			},
		},
	}

	var out powerSupervisorResourceModel
	if d := r.powerSupervisorToModel(resp, &out, "default"); d.HasError() {
		t.Fatalf("powerSupervisorToModel: %v", d)
	}
	if out.ID.ValueString() != "000000000000000000000001" {
		t.Errorf("ID = %q", out.ID.ValueString())
	}
	if out.ConsecutiveFailures.ValueInt64() != 2 {
		t.Errorf("ConsecutiveFailures = %d, want 2", out.ConsecutiveFailures.ValueInt64())
	}
	if out.Site.ValueString() != "default" {
		t.Errorf("Site = %q, want default", out.Site.ValueString())
	}

	var sources []struct {
		ClientPsuIndex   int64  `tfsdk:"client_psu_index"`
		PowerSourceIndex int64  `tfsdk:"power_source_index"`
		PowerSourceMAC   string `tfsdk:"power_source_mac"`
		PowerSourceType  string `tfsdk:"power_source_type"`
	}
	if d := out.PowerSources.ElementsAs(ctx, &sources, false); d.HasError() {
		t.Fatalf("reading power_sources: %v", d)
	}
	if len(sources) != 1 || sources[0].PowerSourceMAC != "f4:e2:c6:ad:4f:82" ||
		sources[0].PowerSourceType != "poe_port" || sources[0].PowerSourceIndex != 4 {
		t.Errorf("power_sources not read back correctly: %+v", sources)
	}
}

func TestNewPowerSupervisorResource(t *testing.T) {
	tests := []struct {
		name string
		want fwresource.Resource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewPowerSupervisorResource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewPowerSupervisorResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewPowerSupervisorListResource(t *testing.T) {
	tests := []struct {
		name string
		want fwlist.ListResource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewPowerSupervisorListResource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewPowerSupervisorListResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_powerSourceAttrTypes(t *testing.T) {
	tests := []struct {
		name string
		want map[string]attr.Type
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := powerSourceAttrTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("powerSourceAttrTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_powerSupervisorResource_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.MetadataRequest
		resp *fwresource.MetadataResponse
	}
	tests := []struct {
		name string
		r    *powerSupervisorResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Metadata(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_powerSupervisorResource_IdentitySchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwresource.IdentitySchemaRequest
		resp *fwresource.IdentitySchemaResponse
	}
	tests := []struct {
		name string
		r    *powerSupervisorResource
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

func Test_powerSupervisorResource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.SchemaRequest
		resp *fwresource.SchemaResponse
	}
	tests := []struct {
		name string
		r    *powerSupervisorResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Schema(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_powerSupervisorResource_UpgradeState(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		r    *powerSupervisorResource
		args args
		want map[int64]fwresource.StateUpgrader
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.UpgradeState(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("powerSupervisorResource.UpgradeState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_powerSupervisorResource_Configure(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ConfigureRequest
		resp *fwresource.ConfigureResponse
	}
	tests := []struct {
		name string
		r    *powerSupervisorResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Configure(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_powerSupervisorResource_Create(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.CreateRequest
		resp *fwresource.CreateResponse
	}
	tests := []struct {
		name string
		r    *powerSupervisorResource
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

func Test_powerSupervisorResource_Read(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ReadRequest
		resp *fwresource.ReadResponse
	}
	tests := []struct {
		name string
		r    *powerSupervisorResource
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

func Test_powerSupervisorResource_Update(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.UpdateRequest
		resp *fwresource.UpdateResponse
	}
	tests := []struct {
		name string
		r    *powerSupervisorResource
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

func Test_powerSupervisorResource_Delete(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.DeleteRequest
		resp *fwresource.DeleteResponse
	}
	tests := []struct {
		name string
		r    *powerSupervisorResource
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

func Test_powerSupervisorResource_ImportState(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ImportStateRequest
		resp *fwresource.ImportStateResponse
	}
	tests := []struct {
		name string
		r    *powerSupervisorResource
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

func Test_powerSupervisorResource_modelToPowerSupervisor(t *testing.T) {
	type args struct {
		model *powerSupervisorResourceModel
	}
	tests := []struct {
		name string
		r    *powerSupervisorResource
		args args
		want *unifi.PowerSupervisor
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.modelToPowerSupervisor(tt.args.model); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("powerSupervisorResource.modelToPowerSupervisor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_powerSupervisorResource_powerSupervisorToModel(t *testing.T) {
	type args struct {
		supervisor *unifi.PowerSupervisor
		model      *powerSupervisorResourceModel
		site       string
	}
	tests := []struct {
		name string
		r    *powerSupervisorResource
		args args
		want diag.Diagnostics
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.powerSupervisorToModel(tt.args.supervisor, tt.args.model, tt.args.site); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("powerSupervisorResource.powerSupervisorToModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_powerSupervisorResource_ListResourceConfigSchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwlist.ListResourceSchemaRequest
		resp *fwlist.ListResourceSchemaResponse
	}
	tests := []struct {
		name string
		r    *powerSupervisorResource
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

func Test_powerSupervisorResource_List(t *testing.T) {
	type args struct {
		ctx    context.Context
		req    fwlist.ListRequest
		stream *fwlist.ListResultsStream
	}
	tests := []struct {
		name string
		r    *powerSupervisorResource
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
