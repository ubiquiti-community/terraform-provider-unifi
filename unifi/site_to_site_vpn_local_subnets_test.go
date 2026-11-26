package unifi

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func strList(t *testing.T, vals ...string) types.List {
	t.Helper()
	l, d := types.ListValueFrom(context.Background(), types.StringType, vals)
	if d.HasError() {
		t.Fatalf("building list: %v", d)
	}
	return l
}

func TestSiteToSiteVPNLocalSubnets_modelToNetwork(t *testing.T) {
	ctx := context.Background()
	r := &siteToSiteVPNResource{}

	t.Run("selected networks are sent", func(t *testing.T) {
		model := &siteToSiteVPNResourceModel{
			Name:             types.StringValue("vpn"),
			RemoteSubnets:    strList(t, "198.51.100.0/24"),
			LocalSubnetsMode: types.StringValue("selected_networks"),
			LocalNetworkIDs:  strList(t, "net-a", "net-b"),
			LocalSubnets:     types.ListUnknown(types.StringType),
		}
		network, diags := r.modelToNetwork(ctx, model)
		if diags.HasError() {
			t.Fatalf("modelToNetwork: %v", diags)
		}
		if network.LocalVPNSubnetsMode == nil || *network.LocalVPNSubnetsMode != "selected_networks" {
			t.Errorf("LocalVPNSubnetsMode = %v", network.LocalVPNSubnetsMode)
		}
		if got := network.LocalVPNNetworkIDs; len(got) != 2 || got[0] != "net-a" ||
			got[1] != "net-b" {
			t.Errorf("LocalVPNNetworkIDs = %v", got)
		}
		if network.LocalVPNSubnets != nil {
			t.Errorf("LocalVPNSubnets = %v, want nil for an unknown list", network.LocalVPNSubnets)
		}
	})

	t.Run("custom subnets are sent", func(t *testing.T) {
		model := &siteToSiteVPNResourceModel{
			Name:             types.StringValue("vpn"),
			RemoteSubnets:    strList(t, "198.51.100.0/24"),
			LocalSubnetsMode: types.StringValue("custom"),
			LocalSubnets:     strList(t, "192.0.2.0/25"),
			LocalNetworkIDs:  types.ListNull(types.StringType),
		}
		network, diags := r.modelToNetwork(ctx, model)
		if diags.HasError() {
			t.Fatalf("modelToNetwork: %v", diags)
		}
		if got := network.LocalVPNSubnets; len(got) != 1 || got[0] != "192.0.2.0/25" {
			t.Errorf("LocalVPNSubnets = %v", got)
		}
		if network.LocalVPNNetworkIDs != nil {
			t.Errorf("LocalVPNNetworkIDs = %v, want nil", network.LocalVPNNetworkIDs)
		}
	})

	t.Run("unset attributes send nothing", func(t *testing.T) {
		model := &siteToSiteVPNResourceModel{
			Name:             types.StringValue("vpn"),
			RemoteSubnets:    strList(t, "198.51.100.0/24"),
			LocalSubnetsMode: types.StringNull(),
			LocalNetworkIDs:  types.ListNull(types.StringType),
			LocalSubnets:     types.ListNull(types.StringType),
		}
		network, diags := r.modelToNetwork(ctx, model)
		if diags.HasError() {
			t.Fatalf("modelToNetwork: %v", diags)
		}
		if network.LocalVPNSubnetsMode != nil || network.LocalVPNNetworkIDs != nil ||
			network.LocalVPNSubnets != nil {
			t.Errorf("expected no local_vpn_* fields, got %+v", network)
		}
	})
}

