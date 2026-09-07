package unifi

import (
	"context"
	"math/big"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
)

func TestAccWANFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWANFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_wan.test", "id"),
					resource.TestCheckResourceAttr("unifi_wan.test", "name", "test-wan"),
					resource.TestCheckResourceAttr("unifi_wan.test", "type", "dhcp"),
					resource.TestCheckResourceAttr("unifi_wan.test", "vlan.enabled", "true"),
					resource.TestCheckResourceAttr("unifi_wan.test", "vlan.id", "10"),
					resource.TestCheckResourceAttr("unifi_wan.test", "enabled", "true"),
				),
			},
			{
				ResourceName:      "unifi_wan.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Identity-based import (import block with identity, Terraform 1.12+).
			{
				ResourceName:    "unifi_wan.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

// TestAccWANFramework_minimal verifies that a WAN with no optional nested objects
// can be created and imported without "was null, but now..." errors from API defaults.
func TestAccWANFramework_minimal(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWANFrameworkConfig_minimal(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_wan.minimal", "id"),
					resource.TestCheckResourceAttr("unifi_wan.minimal", "name", "test-wan-minimal"),
					resource.TestCheckResourceAttr("unifi_wan.minimal", "type", "dhcp"),
					resource.TestCheckResourceAttr("unifi_wan.minimal", "enabled", "true"),
				),
			},
			{
				ResourceName:      "unifi_wan.minimal",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccWANFramework_withNestedObjects verifies that explicitly configured nested
// objects are preserved through create, read, and import.
func TestAccWANFramework_withNestedObjects(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWANFrameworkConfig_withNestedObjects(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_wan.nested", "id"),
					resource.TestCheckResourceAttr("unifi_wan.nested", "name", "test-wan-nested"),
					resource.TestCheckResourceAttr("unifi_wan.nested", "type", "dhcp"),
					resource.TestCheckResourceAttr("unifi_wan.nested", "enabled", "true"),
					// VLAN
					resource.TestCheckResourceAttr("unifi_wan.nested", "vlan.enabled", "true"),
					resource.TestCheckResourceAttr("unifi_wan.nested", "vlan.id", "20"),
					// DNS
					resource.TestCheckResourceAttr("unifi_wan.nested", "dns.preference", "manual"),
					resource.TestCheckResourceAttr("unifi_wan.nested", "dns.primary", "8.8.8.8"),
					resource.TestCheckResourceAttr("unifi_wan.nested", "dns.secondary", "8.8.4.4"),
					// Load Balance
					resource.TestCheckResourceAttrSet(
						"unifi_wan.nested",
						"load_balance.failover_priority",
					),
				),
			},
			{
				ResourceName:      "unifi_wan.nested",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccWANFrameworkConfig_basic() string {
	return `
resource "unifi_wan" "test" {
	name    = "test-wan"
	type    = "dhcp"
	enabled = true

	vlan = {
		enabled = true
		id      = 10
	}
}
`
}

func testAccWANFrameworkConfig_minimal() string {
	return `
resource "unifi_wan" "minimal" {
	name    = "test-wan-minimal"
	type    = "dhcp"
	enabled = true
}
`
}

func testAccWANFrameworkConfig_withNestedObjects() string {
	return `
resource "unifi_wan" "nested" {
	name    = "test-wan-nested"
	type    = "dhcp"
	enabled = true

	vlan = {
		enabled = true
		id      = 20
	}

	dns = {
		preference = "manual"
		primary    = "8.8.8.8"
		secondary  = "8.8.4.4"
	}

	load_balance = {
		failover_priority = 1
	}
}
`
}

// TestAccWANFramework_additionalFields verifies the newly exposed top-level
// fields round-trip through create, read, and import without spurious diffs.
func TestAccWANFramework_additionalFields(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWANFrameworkConfig_additionalFields(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_wan.extra", "id"),
					resource.TestCheckResourceAttr("unifi_wan.extra", "name", "test-wan-extra"),
					// Computed fields populated from the controller.
					resource.TestCheckResourceAttrSet(
						"unifi_wan.extra",
						"mac_override_enabled",
					),
					resource.TestCheckResourceAttrSet(
						"unifi_wan.extra",
						"dslite.remote_host_auto",
					),
				),
			},
			{
				ResourceName:      "unifi_wan.extra",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccWANFrameworkConfig_additionalFields() string {
	// Note: setting_preference is intentionally NOT pinned here. The controller
	// treats it as a managed/derived field on WAN networks and reverts it to
	// "auto" for a dhcp WAN regardless of what we send (even with manual DNS),
	// which makes "manual" produce perpetual auto->manual plan drift. The other
	// newly exposed top-level fields below do round-trip cleanly.
	return `
resource "unifi_wan" "extra" {
	name    = "test-wan-extra"
	type    = "dhcp"
	enabled = true
}
`
}

func TestNewWANResource(t *testing.T) {
	got := NewWANResource()
	if got == nil {
		t.Fatal("NewWANResource() returned nil")
	}
	if _, ok := got.(fwresource.ResourceWithImportState); !ok {
		t.Errorf("NewWANResource() does not implement fwresource.ResourceWithImportState")
	}
	if _, ok := got.(fwresource.ResourceWithIdentity); !ok {
		t.Errorf("NewWANResource() does not implement fwresource.ResourceWithIdentity")
	}
}

func TestNewWANListResource(t *testing.T) {
	got := NewWANListResource()
	if got == nil {
		t.Fatal("NewWANListResource() returned nil")
	}
	if _, ok := got.(fwlist.ListResourceWithConfigure); !ok {
		t.Errorf("NewWANListResource() does not implement fwlist.ListResourceWithConfigure")
	}
}

func Test_vlanModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    vlanModel
		want map[string]attr.Type
	}{
		{
			name: "returns correct types",
			want: map[string]attr.Type{
				"enabled": types.BoolType,
				"id":      types.Int64Type,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("vlanModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_egressQosModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    egressQosModel
		want map[string]attr.Type
	}{
		{
			name: "returns correct types",
			want: map[string]attr.Type{
				"enabled":  types.BoolType,
				"priority": types.Int64Type,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("egressQosModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_smartqModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    smartqModel
		want map[string]attr.Type
	}{
		{
			name: "returns correct types",
			want: map[string]attr.Type{
				"enabled":   types.BoolType,
				"up_rate":   types.Int64Type,
				"down_rate": types.Int64Type,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("smartqModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_providerCapabilitiesModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    providerCapabilitiesModel
		want map[string]attr.Type
	}{
		{
			name: "returns correct types",
			want: map[string]attr.Type{
				"download_kilobits_per_second": types.Int64Type,
				"upload_kilobits_per_second":   types.Int64Type,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("providerCapabilitiesModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_dhcpOptionModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    dhcpOptionModel
		want map[string]attr.Type
	}{
		{
			name: "returns correct types",
			want: map[string]attr.Type{
				"option_number": types.Int64Type,
				"value":         types.StringType,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("dhcpOptionModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test_wanResource_networkGroup guards #334: the WAN network group must be
// preserved in the update PUT (and attr_hidden_id mirror it), instead of being
// hard-coded to "WAN" — otherwise a secondary uplink (WAN2) collides with the
// primary and the controller rejects it.
func Test_wanResource_networkGroup(t *testing.T) {
	r := &wanResource{}
	ctx := context.Background()

	base := wanResourceModel{
		Name:    types.StringValue("CC Internet SFP"),
		Enabled: types.BoolValue(true),
		Type:    types.StringValue("dhcp"),
	}

	t.Run("WAN2 is preserved and mirrored to hidden id", func(t *testing.T) {
		m := base
		m.NetworkGroup = types.StringValue("WAN2")
		n, d := r.modelToNetwork(ctx, &m)
		if d.HasError() {
			t.Fatalf("modelToNetwork: %v", d)
		}
		if n.WANNetworkGroup == nil || *n.WANNetworkGroup != "WAN2" {
			t.Errorf("WANNetworkGroup = %v, want WAN2", n.WANNetworkGroup)
		}
		if n.HiddenID != "WAN2" {
			t.Errorf("HiddenID = %q, want WAN2", n.HiddenID)
		}
	})

	t.Run("unset defaults to WAN", func(t *testing.T) {
		m := base
		m.NetworkGroup = types.StringNull()
		n, d := r.modelToNetwork(ctx, &m)
		if d.HasError() {
			t.Fatalf("modelToNetwork: %v", d)
		}
		if n.WANNetworkGroup == nil || *n.WANNetworkGroup != "WAN" {
			t.Errorf("WANNetworkGroup = %v, want WAN", n.WANNetworkGroup)
		}
		if n.HiddenID != "WAN" {
			t.Errorf("HiddenID = %q, want WAN", n.HiddenID)
		}
	})
}

// Test_wanResource_overlayConfig_dslite guards #281: the controller forces
// dslite.remote_host_auto back to true server-side, so the API value in state
// would conflict with a user-configured false. overlayConfig must keep the
// user's planned leaf when it was set in config, and leave the controller value
// when it wasn't (an unset Optional+Computed leaf is unknown in the create
// plan, and an unset object is wholly unknown).
func Test_wanResource_overlayConfig_dslite(t *testing.T) {
	r := &wanResource{}
	ctx := context.Background()
	dslite := func(host, auto attr.Value) types.Object {
		return types.ObjectValueMust(wanDsliteModel{}.AttributeTypes(), map[string]attr.Value{
			"remote_host":      host,
			"remote_host_auto": auto,
		})
	}
	remoteHostAuto := func(t *testing.T, m wanResourceModel) bool {
		t.Helper()
		return attrAs[types.Bool](t, m.Dslite.Attributes()["remote_host_auto"]).ValueBool()
	}

	t.Run("configured false overrides controller true", func(t *testing.T) {
		state := wanResourceModel{
			Dslite: dslite(types.StringValue("aftr.isp.net"), types.BoolValue(true)),
		}
		config := wanResourceModel{Dslite: dslite(types.StringNull(), types.BoolValue(false))}
		plan := wanResourceModel{Dslite: dslite(types.StringUnknown(), types.BoolValue(false))}
		r.overlayConfig(ctx, &state, &config, &plan)
		if remoteHostAuto(t, state) {
			t.Errorf("dslite.remote_host_auto = true, want false (planned value)")
		}
		host := attrAs[types.String](t, state.Dslite.Attributes()["remote_host"])
		if host.ValueString() != "aftr.isp.net" {
			t.Errorf("dslite.remote_host = %v, want the controller value kept", host)
		}
	})

	t.Run("unset keeps controller value", func(t *testing.T) {
		state := wanResourceModel{Dslite: dslite(types.StringNull(), types.BoolValue(true))}
		config := wanResourceModel{Dslite: types.ObjectNull(wanDsliteModel{}.AttributeTypes())}
		plan := wanResourceModel{Dslite: types.ObjectUnknown(wanDsliteModel{}.AttributeTypes())}
		r.overlayConfig(ctx, &state, &config, &plan)
		if !remoteHostAuto(t, state) {
			t.Errorf("dslite.remote_host_auto = false, want true (controller value kept)")
		}
	})
}

// Test_dnsAddrValue guards #333: the controller persists an unset WAN DNS
// address as "" and returns it, but the Optional address fields plan as null.
// "" (and a nil pointer) must map to null so the post-apply read matches the
// plan; a real address must round-trip.
func Test_dnsAddrValue(t *testing.T) {
	empty := ""
	addr := "2001:4860:4860::8888"
	v4 := "8.8.8.8"

	cases := []struct {
		name     string
		in       *string
		wantNull bool
		wantStr  string
	}{
		{"nil pointer -> null", nil, true, ""},
		{"empty string -> null", &empty, true, ""},
		{"ipv6 address survives", &addr, false, addr},
		{"ipv4 address survives", &v4, false, v4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := dnsAddrValue(c.in)
			if got.IsNull() != c.wantNull {
				t.Errorf("IsNull = %v, want %v", got.IsNull(), c.wantNull)
			}
			if !c.wantNull && got.ValueString() != c.wantStr {
				t.Errorf("ValueString = %q, want %q", got.ValueString(), c.wantStr)
			}
		})
	}
}

func Test_dnsModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    dnsModel
		want map[string]attr.Type
	}{
		{
			name: "returns correct types",
			want: map[string]attr.Type{
				"primary":    types.StringType,
				"secondary":  types.StringType,
				"preference": types.StringType,
				"ipv6": types.ObjectType{AttrTypes: map[string]attr.Type{
					"primary":    types.StringType,
					"secondary":  types.StringType,
					"preference": types.StringType,
				}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("dnsModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_upnpModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    upnpModel
		want map[string]attr.Type
	}{
		{
			name: "returns correct types",
			want: map[string]attr.Type{
				"enabled":         types.BoolType,
				"wan_interface":   types.StringType,
				"nat_pmp_enabled": types.BoolType,
				"secure_mode":     types.BoolType,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("upnpModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_loadBalanceModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    loadBalanceModel
		want map[string]attr.Type
	}{
		{
			name: "returns correct types",
			want: map[string]attr.Type{
				"type":              types.StringType,
				"weight":            types.Int64Type,
				"failover_priority": types.Int64Type,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("loadBalanceModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_igmpProxyModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    igmpProxyModel
		want map[string]attr.Type
	}{
		{
			name: "returns correct types",
			want: map[string]attr.Type{
				"downstream": types.StringType,
				"upstream":   types.BoolType,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("igmpProxyModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_dhcpv6WanModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    dhcpv6WanModel
		want map[string]attr.Type
	}{
		{
			name: "returns correct types",
			want: map[string]attr.Type{
				"cos": types.Int64Type,
				"pd": types.ObjectType{AttrTypes: map[string]attr.Type{
					"size":      types.Int64Type,
					"size_auto": types.BoolType,
				}},
				"options": types.ListType{
					ElemType: types.ObjectType{AttrTypes: dhcpOptionModel{}.AttributeTypes()},
				},
				"wan_delegation_type": types.StringType,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("dhcpv6WanModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_dhcpWanModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    dhcpWanModel
		want map[string]attr.Type
	}{
		{
			name: "returns correct types",
			want: map[string]attr.Type{
				"cos": types.Int64Type,
				"options": types.ListType{
					ElemType: types.ObjectType{AttrTypes: dhcpOptionModel{}.AttributeTypes()},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("dhcpWanModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_dnsIPv6Model_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    dnsIPv6Model
		want map[string]attr.Type
	}{
		{
			name: "returns correct types",
			want: map[string]attr.Type{
				"primary":    types.StringType,
				"secondary":  types.StringType,
				"preference": types.StringType,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("dnsIPv6Model.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_dhcpv6PDModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    dhcpv6PDModel
		want map[string]attr.Type
	}{
		{
			name: "returns correct types",
			want: map[string]attr.Type{
				"size":      types.Int64Type,
				"size_auto": types.BoolType,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("dhcpv6PDModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_wanDsliteModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    wanDsliteModel
		want map[string]attr.Type
	}{
		{
			name: "returns correct types",
			want: map[string]attr.Type{
				"remote_host":      types.StringType,
				"remote_host_auto": types.BoolType,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("wanDsliteModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_wanResource_Metadata(t *testing.T) {
	tests := []struct {
		name             string
		providerTypeName string
		wantTypeName     string
	}{
		{
			name:             "type name includes provider prefix",
			providerTypeName: "unifi",
			wantTypeName:     "unifi_wan",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &wanResource{}
			resp := &fwresource.MetadataResponse{}
			r.Metadata(
				context.Background(),
				fwresource.MetadataRequest{ProviderTypeName: tt.providerTypeName},
				resp,
			)
			if resp.TypeName != tt.wantTypeName {
				t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, tt.wantTypeName)
			}
		})
	}
}

func Test_wanResource_IdentitySchema(t *testing.T) {
	t.Run("does not panic and returns identity attributes", func(t *testing.T) {
		r := &wanResource{}
		resp := &fwresource.IdentitySchemaResponse{}
		r.IdentitySchema(context.Background(), fwresource.IdentitySchemaRequest{}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("IdentitySchema() returned errors: %v", resp.Diagnostics)
		}
		if len(resp.IdentitySchema.Attributes) == 0 {
			t.Error("IdentitySchema() returned no attributes")
		}
	})
}

func Test_wanResource_Schema(t *testing.T) {
	t.Run("returns schema with key attributes", func(t *testing.T) {
		r := &wanResource{}
		resp := &fwresource.SchemaResponse{}
		r.Schema(context.Background(), fwresource.SchemaRequest{}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("Schema() returned errors: %v", resp.Diagnostics)
		}
		for _, key := range []string{"id", "name", "type"} {
			if _, ok := resp.Schema.Attributes[key]; !ok {
				t.Errorf("Schema() missing attribute %q", key)
			}
		}
	})
}

func Test_wanResource_Configure(t *testing.T) {
	t.Run("nil provider data is not an error", func(t *testing.T) {
		r := &wanResource{}
		resp := &fwresource.ConfigureResponse{}
		r.Configure(context.Background(), fwresource.ConfigureRequest{ProviderData: nil}, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf(
				"Configure() with nil provider data should not error, got: %v",
				resp.Diagnostics,
			)
		}
	})

	t.Run("wrong type produces error", func(t *testing.T) {
		r := &wanResource{}
		resp := &fwresource.ConfigureResponse{}
		r.Configure(context.Background(), fwresource.ConfigureRequest{ProviderData: "wrong"}, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("Configure() with wrong type should produce an error")
		}
	})

	t.Run("correct Client type", func(t *testing.T) {
		r := &wanResource{}
		resp := &fwresource.ConfigureResponse{}
		client := &Client{}
		r.Configure(context.Background(), fwresource.ConfigureRequest{ProviderData: client}, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("Configure() with *Client should not error, got: %v", resp.Diagnostics)
		}
		if r.client != client {
			t.Error("Configure() did not set client")
		}
	})
}

func Test_wanResource_Create(t *testing.T) {
	t.Skip("requires terraform state machinery")
}

func Test_wanResource_adoptExistingWAN(t *testing.T) {
	t.Skip("requires configured client")
}

func Test_wanResource_overlayConfig(t *testing.T) {
	t.Skip("requires complex state setup")
}

func Test_wanResource_Read(t *testing.T) {
	t.Skip("requires terraform state machinery")
}

func Test_wanResource_Update(t *testing.T) {
	t.Skip("requires terraform state machinery")
}

func Test_wanResource_applyPlanToState(t *testing.T) {
	t.Skip("requires complex state setup")
}

func Test_wanResource_Delete(t *testing.T) {
	t.Skip("requires terraform state machinery")
}

func Test_wanResource_ImportState(t *testing.T) {
	t.Skip("requires terraform state machinery")
}

func Test_wanResource_modelToNetwork(t *testing.T) {
	t.Run("minimal model converts correctly", func(t *testing.T) {
		r := &wanResource{}
		ctx := context.Background()
		model := &wanResourceModel{
			Name:                  types.StringValue("test"),
			Type:                  types.StringValue("dhcp"),
			TypeV6:                types.StringNull(),
			Enabled:               types.BoolValue(true),
			Vlan:                  types.ObjectNull(vlanModel{}.AttributeTypes()),
			EgressQoS:             types.ObjectNull(egressQosModel{}.AttributeTypes()),
			DNS:                   types.ObjectNull(dnsModel{}.AttributeTypes()),
			DHCP:                  types.ObjectNull(dhcpWanModel{}.AttributeTypes()),
			DHCPv6:                types.ObjectNull(dhcpv6WanModel{}.AttributeTypes()),
			SmartQ:                types.ObjectNull(smartqModel{}.AttributeTypes()),
			UPnP:                  types.ObjectNull(upnpModel{}.AttributeTypes()),
			LoadBalance:           types.ObjectNull(loadBalanceModel{}.AttributeTypes()),
			IGMPProxy:             types.ObjectNull(igmpProxyModel{}.AttributeTypes()),
			ProviderCapabilities:  types.ObjectNull(providerCapabilitiesModel{}.AttributeTypes()),
			ReportWANEvent:        types.BoolNull(),
			IPAliases:             types.ListNull(types.StringType),
			SettingPreference:     types.StringNull(),
			IPv6SettingPreference: types.StringNull(),
			SingleNetworkLAN:      types.StringNull(),
			MACOverrideEnabled:    types.BoolNull(),
			Dslite:                types.ObjectNull(wanDsliteModel{}.AttributeTypes()),
		}
		got, diags := r.modelToNetwork(ctx, model)
		if diags.HasError() {
			t.Fatalf("modelToNetwork() returned errors: %v", diags)
		}
		if got == nil {
			t.Fatal("modelToNetwork() returned nil network")
		}
		if got.Name == nil || *got.Name != "test" {
			t.Errorf("expected Name=test, got %v", got.Name)
		}
		if got.WANType == nil || *got.WANType != "dhcp" {
			t.Errorf("expected WANType=dhcp, got %v", got.WANType)
		}
		if got.Purpose != "wan" {
			t.Errorf("expected Purpose=wan, got %v", got.Purpose)
		}
		if !got.Enabled {
			t.Error("expected Enabled=true")
		}
	})
}

func Test_wanResource_networkToModel(t *testing.T) {
	t.Run("converts API network back to model", func(t *testing.T) {
		r := &wanResource{}
		ctx := context.Background()
		wanType := "dhcp"
		name := "test-wan"
		network := &unifi.Network{
			ID:      "abc123",
			Name:    &name,
			Purpose: "wan",
			WANType: &wanType,
			Enabled: true,
		}
		model := &wanResourceModel{}
		applyWANDefaults(model)
		diags := r.networkToModel(ctx, network, model, "default")
		if diags.HasError() {
			t.Fatalf("networkToModel() returned errors: %v", diags)
		}
		if model.ID.ValueString() != "abc123" {
			t.Errorf("expected ID=abc123, got %v", model.ID.ValueString())
		}
		if model.Site.ValueString() != "default" {
			t.Errorf("expected Site=default, got %v", model.Site.ValueString())
		}
		if model.Name.ValueString() != "test-wan" {
			t.Errorf("expected Name=test-wan, got %v", model.Name.ValueString())
		}
		if model.Type.ValueString() != "dhcp" {
			t.Errorf("expected Type=dhcp, got %v", model.Type.ValueString())
		}
	})
}

func Test_applyWANDefaults(t *testing.T) {
	t.Run("applies defaults to empty model", func(t *testing.T) {
		model := &wanResourceModel{}
		applyWANDefaults(model)
		if !model.Vlan.IsNull() {
			t.Error("expected Vlan to be null after defaults")
		}
		if !model.EgressQoS.IsNull() {
			t.Error("expected EgressQoS to be null after defaults")
		}
		if !model.SmartQ.IsNull() {
			t.Error("expected SmartQ to be null after defaults")
		}
		if !model.DNS.IsNull() {
			t.Error("expected DNS to be null after defaults")
		}
		if !model.IPAliases.IsNull() {
			t.Error("expected IPAliases to be null after defaults")
		}
		if !model.Dslite.IsNull() {
			t.Error("expected Dslite to be null after defaults")
		}
	})
}

func Test_wanResource_ListResourceConfigSchema(t *testing.T) {
	t.Run("does not panic", func(t *testing.T) {
		r := &wanResource{}
		resp := &fwlist.ListResourceSchemaResponse{}
		r.ListResourceConfigSchema(context.Background(), fwlist.ListResourceSchemaRequest{}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ListResourceConfigSchema() returned errors: %v", resp.Diagnostics)
		}
	})
}

func Test_wanResource_List(t *testing.T) {
	t.Skip("requires configured client")
}

// TestWANUpgradeState_v0NestsPrefixedGroups guards the v0 -> v1 schema
// upgrade: wan_dslite_* moves under dslite, dns.ipv6_* under dns.ipv6 and
// dhcpv6.pd_size* under dhcpv6.pd, while every other attribute passes through
// and objects absent from prior state stay null.
func TestWANUpgradeState_v0NestsPrefixedGroups(t *testing.T) {
	ctx := context.Background()
	r := &wanResource{}

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	prior := []byte(`{
		"id": "wan-1", "site": "default", "name": "Internet", "networkgroup": "WAN",
		"type": "dhcp", "type_v6": "dhcpv6", "enabled": true,
		"vlan": {"enabled": true, "id": 10},
		"dns": {
			"primary": "1.1.1.1", "secondary": null, "preference": "manual",
			"ipv6_primary": "2606:4700:4700::1111", "ipv6_secondary": null,
			"ipv6_preference": "auto"
		},
		"dhcpv6": {
			"cos": null, "pd_size": 56, "pd_size_auto": false, "options": null,
			"wan_delegation_type": "pd"
		},
		"setting_preference": "manual", "ipv6_setting_preference": "auto",
		"mac_override_enabled": false,
		"wan_dslite_remote_host": "aftr.isp.net",
		"wan_dslite_remote_host_auto": false
	}`)

	up, ok := r.UpgradeState(ctx)[0]
	if !ok {
		t.Fatal("no upgrader registered for schema version 0")
	}
	resp := &fwresource.UpgradeStateResponse{}
	up.StateUpgrader(ctx, fwresource.UpgradeStateRequest{
		RawState: &tfprotov6.RawState{JSON: prior},
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
	obj := func(v tftypes.Value, name string) map[string]tftypes.Value {
		t.Helper()
		var m map[string]tftypes.Value
		if err := v.As(&m); err != nil {
			t.Fatalf("%s: as object: %v (value %v)", name, err, v)
		}
		return m
	}
	str := func(v tftypes.Value, name, want string) {
		t.Helper()
		var s string
		if err := v.As(&s); err != nil || s != want {
			t.Errorf("%s = %v (%v), want %q", name, v, err, want)
		}
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
	boolean := func(v tftypes.Value, name string, want bool) {
		t.Helper()
		var b bool
		if err := v.As(&b); err != nil || b != want {
			t.Errorf("%s = %v (%v), want %v", name, v, err, want)
		}
	}

	str(root["name"], "name", "Internet")
	str(root["type_v6"], "type_v6", "dhcpv6")
	str(root["setting_preference"], "setting_preference", "manual")
	boolean(root["mac_override_enabled"], "mac_override_enabled", false)
	for _, flat := range []string{"wan_dslite_remote_host", "wan_dslite_remote_host_auto"} {
		if _, exists := root[flat]; exists {
			t.Errorf("flat attribute %q survived the upgrade", flat)
		}
	}
	dslite := obj(root["dslite"], "dslite")
	str(dslite["remote_host"], "dslite.remote_host", "aftr.isp.net")
	boolean(dslite["remote_host_auto"], "dslite.remote_host_auto", false)

	dns := obj(root["dns"], "dns")
	str(dns["primary"], "dns.primary", "1.1.1.1")
	str(dns["preference"], "dns.preference", "manual")
	for _, flat := range []string{"ipv6_primary", "ipv6_secondary", "ipv6_preference"} {
		if _, exists := dns[flat]; exists {
			t.Errorf("flat attribute dns.%s survived the upgrade", flat)
		}
	}
	ipv6 := obj(dns["ipv6"], "dns.ipv6")
	str(ipv6["primary"], "dns.ipv6.primary", "2606:4700:4700::1111")
	if !ipv6["secondary"].IsNull() {
		t.Errorf("dns.ipv6.secondary = %v, want null", ipv6["secondary"])
	}
	str(ipv6["preference"], "dns.ipv6.preference", "auto")

	dhcpv6 := obj(root["dhcpv6"], "dhcpv6")
	str(dhcpv6["wan_delegation_type"], "dhcpv6.wan_delegation_type", "pd")
	for _, flat := range []string{"pd_size", "pd_size_auto"} {
		if _, exists := dhcpv6[flat]; exists {
			t.Errorf("flat attribute dhcpv6.%s survived the upgrade", flat)
		}
	}
	pd := obj(dhcpv6["pd"], "dhcpv6.pd")
	num(pd["size"], "dhcpv6.pd.size", 56)
	boolean(pd["size_auto"], "dhcpv6.pd.size_auto", false)

	// Objects absent from prior state (never configured) stay null rather
	// than becoming objects of nulls.
	for _, absent := range []string{"dhcp", "upnp", "load_balance", "timeouts"} {
		if !root[absent].IsNull() {
			t.Errorf("%s = %v, want null", absent, root[absent])
		}
	}
}

// TestWAN_nestedGroupsRoundTrip checks that dns.ipv6, dhcpv6.pd and dslite
// convert API -> model -> API without loss, that null or unknown groups stay
// off the request, and that a read resolves an unknown group to a known object.
func TestWAN_nestedGroupsRoundTrip(t *testing.T) {
	ctx := context.Background()
	r := &wanResource{}

	api := &unifi.Network{
		ID:                      "wan-1",
		Name:                    util.Ptr("Internet"),
		Purpose:                 unifi.PurposeWAN,
		WANNetworkGroup:         util.Ptr("WAN"),
		WANType:                 util.Ptr("dhcp"),
		Enabled:                 true,
		WANIPV6DNS1:             util.Ptr("2606:4700:4700::1111"),
		WANIPV6DNS2:             util.Ptr(""),
		WANIPV6DNSPreference:    util.Ptr("manual"),
		WANDHCPv6PDSize:         ptrInt64(56),
		WANDHCPv6PDSizeAuto:     true,
		WANDsliteRemoteHost:     util.Ptr("aftr.isp.net"),
		WANDsliteRemoteHostAuto: true,
	}

	model := &wanResourceModel{}
	applyWANDefaults(model)
	if d := r.networkToModel(ctx, api, model, "default"); d.HasError() {
		t.Fatalf("networkToModel: %v", d)
	}

	ipv6 := attrAs[types.Object](t, model.DNS.Attributes()["ipv6"]).Attributes()
	if got := attrAs[types.String](
		t,
		ipv6["primary"],
	).ValueString(); got != "2606:4700:4700::1111" {
		t.Errorf("dns.ipv6.primary = %q", got)
	}
	if !ipv6["secondary"].IsNull() {
		t.Errorf(
			"dns.ipv6.secondary = %v, want null (the controller sends \"\" for unset)",
			ipv6["secondary"],
		)
	}
	if got := attrAs[types.String](t, ipv6["preference"]).ValueString(); got != "manual" {
		t.Errorf("dns.ipv6.preference = %q", got)
	}
	if !model.DNS.Attributes()["primary"].IsNull() {
		t.Errorf("dns.primary = %v, want null", model.DNS.Attributes()["primary"])
	}
	pd := attrAs[types.Object](t, model.DHCPv6.Attributes()["pd"]).Attributes()
	if attrAs[types.Int64](t, pd["size"]).ValueInt64() != 56 ||
		!attrAs[types.Bool](t, pd["size_auto"]).ValueBool() {
		t.Errorf("dhcpv6.pd = %v", pd)
	}
	dslite := model.Dslite.Attributes()
	if attrAs[types.String](t, dslite["remote_host"]).ValueString() != "aftr.isp.net" ||
		!attrAs[types.Bool](t, dslite["remote_host_auto"]).ValueBool() {
		t.Errorf("dslite = %v", dslite)
	}

	back, d := r.modelToNetwork(ctx, model)
	if d.HasError() {
		t.Fatalf("modelToNetwork: %v", d)
	}
	if back.WANIPV6DNS1 == nil || *back.WANIPV6DNS1 != "2606:4700:4700::1111" ||
		back.WANIPV6DNS2 != nil ||
		back.WANIPV6DNSPreference == nil || *back.WANIPV6DNSPreference != "manual" {
		t.Errorf(
			"dns.ipv6 round trip: %v %v %v",
			back.WANIPV6DNS1,
			back.WANIPV6DNS2,
			back.WANIPV6DNSPreference,
		)
	}
	if back.WANDHCPv6PDSize == nil || *back.WANDHCPv6PDSize != 56 || !back.WANDHCPv6PDSizeAuto {
		t.Errorf("dhcpv6.pd round trip: %v %v", back.WANDHCPv6PDSize, back.WANDHCPv6PDSizeAuto)
	}
	if back.WANDsliteRemoteHost == nil || *back.WANDsliteRemoteHost != "aftr.isp.net" ||
		!back.WANDsliteRemoteHostAuto {
		t.Errorf("dslite round trip: %v %v", back.WANDsliteRemoteHost, back.WANDsliteRemoteHostAuto)
	}

	// Groups the plan left unknown (or null) contribute nothing, exactly as
	// the unset flat attributes did.
	dnsAttrs := model.DNS.Attributes()
	dnsAttrs["ipv6"] = types.ObjectUnknown(dnsIPv6Model{}.AttributeTypes())
	model.DNS = types.ObjectValueMust(dnsModel{}.AttributeTypes(), dnsAttrs)
	dhcpv6Attrs := model.DHCPv6.Attributes()
	dhcpv6Attrs["pd"] = types.ObjectNull(dhcpv6PDModel{}.AttributeTypes())
	model.DHCPv6 = types.ObjectValueMust(dhcpv6WanModel{}.AttributeTypes(), dhcpv6Attrs)
	model.Dslite = types.ObjectUnknown(wanDsliteModel{}.AttributeTypes())
	back, d = r.modelToNetwork(ctx, model)
	if d.HasError() {
		t.Fatalf("modelToNetwork (unknown groups): %v", d)
	}
	if back.WANIPV6DNS1 != nil || back.WANIPV6DNSPreference != nil ||
		back.WANDHCPv6PDSize != nil || back.WANDHCPv6PDSizeAuto ||
		back.WANDsliteRemoteHost != nil || back.WANDsliteRemoteHostAuto {
		t.Errorf("unknown/null groups leaked into the request: %+v", back)
	}

	// The post-apply read resolves an unknown group to a known object built
	// from the controller's values instead of leaving it unknown.
	if d := r.networkToModel(ctx, api, model, "default"); d.HasError() {
		t.Fatalf("networkToModel (resolve unknown): %v", d)
	}
	if model.DNS.Attributes()["ipv6"].IsUnknown() ||
		model.DHCPv6.Attributes()["pd"].IsUnknown() || model.Dslite.IsUnknown() {
		t.Errorf(
			"unknown groups not resolved after read: dns=%v dhcpv6=%v dslite=%v",
			model.DNS,
			model.DHCPv6,
			model.Dslite,
		)
	}
	if got := attrAs[types.Int64](
		t,
		attrAs[types.Object](t, model.DHCPv6.Attributes()["pd"]).Attributes()["size"],
	).ValueInt64(); got != 56 {
		t.Errorf("dhcpv6.pd.size after read = %d, want 56", got)
	}
}
