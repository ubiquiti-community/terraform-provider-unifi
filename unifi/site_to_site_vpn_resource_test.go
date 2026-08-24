package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/iptypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
	r := NewSiteToSiteVPNResource()
	if r == nil {
		t.Fatal("NewSiteToSiteVPNResource() returned nil")
	}
	if _, ok := r.(fwresource.ResourceWithImportState); !ok {
		t.Error("expected ResourceWithImportState interface")
	}
}

func TestNewSiteToSiteVPNListResource(t *testing.T) {
	r := NewSiteToSiteVPNListResource()
	if r == nil {
		t.Fatal("NewSiteToSiteVPNListResource() returned nil")
	}
	if _, ok := r.(fwlist.ListResourceWithConfigure); !ok {
		t.Error("expected ListResourceWithConfigure interface")
	}
}

func Test_siteToSiteVPNResource_Metadata(t *testing.T) {
	for _, tt := range []struct{ provider, want string }{
		{"unifi", "unifi_site_to_site_vpn"},
		{"test", "test_site_to_site_vpn"},
	} {
		t.Run(tt.provider, func(t *testing.T) {
			r := &siteToSiteVPNResource{}
			resp := &fwresource.MetadataResponse{}
			r.Metadata(
				context.Background(),
				fwresource.MetadataRequest{ProviderTypeName: tt.provider},
				resp,
			)
			if resp.TypeName != tt.want {
				t.Errorf("got %q, want %q", resp.TypeName, tt.want)
			}
		})
	}
}

func Test_siteToSiteVPNResource_IdentitySchema(t *testing.T) {
	r := &siteToSiteVPNResource{}
	resp := &fwresource.IdentitySchemaResponse{}
	r.IdentitySchema(context.Background(), fwresource.IdentitySchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("IdentitySchema() produced errors: %v", resp.Diagnostics)
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Error("IdentitySchema missing 'id' attribute")
	}
	if _, ok := resp.IdentitySchema.Attributes["site"]; !ok {
		t.Error("IdentitySchema missing 'site' attribute")
	}
}

// siteToSiteVPNImportHarness builds the empty state and null identity
// containers the framework hands to ImportState, so the import logic can be
// unit tested without a live controller.
func siteToSiteVPNImportHarness(t *testing.T) (tfsdk.State, tfsdk.ResourceIdentity) {
	t.Helper()
	ctx := context.Background()
	r := &siteToSiteVPNResource{}

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}

	var idResp fwresource.IdentitySchemaResponse
	r.IdentitySchema(ctx, fwresource.IdentitySchemaRequest{}, &idResp)
	if idResp.Diagnostics.HasError() {
		t.Fatalf("identity schema: %v", idResp.Diagnostics)
	}

	state := tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
	}
	identity := tfsdk.ResourceIdentity{
		Schema: idResp.IdentitySchema,
		Raw:    tftypes.NewValue(idResp.IdentitySchema.Type().TerraformType(ctx), nil),
	}
	return state, identity
}

// siteToSiteVPNIdentityValue builds a populated identity container for
// identity-based import requests. Pass "" to leave site null.
func siteToSiteVPNIdentityValue(t *testing.T, id, site string) tfsdk.ResourceIdentity {
	t.Helper()
	ctx := context.Background()
	_, identity := siteToSiteVPNImportHarness(t)

	siteVal := tftypes.NewValue(tftypes.String, nil)
	if site != "" {
		siteVal = tftypes.NewValue(tftypes.String, site)
	}
	identity.Raw = tftypes.NewValue(
		identity.Schema.Type().TerraformType(ctx),
		map[string]tftypes.Value{
			"id":   tftypes.NewValue(tftypes.String, id),
			"site": siteVal,
		},
	)
	return identity
}

