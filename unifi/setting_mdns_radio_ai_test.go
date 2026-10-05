package unifi

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi/settings"
)

func Test_settingResource_mdnsRadioAiSchema(t *testing.T) {
	resp := &fwresource.SchemaResponse{}
	(&settingResource{}).Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() produced errors: %v", resp.Diagnostics)
	}
	for _, a := range []string{"mdns", "radio_ai"} {
		if _, ok := resp.Schema.Attributes[a]; !ok {
			t.Errorf("missing attribute %q", a)
		}
	}
}

// mDNS as UniFi Network 10.6 stores it when reflected on a subset of networks.
func liveMdns() *settings.Mdns {
	return &settings.Mdns{
		Mode:                 "all",
		EnabledFor:           "some",
		EnabledForNetworkIDs: []string{"net-default", "net-iot", "net-helena"},
		PredefinedServices:   []settings.SettingMdnsPredefinedServices{},
		CustomServices:       []settings.SettingMdnsCustomServices{},
	}
}

func Test_settingResource_mdnsRoundTrip(t *testing.T) {
	r, ctx := &settingResource{}, context.Background()
	var diags diag.Diagnostics

	model := r.mdnsSettingToModel(ctx, liveMdns(), &diags)
	got := r.mdnsModelToSetting(ctx, &model, &settings.Mdns{}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !reflect.DeepEqual(got, liveMdns()) {
		t.Errorf("round trip = %+v, want %+v", got, liveMdns())
	}
}

func Test_settingResource_mdnsOverlayKeepsUnsetFields(t *testing.T) {
	r, ctx := &settingResource{}, context.Background()
	var diags diag.Diagnostics

	model := settingMdnsModel{
		Mode:                 types.StringNull(),
		EnabledFor:           types.StringValue("all"),
		EnabledForNetworkIDs: types.ListUnknown(types.StringType),
		PredefinedServices:   types.ListNull(types.StringType),
		CustomServices: types.ListNull(
			types.ObjectType{AttrTypes: mdnsCustomServiceAttrTypes},
		),
	}
	got := r.mdnsModelToSetting(ctx, &model, liveMdns(), &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.EnabledFor != "all" {
		t.Errorf("EnabledFor = %q, want all", got.EnabledFor)
	}
	if got.Mode != "all" || len(got.EnabledForNetworkIDs) != 3 {
		t.Errorf("unset fields were not kept: %+v", got)
	}
}

func ptr64(v int64) *int64 { return &v }

// Radio AI as UniFi Network 10.6 stores it with a custom channel preset.
func liveRadioAi() *settings.RadioAi {
	auto := false
	return &settings.RadioAi{
		Enabled:                     true,
		AutoEnabled:                 &auto,
		CronExpr:                    "0 4 * * *",
		SettingPreference:           "auto",
		AutoChannelPresetsType:      "custom",
		AutoAdjustChannelsToCountry: true,
		Optimize:                    []string{"channel"},
		Radios:                      []string{"ng", "na"},
		ChannelsNg:                  []int64{1, 6, 11},
		ChannelsNa:                  []int64{36, 40, 44, 48},
		Channels6E:                  []int64{1, 5},
		HtModesNg:                   []int64{20},
		HtModesNa:                   []int64{20, 40},
		ExcludeDevices:              []string{},
		HighPriorityDevices:         []string{},
		ChannelsBlacklist: []settings.SettingRadioAiChannelsBlacklist{
			{Radio: "ng", Channel: ptr64(2), ChannelWidth: ptr64(20)},
		},
		RadiosConfiguration: []settings.SettingRadioAiRadiosConfiguration{
			{Radio: "ng", ChannelWidth: ptr64(20), Dfs: false},
			{Radio: "na", ChannelWidth: ptr64(160), Dfs: true},
		},
		UseXy: true,
	}
}

func Test_settingResource_radioAiRoundTrip(t *testing.T) {
	r, ctx := &settingResource{}, context.Background()
	var diags diag.Diagnostics

	model := r.radioAiSettingToModel(ctx, liveRadioAi(), &diags)
	// UseXy is not exposed; the base carries it.
	got := r.radioAiModelToSetting(ctx, &model, &settings.RadioAi{UseXy: true}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !reflect.DeepEqual(got, liveRadioAi()) {
		t.Errorf("round trip =\n%+v\nwant\n%+v", got, liveRadioAi())
	}
}

func Test_settingResource_radioAiOverlayKeepsUnsetFields(t *testing.T) {
	r, ctx := &settingResource{}, context.Background()
	var diags diag.Diagnostics

	model := r.radioAiSettingToModel(ctx, liveRadioAi(), &diags)
	// Only the width list is configured; everything else is left to the controller.
	for _, l := range []*types.List{
		&model.Optimize, &model.Radios, &model.ChannelsNg, &model.ChannelsNa, &model.Channels6e,
		&model.HtModesNg, &model.ExcludeDevices, &model.HighPriorityDevices,
	} {
		*l = types.ListNull(l.ElementType(ctx))
	}
	model.ChannelsBlacklist = types.ListNull(types.ObjectType{AttrTypes: radioAiBlacklistAttrTypes})
	model.RadiosConfiguration = types.ListNull(
		types.ObjectType{AttrTypes: radioAiRadioConfigAttrTypes},
	)
	model.Enabled, model.AutoEnabled = types.BoolNull(), types.BoolNull()
	model.HtModesNa, _ = types.ListValueFrom(ctx, types.Int64Type, []int64{20, 40, 80})

	got := r.radioAiModelToSetting(ctx, &model, liveRadioAi(), &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	want := liveRadioAi()
	want.HtModesNa = []int64{20, 40, 80}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("overlay =\n%+v\nwant\n%+v", got, want)
	}
}

// After an import, attributes left out of config are planned as null; the
// read-back must keep them null instead of filling them from the controller.
func Test_keepNullAttributes(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	r := &settingResource{}

	freshModel := r.mdnsSettingToModel(ctx, liveMdns(), &diags)
	fresh, d := types.ObjectValueFrom(ctx, mdnsAttrTypes, freshModel)
	diags.Append(d...)

	priorModel := freshModel
	priorModel.Mode = types.StringValue("auto") // configured, stale
	priorModel.PredefinedServices = types.ListNull(types.StringType)
	prior, d := types.ObjectValueFrom(ctx, mdnsAttrTypes, priorModel)
	diags.Append(d...)

	got := keepNullAttributes(prior, fresh, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	attrs := got.Attributes()
	if !attrs["predefined_services"].IsNull() {
		t.Error("predefined_services should stay null")
	}
	if mode, ok := attrs["mode"].(types.String); !ok || mode.ValueString() != "all" {
		t.Error("mode should be refreshed from the controller")
	}
}
