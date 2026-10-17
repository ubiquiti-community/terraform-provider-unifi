package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestPortProfileUnsetForwardPlansUnknown checks that an unset forward plans
// unknown: the controller stores it in step with tagged_vlan_mgmt, so a planned
// default would contradict it for some modes.
func TestPortProfileUnsetForwardPlansUnknown(t *testing.T) {
	ctx := context.Background()
	r := &portProfileResource{}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	f := planFixture{ctx: ctx, t: t}
	_, config := f.object(schemaResp.Schema, "")

	planned := planResourceChange(t, "unifi_port_profile", schemaResp.Schema,
		tftypes.NewValue(schemaType, nil), config, config)
	var attrs map[string]tftypes.Value
	if err := planned.As(&attrs); err != nil {
		t.Fatalf("planned as object: %v", err)
	}
	if v := attrs["forward"]; v.IsKnown() {
		t.Errorf("forward plans %v, want unknown so it can follow tagged_vlan_mgmt", v)
	}
}
