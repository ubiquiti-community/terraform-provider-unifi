package unifi

import (
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func stringSet(t *testing.T, ids ...string) types.Set {
	t.Helper()
	vals := make([]attr.Value, len(ids))
	for i, id := range ids {
		vals[i] = types.StringValue(id)
	}
	s, d := types.SetValue(types.StringType, vals)
	if d.HasError() {
		t.Fatalf("building string set: %v", d)
	}
	return s
}

func setStrings(t *testing.T, s types.Set) []string {
	t.Helper()
	var out []string
	if d := s.ElementsAs(context.Background(), &out, false); d.HasError() {
		t.Fatalf("reading string set: %v", d)
	}
	return out
}

// A configured tagged_networkconf_ids must reach the API payload. Before, it was
// never copied into DevicePortOverrides, so the mgmt_network_id advice to tag
// the VLAN on the uplink first silently did nothing.
func TestFrameworkToPortOverrides_SendsTaggedNetworkIDs(t *testing.T) {
	r := &deviceResource{}
	set := portOverrideSetWith(t, map[string]attr.Value{
		"index":                  types.Int64Value(5),
		"tagged_vlan_mgmt":       types.StringValue("custom"),
		"tagged_networkconf_ids": stringSet(t, "net-b", "net-a"),
	})

	pos, diags := r.frameworkToPortOverrides(context.Background(), set)
	if diags.HasError() {
		t.Fatalf("frameworkToPortOverrides errored: %v", diags)
	}
	if len(pos) != 1 {
		t.Fatalf("got %d port overrides, want 1", len(pos))
	}
	if !sameStringSet(pos[0].TaggedNetworkIDs, []string{"net-a", "net-b"}) ||
		len(pos[0].TaggedNetworkIDs) != 2 {
		t.Errorf("TaggedNetworkIDs = %v, want [net-a net-b]", pos[0].TaggedNetworkIDs)
	}

	b, err := json.Marshal(pos[0])
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(b, &wire); err != nil {
		t.Fatal(err)
	}
	if _, ok := wire["tagged_networkconf_ids"]; !ok {
		t.Errorf("tagged_networkconf_ids missing from payload: %s", b)
	}
}

// Leaving tagged_networkconf_ids unset must not put it on the wire, and an
// explicit empty set must yield a non-nil empty slice so the merge does not
// mistake it for "unset" and carry the controller's value forward.
func TestFrameworkToPortOverrides_TaggedNetworkIDsUnsetAndEmpty(t *testing.T) {
	r := &deviceResource{}

	unset := portOverrideSetWith(t, map[string]attr.Value{"index": types.Int64Value(1)})
	pos, diags := r.frameworkToPortOverrides(context.Background(), unset)
	if diags.HasError() {
		t.Fatalf("frameworkToPortOverrides errored: %v", diags)
	}
	if pos[0].TaggedNetworkIDs != nil {
		t.Errorf("unset: TaggedNetworkIDs = %#v, want nil", pos[0].TaggedNetworkIDs)
	}
	b, err := json.Marshal(pos[0])
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(b, &wire); err != nil {
		t.Fatal(err)
	}
	if _, ok := wire["tagged_networkconf_ids"]; ok {
		t.Errorf("unset: tagged_networkconf_ids must not be sent: %s", b)
	}

	empty := portOverrideSetWith(t, map[string]attr.Value{
		"index":                  types.Int64Value(1),
		"tagged_networkconf_ids": stringSet(t),
	})
	pos, diags = r.frameworkToPortOverrides(context.Background(), empty)
	if diags.HasError() {
		t.Fatalf("frameworkToPortOverrides errored: %v", diags)
	}
	if pos[0].TaggedNetworkIDs == nil || len(pos[0].TaggedNetworkIDs) != 0 {
		t.Errorf("empty: TaggedNetworkIDs = %#v, want non-nil empty", pos[0].TaggedNetworkIDs)
	}
}

func TestMergePortOverridesByIndex_TaggedNetworkIDs(t *testing.T) {
	current := unifi.DevicePortOverrides{
		PortIDX:          ptrInt64(5),
		Name:             "Uplink",
		TaggedVLANMgmt:   "custom",
		TaggedNetworkIDs: []string{"net-b", "net-a"},
	}

	tests := []struct {
		name     string
		declared []string
		want     []string
	}{
		// Not configured: keep the controller's tagged networks rather than
		// deleting them by sending the port without the field.
		{name: "unset carries controller value", declared: nil, want: []string{"net-b", "net-a"}},
		// Same networks in another order: keep the controller's order so the
		// entry marshals identically and no write is triggered.
		{
			name:     "same set keeps controller order",
			declared: []string{"net-a", "net-b"},
			want:     []string{"net-b", "net-a"},
		},
		{
			name:     "changed set is sent",
			declared: []string{"net-a", "net-c"},
			want:     []string{"net-a", "net-c"},
		},
		{name: "empty set clears", declared: []string{}, want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			declared := unifi.DevicePortOverrides{
				PortIDX:          ptrInt64(5),
				Name:             "Uplink",
				TaggedVLANMgmt:   "custom",
				TaggedNetworkIDs: tt.declared,
			}
			got := mergePortOverridesByIndex(
				[]unifi.DevicePortOverrides{current},
				[]unifi.DevicePortOverrides{declared},
			)
			if !reflect.DeepEqual(got[0].TaggedNetworkIDs, tt.want) {
				t.Errorf("TaggedNetworkIDs = %#v, want %#v", got[0].TaggedNetworkIDs, tt.want)
			}
		})
	}
}

