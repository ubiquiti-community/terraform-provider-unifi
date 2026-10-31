package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestPlanPortForwardWanIPAddressOmitted covers a rule whose configuration sets
// wan without ip_address. The controller stores and returns "any" for such a
// rule, so the prior state holds "any"; the plan must keep it instead of
// proposing "any" -> null on every run.
func TestPlanPortForwardWanIPAddressOmitted(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	(&portForwardResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}
	s := schemaResp.Schema
	wanAttr, ok := s.Attributes["wan"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("wan is %T, want SingleNestedAttribute", s.Attributes["wan"])
	}
	wanType := wanAttr.GetType().TerraformType(ctx)

	wan := func(ip tftypes.Value) tftypes.Value {
		return tftypes.NewValue(wanType, map[string]tftypes.Value{
			"interface":  tftypes.NewValue(tftypes.String, "wan"),
			"ip_address": ip,
			"port":       tftypes.NewValue(tftypes.String, "443"),
		})
	}
	withWan := func(obj, w tftypes.Value) tftypes.Value {
		var attrs map[string]tftypes.Value
		if err := obj.As(&attrs); err != nil {
			t.Fatalf("as object: %v", err)
		}
		attrs["wan"] = w
		return tftypes.NewValue(obj.Type(), attrs)
	}

	nullIP := tftypes.NewValue(tftypes.String, nil)
	anyIP := tftypes.NewValue(tftypes.String, "any")
	f := planFixture{ctx: ctx, t: t}
	prior, config := f.object(s, "")
	prior = withWan(prior, wan(anyIP))
	config = withWan(config, wan(nullIP))

	// Terraform core proposes the prior value for a null Computed leaf and the
	// configured null for an Optional-only one.
	proposedIP := nullIP
	if wanAttr.Attributes["ip_address"].IsComputed() {
		proposedIP = anyIP
	}
	proposed := withWan(prior, wan(proposedIP))

	planned := planResourceChange(t, "unifi_port_forward", s, prior, proposed, config)

	var attrs map[string]tftypes.Value
	if err := planned.As(&attrs); err != nil {
		t.Fatalf("as object: %v", err)
	}
	plannedWan := plannedObject(t, attrs["wan"], "wan")
	var ip string
	if err := plannedWan["ip_address"].As(&ip); err != nil || ip != "any" {
		t.Errorf(
			"wan.ip_address planned as %v (%v), want \"any\" from state",
			plannedWan["ip_address"],
			err,
		)
	}
}
