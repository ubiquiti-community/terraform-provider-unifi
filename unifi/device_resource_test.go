package unifi

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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

// Test_resolvePortOverridesForUpdate_zeroDeclaredEchoesCurrent guards #438: a
// switch with live port_overrides, updated while config declares zero
// port_override blocks, must echo the controller's current overrides — not send
// `port_overrides: null` (rejected with api.err.InvalidPayload) and not send `[]`
// (which would silently wipe every live override).
func Test_resolvePortOverridesForUpdate_zeroDeclaredEchoesCurrent(t *testing.T) {
	current := []unifi.DevicePortOverrides{
		{PortIDX: ptrInt64(1), NATiveNetworkID: "vlan-a"},
		{PortIDX: ptrInt64(2), NATiveNetworkID: "vlan-b"},
	}
	currentDevice := &unifi.Device{PortOverrides: current}
	deviceReq := &unifi.Device{PortOverrides: nil}

	got := resolvePortOverridesForUpdate(currentDevice, deviceReq)
	if len(got) != len(current) {
		t.Fatalf(
			"resolvePortOverridesForUpdate() length = %d, want %d (must echo current, not null/empty): %+v",
			len(got),
			len(current),
			got,
		)
	}
	byIdx := indexOverrides(got)
	if byIdx[1].NATiveNetworkID != "vlan-a" || byIdx[2].NATiveNetworkID != "vlan-b" {
		t.Errorf("current overrides not echoed unchanged: %+v", got)
	}

	minimalDevice := buildMinimalUpdateDevice(deviceReq, currentDevice, got)
	if minimalDevice.PortOverrides == nil {
		t.Error(
			"buildMinimalUpdateDevice() PortOverrides is nil, want the live overrides echoed (would marshal to `port_overrides: null`)",
		)
	}
	if len(minimalDevice.PortOverrides) != len(current) {
		t.Errorf(
			"buildMinimalUpdateDevice() PortOverrides length = %d, want %d",
			len(minimalDevice.PortOverrides),
			len(current),
		)
	}
}