func Test_siteToSiteVPNResource_ImportState(t *testing.T) {
	ctx := context.Background()
	const oid = "0123456789abcdef01234567"

	newRes := func() *siteToSiteVPNResource {
		return &siteToSiteVPNResource{client: &Client{Site: "default"}}
	}

	getString := func(t *testing.T, get func(context.Context, path.Path, any) diag.Diagnostics, p path.Path) types.String {
		t.Helper()
		var v types.String
		if d := get(ctx, p, &v); d.HasError() {
			t.Fatalf("get %s: %v", p, d)
		}
		return v
	}

	t.Run("identity import defaults omitted site to provider site", func(t *testing.T) {
		r := newRes()
		state, _ := siteToSiteVPNImportHarness(t)
		reqIdentity := siteToSiteVPNIdentityValue(t, oid, "")
		respIdentity := siteToSiteVPNIdentityValue(t, oid, "")
		resp := &fwresource.ImportStateResponse{State: state, Identity: &respIdentity}

		r.ImportState(ctx, fwresource.ImportStateRequest{ID: "", Identity: &reqIdentity}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diags: %v", resp.Diagnostics)
		}
		if got := getString(t, resp.State.GetAttribute, path.Root("id")); got.ValueString() != oid {
			t.Errorf("state id = %v, want %s", got, oid)
		}
		if got := getString(
			t,
			resp.State.GetAttribute,
			path.Root("site"),
		); got.ValueString() != "default" {
			t.Errorf("state site = %v, want default", got)
		}
		if got := getString(
			t,
			resp.Identity.GetAttribute,
			path.Root("site"),
		); got.ValueString() != "default" {
			t.Errorf("identity site = %v, want default", got)
		}
	})

	t.Run("string import by id mirrors identity", func(t *testing.T) {
		r := newRes()
		state, respIdentity := siteToSiteVPNImportHarness(t)
		resp := &fwresource.ImportStateResponse{State: state, Identity: &respIdentity}

		r.ImportState(ctx, fwresource.ImportStateRequest{ID: oid}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diags: %v", resp.Diagnostics)
		}
		if got := getString(t, resp.State.GetAttribute, path.Root("id")); got.ValueString() != oid {
			t.Errorf("state id = %v, want %s", got, oid)
		}
		if got := getString(
			t,
			resp.Identity.GetAttribute,
			path.Root("id"),
		); got.ValueString() != oid {
			t.Errorf("identity id = %v, want %s", got, oid)
		}
		if got := getString(
			t,
			resp.Identity.GetAttribute,
			path.Root("site"),
		); got.ValueString() != "default" {
			t.Errorf("identity site = %v, want default", got)
		}
	})

	t.Run("string import with site prefix", func(t *testing.T) {
		r := newRes()
		state, respIdentity := siteToSiteVPNImportHarness(t)
		resp := &fwresource.ImportStateResponse{State: state, Identity: &respIdentity}

		r.ImportState(ctx, fwresource.ImportStateRequest{ID: "other:" + oid}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diags: %v", resp.Diagnostics)
		}
		if got := getString(
			t,
			resp.State.GetAttribute,
			path.Root("site"),
		); got.ValueString() != "other" {
			t.Errorf("state site = %v, want other", got)
		}
		if got := getString(
			t,
			resp.Identity.GetAttribute,
			path.Root("site"),
		); got.ValueString() != "other" {
			t.Errorf("identity site = %v, want other", got)
		}
	})
}

func TestAccSiteToSiteVPN_import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSiteToSiteVPNConfig_writeOnlyPSK(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_site_to_site_vpn.test",
						"name",
						"tfacc-s2s-vpn",
					),
					resource.TestCheckResourceAttr(
						"unifi_site_to_site_vpn.test",
						"peer_ip",
						"203.0.113.9",
					),
					resource.TestCheckResourceAttr(
						"unifi_site_to_site_vpn.test",
						"remote_subnets.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_site_to_site_vpn.test",
						"remote_subnets.0",
						"192.0.2.0/24",
					),
					resource.TestCheckResourceAttrSet("unifi_site_to_site_vpn.test", "id"),
				),
			},
			// String-ID import. The pre-shared key is write-only, so it is
			// null in both the prior and imported state.
			{
				ResourceName:      "unifi_site_to_site_vpn.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Identity-based import (import block with identity, Terraform 1.12+).
			{
				ResourceName:    "unifi_site_to_site_vpn.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func testAccSiteToSiteVPNConfig_writeOnlyPSK() string {
	return `
resource "unifi_site_to_site_vpn" "test" {
  name              = "tfacc-s2s-vpn"
  peer_ip           = "203.0.113.9"
  pre_shared_key_wo = "tfacc-s2s-psk-secret"
  remote_subnets    = ["192.0.2.0/24"]
}
`
}

func Test_siteToSiteVPNResource_Schema(t *testing.T) {
	r := &siteToSiteVPNResource{}
	resp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() produced errors: %v", resp.Diagnostics)
	}
	for _, attr := range []string{
		"id", "site", "name", "enabled", "interface", "peer_ip",
		"pre_shared_key", "remote_subnets", "profile", "timeouts",
	} {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("missing attribute %q", attr)
		}
	}
}

