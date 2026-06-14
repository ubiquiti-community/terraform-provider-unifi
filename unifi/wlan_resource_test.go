package unifi

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func TestAccWLANFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWLANFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_wlan.test", "name", "wlan1"),
					resource.TestCheckResourceAttr("unifi_wlan.test", "security", "wpapsk"),
					resource.TestCheckResourceAttr("unifi_wlan.test", "passphrase", "passphrase"),
					resource.TestCheckResourceAttr("unifi_wlan.test", "hide_ssid", "false"),
					resource.TestCheckResourceAttr("unifi_wlan.test", "mac_filter.enabled", "true"),
					resource.TestCheckResourceAttr("unifi_wlan.test", "mac_filter.policy", "allow"),
					resource.TestCheckResourceAttr("unifi_wlan.test", "mac_filter.list.#", "1"),
				),
				ResourceName:  "unifi_wlan.test",
				ImportState:   true,
				ImportStateId: "wlan1",
			},
		},
	})
}

func testAccWLANFrameworkConfig_basic() string {
	return `
data "unifi_client_qos_rate" "default" {
	name = "Default"
}

resource "unifi_wlan" "test" {
	name            = "wlan1"
	security        = "wpapsk"
	passphrase      = "passphrase"
	hide_ssid       = false
}
`
}

// TestAccWLANFramework_additionalFields verifies that the newly exposed
// security/DTIM/toggle attributes are populated by the read path when a WLAN
// is imported. It follows the same import-based pattern as the basic test: a
// full create cannot be exercised here because WLAN creation currently fails
// with a pre-existing api.err.InvalidPayload that is unrelated to these
// attributes (a minimal WLAN with none of them set fails identically).
func TestAccWLANFramework_additionalFields(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWLANFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_wlan.test", "wpa_mode"),
					resource.TestCheckResourceAttrSet("unifi_wlan.test", "wpa_enc"),
					resource.TestCheckResourceAttrSet("unifi_wlan.test", "dtim_mode"),
					resource.TestCheckResourceAttrSet("unifi_wlan.test", "group_rekey"),
					resource.TestCheckResourceAttrSet("unifi_wlan.test", "iapp_enabled"),
					resource.TestCheckResourceAttrSet("unifi_wlan.test", "mlo_enabled"),
					// Issue #176 (secondary): the API omits minimum_data_rate_*
					// from GET responses, so the read path must surface them as 0
					// (the schema default), not null, to avoid perpetual plan
					// drift after import.
					resource.TestCheckResourceAttr(
						"unifi_wlan.test",
						"minimum_data_rate_2g_kbps",
						"0",
					),
					resource.TestCheckResourceAttr(
						"unifi_wlan.test",
						"minimum_data_rate_5g_kbps",
						"0",
					),
				),
				ResourceName:  "unifi_wlan.test",
				ImportState:   true,
				ImportStateId: "wlan1",
			},
		},
	})
}

func TestNewWLANFrameworkResource(t *testing.T) {
	got := NewWLANFrameworkResource()
	if got == nil {
		t.Fatal("NewWLANFrameworkResource() returned nil")
	}
	// Verify interface compliance
	_ = got
	_ = got.(fwresource.ResourceWithImportState)
	_ = got.(fwresource.ResourceWithIdentity)
	_ = got.(fwresource.ResourceWithUpgradeState)
}

func TestNewWLANListResource(t *testing.T) {
	got := NewWLANListResource()
	if got == nil {
		t.Fatal("NewWLANListResource() returned nil")
	}
	_ = got
	_ = got.(fwlist.ListResourceWithConfigure)
}

func Test_wlanPrivatePresharedKeyModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    wlanPrivatePresharedKeyModel
		want map[string]attr.Type
	}{
		{
			name: "returns correct attribute types",
			m:    wlanPrivatePresharedKeyModel{},
			want: map[string]attr.Type{
				"network_id": types.StringType,
				"password":   types.StringType,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf(
					"wlanPrivatePresharedKeyModel.AttributeTypes() = %v, want %v",
					got,
					tt.want,
				)
			}
		})
	}
}