// Test_resolvePortOverridesForUpdate_noCurrentOverridesMirrorsDevice guards the
// #436/#427 case: a device with no current overrides at all (an AP/gateway) and
// zero declared blocks must mirror the existing device's representation exactly
// — nil when the device reports null, [] when it reports [] — so go-unifi's
// diff-based UpdateDevice drops the unchanged key from the PUT instead of
// manufacturing a null→[] change some controllers reject with api.err.Invalid.
func Test_resolvePortOverridesForUpdate_noCurrentOverridesMirrorsDevice(t *testing.T) {
	t.Run("device reports null", func(t *testing.T) {
		currentDevice := &unifi.Device{PortOverrides: nil}
		deviceReq := &unifi.Device{PortOverrides: nil}

		got := resolvePortOverridesForUpdate(currentDevice, deviceReq)
		minimalDevice := buildMinimalUpdateDevice(deviceReq, currentDevice, got)
		if minimalDevice.PortOverrides != nil {
			t.Errorf(
				"buildMinimalUpdateDevice() PortOverrides = %#v, want nil to mirror the "+
					"device's null so the diff drops the key",
				minimalDevice.PortOverrides,
			)
		}
	})

	t.Run("device reports empty array", func(t *testing.T) {
		currentDevice := &unifi.Device{PortOverrides: []unifi.DevicePortOverrides{}}
		deviceReq := &unifi.Device{PortOverrides: nil}

		got := resolvePortOverridesForUpdate(currentDevice, deviceReq)
		minimalDevice := buildMinimalUpdateDevice(deviceReq, currentDevice, got)
		if minimalDevice.PortOverrides == nil {
			t.Error(
				"buildMinimalUpdateDevice() PortOverrides is nil, want a non-nil empty " +
					"slice to mirror the device's [] (nil would marshal to `port_overrides: null`)",
			)
		}
		if len(minimalDevice.PortOverrides) != 0 {
			t.Errorf(
				"buildMinimalUpdateDevice() PortOverrides length = %d, want 0",
				len(minimalDevice.PortOverrides),
			)
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

// Test_buildMinimalUpdateDevice_mgmtNetworkID guards #329: a configured
// mgmt_network_id (the UI "Network Override") must travel in the minimal PUT body,
// and a null value must stay off the wire so it never reintroduces the #177
// zero-value rejection. modelToAPIDevice sets deviceReq.MgmtNetworkID only when
// configured, so nullness is represented here by an empty value on deviceReq.
func Test_buildMinimalUpdateDevice_mgmtNetworkID(t *testing.T) {
	t.Run("configured mgmt_network_id is sent in the PUT body", func(t *testing.T) {
		deviceReq := &unifi.Device{
			ID:            "dev-1",
			Type:          "usw",
			MAC:           "aa:bb:cc:dd:ee:ff",
			Name:          "Test Switch",
			MgmtNetworkID: "net-mgmt",
		}
		body := buildMinimalUpdateDevice(deviceReq, nil, nil)
		if body.MgmtNetworkID != "net-mgmt" {
			t.Fatalf(
				"MgmtNetworkID = %q, want %q (override dropped from PUT, #329)",
				body.MgmtNetworkID,
				"net-mgmt",
			)
		}
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if !strings.Contains(string(raw), `"mgmt_network_id":"net-mgmt"`) {
			t.Errorf("PUT body missing mgmt_network_id: %s", raw)
		}
	})

	t.Run("null mgmt_network_id stays off the wire (no #177 regression)", func(t *testing.T) {
		deviceReq := &unifi.Device{
			ID:   "dev-1",
			Type: "usw",
			MAC:  "aa:bb:cc:dd:ee:ff",
			Name: "Test Switch",
		}
		body := buildMinimalUpdateDevice(deviceReq, nil, nil)
		if body.MgmtNetworkID != "" {
			t.Errorf("MgmtNetworkID = %q, want empty for a null override", body.MgmtNetworkID)
		}
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if strings.Contains(string(raw), "mgmt_network_id") {
			t.Errorf("null mgmt_network_id leaked into PUT body: %s", raw)
		}
	})
}

// Test_buildMinimalUpdateDevice_switchVLANEnabled guards the switch_vlan_enabled
// bug class: a configured "Port VLAN" toggle (true) must travel in the minimal
// PUT body, else the controller keeps its old value and the post-apply read
// conflicts with the plan. Being `omitempty`, a false stays off the wire and
// doesn't disturb the controller default.
func Test_buildMinimalUpdateDevice_switchVLANEnabled(t *testing.T) {
	t.Run("configured switch_vlan_enabled is sent in the PUT body", func(t *testing.T) {
		deviceReq := &unifi.Device{
			ID:                "dev-1",
			Type:              "uap",
			MAC:               "aa:bb:cc:dd:ee:ff",
			Name:              "Test AP",
			SwitchVLANEnabled: true,
		}
		body := buildMinimalUpdateDevice(deviceReq, nil, nil)
		if !body.SwitchVLANEnabled {
			t.Fatal("SwitchVLANEnabled = false, want true (toggle dropped from PUT)")
		}
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if !strings.Contains(string(raw), `"switch_vlan_enabled":true`) {
			t.Errorf("PUT body missing switch_vlan_enabled: %s", raw)
		}
	})

	t.Run("false switch_vlan_enabled stays off the wire (omitempty)", func(t *testing.T) {
		deviceReq := &unifi.Device{
			ID:   "dev-1",
			Type: "uap",
			MAC:  "aa:bb:cc:dd:ee:ff",
			Name: "Test AP",
		}
		body := buildMinimalUpdateDevice(deviceReq, nil, nil)
		if body.SwitchVLANEnabled {
			t.Errorf("SwitchVLANEnabled = true, want false when unconfigured")
		}
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if strings.Contains(string(raw), "switch_vlan_enabled") {
			t.Errorf("false switch_vlan_enabled leaked into PUT body: %s", raw)
		}
	})
}

// Test_buildMinimalUpdateDevice_vwireEnabled guards the radio_table[].vwire_enabled
// bug class (the UI "Mesh Parent" toggle): the hand-listed minimal PUT never
// copied radio_table across, so every radio sub-field — vwire_enabled included —
// was dropped, the controller kept its old value, and the post-apply read
// conflicted with the plan. Being `omitempty` at every level, an empty
// radio_table stays off the wire and doesn't disturb the controller default.
func Test_buildMinimalUpdateDevice_vwireEnabled(t *testing.T) {
	t.Run("configured vwire_enabled is sent in the PUT body", func(t *testing.T) {
		deviceReq := &unifi.Device{
			ID:   "dev-1",
			Type: "uap",
			MAC:  "aa:bb:cc:dd:ee:ff",
			Name: "Test AP",
			RadioTable: []unifi.DeviceRadioTable{
				{Name: "wifi0", Radio: "ng", VwireEnabled: true},
				{Name: "wifi1", Radio: "na", VwireEnabled: true},
			},
		}
		body := buildMinimalUpdateDevice(deviceReq, nil, nil)
		if len(body.RadioTable) != 2 {
			t.Fatalf(
				"RadioTable len = %d, want 2 (radio_table dropped from PUT)",
				len(body.RadioTable),
			)
		}
		for _, radio := range body.RadioTable {
			if !radio.VwireEnabled {
				t.Fatalf(
					"radio %q VwireEnabled = false, want true (toggle dropped from PUT)",
					radio.Name,
				)
			}
		}
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if !strings.Contains(string(raw), `"vwire_enabled":true`) {
			t.Errorf("PUT body missing vwire_enabled: %s", raw)
		}
	})

	t.Run("false vwire_enabled stays off the wire (omitempty)", func(t *testing.T) {
		deviceReq := &unifi.Device{
			ID:   "dev-1",
			Type: "uap",
			MAC:  "aa:bb:cc:dd:ee:ff",
			Name: "Test AP",
			RadioTable: []unifi.DeviceRadioTable{
				{Name: "wifi0", Radio: "ng"},
			},
		}
		body := buildMinimalUpdateDevice(deviceReq, nil, nil)
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if strings.Contains(string(raw), "vwire_enabled") {
			t.Errorf("false vwire_enabled leaked into PUT body: %s", raw)
		}
	})

	t.Run("nil radio_table stays off the wire (omitempty)", func(t *testing.T) {
		deviceReq := &unifi.Device{
			ID:   "dev-1",
			Type: "usw",
			MAC:  "aa:bb:cc:dd:ee:ff",
			Name: "Test Switch",
		}
		body := buildMinimalUpdateDevice(deviceReq, nil, nil)
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if strings.Contains(string(raw), "radio_table") {
			t.Errorf("empty radio_table leaked into PUT body: %s", raw)
		}
	})
}

// Test_buildMinimalUpdateDevice_meshStaVapEnabled guards the top-level
// mesh_sta_vap_enabled bug class (the UI "Mesh Connect" toggle): a configured
// true must travel in the minimal PUT body, else the controller keeps its old
// value and the post-apply read conflicts with the plan. Being `omitempty`, a
// false stays off the wire and doesn't disturb the controller default.
func Test_buildMinimalUpdateDevice_meshStaVapEnabled(t *testing.T) {
	t.Run("configured mesh_sta_vap_enabled is sent in the PUT body", func(t *testing.T) {
		deviceReq := &unifi.Device{
			ID:                "dev-1",
			Type:              "uap",
			MAC:               "aa:bb:cc:dd:ee:ff",
			Name:              "Test AP",
			MeshStaVapEnabled: true,
		}
		body := buildMinimalUpdateDevice(deviceReq, nil, nil)
		if !body.MeshStaVapEnabled {
			t.Fatal("MeshStaVapEnabled = false, want true (toggle dropped from PUT)")
		}
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if !strings.Contains(string(raw), `"mesh_sta_vap_enabled":true`) {
			t.Errorf("PUT body missing mesh_sta_vap_enabled: %s", raw)
		}
	})

	t.Run("false mesh_sta_vap_enabled stays off the wire (omitempty)", func(t *testing.T) {
		deviceReq := &unifi.Device{
			ID:   "dev-1",
			Type: "uap",
			MAC:  "aa:bb:cc:dd:ee:ff",
			Name: "Test AP",
		}
		body := buildMinimalUpdateDevice(deviceReq, nil, nil)
		if body.MeshStaVapEnabled {
			t.Errorf("MeshStaVapEnabled = true, want false when unconfigured")
		}
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if strings.Contains(string(raw), "mesh_sta_vap_enabled") {
			t.Errorf("false mesh_sta_vap_enabled leaked into PUT body: %s", raw)
		}
	})
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
			// Classic string import. ImportStateVerify replays the import with the
			// `id` attribute value, exercising the ID-or-MAC fallback.
			{
				ResourceName:            "unifi_device.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"allow_adoption", "forget_on_destroy"},
			},
			// Classic string import by MAC address (the documented import ID).
			{
				ResourceName:            "unifi_device.test",
				ImportState:             true,
				ImportStateId:           "00:27:22:00:00:02",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"allow_adoption", "forget_on_destroy"},
			},
			// The post-import refresh defaults the provider-only flags
			// allow_adoption/forget_on_destroy to true, so the identity-import
			// step must follow a config using those defaults for its plan to be
			// empty. Switch to that config first.
			{
				Config: testAccDeviceFrameworkConfig_importDefaults(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_device.test",
						"forget_on_destroy",
						"true",
					),
				),
			},
			// Identity-based import (import block with identity, Terraform 1.12+).
			{
				ResourceName:    "unifi_device.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
			// Return to forget_on_destroy = false so the final destroy does not
			// forget the shared simulated device.
			{
				Config: testAccDeviceFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_device.test",
						"forget_on_destroy",
						"false",
					),
				),
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

// testAccDeviceFrameworkConfig_importDefaults leaves the provider-only flags at
// their schema defaults (true), matching what a fresh import writes to state, so
// a post-import plan against this config is empty.
func testAccDeviceFrameworkConfig_importDefaults() string {
	return `
resource "unifi_device" "test" {
	mac  = "00:27:22:00:00:02"
	name = "Test Device"
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

// TestDeviceNetworkconfIDsAreSets guards #384: the port_override networkconf_ids
// attributes are order-insensitive Sets, and a v1->v2 state upgrader exists so
// existing List state migrates instead of erroring on refresh.
func TestDeviceNetworkconfIDsAreSets(t *testing.T) {
	ctx := context.Background()
	r := &deviceResource{}

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)

	if schemaResp.Schema.Version != 2 {
		t.Errorf("device schema Version = %d, want 2", schemaResp.Schema.Version)
	}

	block, ok := schemaResp.Schema.Blocks["port_override"].(schema.SetNestedBlock)
	if !ok {
		t.Fatalf(
			"port_override is not a SetNestedBlock: %T",
			schemaResp.Schema.Blocks["port_override"],
		)
	}
	for _, name := range []string{
		"excluded_networkconf_ids",
		"multicast_router_networkconf_ids",
		"tagged_networkconf_ids",
	} {
		if _, ok := block.NestedObject.Attributes[name].(schema.SetAttribute); !ok {
			t.Errorf(
				"port_override.%s must be a SetAttribute, got %T",
				name, block.NestedObject.Attributes[name],
			)
		}
	}

	ups := r.UpgradeState(ctx)
	for _, v := range []int64{0, 1} {
		if _, ok := ups[v]; !ok {
			t.Errorf("UpgradeState is missing an upgrader for schema version %d", v)
		}
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

// TestReconcilePortOverrides_NativeNetworkClearedRoundTrips guards #410: the
// controller reports "" for a port's native_networkconf_id when the native
// network is explicitly set to None (the device_resource analogue of #383's
// unifi_port_profile fix). reconcilePortOverrides must surface that as a known
// empty string (not null) for any port the user explicitly configured, so an
// explicit native_networkconf_id = "" round-trips instead of drifting back to
// "unset" on every subsequent plan.
func TestReconcilePortOverrides_NativeNetworkClearedRoundTrips(t *testing.T) {
	ctx := context.Background()
	r := &deviceResource{}

	priorModel := portOverrideModel{
		Index:           types.Int64Value(1),
		NativeNetworkID: types.StringValue(""),
		// Zero-value collections carry no element type and fail the
		// ObjectValueFrom type check; use typed nulls.
		AggregateMembers:          types.ListNull(types.Int64Type),
		ExcludedNetworkIDs:        types.SetNull(types.StringType),
		MulticastRouterNetworkIDs: types.SetNull(types.StringType),
		PortSecurityMACAddress:    types.ListNull(types.StringType),
		TaggedNetworkIDs:          types.SetNull(types.StringType),
	}
	priorObj, diags := types.ObjectValueFrom(ctx, priorModel.AttributeTypes(), priorModel)
	if diags.HasError() {
		t.Fatalf("building prior object: %v", diags)
	}
	priorSet, diags := types.SetValue(
		types.ObjectType{AttrTypes: portOverrideAttrTypes()},
		[]attr.Value{priorObj},
	)
	if diags.HasError() {
		t.Fatalf("building prior set: %v", diags)
	}

	got, diags := r.reconcilePortOverrides(ctx, priorSet, []unifi.DevicePortOverrides{
		{PortIDX: ptrInt64(1), NATiveNetworkID: ""},
	})
	if diags.HasError() {
		t.Fatalf("reconcilePortOverrides returned errors: %v", diags)
	}

	var reconciled []portOverrideModel
	diags = got.ElementsAs(ctx, &reconciled, false)
	if diags.HasError() {
		t.Fatalf("reading back reconciled set: %v", diags)
	}
	if len(reconciled) != 1 {
		t.Fatalf("reconciled length = %d, want 1: %+v", len(reconciled), reconciled)
	}
	if reconciled[0].NativeNetworkID.IsNull() || reconciled[0].NativeNetworkID.IsUnknown() ||
		reconciled[0].NativeNetworkID.ValueString() != "" {
		t.Errorf(
			"native_networkconf_id: want known empty string, got %#v",
			reconciled[0].NativeNetworkID,
		)
	}
}

// TestReconcilePortOverrides_NativeNetworkAssignedKept is the companion to
// #410: a controller-assigned native network ID must still surface its value.
func TestReconcilePortOverrides_NativeNetworkAssignedKept(t *testing.T) {
	ctx := context.Background()
	r := &deviceResource{}

	priorModel := portOverrideModel{
		Index:           types.Int64Value(1),
		NativeNetworkID: types.StringValue("net-old"),
		// Zero-value collections carry no element type and fail the
		// ObjectValueFrom type check; use typed nulls.
		AggregateMembers:          types.ListNull(types.Int64Type),
		ExcludedNetworkIDs:        types.SetNull(types.StringType),
		MulticastRouterNetworkIDs: types.SetNull(types.StringType),
		PortSecurityMACAddress:    types.ListNull(types.StringType),
		TaggedNetworkIDs:          types.SetNull(types.StringType),
	}
	priorObj, diags := types.ObjectValueFrom(ctx, priorModel.AttributeTypes(), priorModel)
	if diags.HasError() {
		t.Fatalf("building prior object: %v", diags)
	}
	priorSet, diags := types.SetValue(
		types.ObjectType{AttrTypes: portOverrideAttrTypes()},
		[]attr.Value{priorObj},
	)
	if diags.HasError() {
		t.Fatalf("building prior set: %v", diags)
	}

	got, diags := r.reconcilePortOverrides(ctx, priorSet, []unifi.DevicePortOverrides{
		{PortIDX: ptrInt64(1), NATiveNetworkID: "net-123"},
	})
	if diags.HasError() {
		t.Fatalf("reconcilePortOverrides returned errors: %v", diags)
	}

	var reconciled []portOverrideModel
	diags = got.ElementsAs(ctx, &reconciled, false)
	if diags.HasError() {
		t.Fatalf("reading back reconciled set: %v", diags)
	}
	if len(reconciled) != 1 {
		t.Fatalf("reconciled length = %d, want 1: %+v", len(reconciled), reconciled)
	}
	if reconciled[0].NativeNetworkID.ValueString() != "net-123" {
		t.Errorf(
			"native_networkconf_id: want net-123, got %q",
			reconciled[0].NativeNetworkID.ValueString(),
		)
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

// TestReconcilePortOverrides_ResolvesUnknownOptionalComputedAttrs is a
// regression test for #431. Optional+Computed attributes left out of config
// (e.g. flow_control_enabled) plan as unknown, and the six field guards in
// reconcilePortOverrides only handle "configured" (non-null) or "absent"
// (null) prior values — an unknown value falls through both and reaches
// ObjectValueFrom untouched, which the framework rejects with "produced an
// unexpected new value: ... is unknown after apply". reconcilePortOverrides
// must resolve any attribute still unknown from the API response.
func TestReconcilePortOverrides_ResolvesUnknownOptionalComputedAttrs(t *testing.T) {
	ctx := context.Background()
	r := &deviceResource{}

	baseline, diags := r.portOverridesToFramework(ctx, []unifi.DevicePortOverrides{
		{PortIDX: ptrInt64(7), Name: "Port 7", FlowControlEnabled: false},
	})
	if diags.HasError() {
		t.Fatalf("portOverridesToFramework errored: %v", diags.Errors())
	}

	elems := baseline.Elements()
	if len(elems) != 1 {
		t.Fatalf("expected 1 port_override element, got %d", len(elems))
	}
	obj, ok := elems[0].(types.Object)
	if !ok {
		t.Fatalf("expected port_override element to be types.Object, got %T", elems[0])
	}

	var model portOverrideModel
	if diags = obj.As(ctx, &model, basetypes.ObjectAsOptions{}); diags.HasError() {
		t.Fatalf("Object.As errored: %v", diags.Errors())
	}

	// Simulate the plan: flow_control_enabled was left out of config, so it
	// plans as unknown rather than the null/known value portOverridesToFramework
	// (a Read) would have produced.
	model.FlowControlEnabled = types.BoolUnknown()

	objVal, objDiags := types.ObjectValueFrom(ctx, model.AttributeTypes(), model)
	if objDiags.HasError() {
		t.Fatalf("ObjectValueFrom errored: %v", objDiags.Errors())
	}

	prior, setDiags := types.SetValue(
		types.ObjectType{AttrTypes: portOverrideAttrTypes()},
		[]attr.Value{objVal},
	)
	if setDiags.HasError() {
		t.Fatalf("SetValue errored: %v", setDiags.Errors())
	}

	got, gotDiags := r.reconcilePortOverrides(ctx, prior, []unifi.DevicePortOverrides{
		{PortIDX: ptrInt64(7), Name: "Port 7", FlowControlEnabled: true},
	})
	if gotDiags.HasError() {
		t.Fatalf("reconcilePortOverrides errored: %v", gotDiags.Errors())
	}

	gotElems := got.Elements()
	if len(gotElems) != 1 {
		t.Fatalf("expected 1 reconciled element, got %d", len(gotElems))
	}
	gotObj, ok := gotElems[0].(types.Object)
	if !ok {
		t.Fatalf("expected reconciled element to be types.Object, got %T", gotElems[0])
	}

	flowControl, ok := gotObj.Attributes()["flow_control_enabled"]
	if !ok {
		t.Fatal("reconciled port_override is missing flow_control_enabled")
	}
	if flowControl.IsUnknown() {
		t.Fatal("flow_control_enabled is still unknown after reconcile — apply would fail with " +
			`"Provider returned invalid result object after apply" (#431)`)
	}
	boolVal, ok := flowControl.(types.Bool)
	if !ok {
		t.Fatalf("expected flow_control_enabled to be types.Bool, got %T", flowControl)
	}
	if !boolVal.ValueBool() {
		t.Errorf("flow_control_enabled = %v, want true (from API response)", boolVal)
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

// ---------------------------------------------------------------------------
// #427: declared radio_table entries must not zero-fill undeclared sub-fields
// ---------------------------------------------------------------------------

// unknownRadioTableModel returns a radioTableModel with every sub-attribute
// Unknown — the exact shape the plan produces for a radio_table entry whose
// Optional+Computed sub-attributes the practitioner did not declare.
func unknownRadioTableModel() radioTableModel {
	return radioTableModel{
		Radio:                  types.StringUnknown(),
		Channel:                types.StringUnknown(),
		Ht:                     types.Int64Unknown(),
		TxPower:                types.StringUnknown(),
		TxPowerMode:            types.StringUnknown(),
		MinRssiEnabled:         types.BoolUnknown(),
		MinRssi:                types.Int64Unknown(),
		AntennaGain:            types.Int64Unknown(),
		AntennaID:              types.Int64Unknown(),
		AssistedRoamingEnabled: types.BoolUnknown(),
		AssistedRoamingRssi:    types.Int64Unknown(),
		Dfs:                    types.BoolUnknown(),
		HardNoiseFloorEnabled:  types.BoolUnknown(),
		LoadbalanceEnabled:     types.BoolUnknown(),
		Maxsta:                 types.Int64Unknown(),
		Name:                   types.StringUnknown(),
		SensLevel:              types.Int64Unknown(),
		SensLevelEnabled:       types.BoolUnknown(),
		VwireEnabled:           types.BoolUnknown(),
	}
}

func radioTableListFromModels(t *testing.T, models ...radioTableModel) types.List {
	t.Helper()
	ctx := context.Background()
	elems := make([]attr.Value, 0, len(models))
	for _, m := range models {
		obj, diags := types.ObjectValueFrom(ctx, radioTableAttrTypes(), m)
		if diags.HasError() {
			t.Fatalf("building radio object: %v", diags)
		}
		elems = append(elems, obj)
	}
	list, diags := types.ListValue(
		types.ObjectType{AttrTypes: radioTableAttrTypes()},
		elems,
	)
	if diags.HasError() {
		t.Fatalf("building radio list: %v", diags)
	}
	return list
}

// Test_frameworkToRadioTable_unknownStaysOffTheWire is the exact repro of #427:
// a radio_table entry declared with only radio/channel/ht/tx_power_mode leaves
// every other Optional+Computed sub-attribute Unknown (not Null) in the plan.
// ValueInt64Pointer() maps Unknown to a pointer at the Go zero value, so the
// PUT carried "antenna_gain":0,"antenna_id":0,"min_rssi":0,… — rejected by the
// controller with api.err.InvalidPayload (400). Unknown must serialize exactly
// like Null: off the wire.
func Test_frameworkToRadioTable_unknownStaysOffTheWire(t *testing.T) {
	ctx := context.Background()
	r := &deviceResource{}

	declared := unknownRadioTableModel()
	declared.Radio = types.StringValue("ng")
	declared.Channel = types.StringValue("11")
	declared.Ht = types.Int64Value(20)
	declared.TxPowerMode = types.StringValue("auto")

	radios, diags := r.frameworkToRadioTable(ctx, radioTableListFromModels(t, declared))
	if diags.HasError() {
		t.Fatalf("frameworkToRadioTable: %v", diags)
	}
	if len(radios) != 1 {
		t.Fatalf("got %d radios, want 1", len(radios))
	}

	raw, err := json.Marshal(radios[0])
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	body := string(raw)

	for _, key := range []string{
		`"min_rssi":`,
		`"sens_level":`,
		`"assisted_roaming_rssi":`,
		`"maxsta":`,
		`"antenna_gain":`,
		`"antenna_id":`,
		`"name":`,
		`"tx_power":`,
	} {
		if strings.Contains(body, key) {
			t.Errorf("undeclared %s zero-filled into PUT body: %s", key, body)
		}
	}
	for _, want := range []string{
		`"radio":"ng"`,
		`"channel":"11"`,
		`"ht":20`,
		`"tx_power_mode":"auto"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("declared %s missing from PUT body: %s", want, body)
		}
	}
}

// Test_mergeRadioTableFromDevice guards the echo half of #427: the controller
// requires each radio_table entry to carry the hardware-assigned radio `name`
// (omitting it fails with api.err.MissingValue, key "radio"), and
// hardware-derived antenna fields must match the device. Fields the user left
// unset are filled from the device's current radio table; declared values win;
// fields absent on the device stay off the wire; and the gated RSSI-style
// fields are never merged (sanitizeRadioForUpdate owns those, #378).
func Test_mergeRadioTableFromDevice(t *testing.T) {
	current := []unifi.DeviceRadioTable{
		{
			Radio:       "na",
			Name:        "wifi1",
			Channel:     "auto",
			Ht:          ptrInt64(80),
			AntennaGain: ptrInt64(6),
			AntennaID:   ptrInt64(-1),
			TxPowerMode: "auto",
			MinRssi:     ptrInt64(-90),
		},
		{
			Radio:       "ng",
			Name:        "wifi0",
			Channel:     "auto",
			Ht:          ptrInt64(20),
			AntennaGain: ptrInt64(4),
			AntennaID:   ptrInt64(-1),
			TxPowerMode: "auto",
		},
	}

	target := []unifi.DeviceRadioTable{
		{Radio: "ng", Channel: "11", Ht: ptrInt64(20), TxPowerMode: "auto"},
		{Radio: "na", Channel: "149", Ht: ptrInt64(80), TxPowerMode: "custom", TxPower: "23"},
		{Radio: "6e", Channel: "117"},
	}

	mergeRadioTableFromDevice(target, current)

	ng := target[0]
	if ng.Name != "wifi0" {
		t.Errorf("ng name = %q, want %q (controller-required, must be echoed)", ng.Name, "wifi0")
	}
	if ng.AntennaGain == nil || *ng.AntennaGain != 4 {
		t.Errorf("ng antenna_gain = %v, want 4 (echoed from device)", ng.AntennaGain)
	}
	if ng.AntennaID == nil || *ng.AntennaID != -1 {
		t.Errorf("ng antenna_id = %v, want -1 (echoed from device)", ng.AntennaID)
	}
	if ng.Channel != "11" {
		t.Errorf(
			"ng channel = %q, want declared %q to win over device %q",
			ng.Channel,
			"11",
			"auto",
		)
	}

	na := target[1]
	if na.Name != "wifi1" {
		t.Errorf("na name = %q, want %q", na.Name, "wifi1")
	}
	if na.TxPower != "23" || na.TxPowerMode != "custom" {
		t.Errorf(
			"na tx_power/mode = %q/%q, want declared 23/custom kept",
			na.TxPower,
			na.TxPowerMode,
		)
	}
	if na.MinRssi != nil {
		t.Errorf(
			"na min_rssi = %v, want nil — gated fields must never be merged back (#378)",
			*na.MinRssi,
		)
	}

	sixE := target[2]
	if sixE.Name != "" || sixE.AntennaGain != nil || sixE.AntennaID != nil {
		t.Errorf("6e radio has no device entry; nothing may be invented: %+v", sixE)
	}
}

// Test_reconcileRadioTableWithPlan guards the post-apply half of #427: the
// controller reports its radios in hardware order with every field populated,
// but Terraform requires the applied value to keep the declared list's count,
// order, and known values. Only Unknown sub-attributes may resolve from the
// controller's entry for the same band.
func Test_reconcileRadioTableWithPlan(t *testing.T) {
	ctx := context.Background()
	r := &deviceResource{}

	plannedNg := unknownRadioTableModel()
	plannedNg.Radio = types.StringValue("ng")
	plannedNg.Channel = types.StringValue("11")
	plannedNg.Ht = types.Int64Value(20)
	plannedNg.TxPowerMode = types.StringValue("auto")

	plannedNa := unknownRadioTableModel()
	plannedNa.Radio = types.StringValue("na")
	plannedNa.Channel = types.StringValue("149")
	plannedNa.Ht = types.Int64Value(80)
	plannedNa.TxPowerMode = types.StringValue("auto")

	// Device order is reversed relative to the declared order.
	actualNa := unknownRadioTableModel()
	actualNa.Radio = types.StringValue("na")
	actualNa.Channel = types.StringValue("149")
	actualNa.Ht = types.Int64Value(80)
	actualNa.TxPowerMode = types.StringValue("auto")
	actualNa.Name = types.StringValue("wifi1")
	actualNa.AntennaGain = types.Int64Value(6)
	actualNa.AntennaID = types.Int64Value(-1)
	actualNa.MinRssi = types.Int64Null()
	actualNa.MinRssiEnabled = types.BoolValue(false)
	actualNa.TxPower = types.StringNull()
	actualNa.AssistedRoamingEnabled = types.BoolValue(false)
	actualNa.AssistedRoamingRssi = types.Int64Null()
	actualNa.Dfs = types.BoolValue(false)
	actualNa.HardNoiseFloorEnabled = types.BoolValue(false)
	actualNa.LoadbalanceEnabled = types.BoolValue(false)
	actualNa.Maxsta = types.Int64Null()
	actualNa.SensLevel = types.Int64Null()
	actualNa.SensLevelEnabled = types.BoolValue(false)
	actualNa.VwireEnabled = types.BoolValue(false)

	actualNg := actualNa
	actualNg.Radio = types.StringValue("ng")
	actualNg.Channel = types.StringValue("11")
	actualNg.Ht = types.Int64Value(20)
	actualNg.Name = types.StringValue("wifi0")
	actualNg.AntennaGain = types.Int64Value(4)

	planned := radioTableListFromModels(t, plannedNg, plannedNa)
	actual := radioTableListFromModels(t, actualNa, actualNg)

	got, diags := r.reconcileRadioTableWithPlan(ctx, planned, actual)
	if diags.HasError() {
		t.Fatalf("reconcileRadioTableWithPlan: %v", diags)
	}

	elems := got.Elements()
	if len(elems) != 2 {
		t.Fatalf("got %d elements, want 2 (planned count preserved)", len(elems))
	}

	var first, second radioTableModel
	firstObj, ok := elems[0].(types.Object)
	if !ok {
		t.Fatalf("first element is %T, want types.Object", elems[0])
	}
	if d := firstObj.As(ctx, &first, basetypes.ObjectAsOptions{}); d.HasError() {
		t.Fatalf("decode first: %v", d)
	}
	secondObj, ok := elems[1].(types.Object)
	if !ok {
		t.Fatalf("second element is %T, want types.Object", elems[1])
	}
	if d := secondObj.As(ctx, &second, basetypes.ObjectAsOptions{}); d.HasError() {
		t.Fatalf("decode second: %v", d)
	}

	if first.Radio.ValueString() != "ng" || second.Radio.ValueString() != "na" {
		t.Errorf("planned order not preserved: got %s, %s",
			first.Radio.ValueString(), second.Radio.ValueString())
	}
	if first.Channel.ValueString() != "11" {
		t.Errorf("declared channel lost: %s", first.Channel)
	}
	if first.Name.ValueString() != "wifi0" {
		t.Errorf("unknown name not resolved from matching band: %s", first.Name)
	}
	if first.AntennaGain.ValueInt64() != 4 {
		t.Errorf("unknown antenna_gain not resolved from matching band: %s", first.AntennaGain)
	}
	if second.Name.ValueString() != "wifi1" {
		t.Errorf("unknown name not resolved for na: %s", second.Name)
	}
	if first.IsUnknownAny() {
		t.Errorf("reconciled first entry still carries Unknown values: %+v", first)
	}
	if second.IsUnknownAny() {
		t.Errorf("reconciled second entry still carries Unknown values: %+v", second)
	}

	// A null/unknown plan (radio_table not declared) keeps the applied value.
	nullPlan := types.ListNull(types.ObjectType{AttrTypes: radioTableAttrTypes()})
	kept, diags := r.reconcileRadioTableWithPlan(ctx, nullPlan, actual)
	if diags.HasError() {
		t.Fatalf("null plan reconcile: %v", diags)
	}
	if !kept.Equal(actual) {
		t.Errorf("null plan must keep applied value")
	}
}

// IsUnknownAny reports whether any sub-attribute of the model is Unknown.
func (m radioTableModel) IsUnknownAny() bool {
	for _, v := range []attr.Value{
		m.Radio, m.Channel, m.Ht, m.TxPower, m.TxPowerMode,
		m.MinRssiEnabled, m.MinRssi, m.AntennaGain, m.AntennaID,
		m.AssistedRoamingEnabled, m.AssistedRoamingRssi, m.Dfs,
		m.HardNoiseFloorEnabled, m.LoadbalanceEnabled, m.Maxsta,
		m.Name, m.SensLevel, m.SensLevelEnabled, m.VwireEnabled,
	} {
		if v.IsUnknown() {
			return true
		}
	}
	return false
}

// TestAccDeviceFramework_radioTable applies the exact configuration shape from
// #427 — radio_table entries declaring only radio/channel/ht/tx_power_mode —
// against a simulated UAP. Before the fix the controller rejected the PUT with
// api.err.InvalidPayload (zero-filled undeclared fields) or, with those fields
// merely omitted, api.err.MissingValue key "radio" (missing hardware radio
// name). Both halves are exercised: unknowns stay off the wire, and the
// controller-required name is echoed from the device.
func TestAccDeviceFramework_radioTable(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDeviceFrameworkConfig_radioTable("6", "36"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_device.test_ap", "adopted", "true"),
					resource.TestCheckResourceAttr("unifi_device.test_ap", "radio_table.#", "2"),
					resource.TestCheckResourceAttr(
						"unifi_device.test_ap",
						"radio_table.0.radio",
						"ng",
					),
					resource.TestCheckResourceAttr(
						"unifi_device.test_ap",
						"radio_table.0.channel",
						"6",
					),
					resource.TestCheckResourceAttr(
						"unifi_device.test_ap",
						"radio_table.1.radio",
						"na",
					),
					resource.TestCheckResourceAttr(
						"unifi_device.test_ap",
						"radio_table.1.channel",
						"36",
					),
					// The hardware radio name must be echoed from the device.
					resource.TestCheckResourceAttrSet("unifi_device.test_ap", "radio_table.0.name"),
					resource.TestCheckResourceAttrSet("unifi_device.test_ap", "radio_table.1.name"),
				),
			},
			{
				// In-place channel change on the declared entries.
				Config: testAccDeviceFrameworkConfig_radioTable("11", "40"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_device.test_ap",
						"radio_table.0.channel",
						"11",
					),
					resource.TestCheckResourceAttr(
						"unifi_device.test_ap",
						"radio_table.1.channel",
						"40",
					),
				),
			},
		},
	})
}

func testAccDeviceFrameworkConfig_radioTable(ngChannel, naChannel string) string {
	return fmt.Sprintf(`
resource "unifi_device" "test_ap" {
	mac  = "00:15:6d:00:00:01"
	name = "Test AP Radio"
	allow_adoption    = true
	forget_on_destroy = false

	radio_table = [
		{
			radio         = "ng"
			channel       = %q
			ht            = 20
			tx_power_mode = "auto"
		},
		{
			radio         = "na"
			channel       = %q
			ht            = 40
			tx_power_mode = "auto"
		},
	]
}
`, ngChannel, naChannel)
}
