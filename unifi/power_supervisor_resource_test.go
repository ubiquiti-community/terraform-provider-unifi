package unifi

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/hwtypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
)

// TestPowerSupervisorModelRoundTrip covers the model ⇄ go-unifi conversion for
// the Device Supervisor resource (#244): settings are sent as the user set them,
// power_sources are not sent (the controller resolves them) but are read back,
// and the computed consecutive_failures / id / power_sources land in the model.
func TestPowerSupervisorModelRoundTrip(t *testing.T) {
	ctx := context.Background()
	r := &powerSupervisorResource{}

	model := powerSupervisorResourceModel{
		DeviceMAC:         hwtypes.NewMACAddressValue("94:2a:6f:d6:ce:fd"),
		Enabled:           types.BoolValue(true),
		HeartbeatInterval: util.DurationValue(30, time.Second),
		SilenceThreshold:  util.DurationValue(600, time.Second),
		PowerOffDuration:  util.DurationValue(90, time.Second),
	}

	api := r.modelToPowerSupervisor(&model)
	if api.ClientMAC != "94:2a:6f:d6:ce:fd" {
		t.Errorf("ClientMAC = %q, want the device MAC", api.ClientMAC)
	}
	if !api.Enabled {
		t.Errorf("Enabled = false, want true")
	}
	if api.Settings.HeartbeatInterval != 30 || api.Settings.SilenceThreshold != 600 ||
		api.Settings.PowerOffDuration != 90 {
		t.Errorf("settings not mapped: %+v", api.Settings)
	}
	if api.PowerSources == nil {
		t.Errorf("PowerSources should be non-nil (sent empty), got nil")
	}

	// Simulate the controller's resting response and read it back.
	resp := &unifi.PowerSupervisor{
		ID:                  "000000000000000000000001",
		ClientMAC:           "94:2a:6f:d6:ce:fd",
		Enabled:             true,
		ConsecutiveFailures: 2,
		Settings: unifi.PowerSupervisorSettings{
			HeartbeatInterval: 30, SilenceThreshold: 600, PowerOffDuration: 90,
		},
		PowerSources: []unifi.PowerSupervisorSource{
			{
				ClientPsuIndex:   1,
				PowerSourceIndex: 4,
				PowerSourceMAC:   "f4:e2:c6:ad:4f:82",
				PowerSourceType:  "poe_port",
			},
		},
	}

	var out powerSupervisorResourceModel
	if d := r.powerSupervisorToModel(resp, &out, "default"); d.HasError() {
		t.Fatalf("powerSupervisorToModel: %v", d)
	}
	if out.ID.ValueString() != "000000000000000000000001" {
		t.Errorf("ID = %q", out.ID.ValueString())
	}
	if out.ConsecutiveFailures.ValueInt64() != 2 {
		t.Errorf("ConsecutiveFailures = %d, want 2", out.ConsecutiveFailures.ValueInt64())
	}
	if out.Site.ValueString() != "default" {
		t.Errorf("Site = %q, want default", out.Site.ValueString())
	}

	var sources []struct {
		ClientPsuIndex   int64  `tfsdk:"client_psu_index"`
		PowerSourceIndex int64  `tfsdk:"power_source_index"`
		PowerSourceMAC   string `tfsdk:"power_source_mac"`
		PowerSourceType  string `tfsdk:"power_source_type"`
	}
	if d := out.PowerSources.ElementsAs(ctx, &sources, false); d.HasError() {
		t.Fatalf("reading power_sources: %v", d)
	}
	if len(sources) != 1 || sources[0].PowerSourceMAC != "f4:e2:c6:ad:4f:82" ||
		sources[0].PowerSourceType != "poe_port" || sources[0].PowerSourceIndex != 4 {
		t.Errorf("power_sources not read back correctly: %+v", sources)
	}
}

func TestNewPowerSupervisorResource(t *testing.T) {
	r := NewPowerSupervisorResource()
	if r == nil {
		t.Fatal("NewPowerSupervisorResource() returned nil")
	}
	if _, ok := r.(fwresource.ResourceWithConfigure); !ok {
		t.Error("expected ResourceWithConfigure interface")
	}
	if _, ok := r.(fwresource.ResourceWithImportState); !ok {
		t.Error("expected ResourceWithImportState interface")
	}
	if _, ok := r.(fwresource.ResourceWithIdentity); !ok {
		t.Error("expected ResourceWithIdentity interface")
	}
	if _, ok := r.(fwresource.ResourceWithUpgradeState); !ok {
		t.Error("expected ResourceWithUpgradeState interface")
	}
}

