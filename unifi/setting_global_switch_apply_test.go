package unifi

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
)

// newSettingHarness imports the site settings from a fake controller holding
// stored, the way an adopted site starts.
func newSettingHarness(
	t *testing.T,
	stored []map[string]any,
) (*updateHarness, func() map[string]map[string]any) {
	t.Helper()
	ctx := context.Background()
	r, puts := newSettingsFakeController(t, stored)
	h := &updateHarness{t: t, ctx: ctx, r: r, typeName: "unifi_setting", id: "default"}

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	h.schema = schemaResp.Schema
	var identityResp fwresource.IdentitySchemaResponse
	r.IdentitySchema(ctx, fwresource.IdentitySchemaRequest{}, &identityResp)
	h.state = tfsdk.State{
		Schema: h.schema,
		Raw:    tftypes.NewValue(h.schema.Type().TerraformType(ctx), nil),
	}
	h.identity = &tfsdk.ResourceIdentity{
		Schema: identityResp.IdentitySchema,
		Raw:    tftypes.NewValue(identityResp.IdentitySchema.Type().TerraformType(ctx), nil),
	}
	h.importAndRead()
	puts()
	return h, puts
}

// applySetting is updateHarness.apply for the settings fake: plan, Update,
// then check the new state against the plan, a refresh, and a second plan.
func applySetting(
	h *updateHarness,
	puts func() map[string]map[string]any,
	config tftypes.Value,
) map[string]map[string]any {
	h.t.Helper()
	planned := h.plan(config)
	if diags := h.update(planned, config); diags.HasError() {
		h.t.Fatalf("Update: %v", diagDetails(diags))
	}
	resolved, err := util.ResolveUnknowns(planned, h.state.Raw)
	if err != nil {
		h.t.Fatalf("resolving the plan from the new state: %v", err)
	}
	h.assertEqual("state after Update differs from the plan", resolved, h.state.Raw)
	h.assertEqual("refresh after Update changed the state", h.state.Raw, h.read().Raw)
	h.assertEqual("second plan is not empty", h.state.Raw, h.plan(config))
	return puts()
}

func globalSwitchConfig(set map[string]attr.Value) attr.Value {
	attrs := map[string]attr.Value{}
	for name, typ := range globalSwitchAttrTypes {
		v, ok := set[name]
		if !ok {
			v, _ = typ.ValueFromTerraform(
				context.Background(),
				tftypes.NewValue(typ.TerraformType(context.Background()), nil),
			)
		}
		attrs[name] = v
	}
	return types.ObjectValueMust(globalSwitchAttrTypes, attrs)
}

func stringList(items ...string) attr.Value {
	elems := make([]attr.Value, len(items))
	for i, s := range items {
		elems[i] = types.StringValue(s)
	}
	return types.ListValueMust(types.StringType, elems)
}

// TestSettingGlobalSwitchApplyCycles runs each global_switch configuration
// through plan, Update, refresh and a second plan against an adopted site, and
// checks what reaches the controller. acl_device_isolation is absent from a
// controller that never set it, and reads back as null.
func TestSettingGlobalSwitchApplyCycles(t *testing.T) {
	stored := func(acl ...string) []map[string]any {
		gs := map[string]any{
			"key":                    "global_switch",
			"stp_version":            "rstp",
			"dhcp_snoop":             true,
			"jumboframe_enabled":     false,
			"dot1x_portctrl_enabled": false,
		}
		if acl != nil {
			list := make([]any, len(acl))
			for i, s := range acl {
				list[i] = s
			}
			gs["acl_device_isolation"] = list
		}
		return []map[string]any{gs}
	}
	none := []string(nil)

	cases := []struct {
		name   string
		stored []map[string]any
		gs     map[string]attr.Value // nil omits the block
		want   map[string]map[string]any
	}{
		{
			name:   "omitted, acl absent",
			stored: stored(none...),
		},
		{
			name:   "omitted, acl empty",
			stored: stored([]string{}...),
		},
		{
			name:   "omitted, acl set",
			stored: stored("net-a"),
		},
		{
			name:   "configured unchanged, acl absent",
			stored: stored(none...),
			gs:     map[string]attr.Value{"dhcp_snoop": types.BoolValue(true)},
		},
		{
			name:   "configured change, acl absent",
			stored: stored(none...),
			gs:     map[string]attr.Value{"dhcp_snoop": types.BoolValue(false)},
			want: map[string]map[string]any{
				"global_switch": {"key": "global_switch", "dhcp_snoop": false},
			},
		},
		{
			name:   "configured change, acl set",
			stored: stored("net-a"),
			gs:     map[string]attr.Value{"jumboframe_enabled": types.BoolValue(true)},
			want: map[string]map[string]any{
				"global_switch": {"key": "global_switch", "jumboframe_enabled": true},
			},
		},
		{
			name:   "acl set from absent",
			stored: stored(none...),
			gs:     map[string]attr.Value{"acl_device_isolation": stringList("net-a")},
			want: map[string]map[string]any{
				"global_switch": {"key": "global_switch", "acl_device_isolation": []any{"net-a"}},
			},
		},
		{
			name:   "acl empty from absent",
			stored: stored(none...),
			gs:     map[string]attr.Value{"acl_device_isolation": stringList()},
			want: map[string]map[string]any{
				"global_switch": {"key": "global_switch", "acl_device_isolation": []any{}},
			},
		},
		{
			name:   "acl cleared",
			stored: stored("net-a"),
			gs:     map[string]attr.Value{"acl_device_isolation": stringList()},
			want: map[string]map[string]any{
				"global_switch": {"key": "global_switch", "acl_device_isolation": []any{}},
			},
		},
		{
			name:   "acl unchanged",
			stored: stored("net-a"),
			gs:     map[string]attr.Value{"acl_device_isolation": stringList("net-a")},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, puts := newSettingHarness(t, tc.stored)
			set := map[string]attr.Value{"site": types.StringValue("default")}
			if tc.gs != nil {
				set["global_switch"] = globalSwitchConfig(tc.gs)
			}
			got := applySetting(h, puts, h.config(set))
			if len(got) != 0 || len(tc.want) != 0 {
				if !reflect.DeepEqual(got, tc.want) {
					t.Errorf("PUTs = %v, want %v", got, tc.want)
				}
			}
		})
	}

	t.Run("block removed after an apply", func(t *testing.T) {
		h, puts := newSettingHarness(t, stored(none...))
		site := types.StringValue("default")
		applySetting(h, puts, h.config(map[string]attr.Value{
			"site": site,
			"global_switch": globalSwitchConfig(
				map[string]attr.Value{"jumboframe_enabled": types.BoolValue(true)}),
		}))
		if got := applySetting(
			h,
			puts,
			h.config(map[string]attr.Value{"site": site}),
		); len(
			got,
		) != 0 {
			t.Errorf("removing the block wrote %v", got)
		}
	})
}
