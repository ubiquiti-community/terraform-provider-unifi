package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// The controller stores no interim update interval on a profile that never
// set one: the built-in Default profile has none. A static `1h0m0s` Default
// planned `null -> 1h0m0s` on every adopted profile, and ignore_changes could
// not stop it, because Terraform core hands the provider the prior value
// (null) as configuration and the Default filled it in again (#544). The `1h`
// is now planned on create only.
func TestRadiusProfileInterimIntervalDefaultsOnCreateOnly(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	(&radiusProfileResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}
	s := schemaResp.Schema
	schemaType := s.Type().TerraformType(ctx)

	// An adopted profile with no interval on the controller and a
	// configuration that leaves interim_update out.
	f := planFixture{ctx: ctx, t: t, overrides: map[string]attr.Value{
		"interim_update.interval": timetypes.NewGoDurationNull(),
	}}
	prior, config := f.object(s, "")

	interval := func(t *testing.T, planned tftypes.Value) tftypes.Value {
		t.Helper()
		var attrs map[string]tftypes.Value
		if err := planned.As(&attrs); err != nil {
			t.Fatalf("as object: %v", err)
		}
		return plannedObject(t, attrs["interim_update"], "interim_update")["interval"]
	}

	t.Run("adopted profile keeps no interval", func(t *testing.T) {
		planned := planResourceChange(t, "unifi_radius_profile", s, prior, prior, config)
		if v := interval(t, planned); !v.IsNull() {
			t.Errorf("interim_update.interval planned %v, want null as on the controller", v)
		}
		if diffs, err := prior.Diff(planned); err != nil || len(diffs) != 0 {
			t.Errorf("plan differs from prior state: %v (%v)", diffs, err)
		}
	})

	t.Run("ignore_changes keeps no interval", func(t *testing.T) {
		// For an ignored attribute core sends the prior value as configuration.
		var configAttrs, priorAttrs map[string]tftypes.Value
		if err := config.As(&configAttrs); err != nil {
			t.Fatalf("config as object: %v", err)
		}
		if err := prior.As(&priorAttrs); err != nil {
			t.Fatalf("prior as object: %v", err)
		}
		configAttrs["interim_update"] = priorAttrs["interim_update"]
		ignored := tftypes.NewValue(schemaType, configAttrs)

		planned := planResourceChange(t, "unifi_radius_profile", s, prior, prior, ignored)
		if v := interval(t, planned); !v.IsNull() {
			t.Errorf("interim_update.interval planned %v, want null as on the controller", v)
		}
	})

	t.Run("create plans 1h", func(t *testing.T) {
		created := planResourceChange(
			t, "unifi_radius_profile", s, tftypes.NewValue(schemaType, nil), config, config,
		)
		var got string
		if err := interval(t, created).As(&got); err != nil || got != "1h0m0s" {
			t.Errorf("interim_update.interval on create = %q (%v), want 1h0m0s", got, err)
		}
	})
}