func TestNewPowerSupervisorListResource(t *testing.T) {
	r := NewPowerSupervisorListResource()
	if r == nil {
		t.Fatal("NewPowerSupervisorListResource() returned nil")
	}
	if _, ok := r.(fwlist.ListResourceWithConfigure); !ok {
		t.Error("expected fwlist.ListResourceWithConfigure interface")
	}
}

func Test_powerSourceAttrTypes(t *testing.T) {
	got := powerSourceAttrTypes()
	want := map[string]attr.Type{
		"client_psu_index":   types.Int64Type,
		"power_source_index": types.Int64Type,
		"power_source_mac":   types.StringType,
		"power_source_type":  types.StringType,
	}
	for k, wantType := range want {
		if gotType, ok := got[k]; !ok {
			t.Errorf("powerSourceAttrTypes() missing key %q", k)
		} else if gotType != wantType {
			t.Errorf("powerSourceAttrTypes()[%q] = %v, want %v", k, gotType, wantType)
		}
	}
}

func Test_powerSupervisorResource_Metadata(t *testing.T) {
	for _, tt := range []struct{ p, w string }{
		{"unifi", "unifi_power_supervisor"},
		{"test", "test_power_supervisor"},
	} {
		t.Run(tt.p, func(t *testing.T) {
			r := &powerSupervisorResource{}
			resp := &fwresource.MetadataResponse{}
			r.Metadata(
				context.Background(),
				fwresource.MetadataRequest{ProviderTypeName: tt.p},
				resp,
			)
			if resp.TypeName != tt.w {
				t.Errorf("TypeName = %q, want %q", resp.TypeName, tt.w)
			}
		})
	}
}

func Test_powerSupervisorResource_IdentitySchema(t *testing.T) {
	r := &powerSupervisorResource{}
	resp := &fwresource.IdentitySchemaResponse{}
	r.IdentitySchema(context.Background(), fwresource.IdentitySchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("IdentitySchema() returned errors: %v", resp.Diagnostics)
	}
	if len(resp.IdentitySchema.Attributes) == 0 {
		t.Error("IdentitySchema() returned no attributes")
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Error("IdentitySchema() missing 'id' attribute")
	}
}

func Test_powerSupervisorResource_Schema(t *testing.T) {
	r := &powerSupervisorResource{}
	resp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() returned errors: %v", resp.Diagnostics)
	}
	for _, key := range []string{
		"id", "site", "device_mac", "enabled", "heartbeat_interval",
		"silence_threshold", "power_off_duration", "consecutive_failures", "power_sources", "timeouts",
	} {
		if _, ok := resp.Schema.Attributes[key]; !ok {
			t.Errorf("Schema() missing attribute %q", key)
		}
	}
}

func Test_powerSupervisorResource_UpgradeState(t *testing.T) {
	r := &powerSupervisorResource{}
	got := r.UpgradeState(context.Background())
	if got == nil {
		t.Fatal("UpgradeState() returned nil")
	}
	if _, ok := got[0]; !ok {
		t.Error("UpgradeState() missing version 0 upgrader")
	}
}

func Test_powerSupervisorResource_Configure(t *testing.T) {
	for _, tt := range []struct {
		name string
		data any
		err  bool
	}{
		{"nil", nil, false},
		{"wrong", "wrong", true},
		{"ok", &Client{}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := &powerSupervisorResource{}
			resp := &fwresource.ConfigureResponse{}
			r.Configure(
				context.Background(),
				fwresource.ConfigureRequest{ProviderData: tt.data},
				resp,
			)
			if tt.err && !resp.Diagnostics.HasError() {
				t.Error("expected error in diagnostics")
			}
			if !tt.err && resp.Diagnostics.HasError() {
				t.Errorf("unexpected error: %v", resp.Diagnostics)
			}
		})
	}
}

