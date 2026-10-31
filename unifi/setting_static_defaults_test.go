package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// #544: an Optional+Computed attribute that also carries a static Default is
// overwritten by that Default, because the framework applies defaults
// (TransformDefaults) before it runs plan modifiers (SchemaModifyPlan).
//
// unifi_setting makes this worse than elsewhere: it is one resource fronting
// many site-level settings as nested blocks, so declaring a block to manage a
// single attribute pulls in every default in that block. Measured against a
// live site, five disagreed with what the controller stores:
//
//	dpi.enabled               false vs live true  -> disables deep packet inspection
//	syslog.enabled            false vs live true  -> disables remote syslog
//	syslog.log_all_contents   false vs live true
//	auto_speedtest.enabled    false vs live true
//	lcm.sync                  false vs live true
//
// The syslog pair is the sharpest: a configuration that declares the syslog
// block only to set the server address was writing enabled=false alongside it,
// turning off the very logging it was configuring.
func TestSettingNoStaticDefaultsOnLiveManagedAttributes(t *testing.T) {
	var resp resource.SchemaResponse
	(&settingResource{}).Schema(
		context.Background(),
		resource.SchemaRequest{},
		&resp,
	)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}

	for _, c := range []struct{ block, attr string }{
		{"dpi", "enabled"},
		{"syslog", "enabled"},
		{"syslog", "log_all_contents"},
		{"auto_speedtest", "enabled"},
		{"lcm", "sync"},
	} {
		nested, ok := resp.Schema.Attributes[c.block].(schema.SingleNestedAttribute)
		if !ok {
			t.Errorf(
				"block %q is %T, want schema.SingleNestedAttribute",
				c.block, resp.Schema.Attributes[c.block],
			)
			continue
		}

		attr, ok := nested.Attributes[c.attr]
		if !ok {
			t.Errorf("%s.%s is missing from the schema", c.block, c.attr)
			continue
		}
		a, ok := attr.(schema.BoolAttribute)
		if !ok {
			t.Errorf("%s.%s is %T, want schema.BoolAttribute", c.block, c.attr, attr)
			continue
		}

		if a.Default != nil {
			t.Errorf(
				"%s.%s must not carry a static Default: it is applied before "+
					"plan modifiers, so declaring the %s block for any other "+
					"attribute writes this one too and overwrites the "+
					"controller's value (#544)",
				c.block, c.attr, c.block,
			)
		}
	}
}
