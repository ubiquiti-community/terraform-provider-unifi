package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func testLed(override, color attr.Value, brightness attr.Value) types.Object {
	return types.ObjectValueMust(deviceLedAttrTypes(), map[string]attr.Value{
		"override":   override,
		"color":      color,
		"brightness": brightness,
	})
}

func ledString(t *testing.T, o types.Object, name string) string {
	t.Helper()
	v, ok := o.Attributes()[name].(types.String)
	if !ok {
		t.Fatalf("led.%s is not a string", name)
	}
	return v.ValueString()
}

// The read right after the update can still carry the old LED and band
// steering values; the planned ones must win so the apply does not fail with
// "inconsistent result after apply".
func Test_reassertAsyncApplied(t *testing.T) {
	read := deviceResourceModel{
		Led: testLed(
			types.StringValue("on"), types.StringValue("#ffffff"), types.Int64Value(80)),
		BandsteeringMode: types.StringValue("equal"),
	}
	planned := deviceResourceModel{
		Led: testLed(
			types.StringValue("off"), types.StringValue("#0000ff"), types.Int64Value(100)),
		BandsteeringMode: types.StringValue("prefer_5g"),
	}

	reassertAsyncApplied(context.Background(), &read, &planned)

	if read.BandsteeringMode.ValueString() != "prefer_5g" {
		t.Errorf("bandsteering_mode = %q, want prefer_5g", read.BandsteeringMode.ValueString())
	}
	if !read.Led.Equal(planned.Led) {
		t.Errorf("LED values not re-asserted: %v", read.Led)
	}
}

// Values the configuration leaves unset keep what the controller reported.
func Test_reassertAsyncApplied_keepsReadValuesWhenUnplanned(t *testing.T) {
	read := deviceResourceModel{
		Led: testLed(
			types.StringValue("on"), types.StringValue("#ffffff"), types.Int64Value(80)),
		BandsteeringMode: types.StringValue("equal"),
	}
	planned := deviceResourceModel{
		Led:              testLed(types.StringNull(), types.StringUnknown(), types.Int64Null()),
		BandsteeringMode: types.StringUnknown(),
	}

	reassertAsyncApplied(context.Background(), &read, &planned)

	brightness, _ := read.Led.Attributes()["brightness"].(types.Int64)
	if read.BandsteeringMode.ValueString() != "equal" ||
		ledString(t, read.Led, "override") != "on" ||
		ledString(t, read.Led, "color") != "#ffffff" ||
		brightness.ValueInt64() != 80 {
		t.Errorf("read values were overwritten: %+v", read)
	}

	planned.Led = types.ObjectNull(deviceLedAttrTypes())
	reassertAsyncApplied(context.Background(), &read, &planned)
	if ledString(t, read.Led, "override") != "on" {
		t.Errorf("null planned led overwrote the read: %v", read.Led)
	}
}