func Test_powerSupervisorResource_modelToPowerSupervisor(t *testing.T) {
	r := &powerSupervisorResource{}

	t.Run("maps all settings fields", func(t *testing.T) {
		model := &powerSupervisorResourceModel{
			DeviceMAC:         hwtypes.NewMACAddressValue("aa:bb:cc:dd:ee:ff"),
			Enabled:           types.BoolValue(true),
			HeartbeatInterval: util.DurationValue(60, time.Second),
			SilenceThreshold:  util.DurationValue(300, time.Second),
			PowerOffDuration:  util.DurationValue(120, time.Second),
		}
		got := r.modelToPowerSupervisor(model)
		if got == nil {
			t.Fatal("modelToPowerSupervisor() returned nil")
		}
		if got.ClientMAC != "aa:bb:cc:dd:ee:ff" {
			t.Errorf("ClientMAC = %q, want aa:bb:cc:dd:ee:ff", got.ClientMAC)
		}
		if !got.Enabled {
			t.Error("Enabled should be true")
		}
		if got.Settings.HeartbeatInterval != 60 {
			t.Errorf("HeartbeatInterval = %d, want 60", got.Settings.HeartbeatInterval)
		}
		if got.Settings.SilenceThreshold != 300 {
			t.Errorf("SilenceThreshold = %d, want 300", got.Settings.SilenceThreshold)
		}
		if got.Settings.PowerOffDuration != 120 {
			t.Errorf("PowerOffDuration = %d, want 120", got.Settings.PowerOffDuration)
		}
		if got.PowerSources == nil {
			t.Error("PowerSources should be non-nil empty slice")
		}
	})

	t.Run("disabled supervisor", func(t *testing.T) {
		model := &powerSupervisorResourceModel{
			DeviceMAC:         hwtypes.NewMACAddressValue("11:22:33:44:55:66"),
			Enabled:           types.BoolValue(false),
			HeartbeatInterval: util.DurationValue(30, time.Second),
			SilenceThreshold:  util.DurationValue(900, time.Second),
			PowerOffDuration:  util.DurationValue(60, time.Second),
		}
		got := r.modelToPowerSupervisor(model)
		if got.Enabled {
			t.Error("Enabled should be false")
		}
		if got.ClientMAC != "11:22:33:44:55:66" {
			t.Errorf("ClientMAC = %q", got.ClientMAC)
		}
	})
}

func Test_powerSupervisorResource_powerSupervisorToModel(t *testing.T) {
	r := &powerSupervisorResource{}

	t.Run("populates all fields", func(t *testing.T) {
		supervisor := &unifi.PowerSupervisor{
			ID:                  "sup-123",
			ClientMAC:           "aa:bb:cc:dd:ee:ff",
			Enabled:             true,
			ConsecutiveFailures: 3,
			Settings: unifi.PowerSupervisorSettings{
				HeartbeatInterval: 60,
				SilenceThreshold:  900,
				PowerOffDuration:  120,
			},
			PowerSources: []unifi.PowerSupervisorSource{
				{
					ClientPsuIndex:   0,
					PowerSourceIndex: 2,
					PowerSourceMAC:   "de:ad:be:ef:00:01",
					PowerSourceType:  "poe_port",
				},
			},
		}
		var model powerSupervisorResourceModel
		diags := r.powerSupervisorToModel(supervisor, &model, "site1")
		if diags.HasError() {
			t.Fatalf("powerSupervisorToModel() errors: %v", diags)
		}
		if model.ID.ValueString() != "sup-123" {
			t.Errorf("ID = %q, want sup-123", model.ID.ValueString())
		}
		if model.Site.ValueString() != "site1" {
			t.Errorf("Site = %q, want site1", model.Site.ValueString())
		}
		if model.DeviceMAC.ValueString() != "aa:bb:cc:dd:ee:ff" {
			t.Errorf("DeviceMAC = %q", model.DeviceMAC.ValueString())
		}
		if !model.Enabled.ValueBool() {
			t.Error("Enabled should be true")
		}
		if model.ConsecutiveFailures.ValueInt64() != 3 {
			t.Errorf("ConsecutiveFailures = %d, want 3", model.ConsecutiveFailures.ValueInt64())
		}
		if model.PowerSources.IsNull() || model.PowerSources.IsUnknown() {
			t.Error("PowerSources should be a non-null list")
		}
		if len(model.PowerSources.Elements()) != 1 {
			t.Errorf("PowerSources len = %d, want 1", len(model.PowerSources.Elements()))
		}
	})

	t.Run("empty power sources", func(t *testing.T) {
		supervisor := &unifi.PowerSupervisor{
			ID:           "sup-456",
			ClientMAC:    "11:22:33:44:55:66",
			Enabled:      false,
			PowerSources: []unifi.PowerSupervisorSource{},
		}
		var model powerSupervisorResourceModel
		diags := r.powerSupervisorToModel(supervisor, &model, "default")
		if diags.HasError() {
			t.Fatalf("powerSupervisorToModel() errors: %v", diags)
		}
		if len(model.PowerSources.Elements()) != 0 {
			t.Errorf(
				"PowerSources should be empty, got %d elements",
				len(model.PowerSources.Elements()),
			)
		}
	})
}

