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

func Test_settingResource_mdnsSchema(t *testing.T) {
	resp := &fwresource.SchemaResponse{}
	(&settingResource{}).Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() produced errors: %v", resp.Diagnostics)
	}
	for _, a := range []string{"mdns"} {
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
	// enabled_for_network_ids is read-only: the stored list is sent back as is.
	got := r.mdnsModelToSetting(ctx, &model, liveMdns(), &diags)
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
