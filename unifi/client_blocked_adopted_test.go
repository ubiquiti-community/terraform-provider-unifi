package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// blocked was Optional+Computed with a static Default(false) next to
// UseStateForUnknown. The framework applies a Default before plan modifiers,
// so the modifier never fired and an adopted blocked client, with blocked
// left out of configuration, planned true -> false: unblocking it (#544).
func TestClientBlockedKeepsControllerValue(t *testing.T) {
	ctx := context.Background()
	s := resourceSchema(t, &clientResource{})
	if a, ok := s.Attributes["blocked"].(schema.BoolAttribute); !ok || a.Default != nil {
		t.Fatalf("blocked must have no static Default")
	}

	f := planFixture{ctx: ctx, t: t, overrides: map[string]attr.Value{
		"blocked": types.BoolValue(true),
	}}
	prior, config := f.object(s, "")
	planned := planResourceChange(t, "unifi_client", s, prior, prior, config)
	if diffs, err := prior.Diff(planned); err != nil || len(diffs) != 0 {
		for _, d := range diffs {
			t.Errorf("adopted client: %s planned %v -> %v", d.Path, d.Value1, d.Value2)
		}
	}

	// On create the attribute is unknown, and nothing is sent for it, so
	// taking over an existing blocked client (allow_existing) keeps it blocked.
	created := planResourceChange(t, "unifi_client", s,
		tftypes.NewValue(s.Type().TerraformType(ctx), nil), config, config)
	var createdAttrs map[string]tftypes.Value
	if err := created.As(&createdAttrs); err != nil {
		t.Fatalf("as object: %v", err)
	}
	if v := createdAttrs["blocked"]; v.IsKnown() {
		t.Errorf("create: blocked = %v, want unknown (taken from the controller)", v)
	}
	if p := clientKnownBoolPointer(types.BoolUnknown()); p != nil {
		t.Errorf("unknown blocked sent as %v, want omitted", *p)
	}
	blocked := true
	merged := (&clientResource{}).mergeClient(
		&unifi.Client{MAC: "02:00:00:00:00:01", Blocked: &blocked},
		&unifi.Client{MAC: "02:00:00:00:00:01"},
	)
	if merged.Blocked == nil || !*merged.Blocked {
		t.Errorf("merge with no planned blocked: got %v, want the existing true", merged.Blocked)
	}
}