func Test_siteToSiteVPNResource_UpgradeState(t *testing.T) {
	r := &siteToSiteVPNResource{}
	upgraders := r.UpgradeState(context.Background())
	if _, ok := upgraders[0]; !ok {
		t.Error("expected state upgrader for version 0")
	}
}

func Test_siteToSiteVPNResource_ConfigValidators(t *testing.T) {
	r := &siteToSiteVPNResource{}
	validators := r.ConfigValidators(context.Background())
	if len(validators) != 1 {
		t.Fatalf("expected one ConfigValidator, got %d", len(validators))
	}
	if _, ok := validators[0].(*siteToSiteVPNRemoteSubnetsValidator); !ok {
		t.Errorf("unexpected ConfigValidator type %T", validators[0])
	}
}

func TestSiteToSiteVPNDynamicRoutingRoundTrip(t *testing.T) {
	ctx := context.Background()
	r := &siteToSiteVPNResource{}

	model := &siteToSiteVPNResourceModel{
		Name:           types.StringValue("dynamic-vpn"),
		DynamicRouting: types.BoolValue(true),
		RemoteSubnets:  types.ListValueMust(types.StringType, nil),
	}

	network, diags := r.modelToNetwork(ctx, model)
	if diags.HasError() {
		t.Fatalf("modelToNetwork: %v", diags)
	}
	if !network.IPSecDynamicRouting {
		t.Error("IPSecDynamicRouting = false, want true")
	}
	if len(network.RemoteVPNSubnets) != 0 {
		t.Errorf("RemoteVPNSubnets = %v, want empty", network.RemoteVPNSubnets)
	}

	out := &siteToSiteVPNResourceModel{}
	if diags := r.networkToModel(ctx, network, out, "default"); diags.HasError() {
		t.Fatalf("networkToModel: %v", diags)
	}
	if !out.DynamicRouting.ValueBool() {
		t.Error("DynamicRouting = false, want true")
	}
	if len(out.RemoteSubnets.Elements()) != 0 {
		t.Errorf("RemoteSubnets = %v, want empty", out.RemoteSubnets.Elements())
	}
}