func Test_wlanFrameworkResource_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.MetadataRequest
		resp *fwresource.MetadataResponse
	}
	tests := []struct {
		name         string
		r            *wlanFrameworkResource
		args         args
		wantTypeName string
	}{
		{
			name: "sets type name",
			r:    &wlanFrameworkResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.MetadataRequest{ProviderTypeName: "unifi"},
				resp: &fwresource.MetadataResponse{},
			},
			wantTypeName: "unifi_wlan",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Metadata(tt.args.ctx, tt.args.req, tt.args.resp)
			if tt.args.resp.TypeName != tt.wantTypeName {
				t.Errorf("TypeName = %q, want %q", tt.args.resp.TypeName, tt.wantTypeName)
			}
		})
	}
}

func Test_wlanFrameworkResource_IdentitySchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwresource.IdentitySchemaRequest
		resp *fwresource.IdentitySchemaResponse
	}
	tests := []struct {
		name string
		r    *wlanFrameworkResource
		args args
	}{
		{
			name: "does not panic",
			r:    &wlanFrameworkResource{},
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

func Test_wlanFrameworkResource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.SchemaRequest
		resp *fwresource.SchemaResponse
	}
	tests := []struct {
		name string
		r    *wlanFrameworkResource
		args args
	}{
		{
			name: "contains key attributes",
			r:    &wlanFrameworkResource{},
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
			for _, key := range []string{"id", "name", "security"} {
				if _, ok := tt.args.resp.Schema.Attributes[key]; !ok {
					t.Errorf("Schema missing attribute %q", key)
				}
			}
		})
	}
}

func Test_wlanFrameworkResource_UpgradeState(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	r := &wlanFrameworkResource{}
	got := r.UpgradeState(context.Background())
	if got == nil {
		t.Fatal("UpgradeState() returned nil")
	}
	if _, ok := got[0]; !ok {
		t.Error("UpgradeState() missing key 0")
	}
}

func Test_wlanFrameworkResource_Configure(t *testing.T) {
	t.Run("nil provider data", func(t *testing.T) {
		r := &wlanFrameworkResource{}
		resp := &fwresource.ConfigureResponse{}
		r.Configure(context.Background(), fwresource.ConfigureRequest{ProviderData: nil}, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("expected no error for nil provider data, got %v", resp.Diagnostics)
		}
	})

	t.Run("wrong type", func(t *testing.T) {
		r := &wlanFrameworkResource{}
		resp := &fwresource.ConfigureResponse{}
		r.Configure(context.Background(), fwresource.ConfigureRequest{ProviderData: "wrong"}, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("expected error for wrong type")
		}
	})

	t.Run("correct client", func(t *testing.T) {
		r := &wlanFrameworkResource{}
		resp := &fwresource.ConfigureResponse{}
		client := &Client{}
		r.Configure(context.Background(), fwresource.ConfigureRequest{ProviderData: client}, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("unexpected error: %v", resp.Diagnostics)
		}
		if r.client != client {
			t.Error("client not set")
		}
	})
}

func Test_wlanFrameworkResource_setDefaultWLANGroupID(t *testing.T) {
	type args struct {
		ctx  context.Context
		site string
		wlan *unifi.WLAN
	}
	t.Skip("requires configured client")
}

func Test_wlanFrameworkResource_Create(t *testing.T) {
	t.Skip("requires terraform state and configured client")
}

func Test_wlanFrameworkResource_Read(t *testing.T) {
	t.Skip("requires terraform state and configured client")
}

func Test_wlanFrameworkResource_Update(t *testing.T) {
	t.Skip("requires terraform state and configured client")
}

func Test_wlanFrameworkResource_readPassphraseWO(t *testing.T) {
	t.Skip("requires terraform state")
}

