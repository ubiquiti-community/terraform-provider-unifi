package unifi

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/hwtypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func TestParseDevicePortImportID(t *testing.T) {
	tests := []struct {
		name      string
		importID  string
		wantSite  string
		wantMAC   string
		wantIndex int64
		wantErr   bool
	}{
		{
			name:      "mac and index",
			importID:  "a8:9c:6c:08:ea:3b/2",
			wantMAC:   "a8:9c:6c:08:ea:3b",
			wantIndex: 2,
		},
		{
			name:      "dash-separated upper-case MAC is normalised",
			importID:  "A8-9C-6C-08-EA-3B/9",
			wantMAC:   "a8:9c:6c:08:ea:3b",
			wantIndex: 9,
		},
		{
			name:      "site prefix",
			importID:  "branch/a8:9c:6c:08:ea:3b/12",
			wantSite:  "branch",
			wantMAC:   "a8:9c:6c:08:ea:3b",
			wantIndex: 12,
		},
		{name: "no separator", importID: "a8:9c:6c:08:ea:3b2", wantErr: true},
		{name: "empty index", importID: "a8:9c:6c:08:ea:3b/", wantErr: true},
		{name: "empty mac", importID: "/2", wantErr: true},
		{name: "empty site", importID: "/a8:9c:6c:08:ea:3b/2", wantErr: true},
		{name: "not a MAC", importID: "a8:9c:6c/2", wantErr: true},
		{name: "non-numeric index", importID: "a8:9c:6c:08:ea:3b/nine", wantErr: true},
		{name: "index zero", importID: "a8:9c:6c:08:ea:3b/0", wantErr: true},
		{name: "negative index", importID: "a8:9c:6c:08:ea:3b/-1", wantErr: true},
		{name: "colon separator", importID: "a8:9c:6c:08:ea:3b:2", wantErr: true},
		{name: "too many parts", importID: "a/b/a8:9c:6c:08:ea:3b/2", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			site, mac, index, err := parseDevicePortImportID(tt.importID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseDevicePortImportID(%q) error = %v, wantErr %v",
					tt.importID, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if site != tt.wantSite {
				t.Errorf("site = %q, want %q", site, tt.wantSite)
			}
			if mac != tt.wantMAC {
				t.Errorf("mac = %q, want %q", mac, tt.wantMAC)
			}
			if index != tt.wantIndex {
				t.Errorf("index = %d, want %d", index, tt.wantIndex)
			}
		})
	}
}

func TestDevicePortExists(t *testing.T) {
	t.Parallel()

	table := []unifi.DevicePortTable{{PortIdx: 1}, {PortIdx: 2}, {PortIdx: 3}}

	cases := []struct {
		name   string
		table  []unifi.DevicePortTable
		index  int64
		wantN  int
		wantOK bool
	}{
		{"listed", table, 2, 3, true},
		{"not listed", table, 4, 3, false},
		{"empty table cannot be checked", nil, 48, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			n, ok := devicePortExists(tc.table, tc.index)
			if n != tc.wantN || ok != tc.wantOK {
				t.Fatalf("devicePortExists(%d) = (%d, %v), want (%d, %v)",
					tc.index, n, ok, tc.wantN, tc.wantOK)
			}
		})
	}
}

func TestFindPortOverride(t *testing.T) {
	overrides := []unifi.DevicePortOverrides{
		{PortIDX: ptrInt64(1), Name: "one"},
		{Name: "no index"},
		{PortIDX: ptrInt64(3), Name: "three"},
	}
	if po, ok := findPortOverride(overrides, 3); !ok || po.Name != "three" {
		t.Errorf("findPortOverride(3) = %v, %v; want the entry named three", po, ok)
	}
	if _, ok := findPortOverride(overrides, 2); ok {
		t.Error("findPortOverride(2) found an entry, want none")
	}
}

// devicePortSchemaType is the Terraform type of the unifi_device_port schema.
func devicePortSchemaType(ctx context.Context, t *testing.T) (rschema.Schema, tftypes.Type) {
	t.Helper()
	var schemaResp fwresource.SchemaResponse
	(&devicePortResource{}).Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}
	return schemaResp.Schema, schemaResp.Schema.Type().TerraformType(ctx)
}

