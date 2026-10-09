package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func ptrNatInt64(v int64) *int64 { return &v }

func newNatTestModel() natResourceModel {
	return natResourceModel{
		SourceFilter:      types.ObjectNull(natFilterAttrTypes()),
		DestinationFilter: types.ObjectNull(natFilterAttrTypes()),
	}
}

func TestNewNatResource(t *testing.T) {
	r := NewNatResource()
	if _, ok := r.(fwresource.ResourceWithConfigure); !ok {
		t.Error("expected ResourceWithConfigure")
	}
	if _, ok := r.(fwresource.ResourceWithImportState); !ok {
		t.Error("expected ResourceWithImportState")
	}
	if _, ok := r.(fwresource.ResourceWithIdentity); !ok {
		t.Error("expected ResourceWithIdentity")
	}
}

func Test_natResource_Metadata(t *testing.T) {
	resp := &fwresource.MetadataResponse{}
	(&natResource{}).Metadata(
		context.Background(),
		fwresource.MetadataRequest{ProviderTypeName: "unifi"},
		resp,
	)
	if resp.TypeName != "unifi_nat" {
		t.Errorf("TypeName = %q, want unifi_nat", resp.TypeName)
	}
}

func Test_natResource_Schema(t *testing.T) {
	ctx := context.Background()
	resp := &fwresource.SchemaResponse{}
	(&natResource{}).Schema(ctx, fwresource.SchemaRequest{}, resp)

	if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Fatalf("schema is invalid: %v", diags)
	}
	for _, name := range []string{
		"id", "site", "description", "enabled", "type", "ip_version", "protocol",
		"in_interface", "out_interface", "ip_address", "port", "exclude", "logging",
		"pppoe_use_base_interface", "rule_index", "setting_preference",
		"source_filter", "destination_filter",
	} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("expected attribute %q in schema", name)
		}
	}
}

func Test_natResource_Configure(t *testing.T) {
	tests := []struct {
		name      string
		req       fwresource.ConfigureRequest
		wantError bool
	}{
		{"nil_provider_data", fwresource.ConfigureRequest{}, false},
		{"wrong_type", fwresource.ConfigureRequest{ProviderData: "wrong"}, true},
		{"correct_client", fwresource.ConfigureRequest{ProviderData: &Client{}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &fwresource.ConfigureResponse{}
			(&natResource{}).Configure(context.Background(), tt.req, resp)
			if resp.Diagnostics.HasError() != tt.wantError {
				t.Errorf("hasError = %v, want %v", resp.Diagnostics.HasError(), tt.wantError)
			}
		})
	}
}

func Test_natModelToAPI(t *testing.T) {
	ctx := context.Background()

	t.Run("filters and ports reach the API", func(t *testing.T) {
		groups := types.SetValueMust(types.StringType, []attr.Value{types.StringValue("fg1")})
		src, diags := types.ObjectValueFrom(ctx, natFilterAttrTypes(), natFilterModel{
			FilterType:       types.StringValue("FIREWALL_GROUPS"),
			Address:          types.StringNull(),
			Port:             types.Int64Null(),
			InvertAddress:    types.BoolValue(true),
			InvertPort:       types.BoolValue(false),
			NetworkConfID:    types.StringNull(),
			FirewallGroupIDs: groups,
		})
		if diags.HasError() {
			t.Fatal(diags)
		}

		m := newNatTestModel()
		m.Description = types.StringValue("web")
		m.Enabled = types.BoolValue(true)
		m.Type = types.StringValue("DNAT")
		m.Protocol = types.StringValue("tcp")
		m.IPAddress = types.StringValue("10.0.0.5")
		m.Port = types.Int64Value(8080)
		m.SourceFilter = src

		got, diags := natModelToAPI(ctx, &m)
		if diags.HasError() {
			t.Fatal(diags)
		}

		if got.Description != "web" || got.Type != "DNAT" || got.Protocol != "tcp" {
			t.Errorf("scalar fields not copied: %+v", got)
		}
		if got.Port == nil || *got.Port != 8080 {
			t.Errorf("Port = %v, want 8080", got.Port)
		}
		if got.SourceFilter == nil || got.SourceFilter.FilterType != "FIREWALL_GROUPS" ||
			!got.SourceFilter.InvertAddress ||
			len(got.SourceFilter.FirewallGroupIDs) != 1 ||
			got.SourceFilter.FirewallGroupIDs[0] != "fg1" {
			t.Errorf("SourceFilter = %+v", got.SourceFilter)
		}
		if got.DestinationFilter == nil || got.DestinationFilter.FilterType != "NONE" {
			t.Errorf("DestinationFilter = %+v, want the NONE default", got.DestinationFilter)
		}
	})

	t.Run("unset values are sent as the UI defaults", func(t *testing.T) {
		m := newNatTestModel()
		m.Description = types.StringValue("x")
		m.Type = types.StringValue("MASQUERADE")
		m.IPVersion = types.StringUnknown()
		m.Protocol = types.StringUnknown()
		m.RuleIndex = types.Int64Unknown()
		m.SettingPreference = types.StringUnknown()

		got, diags := natModelToAPI(ctx, &m)
		if diags.HasError() {
			t.Fatal(diags)
		}
		if got.Version != "IPV4" || got.Protocol != "all" || got.SettingPreference != "manual" {
			t.Errorf("defaults not applied: %+v", got)
		}
		if got.SourceFilter == nil || got.SourceFilter.FilterType != "NONE" ||
			got.DestinationFilter == nil || got.DestinationFilter.FilterType != "NONE" {
			t.Errorf("filters must default to NONE: src=%+v dst=%+v",
				got.SourceFilter, got.DestinationFilter)
		}
		if got.RuleIndex != nil || got.Port != nil {
			t.Errorf("unknown/null ints must be nil: rule_index=%v port=%v",
				got.RuleIndex, got.Port)
		}
	})

	t.Run("configured values win over defaults", func(t *testing.T) {
		m := newNatTestModel()
		m.Description = types.StringValue("x")
		m.Type = types.StringValue("SNAT")
		m.IPVersion = types.StringValue("IPV6")
		m.Protocol = types.StringValue("udp")
		m.SettingPreference = types.StringValue("auto")

		got, diags := natModelToAPI(ctx, &m)
		if diags.HasError() {
			t.Fatal(diags)
		}
		if got.Version != "IPV6" || got.Protocol != "udp" || got.SettingPreference != "auto" {
			t.Errorf("configured values overridden: %+v", got)
		}
	})
}

