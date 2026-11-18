package unifi

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Network 11 keeps the mDNS network selection in the v2 global network
// settings. The legacy mdns setting only mirrors it and ignores writes to it,
// so enabled_for = "some" read back as "all" (#726).

// network11StoredMdns is the legacy mdns setting before the fake mirrors the
// global network settings into it.
func network11StoredMdns() []map[string]any {
	return []map[string]any{
		{
			"key":                     "mdns",
			"mode":                    "auto",
			"enabled_for":             "all",
			"enabled_for_network_ids": []any{},
			"predefined_services":     []any{},
			"custom_services":         []any{},
		},
	}
}

// network11Global is a global network settings object with keys besides the
// mDNS ones, which a write must keep.
func network11Global(enabledFor string, ids ...any) *fakeGlobalNetwork {
	return &fakeGlobalNetwork{Stored: map[string]any{
		"default_security_posture":      "allow_all",
		"igmp_snooping_for":             "some",
		"igmp_snooping_for_network_ids": []any{"net-a"},
		"mdns_enabled_for":              enabledFor,
		"mdns_enabled_for_network_ids":  append([]any{}, ids...),
	}}
}

// mdnsSelection configures only the network selection; ids is a list,
// or tfUnknown for an unset (computed) list.
func mdnsSelection(enabledFor string, ids any) map[string]any {
	return map[string]any{
		"mdns": map[string]any{
			"mode":                    "auto",
			"enabled_for":             enabledFor,
			"enabled_for_network_ids": ids,
			"predefined_services":     nil,
			"custom_services":         nil,
		},
	}
}

func stateNetworkIDs(t *testing.T, o types.Object) []string {
	t.Helper()
	l, _ := o.Attributes()["enabled_for_network_ids"].(types.List)
	var ids []string
	if d := l.ElementsAs(context.Background(), &ids, false); d.HasError() {
		t.Fatalf("reading enabled_for_network_ids: %v", d)
	}
	return ids
}

func Test_settingResource_mdnsNetwork11_writesSelection(t *testing.T) {
	g := network11Global("all", "net-a", "net-b", "net-c")
	r, _ := newSettingsFakeController(t, network11StoredMdns(), g)

	resp := createSettings(t, r, mdnsSelection("some", []any{"net-a", "net-b"}))
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}

	if len(g.Puts) != 1 {
		t.Fatalf("global network PUTs = %d, want 1", len(g.Puts))
	}
	put := g.Puts[0]
	if put["mdns_enabled_for"] != "some" {
		t.Errorf("mdns_enabled_for written = %v", put["mdns_enabled_for"])
	}
	if !reflect.DeepEqual(put["mdns_enabled_for_network_ids"], []any{"net-a", "net-b"}) {
		t.Errorf("mdns_enabled_for_network_ids written = %v", put["mdns_enabled_for_network_ids"])
	}
	// The object also holds IGMP and security settings; they go back unchanged.
	if put["default_security_posture"] != "allow_all" ||
		!reflect.DeepEqual(put["igmp_snooping_for_network_ids"], []any{"net-a"}) {
		t.Errorf("unmodelled keys not kept: %v", put)
	}

	state := stateModel(t, resp)
	if got := strAttr(state.Mdns, "enabled_for"); got != "some" {
		t.Errorf("state mdns.enabled_for = %q, want some", got)
	}
	if got := stateNetworkIDs(t, state.Mdns); !reflect.DeepEqual(got, []string{"net-a", "net-b"}) {
		t.Errorf("state mdns.enabled_for_network_ids = %v", got)
	}
}

func Test_settingResource_mdnsNetwork11_unsetListKeepsSelection(t *testing.T) {
	g := network11Global("all", "net-a", "net-b", "net-c")
	r, _ := newSettingsFakeController(t, network11StoredMdns(), g)

	resp := createSettings(t, r, mdnsSelection("some", tfUnknown))
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
	if len(g.Puts) != 1 ||
		!reflect.DeepEqual(
			g.Puts[0]["mdns_enabled_for_network_ids"],
			[]any{"net-a", "net-b", "net-c"},
		) {
		t.Fatalf("global network PUTs = %v, want the stored list kept", g.Puts)
	}
	if got := stateNetworkIDs(t, stateModel(t, resp).Mdns); len(got) != 3 {
		t.Errorf("state mdns.enabled_for_network_ids = %v", got)
	}
}

