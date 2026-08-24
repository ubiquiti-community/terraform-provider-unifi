package unifi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
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

// TestAccWLANFramework_import creates an open (passphrase-less) WLAN and
// exercises both import paths: the classic string import (by controller id,
// with state verification) and the Terraform 1.12+ identity-based import
// block. An open WLAN is used because the read path deliberately keeps a
// null passphrase null (#392), which would otherwise make the post-import
// plan non-empty for a secured WLAN.
func TestAccWLANFramework_import(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set, skipping acceptance test")
	}

	// user_group_id is required in the schema and the provider exposes no
	// user-group data source, so fetch the default group from the API.
	userGroupID := testAccWLANDefaultUserGroupID(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWLANFrameworkConfig_import(userGroupID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_wlan.test_import",
						"name",
						"tfacc-wlan-import",
					),
					resource.TestCheckResourceAttr("unifi_wlan.test_import", "security", "open"),
					resource.TestCheckResourceAttrSet("unifi_wlan.test_import", "id"),
				),
			},
			// Classic string import by controller id.
			{
				ResourceName:      "unifi_wlan.test_import",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Identity-based import (import block with identity, Terraform 1.12+).
			{
				ResourceName:    "unifi_wlan.test_import",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func testAccWLANFrameworkConfig_import(userGroupID string) string {
	// network_id and ap_group_ids are pinned in config: they are optional,
	// non-computed attributes that the controller always assigns, so leaving
	// them out makes the apply result inconsistent with the plan.
	return fmt.Sprintf(`
data "unifi_ap_group" "default" {
	name = "All APs"
}

data "unifi_network" "default" {
	name = "Default"
}

resource "unifi_wlan" "test_import" {
	name          = "tfacc-wlan-import"
	security      = "open"
	user_group_id = %q
	network_id    = data.unifi_network.default.id
	ap_group_ids  = [data.unifi_ap_group.default.id]
}
`, userGroupID)
}

// testAccWLANDefaultUserGroupID fetches the default user group id straight
// from the controller REST API: the provider has no user-group data source
// and go-unifi exposes no user-group client, but WLANs require one.
func testAccWLANDefaultUserGroupID(t *testing.T) string {
	t.Helper()

	baseURL := os.Getenv("UNIFI_API")

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("creating cookie jar: %s", err)
	}
	httpClient := &http.Client{
		Jar: jar,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	loginBody, err := json.Marshal(map[string]string{
		"username": os.Getenv("UNIFI_USERNAME"),
		"password": os.Getenv("UNIFI_PASSWORD"),
	})
	if err != nil {
		t.Fatalf("marshaling login body: %s", err)
	}
	loginResp, err := httpClient.Post(
		baseURL+"/api/login",
		"application/json",
		bytes.NewReader(loginBody),
	)
	if err != nil {
		t.Fatalf("logging in to controller: %s", err)
	}
	defer loginResp.Body.Close()
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("logging in to controller: status %d", loginResp.StatusCode)
	}

	groupResp, err := httpClient.Get(baseURL + "/api/s/default/rest/usergroup")
	if err != nil {
		t.Fatalf("listing user groups: %s", err)
	}
	defer groupResp.Body.Close()
	if groupResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(groupResp.Body, 2048))
		t.Fatalf("listing user groups: status %d: %s", groupResp.StatusCode, respBody)
	}

	var body struct {
		Data []struct {
			ID       string `json:"_id"`
			Name     string `json:"name"`
			HiddenID string `json:"attr_hidden_id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(groupResp.Body).Decode(&body); err != nil {
		t.Fatalf("decoding user groups: %s", err)
	}

	for _, g := range body.Data {
		if g.HiddenID == "Default" || g.Name == "Default" {
			return g.ID
		}
	}
	if len(body.Data) > 0 {
		return body.Data[0].ID
	}

	t.Fatal("no user groups found on controller")
	return ""
}

func TestNewWLANFrameworkResource(t *testing.T) {
	got := NewWLANFrameworkResource()
	if got == nil {
		t.Fatal("NewWLANFrameworkResource() returned nil")
	}
	// Verify interface compliance
	_ = got
	if _, ok := got.(fwresource.ResourceWithImportState); !ok {
		t.Errorf("does not implement fwresource.ResourceWithImportState")
	}
	if _, ok := got.(fwresource.ResourceWithIdentity); !ok {
		t.Errorf("does not implement fwresource.ResourceWithIdentity")
	}
	if _, ok := got.(fwresource.ResourceWithUpgradeState); !ok {
		t.Errorf("does not implement fwresource.ResourceWithUpgradeState")
	}
}

func TestNewWLANListResource(t *testing.T) {
	got := NewWLANListResource()
	if got == nil {
		t.Fatal("NewWLANListResource() returned nil")
	}
	_ = got
	if _, ok := got.(fwlist.ListResourceWithConfigure); !ok {
		t.Errorf("does not implement fwlist.ListResourceWithConfigure")
	}
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

// Test_wlanFrameworkResource_Schema_computedControllerFields guards #323: fields
// the controller assigns on its own must be Computed so a controller-supplied
// value doesn't trip "inconsistent result after apply". minimum_data_rate_*_kbps
// previously defaulted to 0 (rejected/overridden by the controller in auto mode);
// radius_profile_id and bc_filter_list were Optional-only and got populated by
// the controller.
func Test_wlanFrameworkResource_Schema_computedControllerFields(t *testing.T) {
	resp := &fwresource.SchemaResponse{}
	(&wlanFrameworkResource{}).Schema(context.Background(), fwresource.SchemaRequest{}, resp)

	for _, key := range []string{
		"minimum_data_rate_2g_kbps",
		"minimum_data_rate_5g_kbps",
		"radius_profile_id",
		"bc_filter_list",
	} {
		attr, ok := resp.Schema.Attributes[key]
		if !ok {
			t.Errorf("Schema missing attribute %q", key)
			continue
		}
		if !attr.IsComputed() {
			t.Errorf("attribute %q must be Computed (controller-managed, #323)", key)
		}
	}
}

func Test_wlanFrameworkResource_UpgradeState(t *testing.T) {
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

func TestWLANPrivatePresharedKeys_preservesStateForPartialResponse(t *testing.T) {
	ctx := context.Background()
	ppskType := types.ObjectType{AttrTypes: wlanPrivatePresharedKeyModel{}.AttributeTypes()}
	prior, diags := types.ListValueFrom(ctx, ppskType, []wlanPrivatePresharedKeyModel{
		{NetworkID: types.StringValue("net-a"), Password: types.StringValue("secretpass1")},
		{NetworkID: types.StringValue("net-b"), Password: types.StringValue("secretpass2")},
	})
	if diags.HasError() {
		t.Fatalf("building prior PPSK list: %v", diags)
	}

	for name, keys := range map[string][]unifi.WLANPrivatePresharedKeys{
		"passwords omitted and list reordered": {
			{NetworkID: "net-b"},
			{NetworkID: "net-a"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			model := wlanFrameworkResourceModel{PrivatePresharedKeys: prior}
			wlan := &unifi.WLAN{
				PrivatePresharedKeysEnabled: true,
				PrivatePresharedKeys:        keys,
			}

			if diags := (&wlanFrameworkResource{}).wlanToModel(
				ctx,
				wlan,
				&model,
				"default",
			); diags.HasError() {
				t.Fatalf("wlanToModel: %v", diags)
			}
			if !model.PrivatePresharedKeys.Equal(prior) {
				t.Fatalf("PrivatePresharedKeys = %v, want %v", model.PrivatePresharedKeys, prior)
			}
		})
	}
}

func TestWLANPrivatePresharedKeys_usesControllerChanges(t *testing.T) {
	ctx := context.Background()
	ppskType := types.ObjectType{AttrTypes: wlanPrivatePresharedKeyModel{}.AttributeTypes()}
	list := func(keys ...wlanPrivatePresharedKeyModel) types.List {
		value, diags := types.ListValueFrom(ctx, ppskType, keys)
		if diags.HasError() {
			t.Fatalf("building PPSK list: %v", diags)
		}
		return value
	}
	prior := list(
		wlanPrivatePresharedKeyModel{
			NetworkID: types.StringValue("net-a"),
			Password:  types.StringValue("secretpass1"),
		},
	)
	duplicateBindings := list(
		wlanPrivatePresharedKeyModel{
			NetworkID: types.StringValue("net-a"),
			Password:  types.StringValue("secretpass1"),
		},
		wlanPrivatePresharedKeyModel{
			NetworkID: types.StringValue("net-a"),
			Password:  types.StringValue("secretpass2"),
		},
	)

	tests := []struct {
		name  string
		wlan  *unifi.WLAN
		prior types.List
		want  types.List
	}{
		{
			name:  "disabled",
			wlan:  &unifi.WLAN{},
			prior: prior,
			want:  types.ListNull(ppskType),
		},
		{
			name:  "list omitted",
			wlan:  &unifi.WLAN{PrivatePresharedKeysEnabled: true},
			prior: prior,
			want:  types.ListNull(ppskType),
		},
		{
			name: "explicit deletion",
			wlan: &unifi.WLAN{
				PrivatePresharedKeysEnabled: true,
				PrivatePresharedKeys:        []unifi.WLANPrivatePresharedKeys{},
			},
			prior: prior,
			want:  types.ListNull(ppskType),
		},
		{
			name: "import",
			wlan: &unifi.WLAN{
				PrivatePresharedKeysEnabled: true,
				PrivatePresharedKeys: []unifi.WLANPrivatePresharedKeys{
					{NetworkID: "net-a", Password: "secretpass1"},
				},
			},
			prior: types.ListNull(ppskType),
			want:  prior,
		},
		{
			name: "key added",
			wlan: &unifi.WLAN{
				PrivatePresharedKeysEnabled: true,
				PrivatePresharedKeys: []unifi.WLANPrivatePresharedKeys{
					{NetworkID: "net-a", Password: "secretpass1"},
					{NetworkID: "net-b", Password: "secretpass2"},
				},
			},
			prior: prior,
			want: list(
				wlanPrivatePresharedKeyModel{
					NetworkID: types.StringValue("net-a"),
					Password:  types.StringValue("secretpass1"),
				},
				wlanPrivatePresharedKeyModel{
					NetworkID: types.StringValue("net-b"),
					Password:  types.StringValue("secretpass2"),
				},
			),
		},
		{
			name: "binding changed",
			wlan: &unifi.WLAN{
				PrivatePresharedKeysEnabled: true,
				PrivatePresharedKeys: []unifi.WLANPrivatePresharedKeys{
					{NetworkID: "net-b", Password: "secretpass1"},
				},
			},
			prior: prior,
			want: list(wlanPrivatePresharedKeyModel{
				NetworkID: types.StringValue("net-b"),
				Password:  types.StringValue("secretpass1"),
			}),
		},
		{
			name: "binding changed without password",
			wlan: &unifi.WLAN{
				PrivatePresharedKeysEnabled: true,
				PrivatePresharedKeys: []unifi.WLANPrivatePresharedKeys{
					{NetworkID: "net-b"},
				},
			},
			prior: prior,
			want: list(wlanPrivatePresharedKeyModel{
				NetworkID: types.StringValue("net-b"),
				Password:  types.StringValue(""),
			}),
		},
		{
			name: "password changed",
			wlan: &unifi.WLAN{
				PrivatePresharedKeysEnabled: true,
				PrivatePresharedKeys: []unifi.WLANPrivatePresharedKeys{
					{NetworkID: "net-a", Password: "changedpass1"},
				},
			},
			prior: prior,
			want: list(wlanPrivatePresharedKeyModel{
				NetworkID: types.StringValue("net-a"),
				Password:  types.StringValue("changedpass1"),
			}),
		},
		{
			name: "matching response",
			wlan: &unifi.WLAN{
				PrivatePresharedKeysEnabled: true,
				PrivatePresharedKeys: []unifi.WLANPrivatePresharedKeys{
					{NetworkID: "net-a", Password: "secretpass1"},
				},
			},
			prior: prior,
			want:  prior,
		},
		{
			name: "duplicate bindings match once",
			wlan: &unifi.WLAN{
				PrivatePresharedKeysEnabled: true,
				PrivatePresharedKeys: []unifi.WLANPrivatePresharedKeys{
					{NetworkID: "net-a"},
					{NetworkID: "net-a", Password: "secretpass1"},
				},
			},
			prior: duplicateBindings,
			want:  duplicateBindings,
		},
		{
			name: "duplicate binding replaced",
			wlan: &unifi.WLAN{
				PrivatePresharedKeysEnabled: true,
				PrivatePresharedKeys: []unifi.WLANPrivatePresharedKeys{
					{NetworkID: "net-a"},
					{NetworkID: "net-a", Password: "changedpass1"},
				},
			},
			prior: duplicateBindings,
			want: list(
				wlanPrivatePresharedKeyModel{
					NetworkID: types.StringValue("net-a"),
					Password:  types.StringValue(""),
				},
				wlanPrivatePresharedKeyModel{
					NetworkID: types.StringValue("net-a"),
					Password:  types.StringValue("changedpass1"),
				},
			),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, diags := privatePresharedKeysState(ctx, test.wlan, test.prior)
			if diags.HasError() {
				t.Fatalf("privatePresharedKeysState: %v", diags)
			}
			if !got.Equal(test.want) {
				t.Fatalf("privatePresharedKeysState = %v, want %v", got, test.want)
			}
		})
	}
}

func TestWLANPrivatePresharedKeys_rejectsInvalidPriorState(t *testing.T) {
	_, diags := privatePresharedKeysState(
		context.Background(),
		&unifi.WLAN{
			PrivatePresharedKeysEnabled: true,
			PrivatePresharedKeys: []unifi.WLANPrivatePresharedKeys{
				{NetworkID: "net-a", Password: "secretpass1"},
			},
		},
		types.ListValueMust(types.StringType, []attr.Value{types.StringValue("invalid")}),
	)
	if !diags.HasError() {
		t.Fatal("privatePresharedKeysState accepted invalid prior state")
	}
}

// TestApplyEnhancedIotOverrides guards #283: when enhanced_iot is enabled the
// controller forces iapp_enabled, wpa3_support, wpa3_transition, pmf_mode and
// dtim_ng, so the provider pins them in the plan to avoid an inconsistent-result
// error. When enhanced_iot is false it must be a no-op.
func TestApplyEnhancedIotOverrides(t *testing.T) {
	t.Run("enhanced_iot true forces the controller-managed fields", func(t *testing.T) {
		m := &wlanFrameworkResourceModel{
			EnhancedIot:    types.BoolValue(true),
			IappEnabled:    types.BoolValue(false),
			WPA3Support:    types.BoolValue(true),
			WPA3Transition: types.BoolValue(true),
			PMFMode:        types.StringValue("optional"),
			DTIMNg:         types.Int64Value(3),
		}
		if !applyEnhancedIotOverrides(m) {
			t.Fatal("expected overrides to be applied")
		}
		if !m.IappEnabled.ValueBool() {
			t.Errorf("iapp_enabled = %v, want true", m.IappEnabled.ValueBool())
		}
		if m.WPA3Support.ValueBool() {
			t.Errorf("wpa3_support = %v, want false", m.WPA3Support.ValueBool())
		}
		if m.WPA3Transition.ValueBool() {
			t.Errorf("wpa3_transition = %v, want false", m.WPA3Transition.ValueBool())
		}
		if m.PMFMode.ValueString() != "disabled" {
			t.Errorf("pmf_mode = %q, want disabled", m.PMFMode.ValueString())
		}
		if m.DTIMNg.ValueInt64() != 1 {
			t.Errorf("dtim_ng = %d, want 1", m.DTIMNg.ValueInt64())
		}
	})

	t.Run("enhanced_iot false is a no-op", func(t *testing.T) {
		m := &wlanFrameworkResourceModel{
			EnhancedIot: types.BoolValue(false),
			WPA3Support: types.BoolValue(true),
			PMFMode:     types.StringValue("optional"),
		}
		if applyEnhancedIotOverrides(m) {
			t.Fatal("expected no overrides when enhanced_iot is false")
		}
		if !m.WPA3Support.ValueBool() || m.PMFMode.ValueString() != "optional" {
			t.Errorf("non-IoT fields were modified: wpa3=%v pmf=%q",
				m.WPA3Support.ValueBool(), m.PMFMode.ValueString())
		}
	})
}

func TestAccWLANList_basic(t *testing.T) {
	// WLAN creation requires user_group_id which cannot be reliably resolved in
	// the dockerized test environment; skip until the basic create path works.
	t.Skip("WLAN creation requires user_group_id; skipping list acceptance test")
}

// ---------------------------------------------------------------------------
// #406: unifi_wlan creation with "6g" in wlan_bands
// ---------------------------------------------------------------------------

func Test_missingWLANBands(t *testing.T) {
	tests := []struct {
		name      string
		requested []string
		actual    []string
		want      []string
	}{
		{
			name:      "controller kept everything",
			requested: []string{"2g", "5g", "6g"},
			actual:    []string{"2g", "5g", "6g"},
			want:      nil,
		},
		{
			name:      "controller dropped 6g (the #406 report)",
			requested: []string{"2g", "5g", "6g"},
			actual:    []string{"2g", "5g"},
			want:      []string{"6g"},
		},
		{
			name:      "order does not matter",
			requested: []string{"6g", "2g"},
			actual:    []string{"2g", "6g"},
			want:      nil,
		},
		{
			name:      "no bands requested",
			requested: nil,
			actual:    []string{"2g"},
			want:      nil,
		},
		{
			name:      "controller returned nothing",
			requested: []string{"6g"},
			actual:    nil,
			want:      []string{"6g"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := missingWLANBands(tt.requested, tt.actual)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("missingWLANBands(%v, %v) = %v, want %v",
					tt.requested, tt.actual, got, tt.want)
			}
		})
	}
}

// Test_planToWLAN_keeps6gBand pins down where #406 does NOT come from: the
// provider marshals the full declared band list — "6g" included — into the
// create/update payload, and derives the legacy wlan_band field only from the
// 2g/5g members. The drop reported in #406 happens controller-side on create,
// which is why Create re-asserts the band list when the response is missing a
// requested band.
func Test_planToWLAN_keeps6gBand(t *testing.T) {
	ctx := context.Background()
	r := &wlanFrameworkResource{}

	bands, diags := types.SetValueFrom(ctx, types.StringType, []string{"2g", "5g", "6g"})
	if diags.HasError() {
		t.Fatalf("building band set: %v", diags)
	}

	plan := wlanFrameworkResourceModel{
		Name:      types.StringValue("tfacc-6g"),
		Security:  types.StringValue("wpapsk"),
		WLANBands: bands,
	}

	wlan, diags := r.planToWLAN(ctx, plan)
	if diags.HasError() {
		t.Fatalf("planToWLAN: %v", diags)
	}

	got := map[string]bool{}
	for _, b := range wlan.WLANBands {
		got[b] = true
	}
	for _, want := range []string{"2g", "5g", "6g"} {
		if !got[want] {
			t.Errorf("wlan_bands missing %q in payload: %v", want, wlan.WLANBands)
		}
	}
	if wlan.WLANBand != "both" {
		t.Errorf("legacy wlan_band = %q, want %q (derived from 2g+5g)", wlan.WLANBand, "both")
	}
}

// TestAccWLANFramework_wifi6ghzBand creates a WLAN with the exact band/security
// shape from #406 — wlan_bands ["2g","5g","6g"] with WPA3 transition and
// optional PMF — in a single apply, then removes 6g in-place. On controllers
// that drop 6g at create time, Create's re-assert update kicks in; on
// controllers that keep it (like this one), the path is a plain create. Either
// way the resulting state must carry all three bands.
func TestAccWLANFramework_wifi6ghzBand(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set, skipping acceptance test")
	}

	userGroupID := testAccWLANDefaultUserGroupID(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWLANFrameworkConfig_wifi6ghzBand(userGroupID, `"2g", "5g", "6g"`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_wlan.test_6g", "wlan_bands.#", "3"),
					resource.TestCheckTypeSetElemAttr("unifi_wlan.test_6g", "wlan_bands.*", "2g"),
					resource.TestCheckTypeSetElemAttr("unifi_wlan.test_6g", "wlan_bands.*", "5g"),
					resource.TestCheckTypeSetElemAttr("unifi_wlan.test_6g", "wlan_bands.*", "6g"),
					resource.TestCheckResourceAttr("unifi_wlan.test_6g", "wpa3_support", "true"),
				),
			},
			{
				// Dropping 6g afterwards must be a clean in-place update.
				Config: testAccWLANFrameworkConfig_wifi6ghzBand(userGroupID, `"2g", "5g"`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_wlan.test_6g", "wlan_bands.#", "2"),
					resource.TestCheckTypeSetElemAttr("unifi_wlan.test_6g", "wlan_bands.*", "2g"),
					resource.TestCheckTypeSetElemAttr("unifi_wlan.test_6g", "wlan_bands.*", "5g"),
				),
			},
		},
	})
}

func testAccWLANFrameworkConfig_wifi6ghzBand(userGroupID, bands string) string {
	// network_id and ap_group_ids are pinned in config for the same reason as
	// the import test: they are optional, non-computed attributes the
	// controller always assigns, so leaving them out makes the apply result
	// inconsistent with the plan.
	return fmt.Sprintf(`
data "unifi_ap_group" "default" {
	name = "All APs"
}

data "unifi_network" "default" {
	name = "Default"
}

resource "unifi_wlan" "test_6g" {
	name            = "tfacc-wlan-6g"
	security        = "wpapsk"
	passphrase      = "pwd12345678"
	wpa3_support    = true
	wpa3_transition = true
	pmf_mode        = "optional"
	user_group_id   = %q
	network_id      = data.unifi_network.default.id
	ap_group_ids    = [data.unifi_ap_group.default.id]
	wlan_bands      = [%s]
}
`, userGroupID, bands)
}

// fakeWLANUpdater records UpdateWLAN calls for reassertWLANBands tests.
type fakeWLANUpdater struct {
	calls  int
	gotID  string
	got    *unifi.WLAN
	result *unifi.WLAN
	err    error
}

func (f *fakeWLANUpdater) UpdateWLAN(
	_ context.Context,
	_ string,
	d *unifi.WLAN,
) (*unifi.WLAN, error) {
	f.calls++
	f.gotID = d.ID
	f.got = d
	return f.result, f.err
}

// Test_reassertWLANBands covers the #406 retry: when the controller's create
// response is missing a requested band, the full intended configuration is
// re-asserted exactly once via update; when nothing is missing, no update is
// issued; and an update failure falls back to the create response so Create
// can raise its actionable diagnostic.
func Test_reassertWLANBands(t *testing.T) {
	ctx := context.Background()

	t.Run("controller kept all bands: no update issued", func(t *testing.T) {
		fake := &fakeWLANUpdater{}
		requested := &unifi.WLAN{Name: "w", WLANBands: []string{"2g", "5g", "6g"}}
		created := &unifi.WLAN{ID: "id-1", Name: "w", WLANBands: []string{"6g", "2g", "5g"}}

		got := reassertWLANBands(ctx, fake, "default", requested, created)
		if fake.calls != 0 {
			t.Errorf("UpdateWLAN called %d times, want 0", fake.calls)
		}
		if got != created {
			t.Errorf("result = %+v, want the create response untouched", got)
		}
	})

	t.Run("controller dropped 6g: full band list re-asserted once", func(t *testing.T) {
		reasserted := &unifi.WLAN{ID: "id-1", Name: "w", WLANBands: []string{"2g", "5g", "6g"}}
		fake := &fakeWLANUpdater{result: reasserted}
		requested := &unifi.WLAN{Name: "w", WLANBands: []string{"2g", "5g", "6g"}}
		created := &unifi.WLAN{ID: "id-1", Name: "w", WLANBands: []string{"2g", "5g"}}

		got := reassertWLANBands(ctx, fake, "default", requested, created)
		if fake.calls != 1 {
			t.Fatalf("UpdateWLAN called %d times, want 1", fake.calls)
		}
		if fake.gotID != "id-1" {
			t.Errorf("update sent ID %q, want the created WLAN's id-1", fake.gotID)
		}
		if !reflect.DeepEqual(fake.got.WLANBands, []string{"2g", "5g", "6g"}) {
			t.Errorf("update sent bands %v, want the full requested list", fake.got.WLANBands)
		}
		if got != reasserted {
			t.Errorf("result = %+v, want the reasserted response", got)
		}
	})

	t.Run("update fails: create response kept for the diagnostic", func(t *testing.T) {
		fake := &fakeWLANUpdater{err: fmt.Errorf("api.err.InvalidPayload")}
		requested := &unifi.WLAN{Name: "w", WLANBands: []string{"2g", "6g"}}
		created := &unifi.WLAN{ID: "id-2", Name: "w", WLANBands: []string{"2g"}}

		got := reassertWLANBands(ctx, fake, "default", requested, created)
		if fake.calls != 1 {
			t.Fatalf("UpdateWLAN called %d times, want 1", fake.calls)
		}
		if got != created {
			t.Errorf("result = %+v, want the create response", got)
		}
	})
}