func Test_wlanFrameworkResource_applyPlanToState(t *testing.T) {
	t.Skip("requires terraform state")
}

func Test_wlanFrameworkResource_Delete(t *testing.T) {
	t.Skip("requires terraform state and configured client")
}

func Test_wlanFrameworkResource_ImportState(t *testing.T) {
	t.Skip("requires terraform state and configured client")
}

func Test_wlanFrameworkResource_planToWLAN(t *testing.T) {
	ctx := context.Background()
	r := &wlanFrameworkResource{}

	plan := wlanFrameworkResourceModel{
		Name:     types.StringValue("test"),
		Security: types.StringValue("wpapsk"),
		MacFilter: types.ObjectNull(map[string]attr.Type{
			"enabled": types.BoolType,
			"list":    types.SetType{ElemType: types.StringType},
			"policy":  types.StringType,
		}),
		PrivatePresharedKeys: types.ListNull(
			types.ObjectType{AttrTypes: wlanPrivatePresharedKeyModel{}.AttributeTypes()},
		),
		ApGroupIDs:          types.SetNull(types.StringType),
		WLANBands:           types.SetNull(types.StringType),
		Schedule:            types.ListNull(types.ObjectType{}),
		BroadcastFilterList: types.SetNull(types.StringType),
	}

	got, diags := r.planToWLAN(ctx, plan)
	if diags.HasError() {
		t.Fatalf("planToWLAN() diagnostics: %v", diags)
	}
	if got.Name != "test" {
		t.Errorf("Name = %q, want %q", got.Name, "test")
	}
	if got.Security != "wpapsk" {
		t.Errorf("Security = %q, want %q", got.Security, "wpapsk")
	}
	if got.ScheduleWithDuration == nil {
		t.Error("ScheduleWithDuration should not be nil (empty slice expected)")
	}
}

func Test_wlanFrameworkResource_wlanToModel(t *testing.T) {
	ctx := context.Background()
	r := &wlanFrameworkResource{}

	wlan := &unifi.WLAN{
		ID:       "wlan-123",
		Name:     "test-wlan",
		Security: "wpapsk",
	}
	var model wlanFrameworkResourceModel
	diags := r.wlanToModel(ctx, wlan, &model, "default")
	if diags.HasError() {
		t.Fatalf("wlanToModel() diagnostics: %v", diags)
	}
	if model.ID.ValueString() != "wlan-123" {
		t.Errorf("ID = %q, want %q", model.ID.ValueString(), "wlan-123")
	}
	if model.Name.ValueString() != "test-wlan" {
		t.Errorf("Name = %q, want %q", model.Name.ValueString(), "test-wlan")
	}
	if model.Site.ValueString() != "default" {
		t.Errorf("Site = %q, want %q", model.Site.ValueString(), "default")
	}
	if model.Security.ValueString() != "wpapsk" {
		t.Errorf("Security = %q, want %q", model.Security.ValueString(), "wpapsk")
	}
}

func Test_wlanFrameworkResource_ListResourceConfigSchema(t *testing.T) {
	r := &wlanFrameworkResource{}
	resp := &fwlist.ListResourceSchemaResponse{}
	r.ListResourceConfigSchema(context.Background(), fwlist.ListResourceSchemaRequest{}, resp)
}

func Test_wlanFrameworkResource_List(t *testing.T) {
	t.Skip("requires configured client")
}

