package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The read right after the update can still carry the old LED and band
// steering values; the planned ones must win so the apply does not fail with
// "inconsistent result after apply".
func Test_reassertAsyncApplied(t *testing.T) {
	read := deviceResourceModel{
		LedOverride:                types.StringValue("on"),
		LedOverrideColor:           types.StringValue("#ffffff"),
		LedOverrideColorBrightness: types.Int64Value(80),
		BandsteeringMode:           types.StringValue("equal"),
	}
	planned := deviceResourceModel{
		LedOverride:                types.StringValue("off"),
		LedOverrideColor:           types.StringValue("#0000ff"),
		LedOverrideColorBrightness: types.Int64Value(100),
		BandsteeringMode:           types.StringValue("prefer_5g"),
	}

	reassertAsyncApplied(&read, &planned)

	if read.BandsteeringMode.ValueString() != "prefer_5g" {
		t.Errorf("bandsteering_mode = %q, want prefer_5g", read.BandsteeringMode.ValueString())
	}
	if read.LedOverride.ValueString() != "off" ||
		read.LedOverrideColor.ValueString() != "#0000ff" ||
		read.LedOverrideColorBrightness.ValueInt64() != 100 {
		t.Errorf("LED values not re-asserted: %v %v %v",
			read.LedOverride, read.LedOverrideColor, read.LedOverrideColorBrightness)
	}
}

// Values the configuration leaves unset keep what the controller reported.
func Test_reassertAsyncApplied_keepsReadValuesWhenUnplanned(t *testing.T) {
	read := deviceResourceModel{
		LedOverride:                types.StringValue("on"),
		LedOverrideColor:           types.StringValue("#ffffff"),
		LedOverrideColorBrightness: types.Int64Value(80),
		BandsteeringMode:           types.StringValue("equal"),
	}
	planned := deviceResourceModel{
		LedOverride:                types.StringNull(),
		LedOverrideColor:           types.StringUnknown(),
		LedOverrideColorBrightness: types.Int64Null(),
		BandsteeringMode:           types.StringUnknown(),
	}

	reassertAsyncApplied(&read, &planned)

	if read.BandsteeringMode.ValueString() != "equal" || read.LedOverride.ValueString() != "on" ||
		read.LedOverrideColor.ValueString() != "#ffffff" || read.LedOverrideColorBrightness.ValueInt64() != 80 {
		t.Errorf("read values were overwritten: %+v", read)
	}
}
