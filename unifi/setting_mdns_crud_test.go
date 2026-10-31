package unifi

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi/settings"
)

// mdns as UniFi Network 10.6.106 stores it.
func storedMdns() []map[string]any {
	return []map[string]any{
		{
			"key":                     "mdns",
			"mode":                    "all",
			"enabled_for":             "some",
			"enabled_for_network_ids": []any{"net-default", "net-iot"},
			"predefined_services":     []any{},
			"custom_services":         []any{},
		},
	}
}

// mdnsConfig is the planned value: enabled_for_network_ids is computed
// and unknown until the controller reports it.
func mdnsConfig(enabledFor string) map[string]any {
	return map[string]any{
		"mdns": map[string]any{
			"mode":                    "all",
			"enabled_for":             enabledFor,
			"enabled_for_network_ids": tfUnknown,
			"predefined_services":     []any{"airplay"},
			"custom_services": []any{
				map[string]any{"name": "Printer", "address": "_ipp._tcp.local"},
			},
		},
	}
}

func createSettings(
	t *testing.T,
	r *settingResource,
	attrs map[string]any,
) *fwresource.CreateResponse {
	t.Helper()
	s := resourceSchema(t, r)
	resp := &fwresource.CreateResponse{State: tfState(t, s, nil), Identity: tfIdentity(t, r)}
	r.Create(context.Background(), fwresource.CreateRequest{
		Plan: tfPlan(t, s, attrs), Config: tfConfig(t, s, attrs),
	}, resp)
	return resp
}

func stateModel(t *testing.T, resp *fwresource.CreateResponse) settingResourceModel {
	t.Helper()
	var m settingResourceModel
	if d := resp.State.Get(context.Background(), &m); d.HasError() {
		t.Fatalf("reading state: %v", d)
	}
	return m
}

// storedSettingSet changes a stored setting behind the provider's back.
func storedSettingSet(t *testing.T, r *settingResource, key, field string, v any) {
	t.Helper()
	err := r.client.UpdateSetting(context.Background(), "default", &settings.RawSetting{
		BaseSetting: settings.BaseSetting{Key: key},
		Data:        map[string]any{field: v},
	})
	if err != nil {
		t.Fatalf("changing stored %s: %v", key, err)
	}
}

func strAttr(o types.Object, name string) string {
	s, _ := o.Attributes()[name].(types.String)
	return s.ValueString()
}

func Test_settingResource_createMdns(t *testing.T) {
	r, puts := newSettingsFakeController(t, storedMdns())

	resp := createSettings(t, r, mdnsConfig("some"))
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}

	written := puts()
	mdns := written["mdns"]
	if !reflect.DeepEqual(mdns["predefined_services"], []any{map[string]any{"code": "airplay"}}) {
		t.Errorf("mdns predefined_services = %v", mdns["predefined_services"])
	}
	wantCustom := []any{map[string]any{"name": "Printer", "address": "_ipp._tcp.local"}}
	if !reflect.DeepEqual(mdns["custom_services"], wantCustom) {
		t.Errorf("mdns custom_services = %v, want %v", mdns["custom_services"], wantCustom)
	}
	if got := written["mdns"]["mode"]; got != "all" {
		t.Errorf("mdns mode written = %v", got)
	}

	state := stateModel(t, resp)
	if got := strAttr(state.Mdns, "enabled_for"); got != "some" {
		t.Errorf("state mdns.enabled_for = %q", got)
	}
	if svcs, _ := state.Mdns.Attributes()["custom_services"].(types.List); len(
		svcs.Elements(),
	) != 1 {
		t.Errorf("state mdns.custom_services = %v", svcs)
	}
}

func Test_settingResource_updateAndReadMdns(t *testing.T) {
	r, puts := newSettingsFakeController(t, storedMdns())
	created := createSettings(t, r, mdnsConfig("some"))
	if created.Diagnostics.HasError() {
		t.Fatalf("Create: %v", created.Diagnostics)
	}
	puts()

	s := resourceSchema(t, r)
	attrs := mdnsConfig("all")
	upd := &fwresource.UpdateResponse{State: created.State, Identity: created.Identity}
	r.Update(context.Background(), fwresource.UpdateRequest{
		Plan: tfPlan(t, s, attrs), Config: tfConfig(t, s, attrs), State: created.State,
	}, upd)
	if upd.Diagnostics.HasError() {
		t.Fatalf("Update: %v", upd.Diagnostics)
	}
	written := puts()["mdns"]
	if written["enabled_for"] != "all" {
		t.Errorf("mdns enabled_for written = %v", written["enabled_for"])
	}
	// The list is the controller's; a write never changes it.
	if ids, ok := written["enabled_for_network_ids"]; ok &&
		!reflect.DeepEqual(ids, []any{"net-default", "net-iot"}) {
		t.Errorf("mdns enabled_for_network_ids written = %v", ids)
	}

	// The controller re-derives the list when a network's mdns_enabled flag
	// changes; a read picks that up.
	storedSettingSet(t, r, "mdns", "enabled_for_network_ids", []any{"net-default"})
	read := &fwresource.ReadResponse{State: upd.State, Identity: upd.Identity}
	r.Read(
		context.Background(),
		fwresource.ReadRequest{State: upd.State, Identity: upd.Identity},
		read,
	)
	if read.Diagnostics.HasError() {
		t.Fatalf("Read: %v", read.Diagnostics)
	}
	var m settingResourceModel
	read.State.Get(context.Background(), &m)
	ids, _ := m.Mdns.Attributes()["enabled_for_network_ids"].(types.List)
	if want := []attr.Value{types.StringValue("net-default")}; !reflect.DeepEqual(
		ids.Elements(),
		want,
	) {
		t.Errorf("read mdns.enabled_for_network_ids = %v", ids)
	}
}

