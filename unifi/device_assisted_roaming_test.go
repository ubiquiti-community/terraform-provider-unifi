package unifi

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// TestDeviceDeprecatedAssistedRoamingIsComputed guards #615: the deprecated
// radio_table assisted_roaming_* leaves carry a value from prior state on
// every read. As plain Optional leaves, an omitted one planned that carried
// value away, and every access point planned state -> (known after apply).
func TestDeviceDeprecatedAssistedRoamingIsComputed(t *testing.T) {
	var resp fwresource.SchemaResponse
	(&deviceResource{}).Schema(context.Background(), fwresource.SchemaRequest{}, &resp)

	radioTable, ok := resp.Schema.Attributes["radio_table"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatalf(
			"radio_table is %T, want a ListNestedAttribute",
			resp.Schema.Attributes["radio_table"],
		)
	}
	for _, name := range []string{"assisted_roaming_enabled", "assisted_roaming_rssi"} {
		a := radioTable.NestedObject.Attributes[name]
		if a == nil || !a.IsOptional() || !a.IsComputed() {
			t.Errorf("radio_table.%s must be Optional + Computed", name)
		}
	}
}