// powerSupervisorImportHarness builds the empty state and identity containers
// an ImportState call receives from the framework, so the import entry paths
// can be unit-tested without a live controller.
func powerSupervisorImportHarness(
	t *testing.T,
) (*powerSupervisorResource, tfsdk.State, tfsdk.ResourceIdentity) {
	t.Helper()
	ctx := context.Background()
	r := &powerSupervisorResource{client: &Client{Site: "default"}}

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema() errors: %v", schemaResp.Diagnostics)
	}

	var idResp fwresource.IdentitySchemaResponse
	r.IdentitySchema(ctx, fwresource.IdentitySchemaRequest{}, &idResp)
	if idResp.Diagnostics.HasError() {
		t.Fatalf("IdentitySchema() errors: %v", idResp.Diagnostics)
	}

	state := tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
	}
	identity := tfsdk.ResourceIdentity{
		Schema: idResp.IdentitySchema,
		Raw:    tftypes.NewValue(idResp.IdentitySchema.Type().TerraformType(ctx), nil),
	}
	return r, state, identity
}

// powerSupervisorIdentityValue builds a request identity carrying the given id
// and (optionally null) site, mirroring what Terraform sends for an import
// block with an identity attribute.
func powerSupervisorIdentityValue(
	t *testing.T,
	identity tfsdk.ResourceIdentity,
	id string,
	site any,
) tfsdk.ResourceIdentity {
	t.Helper()
	idType := identity.Schema.Type().TerraformType(context.Background())
	identity.Raw = tftypes.NewValue(idType, map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, id),
		"site": tftypes.NewValue(tftypes.String, site),
	})
	return identity
}

// Test_powerSupervisorResource_ImportState_byIdentity covers the identity
// entry path (import block with identity = {...}, Terraform 1.12+): the id
// (and site when given) must land in state so Read can look the object up.
// The demo-mode controller has no power-supervisors endpoint (Network 10.2+),
// so this path is proven at the unit level; TestAccPowerSupervisor_import
// covers it end-to-end against a capable controller.
func Test_powerSupervisorResource_ImportState_byIdentity(t *testing.T) {
	ctx := context.Background()

	t.Run("id only", func(t *testing.T) {
		r, state, respIdentity := powerSupervisorImportHarness(t)
		reqIdentity := powerSupervisorIdentityValue(
			t, respIdentity, "6899fdbb50f0a10ea1dbf5aa", nil,
		)

		resp := &fwresource.ImportStateResponse{State: state, Identity: &respIdentity}
		r.ImportState(ctx, fwresource.ImportStateRequest{Identity: &reqIdentity}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ImportState errors: %v", resp.Diagnostics)
		}

		var gotID, gotSite types.String
		resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("id"), &gotID)...)
		resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("site"), &gotSite)...)
		if resp.Diagnostics.HasError() {
			t.Fatalf("reading imported state: %v", resp.Diagnostics)
		}
		if gotID.ValueString() != "6899fdbb50f0a10ea1dbf5aa" {
			t.Errorf("id = %q, want 6899fdbb50f0a10ea1dbf5aa", gotID.ValueString())
		}
		// site stays null so Read falls back to the provider default site.
		if !gotSite.IsNull() {
			t.Errorf("site = %v, want null", gotSite)
		}
	})

	t.Run("id and site", func(t *testing.T) {
		r, state, respIdentity := powerSupervisorImportHarness(t)
		reqIdentity := powerSupervisorIdentityValue(
			t, respIdentity, "6899fdbb50f0a10ea1dbf5aa", "site2",
		)

		resp := &fwresource.ImportStateResponse{State: state, Identity: &respIdentity}
		r.ImportState(ctx, fwresource.ImportStateRequest{Identity: &reqIdentity}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ImportState errors: %v", resp.Diagnostics)
		}

		var gotSite types.String
		resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("site"), &gotSite)...)
		if resp.Diagnostics.HasError() {
			t.Fatalf("reading imported state: %v", resp.Diagnostics)
		}
		if gotSite.ValueString() != "site2" {
			t.Errorf("site = %q, want site2", gotSite.ValueString())
		}
	})
}