// TestDevicePortSchema_MatchesPortOverride guards the shared attribute set: a
// unifi_device_port has every port_override attribute of unifi_device, plus its
// own keys, and nothing else.
func TestDevicePortSchema_MatchesPortOverride(t *testing.T) {
	ctx := context.Background()
	sch, _ := devicePortSchemaType(ctx, t)

	own := map[string]bool{"id": true, "site": true, "device_mac": true, "timeouts": true}
	for name := range portOverrideAttrTypes() {
		if _, ok := sch.Attributes[name]; !ok {
			t.Errorf("port_override attribute %q is missing from unifi_device_port", name)
		}
	}
	for name := range sch.Attributes {
		if own[name] {
			continue
		}
		if _, ok := portOverrideAttrTypes()[name]; !ok {
			t.Errorf("unifi_device_port attribute %q is not a port_override attribute", name)
		}
	}
}

// TestDevicePortModel_RoundTrip checks that the resource model, which embeds
// portOverrideModel, converts to and from the framework's state.
func TestDevicePortModel_RoundTrip(t *testing.T) {
	ctx := context.Background()
	sch, typ := devicePortSchemaType(ctx, t)
	objType, ok := typ.(tftypes.Object)
	if !ok {
		t.Fatalf("schema type is %T, want tftypes.Object", typ)
	}

	values := map[string]tftypes.Value{}
	for name, attrType := range objType.AttributeTypes {
		values[name] = tftypes.NewValue(attrType, nil)
	}
	values["device_mac"] = tftypes.NewValue(tftypes.String, "AA-BB-CC-DD-EE-FF")
	values["index"] = tftypes.NewValue(tftypes.Number, 8)
	values["name"] = tftypes.NewValue(tftypes.String, "nas")
	values["stp_edge_state"] = tftypes.NewValue(tftypes.String, "enabled")
	values["tagged_networkconf_ids"] = tftypes.NewValue(
		objType.AttributeTypes["tagged_networkconf_ids"],
		[]tftypes.Value{tftypes.NewValue(tftypes.String, "net-a")},
	)

	state := tfsdk.State{Raw: tftypes.NewValue(objType, values), Schema: sch}
	var model devicePortResourceModel
	if diags := state.Get(ctx, &model); diags.HasError() {
		t.Fatalf("state.Get: %v", diags)
	}

	if model.DeviceMAC.ValueString() != "AA-BB-CC-DD-EE-FF" {
		t.Errorf("DeviceMAC = %q", model.DeviceMAC.ValueString())
	}
	if model.Index.ValueInt64() != 8 {
		t.Errorf("Index = %d, want 8", model.Index.ValueInt64())
	}
	if model.Name.ValueString() != "nas" {
		t.Errorf("Name = %q, want nas", model.Name.ValueString())
	}
	if model.StpEdgeState.ValueString() != "enabled" {
		t.Errorf("StpEdgeState = %q, want enabled", model.StpEdgeState.ValueString())
	}
	if !model.OpMode.IsNull() {
		t.Errorf("OpMode = %v, want null", model.OpMode)
	}

	// The embedded port override converts to the API form like a block does.
	po, diags := portOverrideModelToAPI(ctx, model.portOverrideModel)
	if diags.HasError() {
		t.Fatalf("portOverrideModelToAPI: %v", diags)
	}
	if po.PortIDX == nil || *po.PortIDX != 8 || po.Name != "nas" || po.StpEdgeState != "enabled" {
		t.Errorf("api override = %+v", po)
	}
	if len(po.TaggedNetworkIDs) != 1 || po.TaggedNetworkIDs[0] != "net-a" {
		t.Errorf("TaggedNetworkIDs = %v, want [net-a]", po.TaggedNetworkIDs)
	}

	// And back into the state.
	model.ID = types.StringValue(devicePortID("aa:bb:cc:dd:ee:ff", 8))
	model.Site = types.StringValue("default")
	if diags := state.Set(ctx, model); diags.HasError() {
		t.Fatalf("state.Set: %v", diags)
	}
	var id types.String
	if diags := state.GetAttribute(ctx, path.Root("id"), &id); diags.HasError() {
		t.Fatalf("reading id: %v", diags)
	}
	if id.ValueString() != "aa:bb:cc:dd:ee:ff/8" {
		t.Errorf("id = %q", id.ValueString())
	}
}

