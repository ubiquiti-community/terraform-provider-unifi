package unifi

import (
	"context"
	"maps"
	"strings"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// The magic_site_to_site_vpn document also holds the gateway's WireGuard key
// pair, which go-unifi does not model. Turning Site Magic off must send only
// enabled, or the gateway would lose its keys (#722).
func TestSettingSiteMagicWritesOnlyEnabled(t *testing.T) {
	ctx := context.Background()
	fc := &fakeSettingController{docs: allSettingDocuments}
	r := &settingResource{client: newSettingFakeControllerClient(t, fc)}

	imported := importSettings(ctx, t, r, false)
	if imported.Diagnostics.HasError() {
		t.Fatalf("ImportState: %v", imported.Diagnostics)
	}
	var plan settingResourceModel
	if d := imported.State.Get(ctx, &plan); d.HasError() {
		t.Fatalf("reading imported state: %v", d)
	}
	var magic settingMagicSiteToSiteVpnModel
	if d := plan.MagicSiteToSiteVpn.As(ctx, &magic, basetypes.ObjectAsOptions{}); d.HasError() {
		t.Fatalf("magic_site_to_site_vpn: %v", d)
	}
	if !magic.Enabled.ValueBool() {
		t.Fatalf("magic_site_to_site_vpn.enabled imported as %v, want true", magic.Enabled)
	}
	magic.Enabled = types.BoolValue(false)
	obj, d := types.ObjectValueFrom(ctx, magicSiteToSiteVpnAttrTypes, magic)
	if d.HasError() {
		t.Fatal(d)
	}
	plan.MagicSiteToSiteVpn = obj
	planned := tfsdk.Plan{Schema: imported.State.Schema}
	if d := planned.Set(ctx, &plan); d.HasError() {
		t.Fatalf("building plan: %v", d)
	}

	fc.puts = nil
	resp := &fwresource.UpdateResponse{State: imported.State, Identity: imported.Identity}
	r.Update(ctx, fwresource.UpdateRequest{
		State: imported.State, Plan: planned, Config: tfsdk.Config(planned),
		Identity: imported.Identity,
	}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", resp.Diagnostics)
	}
	if strings.Join(fc.puts, ",") != "magic_site_to_site_vpn" {
		t.Fatalf("update wrote %v, want only magic_site_to_site_vpn", fc.puts)
	}
	body := string(fc.bodies["magic_site_to_site_vpn"])
	if !strings.Contains(body, `"enabled":false`) || strings.Contains(body, "key\":\"site-magic") ||
		strings.Contains(body, "public_key") || strings.Contains(body, "x_private_key") {
		t.Errorf("magic_site_to_site_vpn body = %s, want only enabled", body)
	}
}

// Controllers before Teleport and Site Magic store neither key. A refresh of a
// configuration that sets them reads them as unset, so the plan writes them,
// instead of failing.
func TestSettingReadsMissingTeleportAndSiteMagicAsUnset(t *testing.T) {
	ctx := context.Background()
	fc := &fakeSettingController{docs: allSettingDocuments}
	r := &settingResource{client: newSettingFakeControllerClient(t, fc)}

	imported := importSettings(ctx, t, r, false)
	if imported.Diagnostics.HasError() {
		t.Fatalf("ImportState: %v", imported.Diagnostics)
	}
	fc.docs = maps.Clone(allSettingDocuments)
	delete(fc.docs, "teleport")
	delete(fc.docs, "magic_site_to_site_vpn")

	resp := &fwresource.ReadResponse{State: imported.State, Identity: imported.Identity}
	r.Read(ctx, fwresource.ReadRequest{State: imported.State, Identity: imported.Identity}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var got settingResourceModel
	if d := resp.State.Get(ctx, &got); d.HasError() {
		t.Fatal(d)
	}
	if !got.Teleport.IsNull() || !got.MagicSiteToSiteVpn.IsNull() {
		t.Errorf("teleport = %v, magic_site_to_site_vpn = %v; want both unset",
			got.Teleport, got.MagicSiteToSiteVpn)
	}
	if got.Ntp.IsNull() {
		t.Error("ntp read as unset; only the missing sections should be")
	}
}