func Test_natAPIToModel(t *testing.T) {
	ctx := context.Background()

	t.Run("empty values become null", func(t *testing.T) {
		m := newNatTestModel()
		diags := natAPIToModel(ctx, &unifi.Nat{
			ID:          "abc",
			Description: "rule",
			Type:        "SNAT",
			Enabled:     true,
		}, &m, "default")
		if diags.HasError() {
			t.Fatal(diags)
		}

		if m.ID.ValueString() != "abc" || m.Site.ValueString() != "default" {
			t.Errorf("id/site = %q/%q", m.ID.ValueString(), m.Site.ValueString())
		}
		if !m.InInterface.IsNull() || !m.IPAddress.IsNull() || !m.Port.IsNull() ||
			!m.RuleIndex.IsNull() {
			t.Errorf("empty API values must be null: %+v", m)
		}
	})

	t.Run("NONE filter stays null when not configured", func(t *testing.T) {
		m := newNatTestModel()
		diags := natAPIToModel(ctx, &unifi.Nat{
			ID:           "abc",
			SourceFilter: &unifi.NatSourceFilter{FilterType: "NONE"},
		}, &m, "default")
		if diags.HasError() {
			t.Fatal(diags)
		}
		if !m.SourceFilter.IsNull() {
			t.Errorf("SourceFilter = %v, want null", m.SourceFilter)
		}
	})

	t.Run("NONE filter is kept when configured", func(t *testing.T) {
		configured, d := types.ObjectValueFrom(ctx, natFilterAttrTypes(), natFilterModel{
			FilterType:       types.StringValue("NONE"),
			Address:          types.StringNull(),
			Port:             types.Int64Null(),
			InvertAddress:    types.BoolValue(false),
			InvertPort:       types.BoolValue(false),
			NetworkConfID:    types.StringNull(),
			FirewallGroupIDs: types.SetNull(types.StringType),
		})
		if d.HasError() {
			t.Fatal(d)
		}

		m := newNatTestModel()
		m.SourceFilter = configured
		diags := natAPIToModel(ctx, &unifi.Nat{
			ID:           "abc",
			SourceFilter: &unifi.NatSourceFilter{FilterType: "NONE"},
		}, &m, "default")
		if diags.HasError() {
			t.Fatal(diags)
		}
		if m.SourceFilter.IsNull() {
			t.Error("a configured NONE filter must not be dropped")
		}
	})

	t.Run("round trip preserves a destination filter", func(t *testing.T) {
		api := &unifi.Nat{
			ID:   "abc",
			Type: "DNAT",
			DestinationFilter: &unifi.NatDestinationFilter{
				FilterType:    "ADDRESS_AND_PORT",
				Address:       "203.0.113.0/24",
				Port:          ptrNatInt64(443),
				InvertAddress: true,
			},
		}

		m := newNatTestModel()
		if diags := natAPIToModel(ctx, api, &m, "default"); diags.HasError() {
			t.Fatal(diags)
		}
		back, diags := natModelToAPI(ctx, &m)
		if diags.HasError() {
			t.Fatal(diags)
		}

		got := back.DestinationFilter
		if got == nil || got.FilterType != "ADDRESS_AND_PORT" ||
			got.Address != "203.0.113.0/24" || got.Port == nil || *got.Port != 443 ||
			!got.InvertAddress {
			t.Errorf("DestinationFilter = %+v", got)
		}
	})
}