// Switching to "all" keeps the stored list; the mdns setting then reports an
// empty one, as the controller does while mDNS covers every network.
func Test_settingResource_mdnsNetwork11_allKeepsStoredList(t *testing.T) {
	g := network11Global("some", "net-a")
	r, _ := newSettingsFakeController(t, network11StoredMdns(), g)

	resp := createSettings(t, r, mdnsSelection("all", tfUnknown))
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
	if len(g.Puts) != 1 || g.Puts[0]["mdns_enabled_for"] != "all" ||
		!reflect.DeepEqual(g.Puts[0]["mdns_enabled_for_network_ids"], []any{"net-a"}) {
		t.Fatalf("global network PUTs = %v, want all with the stored list", g.Puts)
	}
	state := stateModel(t, resp)
	if got := strAttr(state.Mdns, "enabled_for"); got != "all" {
		t.Errorf("state mdns.enabled_for = %q, want all", got)
	}
	if got := stateNetworkIDs(t, state.Mdns); len(got) != 0 {
		t.Errorf("state mdns.enabled_for_network_ids = %v, want empty", got)
	}
}

func Test_settingResource_mdnsNetwork11_someWithoutNetworksFails(t *testing.T) {
	g := network11Global("none")
	r, puts := newSettingsFakeController(t, network11StoredMdns(), g)

	resp := createSettings(t, r, mdnsSelection("some", tfUnknown))
	if !resp.Diagnostics.HasError() ||
		!strings.Contains(resp.Diagnostics.Errors()[0].Summary(), "No mDNS Networks Selected") {
		t.Fatalf("Create diagnostics = %v, want No mDNS Networks Selected", resp.Diagnostics)
	}
	if len(g.Puts) != 0 {
		t.Errorf("global network written despite the error: %v", g.Puts)
	}
	if _, ok := puts()["mdns"]; ok {
		t.Error("mdns setting written despite the error")
	}
}

func Test_settingResource_mdnsNetwork11_unchangedSelectionNotWritten(t *testing.T) {
	g := network11Global("some", "net-a")
	r, _ := newSettingsFakeController(t, network11StoredMdns(), g)

	resp := createSettings(t, r, mdnsSelection("some", []any{"net-a"}))
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
	if len(g.Puts) != 0 {
		t.Errorf("unchanged selection written: %v", g.Puts)
	}
}

// Earlier controllers derive the list from the networks' mdns_enabled flags,
// so a list that differs is rejected before anything is written.
func Test_settingResource_mdnsLegacy_rejectsDifferentNetworkIDs(t *testing.T) {
	r, puts := newSettingsFakeController(t, storedMdns())

	resp := createSettings(t, r, mdnsSelection("some", []any{"net-guest"}))
	if !resp.Diagnostics.HasError() ||
		!strings.Contains(resp.Diagnostics.Errors()[0].Summary(), "mDNS Networks Not Writable") {
		t.Fatalf("Create diagnostics = %v, want mDNS Networks Not Writable", resp.Diagnostics)
	}
	if _, ok := puts()["mdns"]; ok {
		t.Error("mdns setting written despite the error")
	}
}

// A configured list the controller returns in another order keeps the
// configured order, so the read-back is not an inconsistent result.
func Test_settingResource_mdnsLegacy_sameNetworkIDsKeepConfiguredOrder(t *testing.T) {
	r, _ := newSettingsFakeController(t, storedMdns())

	resp := createSettings(t, r, mdnsSelection("some", []any{"net-iot", "net-default"}))
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
	got := stateNetworkIDs(t, stateModel(t, resp).Mdns)
	if !reflect.DeepEqual(got, []string{"net-iot", "net-default"}) {
		t.Errorf("state mdns.enabled_for_network_ids = %v, want the configured order", got)
	}
}