// Test_powerSupervisorResource_ImportState_byIDString covers the classic
// string entry path ("id" and "site:id") and asserts the import key is
// mirrored into the resource identity.
func Test_powerSupervisorResource_ImportState_byIDString(t *testing.T) {
	ctx := context.Background()

	t.Run("bare id", func(t *testing.T) {
		r, state, respIdentity := powerSupervisorImportHarness(t)
		resp := &fwresource.ImportStateResponse{State: state, Identity: &respIdentity}
		r.ImportState(ctx, fwresource.ImportStateRequest{ID: "6899fdbb50f0a10ea1dbf5aa"}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ImportState errors: %v", resp.Diagnostics)
		}

		var gotID, gotSite, identityID types.String
		resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("id"), &gotID)...)
		resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("site"), &gotSite)...)
		resp.Diagnostics.Append(
			resp.Identity.GetAttribute(ctx, path.Root("id"), &identityID)...)
		if resp.Diagnostics.HasError() {
			t.Fatalf("reading imported state/identity: %v", resp.Diagnostics)
		}
		if gotID.ValueString() != "6899fdbb50f0a10ea1dbf5aa" {
			t.Errorf("id = %q, want 6899fdbb50f0a10ea1dbf5aa", gotID.ValueString())
		}
		if gotSite.ValueString() != "default" {
			t.Errorf("site = %q, want default (provider site)", gotSite.ValueString())
		}
		if identityID.ValueString() != "6899fdbb50f0a10ea1dbf5aa" {
			t.Errorf("identity id = %q, want the import ID", identityID.ValueString())
		}
	})

	t.Run("site:id", func(t *testing.T) {
		r, state, respIdentity := powerSupervisorImportHarness(t)
		resp := &fwresource.ImportStateResponse{State: state, Identity: &respIdentity}
		r.ImportState(
			ctx,
			fwresource.ImportStateRequest{ID: "site2:6899fdbb50f0a10ea1dbf5aa"},
			resp,
		)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ImportState errors: %v", resp.Diagnostics)
		}

		var gotID, gotSite types.String
		resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("id"), &gotID)...)
		resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("site"), &gotSite)...)
		if resp.Diagnostics.HasError() {
			t.Fatalf("reading imported state: %v", resp.Diagnostics)
		}
		if gotID.ValueString() != "6899fdbb50f0a10ea1dbf5aa" {
			t.Errorf("id = %q, want 6899fdbb50f0a10ea1dbf5aa", gotID.ValueString())
		}
		if gotSite.ValueString() != "site2" {
			t.Errorf("site = %q, want site2", gotSite.ValueString())
		}
	})
}

// TestAccPowerSupervisor_import exercises the full create + string-import +
// identity-import lifecycle. The Device Supervisor collection is a UniFi
// Network 10.2+ feature that the demo-mode controller (Network 10.0) does not
// expose (its v2 power-supervisors endpoint 404s), and creating one requires a
// PoE-powered device, so the test is gated on UNIFI_ACC_POE_DEVICE_MAC naming
// such a device plus a probe of the endpoint.
func TestAccPowerSupervisor_import(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping acceptance test")
	}
	mac := os.Getenv("UNIFI_ACC_POE_DEVICE_MAC")
	if mac == "" {
		t.Skip(
			"UNIFI_ACC_POE_DEVICE_MAC not set; power supervisors need a PoE-powered " +
				"device on a UniFi Network 10.2+ controller",
		)
	}

	// Probe the endpoint so the test skips (rather than fails) on controllers
	// without the Device Supervisor feature.
	ctx := context.Background()
	apiClient, err := unifi.New(ctx, &unifi.Config{
		BaseURL:       os.Getenv("UNIFI_API"),
		Username:      os.Getenv("UNIFI_USERNAME"),
		Password:      os.Getenv("UNIFI_PASSWORD"),
		AllowInsecure: true,
	})
	if err != nil {
		t.Skipf("could not build probe client: %s", err)
	}
	probe := &Client{ApiClient: apiClient, Site: "default"}
	if _, err := probe.ListPowerSupervisors(ctx, probe.Site); err != nil {
		t.Skipf("controller does not support power supervisors: %s", err)
	}

	config := fmt.Sprintf(`
resource "unifi_power_supervisor" "test" {
	device_mac = %q
}
`, mac)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_power_supervisor.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_power_supervisor.test",
						"enabled",
						"true",
					),
				),
			},
			// Classic string import by controller ID (what ImportStateVerify
			// replays); consecutive_failures can tick between reads.
			{
				ResourceName:            "unifi_power_supervisor.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"consecutive_failures"},
			},
			// Classic string import by the supervised device's MAC.
			{
				ResourceName:            "unifi_power_supervisor.test",
				ImportState:             true,
				ImportStateId:           mac,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"consecutive_failures"},
			},
			// Identity-based import (import block with identity, Terraform 1.12+).
			{
				ResourceName:    "unifi_power_supervisor.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func Test_powerSupervisorResource_ListResourceConfigSchema(t *testing.T) {
	r := &powerSupervisorResource{}
	resp := &fwlist.ListResourceSchemaResponse{}
	r.ListResourceConfigSchema(context.Background(), fwlist.ListResourceSchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ListResourceConfigSchema() returned errors: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["site"]; !ok {
		t.Error("ListResourceConfigSchema() missing 'site' attribute")
	}
}
