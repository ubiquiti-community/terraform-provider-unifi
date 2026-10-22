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

// mdns and radio_ai as UniFi Network 10.6.106 stores them: numbers in lists
// are strings, Radio AI carries fields this configuration never sets.
func storedMdnsRadioAi() []map[string]any {
	return []map[string]any{
		{
			"key":                     "mdns",
			"mode":                    "all",
			"enabled_for":             "some",
			"enabled_for_network_ids": []any{"net-default", "net-iot"},
			"predefined_services":     []any{},
			"custom_services":         []any{},
		},
		{
			"key":                             "radio_ai",
			"enabled":                         true,
			"cron_expr":                       "0 4 * * *",
			"setting_preference":              "auto",
			"auto_channel_presets_type":       "custom",
			"auto_adjust_channels_to_country": true,
			"optimize":                        []any{"channel"},
			"radios":                          []any{"ng", "na"},
			"channels_ng":                     []any{"1", "6", "11"},
			"channels_na":                     []any{"36", "40", "44", "48"},
			"ht_modes_ng":                     []any{"20"},
			"ht_modes_na":                     []any{"20", "40"},
			"channels_blacklist": []any{
				map[string]any{"channel": 2, "channel_width": 20, "radio": "ng"},
			},
			"exclude_devices": []any{},
		},
	}
}

// mdnsRadioAiConfig is the planned value: enabled_for_network_ids is computed
// and unknown until the controller reports it.
func mdnsRadioAiConfig(enabledFor string) map[string]any {
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
		"radio_ai": map[string]any{
			"enabled":     true,
			"cron_expr":   "0 3 * * *",
			"radios":      []any{"ng", "na"},
			"channels_ng": []any{1, 6, 11},
			"ht_modes_na": []any{20, 40},
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

// storedSetting returns the setting as the controller holds it after the writes.
func storedSetting(t *testing.T, r *settingResource, key string) map[string]any {
	t.Helper()
	all, err := r.client.ListSettings(context.Background(), "default")
	if err != nil {
		t.Fatalf("listing settings: %v", err)
	}
	for _, s := range all {
		if s.Key == key {
			return s.Data
		}
	}
	t.Fatalf("setting %s not stored", key)
	return nil
}

func strAttr(o types.Object, name string) string {
	s, _ := o.Attributes()[name].(types.String)
	return s.ValueString()
}

func Test_settingResource_createMdnsRadioAi(t *testing.T) {
	r, puts := newSettingsFakeController(t, storedMdnsRadioAi())

	resp := createSettings(t, r, mdnsRadioAiConfig("some"))
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
	if got := written["radio_ai"]["cron_expr"]; got != "0 3 * * *" {
		t.Errorf("radio_ai cron_expr written = %v", got)
	}
	// What the configuration leaves out keeps the controller's value.
	ai := storedSetting(t, r, "radio_ai")
	if ai["auto_channel_presets_type"] != "custom" || ai["channels_blacklist"] == nil {
		t.Errorf("radio_ai lost stored fields: %v", ai)
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
	// Left out of the configuration: not tracked, stays null after read-back.
	if v := state.RadioAi.Attributes()["channels_na"]; !v.IsNull() {
		t.Errorf("state radio_ai.channels_na = %v, want null", v)
	}
	if got := strAttr(state.RadioAi, "cron_expr"); got != "0 3 * * *" {
		t.Errorf("state radio_ai.cron_expr = %q", got)
	}
}

func Test_settingResource_updateAndReadMdns(t *testing.T) {
	r, puts := newSettingsFakeController(t, storedMdnsRadioAi())
	created := createSettings(t, r, mdnsRadioAiConfig("some"))
	if created.Diagnostics.HasError() {
		t.Fatalf("Create: %v", created.Diagnostics)
	}
	puts()

	s := resourceSchema(t, r)
	attrs := mdnsRadioAiConfig("all")
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

// A controller that has never stored mdns or radio_ai: the write starts from
// an empty setting.
func Test_settingResource_createMdnsRadioAiNotStored(t *testing.T) {
	r, puts := newSettingsFakeController(t, nil)
	resp := createSettings(t, r, mdnsRadioAiConfig("some"))
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
	written := puts()
	if written["mdns"]["enabled_for"] != "some" || written["radio_ai"]["enabled"] != true {
		t.Errorf("writes = %v", written)
	}
	state := stateModel(t, resp)
	// The controller reports no auto_enabled; it reads as false.
	if v, _ := state.RadioAi.Attributes()["auto_enabled"].(types.Bool); !v.IsNull() &&
		v.ValueBool() {
		t.Errorf("radio_ai.auto_enabled = %v", v)
	}
}

func Test_settingResource_mdnsRadioAiErrors(t *testing.T) {
	for _, tt := range []struct{ key, fail, want string }{
		{"mdns", "read", "Error Reading mDNS Setting"},
		{"radio_ai", "read", "Error Reading Radio AI Setting"},
		{"mdns", "write", "Error Updating mDNS Setting"},
		{"radio_ai", "write", "Error Updating Radio AI Setting"},
		{"mdns", "readback", "Error Reading mDNS Setting"},
		{"radio_ai", "readback", "Error Reading Radio AI Setting"},
	} {
		t.Run(tt.key+"/"+tt.fail, func(t *testing.T) {
			stored := storedMdnsRadioAi()
			for _, s := range stored {
				if s["key"] == tt.key {
					s["_fail"] = tt.fail
				}
			}
			r, _ := newSettingsFakeController(t, stored)
			resp := createSettings(t, r, mdnsRadioAiConfig("some"))
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
	stored := storedMdnsRadioAi()
	r, _ := newSettingsFakeController(t, stored)
	created := createSettings(t, r, mdnsRadioAiConfig("some"))
	if created.Diagnostics.HasError() {
		t.Fatalf("Create: %v", created.Diagnostics)
	}
	stored[0]["_fail"] = "write"

	s := resourceSchema(t, r)
	attrs := mdnsRadioAiConfig("all")
	upd := &fwresource.UpdateResponse{State: created.State, Identity: created.Identity}
	r.Update(context.Background(), fwresource.UpdateRequest{
		Plan: tfPlan(t, s, attrs), Config: tfConfig(t, s, attrs), State: created.State,
	}, upd)
	if !upd.Diagnostics.HasError() {
		t.Fatal("Update succeeded, want a write error")
	}
}

// Only one of the two blocks configured: the other one stays null.
func Test_settingResource_mdnsOrRadioAiAlone(t *testing.T) {
	for _, block := range []string{"mdns", "radio_ai"} {
		t.Run(block, func(t *testing.T) {
			r, puts := newSettingsFakeController(t, storedMdnsRadioAi())
			attrs := map[string]any{block: mdnsRadioAiConfig("some")[block]}
			resp := createSettings(t, r, attrs)
			if resp.Diagnostics.HasError() {
				t.Fatalf("Create: %v", resp.Diagnostics)
			}
			if got := puts(); len(got) != 1 || got[block] == nil {
				t.Errorf("writes = %v, want only %s", got, block)
			}
			state := stateModel(t, resp)
			other := state.RadioAi
			if block == "radio_ai" {
				other = state.Mdns
			}
			if !other.IsNull() {
				t.Errorf("unconfigured block = %v, want null", other)
			}
		})
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
