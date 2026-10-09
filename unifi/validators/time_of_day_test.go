package validators

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestTimeOfDay(t *testing.T) {
	cases := map[string]struct {
		value   types.String
		wantErr bool
	}{
		"null":         {types.StringNull(), false},
		"unknown":      {types.StringUnknown(), false},
		"midnight":     {types.StringValue("00:00"), false},
		"evening":      {types.StringValue("22:30"), false},
		"last minute":  {types.StringValue("23:59"), false},
		"bad hour":     {types.StringValue("24:00"), true},
		"bad minute":   {types.StringValue("12:60"), true},
		"no padding":   {types.StringValue("7:30"), true},
		"with seconds": {types.StringValue("07:30:00"), true},
		"empty":        {types.StringValue(""), true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			TimeOfDay().ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("begins"),
				ConfigValue: tc.value,
			}, resp)
			if resp.Diagnostics.HasError() != tc.wantErr {
				t.Errorf(
					"%q: error=%v, want %v",
					tc.value.ValueString(),
					resp.Diagnostics,
					tc.wantErr,
				)
			}
		})
	}
}
