package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi/settings"
)

// #493: every bool on settings.Ips and settings.Usg serializes without
// omitempty, so building the outgoing document from the model alone writes
// false for each attribute the configuration does not manage - disabling an
// existing honeypot or content-filtering blocking page as a side effect of an
// unrelated change. The write path now starts from the controller's current
// document and overlays only what is configured.
func TestSettingIpsPreservesUnmanagedFields(t *testing.T) {
	ctx := context.Background()
	r := &settingResource{}

	// What the controller holds: features enabled out of band.
	live := &settings.Ips{
		IPsMode:                             "ids",
		HoneypotEnabled:                     true,
		ContentFilteringBlockingPageEnabled: true,
		MemoryOptimized:                     true,
		RestrictTorrents:                    true,
		EnabledCategories:                   []string{"malware"},
	}

	// A configuration that manages only ips_mode.
	model := &settingIpsModel{
		IPSMode:                             types.StringValue("ips"),
		HoneypotEnabled:                     types.BoolNull(),
		ContentFilteringBlockingPageEnabled: types.BoolNull(),
		MemoryOptimized:                     types.BoolNull(),
		RestrictTorrents:                    types.BoolNull(),
		EnabledCategories:                   types.ListNull(types.StringType),
		EnabledNetworks:                     types.ListNull(types.StringType),
		Honeypot: types.ListNull(
			types.ObjectType{AttrTypes: ipsHoneypotAttrTypes},
		),
		SuppressionWhitelist: types.ListNull(
			types.ObjectType{AttrTypes: ipsWhitelistAttrTypes},
		),
		SuppressionAlerts: types.ListNull(
			types.ObjectType{AttrTypes: ipsAlertAttrTypes},
		),
	}

	var diags diag.Diagnostics
	got, _ := r.ipsModelToSetting(ctx, model, &diags, live)
	if diags.HasError() {
		t.Fatalf("conversion: %v", diags)
	}

	if got.IPsMode != "ips" {
		t.Errorf("ips_mode = %q, want the configured ips", got.IPsMode)
	}
	for _, c := range []struct {
		name string
		got  bool
	}{
		{"honeypot_enabled", got.HoneypotEnabled},
		{"content_filtering_blocking_page_enabled", got.ContentFilteringBlockingPageEnabled},
		{"memory_optimized", got.MemoryOptimized},
		{"restrict_torrents", got.RestrictTorrents},
	} {
		if !c.got {
			t.Errorf("%s = false, want the controller's true preserved", c.name)
		}
	}
	if len(got.EnabledCategories) != 1 {
		t.Errorf(
			"enabled_categories = %v, want the controller's list preserved",
			got.EnabledCategories,
		)
	}
}

// Same guarantee for the usg block, which carries twenty such bools.
func TestSettingUsgPreservesUnmanagedFields(t *testing.T) {
	ctx := context.Background()
	r := &settingResource{}

	live := &settings.Usg{
		FtpModule:                true,
		GreModule:                true,
		H323Module:               true,
		BroadcastPing:            true,
		LldpEnableAll:            true,
		MdnsEnabled:              true,
		EchoServer:               "echo.example.net",
		TimeoutSettingPreference: "manual",
	}

	// Manages only ftp_module, flipping it off.
	model := &settingUSGModel{
		FtpModule:  types.BoolValue(false),
		GreModule:  types.BoolNull(),
		H323Module: types.BoolNull(),
	}

	got := r.usgModelToSetting(ctx, model, live)

	if got.FtpModule {
		t.Error("ftp_module = true, want the configured false applied")
	}
	for _, c := range []struct {
		name string
		got  bool
	}{
		{"gre_module", got.GreModule},
		{"h323_module", got.H323Module},
		{"broadcast_ping", got.BroadcastPing},
		{"lldp_enable_all", got.LldpEnableAll},
		{"mdns_enabled", got.MdnsEnabled},
	} {
		if !c.got {
			t.Errorf("%s = false, want the controller's true preserved", c.name)
		}
	}
	if got.EchoServer != "echo.example.net" {
		t.Errorf("echo_server = %q, want the controller's value preserved", got.EchoServer)
	}
}

// With a nil base - which is what the pre-#493 code effectively did, building
// the document from the model alone - the controller's values are lost. This
// pins the difference the fix makes, so a regression to the old shape fails
// here rather than silently disabling features on someone's site.
func TestSettingIpsNilBaseLosesUnmanagedFields(t *testing.T) {
	ctx := context.Background()
	r := &settingResource{}
	var diags diag.Diagnostics

	model := &settingIpsModel{
		IPSMode:                             types.StringValue("ips"),
		HoneypotEnabled:                     types.BoolNull(),
		ContentFilteringBlockingPageEnabled: types.BoolNull(),
		MemoryOptimized:                     types.BoolNull(),
		RestrictTorrents:                    types.BoolNull(),
		EnabledCategories:                   types.ListNull(types.StringType),
		EnabledNetworks:                     types.ListNull(types.StringType),
		Honeypot: types.ListNull(
			types.ObjectType{AttrTypes: ipsHoneypotAttrTypes},
		),
		SuppressionWhitelist: types.ListNull(
			types.ObjectType{AttrTypes: ipsWhitelistAttrTypes},
		),
		SuppressionAlerts: types.ListNull(types.ObjectType{AttrTypes: ipsAlertAttrTypes}),
	}

	got, _ := r.ipsModelToSetting(ctx, model, &diags, nil)
	if diags.HasError() {
		t.Fatalf("conversion: %v", diags)
	}
	// Every unmanaged bool is false: exactly the payload #493 reported.
	if got.HoneypotEnabled || got.ContentFilteringBlockingPageEnabled ||
		got.MemoryOptimized || got.RestrictTorrents {
		t.Error("expected the model-only document to zero the unmanaged bools")
	}
}

// A nil base (the live document could not be read) must still work, falling
// back to the previous behaviour rather than panicking.
func TestSettingIpsNilBase(t *testing.T) {
	ctx := context.Background()
	r := &settingResource{}
	var diags diag.Diagnostics

	model := &settingIpsModel{
		IPSMode:           types.StringValue("ips"),
		EnabledCategories: types.ListNull(types.StringType),
		EnabledNetworks:   types.ListNull(types.StringType),
		Honeypot:          types.ListNull(types.ObjectType{AttrTypes: ipsHoneypotAttrTypes}),
		SuppressionWhitelist: types.ListNull(
			types.ObjectType{AttrTypes: ipsWhitelistAttrTypes},
		),
		SuppressionAlerts: types.ListNull(types.ObjectType{AttrTypes: ipsAlertAttrTypes}),
	}
	got, _ := r.ipsModelToSetting(ctx, model, &diags, nil)
	if diags.HasError() {
		t.Fatalf("conversion: %v", diags)
	}
	if got.IPsMode != "ips" {
		t.Errorf("ips_mode = %q, want ips", got.IPsMode)
	}
}

var _ = attr.Value(nil)