func TestPortOverridesToFramework_ReadsTaggedNetworkIDs(t *testing.T) {
	r := &deviceResource{}
	set, diags := r.portOverridesToFramework(context.Background(), []unifi.DevicePortOverrides{
		{PortIDX: ptrInt64(5), TaggedNetworkIDs: []string{"net-b", "net-a", "net-b"}},
	})
	if diags.HasError() {
		t.Fatalf("portOverridesToFramework errored: %v", diags)
	}
	var models []portOverrideModel
	if d := set.ElementsAs(context.Background(), &models, false); d.HasError() {
		t.Fatalf("reading set: %v", d)
	}
	got := setStrings(t, models[0].TaggedNetworkIDs)
	if !reflect.DeepEqual(got, []string{"net-a", "net-b"}) {
		t.Errorf("tagged_networkconf_ids = %v, want [net-a net-b]", got)
	}
}

func reconcileTagged(t *testing.T, prior types.Set, api []string) types.Set {
	t.Helper()
	r := &deviceResource{}
	priorSet := portOverrideSetWith(t, map[string]attr.Value{
		"index":                  types.Int64Value(5),
		"tagged_networkconf_ids": prior,
	})
	got, diags := r.reconcilePortOverrides(
		context.Background(),
		priorSet,
		[]unifi.DevicePortOverrides{
			{PortIDX: ptrInt64(5), TaggedNetworkIDs: api},
		},
		nil,
	)
	if diags.HasError() {
		t.Fatalf("reconcilePortOverrides errored: %v", diags)
	}
	var models []portOverrideModel
	if d := got.ElementsAs(context.Background(), &models, false); d.HasError() {
		t.Fatalf("reading set: %v", d)
	}
	return models[0].TaggedNetworkIDs
}

// The controller returning the configured networks in a different order (or
// with a duplicate) must read back equal to the configured set: no perpetual
// diff.
func TestReconcilePortOverrides_TaggedNetworkIDsOrderInsensitive(t *testing.T) {
	prior := stringSet(t, "net-a", "net-b")
	got := reconcileTagged(t, prior, []string{"net-b", "net-a", "net-a"})
	if !got.Equal(prior) {
		t.Errorf("read-back %v != configured %v (perpetual diff)", got, prior)
	}
}

func TestReconcilePortOverrides_TaggedNetworkIDsDrift(t *testing.T) {
	got := reconcileTagged(t, stringSet(t, "net-a", "net-b"), []string{"net-a"})
	if !reflect.DeepEqual(setStrings(t, got), []string{"net-a"}) {
		t.Errorf("drift not surfaced: got %v, want [net-a]", got)
	}
}

func TestReconcilePortOverrides_TaggedNetworkIDsEmptyAndUnset(t *testing.T) {
	empty := stringSet(t)
	if got := reconcileTagged(t, empty, nil); !got.Equal(empty) {
		t.Errorf("configured empty set read back as %v, want empty set", got)
	}
	// Unset in config stays null even when the controller has tagged
	// networks, so unmanaged values do not plan a removal.
	null := types.SetNull(types.StringType)
	if got := reconcileTagged(t, null, []string{"net-a"}); !got.IsNull() {
		t.Errorf("unset read back as %v, want null", got)
	}
}