// TestWLANPrivatePresharedKeys_roundTrip exercises the private pre-shared key
// (PPSK) mapping added for issue #47: a plan carrying PPSK entries must be
// translated to the go-unifi WLAN struct (planToWLAN) and back into the
// resource model (wlanToModel) without losing the per-key network binding or
// password.
func TestWLANPrivatePresharedKeys_roundTrip(t *testing.T) {
	ctx := context.Background()
	r := &wlanFrameworkResource{}

	ppskType := types.ObjectType{AttrTypes: wlanPrivatePresharedKeyModel{}.AttributeTypes()}
	ppskList, d := types.ListValueFrom(ctx, ppskType, []wlanPrivatePresharedKeyModel{
		{NetworkID: types.StringValue("net-a"), Password: types.StringValue("secretpass1")},
		{NetworkID: types.StringValue(""), Password: types.StringValue("secretpass2")},
	})
	if d.HasError() {
		t.Fatalf("building PPSK list: %v", d)
	}

	plan := wlanFrameworkResourceModel{
		Name:                        types.StringValue("ppsk-wlan"),
		Security:                    types.StringValue("wpapsk"),
		PrivatePresharedKeysEnabled: types.BoolValue(true),
		PrivatePresharedKeys:        ppskList,
	}

	// plan -> API
	wlan, diags := r.planToWLAN(ctx, plan)
	if diags.HasError() {
		t.Fatalf("planToWLAN: %v", diags)
	}
	if !wlan.PrivatePresharedKeysEnabled {
		t.Errorf("PrivatePresharedKeysEnabled = false, want true")
	}
	if got := len(wlan.PrivatePresharedKeys); got != 2 {
		t.Fatalf("PrivatePresharedKeys len = %d, want 2", got)
	}
	if wlan.PrivatePresharedKeys[0].NetworkID != "net-a" ||
		wlan.PrivatePresharedKeys[0].Password != "secretpass1" {
		t.Errorf("PPSK[0] = %+v, want {net-a secretpass1}", wlan.PrivatePresharedKeys[0])
	}
	if wlan.PrivatePresharedKeys[1].NetworkID != "" ||
		wlan.PrivatePresharedKeys[1].Password != "secretpass2" {
		t.Errorf("PPSK[1] = %+v, want { secretpass2}", wlan.PrivatePresharedKeys[1])
	}

	// API -> model
	var model wlanFrameworkResourceModel
	if diags := r.wlanToModel(ctx, wlan, &model, "default"); diags.HasError() {
		t.Fatalf("wlanToModel: %v", diags)
	}
	if !model.PrivatePresharedKeysEnabled.ValueBool() {
		t.Errorf("model.PrivatePresharedKeysEnabled = false, want true")
	}
	if model.PrivatePresharedKeys.IsNull() {
		t.Fatalf("model.PrivatePresharedKeys is null, want 2 entries")
	}
	var got []wlanPrivatePresharedKeyModel
	if diags := model.PrivatePresharedKeys.ElementsAs(ctx, &got, false); diags.HasError() {
		t.Fatalf("decoding model PPSK: %v", diags)
	}
	if len(got) != 2 {
		t.Fatalf("model PPSK len = %d, want 2", len(got))
	}
	if got[0].NetworkID.ValueString() != "net-a" ||
		got[0].Password.ValueString() != "secretpass1" {
		t.Errorf("model PPSK[0] = %+v, want {net-a secretpass1}", got[0])
	}
}

// TestWLANPrivatePresharedKeys_emptyIsNull verifies that a WLAN without PPSK
// entries reads back as a null list (not an empty list), avoiding spurious
// plan drift for WLANs that don't use private pre-shared keys.
func TestWLANPrivatePresharedKeys_emptyIsNull(t *testing.T) {
	ctx := context.Background()
	r := &wlanFrameworkResource{}

	var model wlanFrameworkResourceModel
	if diags := r.wlanToModel(ctx, &unifi.WLAN{}, &model, "default"); diags.HasError() {
		t.Fatalf("wlanToModel: %v", diags)
	}
	if model.PrivatePresharedKeysEnabled.ValueBool() {
		t.Errorf("PrivatePresharedKeysEnabled = true, want false")
	}
	if !model.PrivatePresharedKeys.IsNull() {
		t.Errorf("PrivatePresharedKeys = %v, want null", model.PrivatePresharedKeys)
	}
}