func TestSiteToSiteVPNLocalSubnets_networkToModel(t *testing.T) {
	ctx := context.Background()
	r := &siteToSiteVPNResource{}

	t.Run("adopted selection is read back", func(t *testing.T) {
		network := &unifi.Network{
			ID:                  "net-1",
			Purpose:             unifi.PurposeSiteVPN,
			LocalVPNSubnetsMode: unifi.Ptr("selected_networks"),
			LocalVPNNetworkIDs:  []string{"net-a"},
		}
		out := &siteToSiteVPNResourceModel{
			LocalNetworkIDs: types.ListNull(types.StringType),
			LocalSubnets:    types.ListNull(types.StringType),
		}
		if diags := r.networkToModel(ctx, network, out, "default"); diags.HasError() {
			t.Fatalf("networkToModel: %v", diags)
		}
		if out.LocalSubnetsMode.ValueString() != "selected_networks" {
			t.Errorf("LocalSubnetsMode = %v", out.LocalSubnetsMode)
		}
		if len(out.LocalNetworkIDs.Elements()) != 1 {
			t.Errorf("LocalNetworkIDs = %v", out.LocalNetworkIDs)
		}
		if !out.LocalSubnets.IsNull() {
			t.Errorf("LocalSubnets = %v, want null", out.LocalSubnets)
		}
	})

	t.Run("absent fields read back as null", func(t *testing.T) {
		out := &siteToSiteVPNResourceModel{
			LocalNetworkIDs: types.ListUnknown(types.StringType),
			LocalSubnets:    types.ListUnknown(types.StringType),
		}
		if diags := r.networkToModel(ctx, &unifi.Network{ID: "net-1"}, out, "default"); diags.HasError() {
			t.Fatalf("networkToModel: %v", diags)
		}
		if !out.LocalSubnetsMode.IsNull() || !out.LocalNetworkIDs.IsNull() ||
			!out.LocalSubnets.IsNull() {
			t.Errorf("expected nulls, got %v %v %v",
				out.LocalSubnetsMode, out.LocalNetworkIDs, out.LocalSubnets)
		}
	})

	t.Run("an explicitly empty list stays empty", func(t *testing.T) {
		out := &siteToSiteVPNResourceModel{
			LocalSubnets: types.ListValueMust(types.StringType, nil),
		}
		if diags := r.networkToModel(ctx, &unifi.Network{ID: "net-1"}, out, "default"); diags.HasError() {
			t.Fatalf("networkToModel: %v", diags)
		}
		if out.LocalSubnets.IsNull() {
			t.Error("LocalSubnets = null, want the configured empty list")
		}
	})
}

func buildLocalSubnetsConfig(
	t *testing.T,
	ctx context.Context,
	mode types.String,
	networkIDs, subnets types.List,
) tfsdk.Config {
	t.Helper()

	var schemaResp fwresource.SchemaResponse
	(&siteToSiteVPNResource{}).Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)

	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatalf("schema terraform type is not an object")
	}
	vals := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, at := range objType.AttributeTypes {
		vals[name] = tftypes.NewValue(at, nil)
	}
	vals["local_subnets_mode"], _ = mode.ToTerraformValue(ctx)
	vals["local_network_ids"], _ = networkIDs.ToTerraformValue(ctx)
	vals["local_subnets"], _ = subnets.ToTerraformValue(ctx)

	return tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(objType, vals),
	}
}

func TestSiteToSiteVPNLocalSubnetsValidator(t *testing.T) {
	ctx := context.Background()
	null := types.ListNull(types.StringType)
	ids := strList(t, "net-a")
	cidrs := strList(t, "192.0.2.0/25")

	for _, tt := range []struct {
		name      string
		mode      types.String
		ids, cidr types.List
		wantErr   bool
	}{
		{"unset", types.StringNull(), null, null, false},
		{"all", types.StringValue("all"), null, null, false},
		{"all with ids", types.StringValue("all"), ids, null, true},
		{"all with subnets", types.StringValue("all"), null, cidrs, true},
		{"selected with ids", types.StringValue("selected_networks"), ids, null, false},
		{"selected without ids", types.StringValue("selected_networks"), null, null, true},
		{"custom with subnets", types.StringValue("custom"), null, cidrs, false},
		{"custom without subnets", types.StringValue("custom"), null, null, true},
		{"unknown mode defers", types.StringUnknown(), null, null, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resp := &fwresource.ValidateConfigResponse{}
			(&siteToSiteVPNLocalSubnetsValidator{}).ValidateResource(
				ctx,
				fwresource.ValidateConfigRequest{
					Config: buildLocalSubnetsConfig(t, ctx, tt.mode, tt.ids, tt.cidr),
				},
				resp,
			)
			if got := resp.Diagnostics.HasError(); got != tt.wantErr {
				t.Errorf("HasError = %v, want %v: %v", got, tt.wantErr, resp.Diagnostics)
			}
		})
	}
}
