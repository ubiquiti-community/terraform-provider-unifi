package unifi

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/iptypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// TestSiteToSiteVPNModelRoundTrip validates the model <-> go-unifi Network
// conversion for the unifi_site_to_site_vpn resource (#78). It is a unit test
// rather than an acceptance test because the dockerized acceptance controller
// has no WAN/peer to establish an IPsec tunnel; the live round-trip is exercised
// against a real controller during development.
func TestSiteToSiteVPNModelRoundTrip(t *testing.T) {
	ctx := context.Background()
	r := &siteToSiteVPNResource{}

	subnets, d := types.ListValueFrom(
		ctx,
		types.StringType,
		[]string{"192.0.2.0/24", "198.51.100.0/24"},
	)
	if d.HasError() {
		t.Fatalf("building remote_subnets: %v", d)
	}
	model := &siteToSiteVPNResourceModel{
		Name:          types.StringValue("HQ-to-Branch"),
		Enabled:       types.BoolValue(true),
		Interface:     types.StringValue("wan"),
		PeerIP:        iptypes.NewIPv4AddressValue("203.0.113.9"),
		KeyExchange:   types.StringValue("ikev2"),
		PreSharedKey:  types.StringValue("s3cret-psk"),
		RemoteSubnets: subnets,
		Profile:       types.StringValue("customized"),
		IKEEncryption: types.StringValue("aes256"),
		IKEDhGroup:    types.Int64Value(14),
		PFS:           types.BoolValue(true),
	}

	network, diags := r.modelToNetwork(ctx, model)
	if diags.HasError() {
		t.Fatalf("modelToNetwork: %v", diags)
	}
	if network.Purpose != unifi.PurposeSiteVPN {
		t.Errorf("Purpose = %q, want %q", network.Purpose, unifi.PurposeSiteVPN)
	}
	if network.VPNType == nil || *network.VPNType != "ipsec-vpn" {
		t.Errorf("VPNType = %v, want ipsec-vpn", network.VPNType)
	}
	if network.IPSecPeerIP == nil || *network.IPSecPeerIP != "203.0.113.9" {
		t.Errorf("IPSecPeerIP = %v", network.IPSecPeerIP)
	}
	if network.IPSecPreSharedKey == nil || *network.IPSecPreSharedKey != "s3cret-psk" {
		t.Errorf("IPSecPreSharedKey not set")
	}
	if network.IPSecDhGroup == nil || *network.IPSecDhGroup != 14 {
		t.Errorf("IPSecDhGroup = %v, want 14", network.IPSecDhGroup)
	}
	if !network.IPSecPfs {
		t.Error("IPSecPfs = false, want true")
	}
	if len(network.RemoteVPNSubnets) != 2 {
		t.Errorf("RemoteVPNSubnets = %v, want 2 entries", network.RemoteVPNSubnets)
	}

	// API -> model: secret is preserved (not re-read), other fields map back.
	apiNetwork := &unifi.Network{
		ID:                "net-1",
		Name:              unifi.Ptr("HQ-to-Branch"),
		Purpose:           unifi.PurposeSiteVPN,
		Enabled:           true,
		VPNType:           unifi.Ptr("ipsec-vpn"),
		IPSecInterface:    unifi.Ptr("wan"),
		IPSecPeerIP:       unifi.Ptr("203.0.113.9"),
		IPSecKeyExchange:  unifi.Ptr("ikev2"),
		IPSecPreSharedKey: unifi.Ptr("echoed-by-controller"),
		IPSecPfs:          true,
		RemoteVPNSubnets:  []string{"192.0.2.0/24", "198.51.100.0/24"},
	}
	out := &siteToSiteVPNResourceModel{
		PreSharedKey: types.StringValue("s3cret-psk"), // prior state value
	}
	if diags := r.networkToModel(ctx, apiNetwork, out, "default"); diags.HasError() {
		t.Fatalf("networkToModel: %v", diags)
	}
	if out.ID.ValueString() != "net-1" {
		t.Errorf("ID = %q, want net-1", out.ID.ValueString())
	}
	if out.PeerIP.ValueString() != "203.0.113.9" {
		t.Errorf("PeerIP = %q", out.PeerIP.ValueString())
	}
	// The controller echoes the PSK on read, but networkToModel must preserve the
	// configured/state value to avoid perpetual diffs.
	if out.PreSharedKey.ValueString() != "s3cret-psk" {
		t.Errorf(
			"PreSharedKey = %q, want preserved s3cret-psk (not the API echo)",
			out.PreSharedKey.ValueString(),
		)
	}
	if l := len(out.RemoteSubnets.Elements()); l != 2 {
		t.Errorf("RemoteSubnets length = %d, want 2", l)
	}
}

