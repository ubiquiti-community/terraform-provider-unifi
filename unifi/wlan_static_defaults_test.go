package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// #543: group_rekey and wpa_enc were Optional+Computed *and* carried a static
// Default. The two contracts conflict and Default wins, so on an adopted SSID
// every undeclared attribute was planned at the constant rather than at the
// controller's value - measured on a live site, that rewrote group rekeying on
// five of six SSIDs and the encryption mode on two.
//
// An Optional+Computed attribute must take the controller's value when the
// configuration does not set one, which is what UseStateForUnknown gives.
func TestWLANNoStaticDefaultsOnComputedAttributes(t *testing.T) {
	var resp resource.SchemaResponse
	(&wlanFrameworkResource{}).Schema(
		context.Background(),
		resource.SchemaRequest{},
		&resp,
	)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}

	for _, name := range []string{"group_rekey", "wpa_enc"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Errorf("attribute %q is missing from the schema", name)
			continue
		}

		switch a := attr.(type) {
		case schema.Int64Attribute:
			if a.Default != nil {
				t.Errorf(
					"%s must not carry a static Default: it overwrites the controller's value on an adopted SSID (#543)",
					name,
				)
			}
			if len(a.PlanModifiers) == 0 {
				t.Errorf("%s should keep UseStateForUnknown so the prior value is held", name)
			}
		case schema.StringAttribute:
			if a.Default != nil {
				t.Errorf(
					"%s must not carry a static Default: it overwrites the controller's value on an adopted SSID (#543)",
					name,
				)
			}
			if len(a.PlanModifiers) == 0 {
				t.Errorf("%s should keep UseStateForUnknown so the prior value is held", name)
			}
		default:
			t.Errorf("%s has unexpected type %T", name, attr)
		}
	}
}
