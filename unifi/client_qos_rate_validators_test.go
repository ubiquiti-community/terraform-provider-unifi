package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// max_up rejected -1 (unlimited) while max_down accepted it, so an unlimited
// upload could not be declared even though every new group is created with it.
func TestClientQosRateLimitsAcceptUnlimited(t *testing.T) {
	ctx := context.Background()
	var resp resource.SchemaResponse
	(&clientQosRateResource{}).Schema(ctx, resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}
	rate, ok := resp.Schema.Attributes["qos_rate"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("qos_rate is %T", resp.Schema.Attributes["qos_rate"])
	}

	for _, name := range []string{"max_down", "max_up"} {
		a, ok := rate.Attributes[name].(schema.Int64Attribute)
		if !ok {
			t.Fatalf("qos_rate.%s is %T", name, rate.Attributes[name])
		}
		for value, valid := range map[int64]bool{
			-1: true, 2: true, 100000: true,
			-2: false, 0: false, 1: false, 100001: false,
		} {
			req := validator.Int64Request{
				Path:        path.Root("qos_rate").AtName(name),
				ConfigValue: types.Int64Value(value),
			}
			var diags validator.Int64Response
			for _, v := range a.Validators {
				v.ValidateInt64(ctx, req, &diags)
			}
			if got := !diags.Diagnostics.HasError(); got != valid {
				t.Errorf("qos_rate.%s = %d: valid = %t, want %t", name, value, got, valid)
			}
		}
	}
}
