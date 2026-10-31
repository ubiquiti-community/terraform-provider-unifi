package unifi

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi/settings"
)

// storedRadioAi is radio_ai as UniFi Network 10.6.109 stores it: channels and
// HT modes as numeric strings, no high_priority_devices key, auto_enabled
// beside enabled.
func storedRadioAi() []map[string]any {
	return []map[string]any{{
		"key":                             "radio_ai",
		"setting_preference":              "auto",
		"auto_adjust_channels_to_country": true,
		"channels_na":                     []any{"36", "40", "44", "48"},
		"ht_modes_na":                     []any{"20", "40"},
		"ht_modes_ng":                     []any{"20"},
		"auto_channel_presets_type":       "conservative",
		"channels_ng":                     []any{"1", "6", "11"},
		"enabled":                         false,
		"default":                         true,
		"optimize":                        []any{"channel"},
		"channels_blacklist": []any{
			map[string]any{"channel": float64(2), "channel_width": float64(20), "radio": "ng"},
		},
		"radios_configuration": []any{
			map[string]any{"dfs": false, "channel_width": float64(40), "radio": "na"},
			map[string]any{"dfs": false, "channel_width": float64(20), "radio": "ng"},
			map[string]any{"dfs": false, "channel_width": float64(160), "radio": "6e"},
		},
		"cron_expr":       "0 3 * * *",
		"auto_enabled":    false,
		"radios":          []any{"ng", "na"},
		"channels_6e":     []any{"5", "21"},
		"exclude_devices": []any{},
	}}
}

// radioAiCurrent is a radio_ai block holding the stored values, with
// high_priority_devices and auto_enabled left as given.
func radioAiCurrent(highPriority, autoEnabled any) map[string]any {
	return map[string]any{
		"enabled":                         false,
		"auto_enabled":                    autoEnabled,
		"cron_expr":                       "0 3 * * *",
		"radios":                          []any{"ng", "na"},
		"optimize":                        []any{"channel"},
		"channels_ng":                     []any{1, 6, 11},
		"channels_na":                     []any{36, 40, 44, 48},
		"channels_6e":                     []any{5, 21},
		"exclude_devices":                 []any{},
		"setting_preference":              "auto",
		"auto_channel_presets_type":       "conservative",
		"auto_adjust_channels_to_country": true,
		"ht_modes_ng":                     []any{20},
		"ht_modes_na":                     []any{20, 40},
		"high_priority_devices":           highPriority,
		"channels_blacklist": []any{
			map[string]any{"radio": "ng", "channel": 2, "channel_width": 20},
		},
		"radios_configuration": []any{
			map[string]any{"radio": "na", "channel_width": 40, "dfs": false},
			map[string]any{"radio": "ng", "channel_width": 20, "dfs": false},
			map[string]any{"radio": "6e", "channel_width": 160, "dfs": false},
		},
	}
}

// Live finding 1: a radio_ai attribute the plan carries as null from prior
// state (state written before the attribute existed) stays null after the
// apply instead of being filled from the controller, which Terraform rejects
// as an inconsistent result. Only the changed key is written.
func TestSettingRadioAiUnsetAttributesStayNull(t *testing.T) {
	r, puts := newSettingsFakeController(t, storedRadioAi())
	s := resourceSchema(t, r)

	prior := map[string]any{"radio_ai": radioAiCurrent(nil, nil)}
	planned := radioAiCurrent(nil, nil)
	planned["cron_expr"] = "0 4 * * *"
	config := radioAiCurrent(nil, nil)
	config["cron_expr"] = "0 4 * * *"
	state := tfState(t, s, prior)

	upd := &fwresource.UpdateResponse{State: state, Identity: tfIdentity(t, r)}
	r.Update(context.Background(), fwresource.UpdateRequest{
		Plan:   tfPlan(t, s, map[string]any{"radio_ai": planned}),
		Config: tfConfig(t, s, map[string]any{"radio_ai": config}),
		State:  state,
	}, upd)
	if upd.Diagnostics.HasError() {
		t.Fatalf("Update: %v", upd.Diagnostics)
	}

	written := puts()["radio_ai"]
	if !reflect.DeepEqual(written, map[string]any{"key": "radio_ai", "cron_expr": "0 4 * * *"}) {
		t.Errorf("radio_ai PUT = %v, want only cron_expr", written)
	}

	var m settingResourceModel
	if d := upd.State.Get(context.Background(), &m); d.HasError() {
		t.Fatalf("state: %v", d)
	}
	attrs := m.RadioAi.Attributes()
	for _, name := range []string{"high_priority_devices", "auto_enabled"} {
		if !attrs[name].IsNull() {
			t.Errorf("unset %s = %v, want null", name, attrs[name])
		}
	}
	if got, _ := attrs["cron_expr"].(types.String); got.ValueString() != "0 4 * * *" {
		t.Errorf("cron_expr = %v", got)
	}

	// A refresh keeps them null too, so no change is planned for them later.
	read := &fwresource.ReadResponse{State: upd.State, Identity: upd.Identity}
	r.Read(context.Background(),
		fwresource.ReadRequest{State: upd.State, Identity: upd.Identity}, read)
	if read.Diagnostics.HasError() {
		t.Fatalf("Read: %v", read.Diagnostics)
	}
	read.State.Get(context.Background(), &m)
	if attrs := m.RadioAi.Attributes(); !attrs["high_priority_devices"].IsNull() ||
		!attrs["auto_enabled"].IsNull() {
		t.Errorf("refreshed radio_ai = %v, want the unset attributes null", m.RadioAi)
	}
}