// TestReconcilePortOverride_LostEntry: a port whose override entry disappeared
// from the controller reads back with its configured attributes unset, so the
// plan shows the drift instead of hiding it.
func TestReconcilePortOverride_LostEntry(t *testing.T) {
	prior := portOverrideModel{
		Index:            types.Int64Value(8),
		Name:             types.StringValue("nas"),
		Forward:          types.StringValue("customize"),
		Autoneg:          types.BoolUnknown(),
		TaggedNetworkIDs: types.SetValueMust(types.StringType, nil),
	}
	got, diags := reconcilePortOverride(prior, unifi.DevicePortOverrides{PortIDX: ptrInt64(8)})
	if diags.HasError() {
		t.Fatalf("reconcilePortOverride: %v", diags)
	}
	if !got.Name.IsNull() {
		t.Errorf("Name = %v, want null", got.Name)
	}
	if !got.Forward.IsNull() {
		t.Errorf("Forward = %v, want null", got.Forward)
	}
	if got.Autoneg.IsUnknown() {
		t.Error("Autoneg is still unknown after reconciliation")
	}
	if got.TaggedNetworkIDs.IsNull() || len(got.TaggedNetworkIDs.Elements()) != 0 {
		t.Errorf("TaggedNetworkIDs = %v, want an empty set", got.TaggedNetworkIDs)
	}
}

func TestDeviceLockKey(t *testing.T) {
	tests := []struct {
		name, mac, id, want string
	}{
		{"MAC is preferred and cleaned", "AA-BB-CC-DD-EE-FF", "device-id", "aa:bb:cc:dd:ee:ff"},
		{"clean MAC is kept", "aa:bb:cc:dd:ee:ff", "device-id", "aa:bb:cc:dd:ee:ff"},
		{"empty MAC falls back to ID", "", "device-id", "device-id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deviceLockKey(tt.mac, tt.id); got != tt.want {
				t.Errorf("deviceLockKey(%q, %q) = %q, want %q", tt.mac, tt.id, got, tt.want)
			}
		})
	}
}

// TestClientLockDevice checks that lockDevice excludes a second holder of the
// same key and does not block an unrelated device.
func TestClientLockDevice(t *testing.T) {
	c := &Client{}
	unlock := c.lockDevice("aa:bb:cc:dd:ee:ff")

	same := make(chan struct{})
	go func() {
		unlock2 := c.lockDevice("aa:bb:cc:dd:ee:ff")
		close(same)
		unlock2()
	}()
	other := make(chan struct{})
	go func() {
		unlock3 := c.lockDevice("11:22:33:44:55:66")
		close(other)
		unlock3()
	}()

	select {
	case <-other:
	case <-time.After(time.Second):
		t.Fatal("lockDevice for another device blocked on an unrelated lock")
	}
	select {
	case <-same:
		t.Fatal("second lockDevice for the same device acquired while the first held it")
	case <-time.After(50 * time.Millisecond):
	}

	unlock()
	select {
	case <-same:
	case <-time.After(time.Second):
		t.Fatal("second lockDevice never acquired after the first released")
	}
}

func newDevicePortImportResponse(
	ctx context.Context,
	t *testing.T,
) *fwresource.ImportStateResponse {
	t.Helper()
	sch, typ := devicePortSchemaType(ctx, t)
	var identityResp fwresource.IdentitySchemaResponse
	(&devicePortResource{}).IdentitySchema(ctx, fwresource.IdentitySchemaRequest{}, &identityResp)
	identityType := identityResp.IdentitySchema.Type().TerraformType(ctx)
	return &fwresource.ImportStateResponse{
		State: tfsdk.State{Raw: tftypes.NewValue(typ, nil), Schema: sch},
		Identity: &tfsdk.ResourceIdentity{
			Raw:    tftypes.NewValue(identityType, nil),
			Schema: identityResp.IdentitySchema,
		},
	}
}