func TestNewSiteToSiteVPNResource(t *testing.T) {
	tests := []struct {
		name string
		want fwresource.Resource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewSiteToSiteVPNResource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSiteToSiteVPNResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewSiteToSiteVPNListResource(t *testing.T) {
	tests := []struct {
		name string
		want fwlist.ListResource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewSiteToSiteVPNListResource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSiteToSiteVPNListResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_siteToSiteVPNResource_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.MetadataRequest
		resp *fwresource.MetadataResponse
	}
	tests := []struct {
		name string
		r    *siteToSiteVPNResource
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

func Test_siteToSiteVPNResource_IdentitySchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwresource.IdentitySchemaRequest
		resp *fwresource.IdentitySchemaResponse
	}
	tests := []struct {
		name string
		r    *siteToSiteVPNResource
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

func Test_siteToSiteVPNResource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.SchemaRequest
		resp *fwresource.SchemaResponse
	}
	tests := []struct {
		name string
		r    *siteToSiteVPNResource
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

func Test_siteToSiteVPNResource_UpgradeState(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		r    *siteToSiteVPNResource
		args args
		want map[int64]fwresource.StateUpgrader
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.UpgradeState(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("siteToSiteVPNResource.UpgradeState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_siteToSiteVPNResource_ConfigValidators(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		r    *siteToSiteVPNResource
		args args
		want []fwresource.ConfigValidator
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.ConfigValidators(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("siteToSiteVPNResource.ConfigValidators() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_siteToSiteVPNResource_Configure(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ConfigureRequest
		resp *fwresource.ConfigureResponse
	}
	tests := []struct {
		name string
		r    *siteToSiteVPNResource
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

func Test_siteToSiteVPNResource_Create(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.CreateRequest
		resp *fwresource.CreateResponse
	}
	tests := []struct {
		name string
		r    *siteToSiteVPNResource
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

func Test_siteToSiteVPNResource_Read(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ReadRequest
		resp *fwresource.ReadResponse
	}
	tests := []struct {
		name string
		r    *siteToSiteVPNResource
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

func Test_siteToSiteVPNResource_Update(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.UpdateRequest
		resp *fwresource.UpdateResponse
	}
	tests := []struct {
		name string
		r    *siteToSiteVPNResource
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

func Test_siteToSiteVPNResource_Delete(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.DeleteRequest
		resp *fwresource.DeleteResponse
	}
	tests := []struct {
		name string
		r    *siteToSiteVPNResource
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

func Test_siteToSiteVPNResource_ImportState(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ImportStateRequest
		resp *fwresource.ImportStateResponse
	}
	tests := []struct {
		name string
		r    *siteToSiteVPNResource
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

func Test_siteToSiteVPNResource_siteOrDefault(t *testing.T) {
	type args struct {
		site types.String
	}
	tests := []struct {
		name string
		r    *siteToSiteVPNResource
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.siteOrDefault(tt.args.site); got != tt.want {
				t.Errorf("siteToSiteVPNResource.siteOrDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_siteToSiteVPNResource_applyPreSharedKeyWO(t *testing.T) {
	type args struct {
		ctx     context.Context
		config  tfsdk.Config
		network *unifi.Network
		diags   *diag.Diagnostics
	}
	tests := []struct {
		name string
		r    *siteToSiteVPNResource
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.applyPreSharedKeyWO(tt.args.ctx, tt.args.config, tt.args.network, tt.args.diags); got != tt.want {
				t.Errorf("siteToSiteVPNResource.applyPreSharedKeyWO() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_siteToSiteVPNResource_modelToNetwork(t *testing.T) {
	type args struct {
		ctx   context.Context
		model *siteToSiteVPNResourceModel
	}
	tests := []struct {
		name  string
		r     *siteToSiteVPNResource
		args  args
		want  *unifi.Network
		want1 diag.Diagnostics
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.r.modelToNetwork(tt.args.ctx, tt.args.model)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("siteToSiteVPNResource.modelToNetwork() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("siteToSiteVPNResource.modelToNetwork() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_siteToSiteVPNResource_networkToModel(t *testing.T) {
	type args struct {
		ctx     context.Context
		network *unifi.Network
		model   *siteToSiteVPNResourceModel
		site    string
	}
	tests := []struct {
		name string
		r    *siteToSiteVPNResource
		args args
		want diag.Diagnostics
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.networkToModel(tt.args.ctx, tt.args.network, tt.args.model, tt.args.site); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("siteToSiteVPNResource.networkToModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_optStr(t *testing.T) {
	type args struct {
		s interface {
			IsNull() bool
			IsUnknown() bool
			ValueString() string
		}
	}
	tests := []struct {
		name string
		args args
		want *string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := optStr(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("optStr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_optInt64(t *testing.T) {
	type args struct {
		v types.Int64
	}
	tests := []struct {
		name string
		args args
		want *int64
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := optInt64(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("optInt64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_stringPtrOrNull(t *testing.T) {
	type args struct {
		v *string
	}
	tests := []struct {
		name string
		args args
		want types.String
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stringPtrOrNull(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("stringPtrOrNull() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_siteToSiteVPNResource_ListResourceConfigSchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwlist.ListResourceSchemaRequest
		resp *fwlist.ListResourceSchemaResponse
	}
	tests := []struct {
		name string
		r    *siteToSiteVPNResource
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

func Test_siteToSiteVPNResource_List(t *testing.T) {
	type args struct {
		ctx    context.Context
		req    fwlist.ListRequest
		stream *fwlist.ListResultsStream
	}
	tests := []struct {
		name string
		r    *siteToSiteVPNResource
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