// Attributes the plan leaves unknown (the block is new to the state) are read
// from the controller: an absent high_priority_devices as [], auto_enabled as
// stored.
func TestSettingRadioAiCreateReadsUnsetAttributes(t *testing.T) {
	r, puts := newSettingsFakeController(t, storedRadioAi())
	s := resourceSchema(t, r)

	config := map[string]any{"enabled": true}
	planned := map[string]any{}
	for name := range radioAiAttrTypes {
		planned[name] = tfUnknown
	}
	planned["enabled"] = true
	attrs := map[string]any{"radio_ai": planned}
	resp := &fwresource.CreateResponse{State: tfState(t, s, nil), Identity: tfIdentity(t, r)}
	r.Create(context.Background(), fwresource.CreateRequest{
		Plan: tfPlan(t, s, attrs), Config: tfConfig(t, s, map[string]any{"radio_ai": config}),
	}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
	if got := puts()["radio_ai"]; !reflect.DeepEqual(
		got, map[string]any{"key": "radio_ai", "enabled": true},
	) {
		t.Errorf("radio_ai PUT = %v, want only enabled", got)
	}
	ra := stateModel(t, resp).RadioAi.Attributes()
	if l, ok := ra["high_priority_devices"].(types.List); !ok || l.IsNull() ||
		len(l.Elements()) != 0 {
		t.Errorf("high_priority_devices = %v, want []", ra["high_priority_devices"])
	}
	if b, ok := ra["auto_enabled"].(types.Bool); !ok || b.IsNull() || b.ValueBool() {
		t.Errorf("auto_enabled = %v, want false", ra["auto_enabled"])
	}
}

// The write payload: channels and HT modes as numeric strings, blacklist and
// radio configuration as objects, auto_enabled as its own key; keys equal to
// the stored value are not sent.
func TestSettingRadioAiWritePayload(t *testing.T) {
	r, puts := newSettingsFakeController(t, storedRadioAi())
	s := resourceSchema(t, r)
	prior := map[string]any{"radio_ai": radioAiCurrent([]any{}, false)}
	block := radioAiCurrent([]any{"aa:bb:cc:dd:ee:ff"}, true)
	block["channels_na"] = []any{36, 40, 52, 100}
	block["ht_modes_na"] = []any{20, 40, 80}
	block["channels_blacklist"] = []any{
		map[string]any{"radio": "na", "channel": 149, "channel_width": 80},
	}
	block["radios_configuration"] = []any{
		map[string]any{"radio": "na", "channel_width": 80, "dfs": true},
	}
	state := tfState(t, s, prior)
	attrs := map[string]any{"radio_ai": block}
	upd := &fwresource.UpdateResponse{State: state, Identity: tfIdentity(t, r)}
	r.Update(context.Background(), fwresource.UpdateRequest{
		Plan: tfPlan(t, s, attrs), Config: tfConfig(t, s, attrs), State: state,
	}, upd)
	if upd.Diagnostics.HasError() {
		t.Fatalf("Update: %v", upd.Diagnostics)
	}
	want := map[string]any{
		"key":                   "radio_ai",
		"auto_enabled":          true,
		"high_priority_devices": []any{"aa:bb:cc:dd:ee:ff"},
		"channels_na":           []any{"36", "40", "52", "100"},
		"ht_modes_na":           []any{"20", "40", "80"},
		"channels_blacklist": []any{
			map[string]any{"channel": float64(149), "channel_width": float64(80), "radio": "na"},
		},
		"radios_configuration": []any{
			map[string]any{"channel_width": float64(80), "dfs": true, "radio": "na"},
		},
	}
	if got := puts()["radio_ai"]; !reflect.DeepEqual(got, want) {
		t.Errorf("radio_ai PUT =\n%v\nwant\n%v", got, want)
	}
	var m settingResourceModel
	upd.State.Get(context.Background(), &m)
	if b, _ := m.RadioAi.Attributes()["auto_enabled"].(types.Bool); !b.ValueBool() {
		t.Errorf("state auto_enabled = %v, want true", b)
	}
}

// Live finding 3: auto_enabled is its own controller key, read and written
// apart from enabled.
func TestRadioAiAutoEnabled(t *testing.T) {
	ctx := context.Background()
	r := &settingResource{}

	raw, d := radioAiRawSetting(ctx, &settingRadioAiModel{AutoEnabled: types.BoolValue(false)})
	if d.HasError() {
		t.Fatalf("radioAiRawSetting: %v", d)
	}
	if !reflect.DeepEqual(raw.Data, map[string]any{"auto_enabled": false}) {
		t.Errorf("payload = %v, want only auto_enabled=false", raw.Data)
	}

	for _, tc := range []struct {
		doc  string
		want types.Bool
	}{
		{`{"key":"radio_ai","enabled":false,"auto_enabled":true}`, types.BoolValue(true)},
		{`{"key":"radio_ai","enabled":true,"auto_enabled":false}`, types.BoolValue(false)},
		{`{"key":"radio_ai","enabled":true}`, types.BoolNull()},
	} {
		var setting settings.RadioAi
		if err := json.Unmarshal([]byte(tc.doc), &setting); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		model, d := r.radioAiSettingToModel(ctx, &setting)
		if d.HasError() {
			t.Fatalf("radioAiSettingToModel: %v", d)
		}
		if !model.AutoEnabled.Equal(tc.want) {
			t.Errorf("%s: auto_enabled = %v, want %v", tc.doc, model.AutoEnabled, tc.want)
		}
	}
}

func TestRawSettingChanges(t *testing.T) {
	stored := map[string]any{
		"cron_expr":   "0 3 * * *",
		"channels_ng": []any{"1", "6", "11"},
		"enabled":     true,
	}
	got, err := rawSettingChanges(stored, map[string]any{
		"cron_expr":    "0 3 * * *",
		"channels_ng":  []string{"1", "6", "11"},
		"enabled":      false,
		"auto_enabled": false,
		"radios":       []string{},
	})
	if err != nil {
		t.Fatalf("rawSettingChanges: %v", err)
	}
	want := map[string]any{"enabled": false, "auto_enabled": false, "radios": []string{}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("changes = %v, want %v", got, want)
	}
	if _, err := rawSettingChanges(
		map[string]any{"x": 1}, map[string]any{"x": make(chan int)},
	); err == nil {
		t.Error("want an error for an unmarshalable value")
	}
}

// Live finding 2: the controller drops DFS channels from channels_na when the
// na radio configuration has dfs = false, so such a configuration fails the
// plan rather than the apply.
func TestRadioAiDfsConflict(t *testing.T) {
	ctx := context.Background()
	obj := func(t *testing.T, v map[string]any) types.Object {
		t.Helper()
		typ := types.ObjectType{AttrTypes: radioAiAttrTypes}
		o, err := typ.ValueFromTerraform(ctx, tfValue(t, typ.TerraformType(ctx), v))
		if err != nil {
			t.Fatalf("building radio_ai: %v", err)
		}
		ro, ok := o.(types.Object)
		if !ok {
			t.Fatalf("radio_ai is %T", o)
		}
		return ro
	}
	naDfs := func(dfs any) map[string]any {
		return map[string]any{"radios_configuration": []any{
			map[string]any{"radio": "ng", "channel_width": 20, "dfs": false},
			map[string]any{"radio": "na", "channel_width": 40, "dfs": dfs},
		}}
	}
	with := func(m map[string]any, channels any) map[string]any {
		out := map[string]any{"channels_na": channels}
		for k, v := range m {
			out[k] = v
		}
		return out
	}

	for name, tc := range map[string]struct {
		config, plan map[string]any
		wantErr      bool
	}{
		"dfs channels with dfs off": {
			config:  with(naDfs(false), []any{36, 52, 100}),
			plan:    with(naDfs(false), []any{36, 52, 100}),
			wantErr: true,
		},
		"dfs off kept from state": {
			config:  map[string]any{"channels_na": []any{36, 144}},
			plan:    with(naDfs(false), []any{36, 144}),
			wantErr: true,
		},
		"no dfs channels": {
			config: with(naDfs(false), []any{36, 40, 48, 149, 165}),
			plan:   with(naDfs(false), []any{36, 40, 48, 149, 165}),
		},
		"dfs on": {
			config: with(naDfs(true), []any{36, 52, 100}),
			plan:   with(naDfs(true), []any{36, 52, 100}),
		},
		"channels_na not configured": {
			config: naDfs(false),
			plan:   with(naDfs(false), []any{36, 52}),
		},
		"radios_configuration unknown": {
			config: map[string]any{"channels_na": []any{52}},
			plan:   map[string]any{"channels_na": []any{52}, "radios_configuration": tfUnknown},
		},
		"no na entry": {
			config: map[string]any{"channels_na": []any{52}},
			plan: with(map[string]any{"radios_configuration": []any{
				map[string]any{"radio": "ng", "channel_width": 20, "dfs": false},
			}}, []any{52}),
		},
		"channels_na unknown": {
			config: map[string]any{"channels_na": tfUnknown},
			plan:   with(naDfs(false), tfUnknown),
		},
	} {
		t.Run(name, func(t *testing.T) {
			d := radioAiDfsConflict(ctx, obj(t, tc.config), obj(t, tc.plan))
			if d.HasError() != tc.wantErr {
				t.Fatalf("diagnostics = %v, want error %v", d, tc.wantErr)
			}
			if tc.wantErr {
				if !strings.Contains(d[0].Detail(), "52") &&
					!strings.Contains(d[0].Detail(), "144") {
					t.Errorf("detail %q does not name the DFS channels", d[0].Detail())
				}
			}
		})
	}

	if d := radioAiDfsConflict(ctx, types.ObjectNull(radioAiAttrTypes),
		obj(t, naDfs(false))); d.HasError() {
		t.Errorf("null config: %v", d)
	}
}

// ModifyPlan runs the DFS check with radios_configuration carried from state.
func TestSettingModifyPlanRejectsDfsChannelsWithDfsOff(t *testing.T) {
	r := &settingResource{}
	s := resourceSchema(t, r)
	prior := map[string]any{"radio_ai": radioAiCurrent([]any{}, false)}
	config := map[string]any{"radio_ai": map[string]any{"channels_na": []any{36, 100}}}
	planned := radioAiCurrent([]any{}, false)
	planned["channels_na"] = []any{36, 100}
	for name := range planned {
		if name != "channels_na" {
			planned[name] = tfUnknown
		}
	}
	plan := tfPlan(t, s, map[string]any{"radio_ai": planned})
	resp := &fwresource.ModifyPlanResponse{Plan: plan}
	r.ModifyPlan(context.Background(), fwresource.ModifyPlanRequest{
		Plan: plan, Config: tfConfig(t, s, config), State: tfState(t, s, prior),
	}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("want a DFS conflict error")
	}
	want := path.Root("radio_ai").AtName("channels_na")
	if got := resp.Diagnostics.Errors()[0]; !strings.Contains(got.Detail(), "100") {
		t.Errorf("error = %v", got)
	} else if p, ok := got.(interface{ Path() path.Path }); !ok || !p.Path().Equal(want) {
		t.Errorf("error is not on %s: %v", want, got)
	}
}