func TestDevicePortImportState(t *testing.T) {
	ctx := context.Background()
	r := &devicePortResource{client: &Client{Site: "default"}}

	t.Run("by id", func(t *testing.T) {
		resp := newDevicePortImportResponse(ctx, t)
		r.ImportState(ctx, fwresource.ImportStateRequest{ID: "A8:9C:6C:08:EA:3B/1"}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ImportState: %v", resp.Diagnostics)
		}
		var mac hwtypes.MACAddress
		var index types.Int64
		var site, id types.String
		resp.State.GetAttribute(ctx, path.Root("device_mac"), &mac)
		resp.State.GetAttribute(ctx, path.Root("index"), &index)
		resp.State.GetAttribute(ctx, path.Root("site"), &site)
		resp.State.GetAttribute(ctx, path.Root("id"), &id)
		if mac.ValueString() != "a8:9c:6c:08:ea:3b" || index.ValueInt64() != 1 ||
			site.ValueString() != "default" {
			t.Errorf("state = %s/%d on %s",
				mac.ValueString(), index.ValueInt64(), site.ValueString())
		}
		if !id.IsNull() {
			t.Errorf("id = %v, want null until the first read", id)
		}
		var identity devicePortIdentityModel
		if diags := resp.Identity.Get(ctx, &identity); diags.HasError() {
			t.Fatalf("identity: %v", diags)
		}
		if identity.DeviceMAC.ValueString() != "a8:9c:6c:08:ea:3b" ||
			identity.Index.ValueInt64() != 1 || identity.Site.ValueString() != "default" {
			t.Errorf("identity = %+v", identity)
		}
	})

	t.Run("by id with site", func(t *testing.T) {
		resp := newDevicePortImportResponse(ctx, t)
		r.ImportState(ctx, fwresource.ImportStateRequest{ID: "branch/a8:9c:6c:08:ea:3b/4"}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ImportState: %v", resp.Diagnostics)
		}
		var site types.String
		resp.State.GetAttribute(ctx, path.Root("site"), &site)
		if site.ValueString() != "branch" {
			t.Errorf("site = %q, want branch", site.ValueString())
		}
	})

	t.Run("by identity", func(t *testing.T) {
		resp := newDevicePortImportResponse(ctx, t)
		identity := &tfsdk.ResourceIdentity{Raw: resp.Identity.Raw, Schema: resp.Identity.Schema}
		diags := identity.Set(ctx, devicePortIdentity("a8:9c:6c:08:ea:3b", 7, ""))
		if diags.HasError() {
			t.Fatalf("building identity: %v", diags)
		}
		r.ImportState(ctx, fwresource.ImportStateRequest{Identity: identity}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ImportState: %v", resp.Diagnostics)
		}
		var index types.Int64
		var site types.String
		resp.State.GetAttribute(ctx, path.Root("index"), &index)
		resp.State.GetAttribute(ctx, path.Root("site"), &site)
		if index.ValueInt64() != 7 || site.ValueString() != "default" {
			t.Errorf("state = port %d on %s, want 7 on default",
				index.ValueInt64(), site.ValueString())
		}
	})

	t.Run("invalid", func(t *testing.T) {
		for _, id := range []string{"not-an-id", "a8:9c:6c:08:ea:3b/0", "a8:9c:6c:08:ea:3b:2"} {
			resp := newDevicePortImportResponse(ctx, t)
			r.ImportState(ctx, fwresource.ImportStateRequest{ID: id}, resp)
			if !resp.Diagnostics.HasError() {
				t.Errorf("ImportState(%q): want an error, got none", id)
			}
		}
	})
}

// Acceptance tests run against a simulated switch of the demo controller.
const testAccDevicePortSwitchMAC = "00:27:22:00:00:01"

