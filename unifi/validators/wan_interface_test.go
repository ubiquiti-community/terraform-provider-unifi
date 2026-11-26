package validators

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestWANInterface(t *testing.T) {
	cases := map[string]struct {
		value   types.String
		wantErr bool
	}{
		"null":       {types.StringNull(), false},
		"unknown":    {types.StringUnknown(), false},
		"first":      {types.StringValue("wan"), false},
		"second":     {types.StringValue("wan2"), false},
		"third":      {types.StringValue("wan3"), false},
		"ninth":      {types.StringValue("wan9"), false},
		"wan1":       {types.StringValue("wan1"), true},
		"tenth":      {types.StringValue("wan10"), true},
		"upper case": {types.StringValue("WAN2"), true},
		"lan":        {types.StringValue("lan"), true},
		"empty":      {types.StringValue(""), true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			WANInterface().ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("interface"),
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