func TestSiteToSiteVPNRemoteSubnetsValid(t *testing.T) {
	empty := types.ListValueMust(types.StringType, nil)
	nonEmpty := types.ListValueMust(
		types.StringType,
		[]attr.Value{types.StringValue("192.0.2.0/24")},
	)

	tests := []struct {
		name           string
		dynamicRouting types.Bool
		remoteSubnets  types.List
		want           bool
	}{
		{"static tunnel with subnet", types.BoolValue(false), nonEmpty, true},
		{"dynamic tunnel with subnet", types.BoolValue(true), nonEmpty, true},
		{"dynamic tunnel without subnets", types.BoolValue(true), empty, true},
		{"static tunnel without subnets", types.BoolValue(false), empty, false},
		{"routing mode omitted", types.BoolNull(), empty, false},
		{"routing mode unknown", types.BoolUnknown(), empty, true},
		{"subnets unknown", types.BoolValue(false), types.ListUnknown(types.StringType), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := siteToSiteVPNRemoteSubnetsValid(
				tt.dynamicRouting,
				tt.remoteSubnets,
			); got != tt.want {
				t.Errorf("siteToSiteVPNRemoteSubnetsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_siteToSiteVPNResource_Configure(t *testing.T) {
	for _, tt := range []struct {
		name    string
		data    any
		wantErr bool
	}{
		{"nil", nil, false},
		{"wrong type", "wrong", true},
		{"correct", &Client{Site: "default"}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := &siteToSiteVPNResource{}
			resp := &fwresource.ConfigureResponse{}
			r.Configure(
				context.Background(),
				fwresource.ConfigureRequest{ProviderData: tt.data},
				resp,
			)
			if tt.wantErr && !resp.Diagnostics.HasError() {
				t.Error("expected error")
			}
			if !tt.wantErr && resp.Diagnostics.HasError() {
				t.Errorf("unexpected: %v", resp.Diagnostics)
			}
		})
	}
}

func Test_siteToSiteVPNResource_siteOrDefault(t *testing.T) {
	t.Run("non-empty site is returned as-is", func(t *testing.T) {
		r := &siteToSiteVPNResource{client: &Client{Site: "fallback"}}
		got := r.siteOrDefault(types.StringValue("custom"))
		if got != "custom" {
			t.Errorf("got %q, want %q", got, "custom")
		}
	})
	t.Run("empty site falls back to client site", func(t *testing.T) {
		r := &siteToSiteVPNResource{client: &Client{Site: "default"}}
		got := r.siteOrDefault(types.StringValue(""))
		if got != "default" {
			t.Errorf("got %q, want %q", got, "default")
		}
	})
}

func Test_siteToSiteVPNResource_modelToNetwork(t *testing.T) {
	ctx := context.Background()
	r := &siteToSiteVPNResource{}

	t.Run("basic fields are set", func(t *testing.T) {
		subnets, d := types.ListValueFrom(ctx, types.StringType, []string{"10.0.0.0/24"})
		if d.HasError() {
			t.Fatalf("building subnets: %v", d)
		}
		model := &siteToSiteVPNResourceModel{
			Name:          types.StringValue("test-vpn"),
			Enabled:       types.BoolValue(true),
			Interface:     types.StringValue("wan"),
			PeerIP:        iptypes.NewIPv4AddressValue("1.2.3.4"),
			KeyExchange:   types.StringValue("ikev2"),
			PreSharedKey:  types.StringValue("psk"),
			RemoteSubnets: subnets,
			PFS:           types.BoolValue(true),
		}
		network, diags := r.modelToNetwork(ctx, model)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if network.Purpose != unifi.PurposeSiteVPN {
			t.Errorf("Purpose = %q, want site-vpn", network.Purpose)
		}
		if network.VPNType == nil || *network.VPNType != "ipsec-vpn" {
			t.Errorf("VPNType = %v, want ipsec-vpn", network.VPNType)
		}
		if !network.Enabled {
			t.Error("Enabled should be true")
		}
		if !network.IPSecPfs {
			t.Error("IPSecPfs should be true")
		}
		if len(network.RemoteVPNSubnets) != 1 {
			t.Errorf("RemoteVPNSubnets length = %d, want 1", len(network.RemoteVPNSubnets))
		}
	})

	t.Run("null optional fields produce nil pointers", func(t *testing.T) {
		subnets, _ := types.ListValueFrom(ctx, types.StringType, []string{"10.0.0.0/24"})
		model := &siteToSiteVPNResourceModel{
			Name:          types.StringValue("vpn"),
			Interface:     types.StringNull(),
			PeerIP:        iptypes.NewIPv4AddressNull(),
			IKEEncryption: types.StringNull(),
			RemoteSubnets: subnets,
		}
		network, diags := r.modelToNetwork(ctx, model)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if network.IPSecInterface != nil {
			t.Errorf("IPSecInterface should be nil for null input, got %v", network.IPSecInterface)
		}
		if network.IPSecEncryption != nil {
			t.Errorf(
				"IPSecEncryption should be nil for null input, got %v",
				network.IPSecEncryption,
			)
		}
	})
}

func Test_siteToSiteVPNResource_networkToModel(t *testing.T) {
	ctx := context.Background()
	r := &siteToSiteVPNResource{}

	t.Run("basic fields are populated", func(t *testing.T) {
		name := "my-vpn"
		iface := "wan"
		network := &unifi.Network{
			ID:             "net-42",
			Name:           &name,
			Purpose:        unifi.PurposeSiteVPN,
			Enabled:        true,
			IPSecInterface: &iface,
			RemoteVPNSubnets: []string{
				"192.168.10.0/24",
			},
		}
		model := &siteToSiteVPNResourceModel{}
		diags := r.networkToModel(ctx, network, model, "site1")
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if model.ID.ValueString() != "net-42" {
			t.Errorf("ID = %q, want net-42", model.ID.ValueString())
		}
		if model.Site.ValueString() != "site1" {
			t.Errorf("Site = %q, want site1", model.Site.ValueString())
		}
		if model.Name.ValueString() != "my-vpn" {
			t.Errorf("Name = %q, want my-vpn", model.Name.ValueString())
		}
		if !model.Enabled.ValueBool() {
			t.Error("Enabled should be true")
		}
		if l := len(model.RemoteSubnets.Elements()); l != 1 {
			t.Errorf("RemoteSubnets length = %d, want 1", l)
		}
	})

	t.Run("nil pointer fields produce null values", func(t *testing.T) {
		network := &unifi.Network{
			ID:              "net-99",
			IPSecInterface:  nil,
			IPSecEncryption: nil,
		}
		model := &siteToSiteVPNResourceModel{}
		diags := r.networkToModel(ctx, network, model, "default")
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if !model.Interface.IsNull() {
			t.Errorf(
				"Interface should be null for nil pointer, got %q",
				model.Interface.ValueString(),
			)
		}
		if !model.IKEEncryption.IsNull() {
			t.Errorf(
				"IKEEncryption should be null for nil pointer, got %q",
				model.IKEEncryption.ValueString(),
			)
		}
	})
}

func Test_optStr(t *testing.T) {
	t.Run("null returns nil", func(t *testing.T) {
		if got := optStr(types.StringNull()); got != nil {
			t.Errorf("optStr(null) = %v, want nil", got)
		}
	})
	t.Run("unknown returns nil", func(t *testing.T) {
		if got := optStr(types.StringUnknown()); got != nil {
			t.Errorf("optStr(unknown) = %v, want nil", got)
		}
	})
	t.Run("empty string returns nil", func(t *testing.T) {
		if got := optStr(types.StringValue("")); got != nil {
			t.Errorf("optStr(\"\") = %v, want nil", got)
		}
	})
	t.Run("non-empty string returns pointer", func(t *testing.T) {
		got := optStr(types.StringValue("ikev2"))
		if got == nil {
			t.Fatal("optStr(\"ikev2\") returned nil")
		}
		if *got != "ikev2" {
			t.Errorf("*got = %q, want ikev2", *got)
		}
	})
}

func Test_optInt64(t *testing.T) {
	t.Run("null returns nil", func(t *testing.T) {
		if got := optInt64(types.Int64Null()); got != nil {
			t.Errorf("optInt64(null) = %v, want nil", got)
		}
	})
	t.Run("unknown returns nil", func(t *testing.T) {
		if got := optInt64(types.Int64Unknown()); got != nil {
			t.Errorf("optInt64(unknown) = %v, want nil", got)
		}
	})
	t.Run("zero returns nil", func(t *testing.T) {
		if got := optInt64(types.Int64Value(0)); got != nil {
			t.Errorf("optInt64(0) = %v, want nil", got)
		}
	})
	t.Run("non-zero returns pointer", func(t *testing.T) {
		got := optInt64(types.Int64Value(14))
		if got == nil {
			t.Fatal("optInt64(14) returned nil")
		}
		if *got != 14 {
			t.Errorf("*got = %d, want 14", *got)
		}
	})
}

func Test_stringPtrOrNull(t *testing.T) {
	t.Run("nil pointer returns null", func(t *testing.T) {
		got := stringPtrOrNull(nil)
		if !got.IsNull() {
			t.Errorf("stringPtrOrNull(nil) = %q, want null", got.ValueString())
		}
	})
	t.Run("empty string pointer returns null", func(t *testing.T) {
		s := ""
		got := stringPtrOrNull(&s)
		if !got.IsNull() {
			t.Errorf("stringPtrOrNull(\"\") = %q, want null", got.ValueString())
		}
	})
	t.Run("non-empty pointer returns value", func(t *testing.T) {
		s := "wan"
		got := stringPtrOrNull(&s)
		if got.IsNull() {
			t.Fatal("stringPtrOrNull(\"wan\") returned null")
		}
		if got.ValueString() != "wan" {
			t.Errorf("got %q, want wan", got.ValueString())
		}
	})
}

func Test_siteToSiteVPNResource_ListResourceConfigSchema(t *testing.T) {
	r := &siteToSiteVPNResource{}
	resp := &fwlist.ListResourceSchemaResponse{}
	r.ListResourceConfigSchema(context.Background(), fwlist.ListResourceSchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("ListResourceConfigSchema() produced errors: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["site"]; !ok {
		t.Error("ListResourceConfigSchema missing 'site' attribute")
	}
}