// A controller that has never stored mdns: the write starts from an empty
// setting.
func Test_settingResource_createMdnsNotStored(t *testing.T) {
	r, puts := newSettingsFakeController(t, nil)
	resp := createSettings(t, r, mdnsConfig("some"))
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
	written := puts()
	if written["mdns"]["enabled_for"] != "some" {
		t.Errorf("writes = %v", written)
	}
}

func Test_settingResource_mdnsErrors(t *testing.T) {
	for _, tt := range []struct{ key, fail, want string }{
		{"mdns", "read", "Error Reading mDNS Setting"},
		{"mdns", "write", "Error Updating mDNS Setting"},
		{"mdns", "readback", "Error Reading mDNS Setting"},
	} {
		t.Run(tt.key+"/"+tt.fail, func(t *testing.T) {
			stored := storedMdns()
			for _, s := range stored {
				if s["key"] == tt.key {
					s["_fail"] = tt.fail
				}
			}
			r, _ := newSettingsFakeController(t, stored)
			resp := createSettings(t, r, mdnsConfig("some"))
			if !resp.Diagnostics.HasError() {
				t.Fatal("Create succeeded, want an error")
			}
			if got := resp.Diagnostics.Errors()[0].Summary(); got != tt.want {
				t.Errorf("error = %q, want %q", got, tt.want)
			}
		})
	}
}

// An update whose write fails reports it and keeps the prior state.
func Test_settingResource_updateMdnsWriteError(t *testing.T) {
	stored := storedMdns()
	r, _ := newSettingsFakeController(t, stored)
	created := createSettings(t, r, mdnsConfig("some"))
	if created.Diagnostics.HasError() {
		t.Fatalf("Create: %v", created.Diagnostics)
	}
	stored[0]["_fail"] = "write"

	s := resourceSchema(t, r)
	attrs := mdnsConfig("all")
	upd := &fwresource.UpdateResponse{State: created.State, Identity: created.Identity}
	r.Update(context.Background(), fwresource.UpdateRequest{
		Plan: tfPlan(t, s, attrs), Config: tfConfig(t, s, attrs), State: created.State,
	}, upd)
	if !upd.Diagnostics.HasError() {
		t.Fatal("Update succeeded, want a write error")
	}
}

// mdns configured alone: only mdns is written and the other blocks stay null.
func Test_settingResource_mdnsAlone(t *testing.T) {
	r, puts := newSettingsFakeController(t, storedMdns())
	resp := createSettings(t, r, mdnsConfig("some"))
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
	if got := puts(); len(got) != 1 || got["mdns"] == nil {
		t.Errorf("writes = %v, want only mdns", got)
	}
	if state := stateModel(t, resp); !state.RadioAi.IsNull() {
		t.Errorf("unconfigured radio_ai = %v, want null", state.RadioAi)
	}
}

// An Update that leaves mdns unchanged does not write it.
func Test_settingResource_updateMdnsUnchangedSkipsWrite(t *testing.T) {
	r, puts := newSettingsFakeController(t, storedMdns())
	created := createSettings(t, r, mdnsConfig("some"))
	if created.Diagnostics.HasError() {
		t.Fatalf("Create: %v", created.Diagnostics)
	}
	puts()

	s := resourceSchema(t, r)
	attrs := mdnsConfig("some")
	upd := &fwresource.UpdateResponse{State: created.State, Identity: created.Identity}
	r.Update(context.Background(), fwresource.UpdateRequest{
		Plan: tfPlan(t, s, attrs), Config: tfConfig(t, s, attrs), State: created.State,
	}, upd)
	if upd.Diagnostics.HasError() {
		t.Fatalf("Update: %v", upd.Diagnostics)
	}
	if got := puts(); got["mdns"] != nil {
		t.Errorf("unchanged mdns was written: %v", got["mdns"])
	}
}

func Test_keepNullAttributes_passesThroughWithoutPrior(t *testing.T) {
	fresh := types.ObjectValueMust(
		map[string]attr.Type{"a": types.StringType},
		map[string]attr.Value{"a": types.StringValue("x")},
	)
	for _, prior := range []types.Object{
		types.ObjectNull(fresh.AttributeTypes(context.Background())),
		types.ObjectUnknown(fresh.AttributeTypes(context.Background())),
	} {
		if got := keepNullAttributes(prior, fresh, nil); !got.Equal(fresh) {
			t.Errorf("keepNullAttributes(%v) = %v, want fresh", prior, got)
		}
	}
}