// The controller (9.5 to 11.0) drops tagged_networkconf_ids from a port override and
// keeps excluded_networkconf_ids: a "custom" port tags every LAN network that
// is not excluded. A unifi_device_port with tagged networks therefore tagged
// every network of the site, and the apply failed with "planned set element
// ... does not correlate with any element in actual". The tagged set must be
// written as the excluded LAN networks, without the native one (the
// controller rejects that with api.err.NativeVlanCannotBeExcluded), and read
// back from them, so a network added later shows up as drift.
func TestDevicePortTaggedNetworks_network11(t *testing.T) {
	ctx := context.Background()
	sw, client := newFakeSwitch(t)
	network := func(id, purpose string) map[string]any {
		return map[string]any{"_id": id, "name": id, "purpose": purpose}
	}
	sw.network11 = true
	sw.networks = []any{
		network("lan", "corporate"), network("workers", "corporate"),
		network("kvm", "corporate"), network("orbs", "corporate"),
		network("guests", "guest"), network("vpn", "site-vpn"), network("wan", "wan"),
	}

	resp := devicePortCreate(t, client, map[string]any{
		"device_mac":             fakeSwitchMAC,
		"index":                  int64(1),
		"native_networkconf_id":  "orbs",
		"tagged_vlan_mgmt":       "custom",
		"tagged_networkconf_ids": []any{"workers"},
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
	stored := overrideAt(sw.overrides(), 1)
	if got, want := stored["excluded_networkconf_ids"], []any{
		"guests",
		"kvm",
		"lan",
	}; !reflect.DeepEqual(
		got,
		want,
	) {
		t.Errorf("stored excluded_networkconf_ids = %v, want %v", got, want)
	}
	assertTagged := func(step string, s tfsdk.State, want ...string) {
		t.Helper()
		var got []string
		if d := s.GetAttribute(ctx, path.Root("tagged_networkconf_ids"), &got); d.HasError() {
			t.Fatalf("%s: reading tagged_networkconf_ids: %v", step, d)
		}
		sort.Strings(got)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: tagged_networkconf_ids = %v, want %v", step, got, want)
		}
	}
	assertTagged("state after Create", resp.State, "workers")

	read := func() tfsdk.State {
		r := &devicePortResource{client: client}
		out := &fwresource.ReadResponse{State: resp.State, Identity: resp.Identity}
		r.Read(ctx, fwresource.ReadRequest{State: resp.State, Identity: resp.Identity}, out)
		if out.Diagnostics.HasError() {
			t.Fatalf("Read: %v", out.Diagnostics)
		}
		return out.State
	}
	assertTagged("refresh", read(), "workers")

	sw.mu.Lock()
	sw.networks = append(sw.networks, network("new", "corporate"))
	sw.mu.Unlock()
	assertTagged("refresh after a network was added", read(), "new", "workers")
}

// unifi_device writes the declared ports onto the controller's entries: a
// port that leaves native_networkconf_id unset keeps the controller's native
// network out of the exclusions, and an unchanged exclusion keeps the
// controller's order so nothing is written.
func TestExcludeUntaggedNetworks(t *testing.T) {
	idx := int64(2)
	current := []unifi.DevicePortOverrides{{
		PortIDX:            &idx,
		NATiveNetworkID:    "orbs",
		ExcludedNetworkIDs: []string{"lan", "kvm"},
	}}
	declared := []unifi.DevicePortOverrides{
		{PortIDX: &idx, TaggedNetworkIDs: []string{"workers"}},
		{Name: "untouched"},
	}
	excludeUntaggedNetworks(current, declared, []string{"kvm", "lan", "orbs", "workers"})
	if got := declared[0].ExcludedNetworkIDs; !reflect.DeepEqual(got, []string{"lan", "kvm"}) {
		t.Errorf("excluded = %v, want the controller's [lan kvm]", got)
	}
	if declared[1].ExcludedNetworkIDs != nil {
		t.Errorf("a port without tagged networks got excluded = %v", declared[1].ExcludedNetworkIDs)
	}
}