func TestAccDevicePort_basic(t *testing.T) {
	const name = "unifi_device_port.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDevicePortConfig(5, "tf-port-5", "auto"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "id", testAccDevicePortSwitchMAC+"/5"),
					resource.TestCheckResourceAttr(name, "device_mac", testAccDevicePortSwitchMAC),
					resource.TestCheckResourceAttr(name, "index", "5"),
					resource.TestCheckResourceAttr(name, "site", "default"),
					resource.TestCheckResourceAttr(name, "name", "tf-port-5"),
					resource.TestCheckResourceAttr(name, "dot1x.ctrl", "auto"),
					resource.TestCheckResourceAttr(name, "op_mode", "switch"),
				),
			},
			{
				Config: testAccDevicePortConfig(5, "tf-port-5-renamed", "force_authorized"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "tf-port-5-renamed"),
					resource.TestCheckResourceAttr(name, "dot1x.ctrl", "force_authorized"),
				),
			},
			// Import by "<device_mac>/<index>". The imported state carries every
			// attribute the controller holds for the port, while the state of
			// the steps above only carries the configured ones.
			{
				ResourceName:            name,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: testAccDevicePortImportIgnore,
			},
			// Import by resource identity (import block with identity, Terraform 1.12+).
			{
				ResourceName:    name,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

// testAccDevicePortImportIgnore lists the attributes an import reads from the
// controller that a create leaves unset: optional attributes without a
// configured value are null after a create and carry the controller's value
// after an import.
var testAccDevicePortImportIgnore = []string{
	"setting_preference", "poe_mode", "port_profile_id", "fec_mode",
	"tagged_vlan_mgmt", "voice_networkconf_id",
	"speed", "mirror_port_idx", "priority_queue1_level", "priority_queue2_level",
	"priority_queue3_level", "priority_queue4_level", "aggregate_members",
	"excluded_networkconf_ids", "multicast_router_networkconf_ids",
	"tagged_networkconf_ids", "timeouts",
}

func TestAccDevicePort_besidePortOverride(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// A unifi_device managing one port and two unifi_device_port resources
			// managing two others on the same switch, applied in one run.
			{
				Config: `
resource "unifi_device" "sw" {
	mac               = "` + testAccDevicePortSwitchMAC + `"
	name              = "Test Device Ports"
	allow_adoption    = true
	forget_on_destroy = false

	port_override {
		index = 1
		name  = "managed-by-device"
	}
}

resource "unifi_device_port" "six" {
	device_mac = unifi_device.sw.mac
	index      = 6
	name       = "managed-by-port-6"
}

resource "unifi_device_port" "seven" {
	device_mac = unifi_device.sw.mac
	index      = 7
	name       = "managed-by-port-7"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_device_port.six", "name", "managed-by-port-6"),
					resource.TestCheckResourceAttr(
						"unifi_device_port.seven", "name", "managed-by-port-7"),
				),
			},
			// A second plan is empty: neither resource overwrote the other's port.
			{
				RefreshState:       true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// testAccDevicePortSwitch adopts the simulated switch the port tests use, so
// the tests do not depend on another test having adopted it first.
// A port the switch does not have is refused before anything is written: the
// simulated US 48 PoE reports 52 ports in its port_table.
func TestAccDevicePort_unknownPort(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDevicePortSwitch() + `
resource "unifi_device_port" "nope" {
	device_mac = unifi_device.sw.mac
	index      = 99
	name       = "does-not-exist"
}
`,
				ExpectError: regexp.MustCompile(`Port Not Found`),
			},
		},
	})
}

func testAccDevicePortSwitch() string {
	return `
resource "unifi_device" "sw" {
	mac               = "` + testAccDevicePortSwitchMAC + `"
	name              = "Test Device Ports"
	allow_adoption    = true
	forget_on_destroy = false
}
`
}

// The simulated switch persists name and the dot1x group; stp_edge_state is
// not stored by the simulator, so the test does not use it.
//
// native_networkconf_id and forward are declared because the controller fills
// them in (default network, "all"): an import then stores them, and the plan
// after the identity import step is empty only when the configuration declares
// them too (an import surfaces every setting the configuration does not cover).
func testAccDevicePortConfig(index int, name, dot1xCtrl string) string {
	return testAccDevicePortSwitch() + `
data "unifi_network" "default" {
	name = "Default"
}

resource "unifi_device_port" "test" {
	device_mac            = unifi_device.sw.mac
	index                 = ` + itoa(index) + `
	name                  = "` + name + `"
	native_networkconf_id = data.unifi_network.default.id
	forward               = "all"
	dot1x = {
		ctrl = "` + dot1xCtrl + `"
	}
}
`
}

func itoa(i int) string {
	return types.Int64Value(int64(i)).String()
}
