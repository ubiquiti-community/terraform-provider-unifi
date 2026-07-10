package unifi

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/hwtypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// testAccAPGroupCheckDestroy verifies that every unifi_ap_group in state has
// been removed from the controller. It is a best-effort check that no-ops when
// no live controller is configured.
func testAccAPGroupCheckDestroy(s *terraform.State) error {
	ctx := context.Background()
	apiURL := os.Getenv("UNIFI_API")
	if apiURL == "" {
		return nil
	}
	apiClient, err := unifi.New(ctx, &unifi.Config{
		BaseURL:       apiURL,
		Username:      os.Getenv("UNIFI_USERNAME"),
		Password:      os.Getenv("UNIFI_PASSWORD"),
		AllowInsecure: true,
	})
	if err != nil {
		return nil //nolint:nilerr // best-effort check; skip when no live client
	}
	c := &Client{ApiClient: apiClient, Site: "default"}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "unifi_ap_group" {
			continue
		}
		site := rs.Primary.Attributes["site"]
		if site == "" {
			site = c.Site
		}
		_, err := c.GetAPGroup(ctx, site, rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("unifi_ap_group %s still exists", rs.Primary.ID)
		}
		if _, ok := err.(*unifi.NotFoundError); !ok {
			return err
		}
	}
	return nil
}

// TestAccAPGroupFramework_basic exercises the full CRUD + import lifecycle of an
// AP group with EMPTY membership. Empty groups are portable: a freshly-booted
// controller has no adopted access points, so any real device MAC would be
// rejected with api.err.InvalidDeviceInApGroup. An empty group has no such
// dependency, which lets create → read → update → import → delete run green
// against any controller. The read/refresh path is the one that previously
// returned HTTP 405 (surfacing as `invalid character '<'` when the HTML error
// page was parsed as JSON); driving it here proves the fix.
func TestAccAPGroupFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccAPGroupCheckDestroy,
		Steps: []resource.TestStep{
			// Create an empty group.
			{
				Config: testAccAPGroupFrameworkConfig_basic("tf-acc-apgroup"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_ap_group.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_ap_group.test",
						"name",
						"tf-acc-apgroup",
					),
					resource.TestCheckResourceAttr("unifi_ap_group.test", "device_macs.#", "0"),
				),
			},
			// Update the name in place; membership stays empty.
			{
				Config: testAccAPGroupFrameworkConfig_basic("tf-acc-apgroup-2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_ap_group.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_ap_group.test",
						"name",
						"tf-acc-apgroup-2",
					),
					resource.TestCheckResourceAttr("unifi_ap_group.test", "device_macs.#", "0"),
				),
			},
			// Import the group and verify the imported state matches.
			{
				ResourceName:      "unifi_ap_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccAPGroupFramework_withDevices asserts MAC semantic-equality: a member
// written in upper/dash form must not produce a spurious plan against the same
// address stored lowercase/colon form on the controller. It is skipped unless
// UNIFI_ACC_AP_MAC names a real adopted access point, since the controller
// rejects membership of any device it has not adopted.
func TestAccAPGroupFramework_withDevices(t *testing.T) {
	mac := os.Getenv("UNIFI_ACC_AP_MAC")
	if mac == "" {
		t.Skip("UNIFI_ACC_AP_MAC not set; skipping adopted-device AP group test")
	}
	upperDashMac := strings.ToUpper(strings.ReplaceAll(mac, ":", "-"))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccAPGroupCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAPGroupFrameworkConfig_withDevice("tf-acc-apgroup-dev", mac),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_ap_group.test", "id"),
					resource.TestCheckResourceAttr("unifi_ap_group.test", "device_macs.#", "1"),
				),
			},
			// The same MAC in UPPER/dash form must be semantically equal, so the
			// plan is empty (no diff) even though the literal string differs.
			{
				Config: testAccAPGroupFrameworkConfig_withDevice(
					"tf-acc-apgroup-dev",
					upperDashMac,
				),
				// PlanOnly with no expected changes: the upper/dash MAC must
				// compare equal to the stored lowercase/colon form (semantic
				// equality), so the plan is empty.
				PlanOnly: true,
			},
		},
	})
}

func testAccAPGroupFrameworkConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "unifi_ap_group" "test" {
	name        = %q
	device_macs = []
}
`, name)
}

func testAccAPGroupFrameworkConfig_withDevice(name, mac string) string {
	return fmt.Sprintf(`
resource "unifi_ap_group" "test" {
	name        = %q
	device_macs = [%q]
}
`, name, mac)
}

func TestNewAPGroupResource(t *testing.T) {
	got := NewAPGroupResource()
	if got == nil {
		t.Fatal("NewAPGroupResource() returned nil")
	}
	if _, ok := got.(fwresource.ResourceWithImportState); !ok {
		t.Errorf("does not implement fwresource.ResourceWithImportState")
	}
	if _, ok := got.(fwresource.ResourceWithIdentity); !ok {
		t.Errorf("does not implement fwresource.ResourceWithIdentity")
	}
}

func TestNewAPGroupListResource(t *testing.T) {
	got := NewAPGroupListResource()
	if got == nil {
		t.Fatal("NewAPGroupListResource() returned nil")
	}
	_ = got
}

func Test_apGroupResource_Metadata(t *testing.T) {
	r := &apGroupResource{}
	resp := &fwresource.MetadataResponse{}
	r.Metadata(
		context.Background(),
		fwresource.MetadataRequest{ProviderTypeName: "unifi"},
		resp,
	)
	if resp.TypeName != "unifi_ap_group" {
		t.Errorf("TypeName = %q, want %q", resp.TypeName, "unifi_ap_group")
	}
}

func Test_apGroupResource_IdentitySchema(t *testing.T) {
	r := &apGroupResource{}
	resp := &fwresource.IdentitySchemaResponse{}
	r.IdentitySchema(context.Background(), fwresource.IdentitySchemaRequest{}, resp)
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Error("IdentitySchema missing 'id' attribute")
	}
}

func Test_apGroupResource_Schema(t *testing.T) {
	r := &apGroupResource{}
	resp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	for _, key := range []string{"id", "site", "name", "device_macs"} {
		if _, ok := resp.Schema.Attributes[key]; !ok {
			t.Errorf("Schema missing attribute %q", key)
		}
	}
}

func Test_apGroupResource_Configure(t *testing.T) {
	tests := []struct {
		name       string
		req        fwresource.ConfigureRequest
		wantErr    bool
		wantClient bool
	}{
		{
			name: "nil provider data",
			req:  fwresource.ConfigureRequest{},
		},
		{
			name:    "wrong type",
			req:     fwresource.ConfigureRequest{ProviderData: "wrong"},
			wantErr: true,
		},
		{
			name:       "correct client",
			req:        fwresource.ConfigureRequest{ProviderData: &Client{}},
			wantClient: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &apGroupResource{}
			resp := &fwresource.ConfigureResponse{}
			r.Configure(context.Background(), tt.req, resp)
			if tt.wantErr && !resp.Diagnostics.HasError() {
				t.Error("expected error diagnostic")
			}
			if !tt.wantErr && resp.Diagnostics.HasError() {
				t.Errorf("unexpected error: %v", resp.Diagnostics)
			}
			if tt.wantClient && r.client == nil {
				t.Error("expected client to be set")
			}
		})
	}
}

func Test_apGroupResource_modelToAPIAPGroup(t *testing.T) {
	ctx := context.Background()
	macsSet, _ := types.SetValueFrom(
		ctx,
		hwtypes.MACAddressType{},
		[]string{"00:11:22:33:44:55", "00:11:22:33:44:66"},
	)
	// Mixed case / dash separators must normalize to lowercase colon form.
	mixedSet, _ := types.SetValueFrom(ctx, hwtypes.MACAddressType{}, []string{"AA-BB-CC-DD-EE-FF"})

	tests := []struct {
		name    string
		model   *apGroupResourceModel
		want    *unifi.APGroup
		wantErr bool
	}{
		{
			name: "basic group",
			model: &apGroupResourceModel{
				Name:       types.StringValue("Test Group"),
				DeviceMacs: macsSet,
			},
			want: &unifi.APGroup{
				Name:       "Test Group",
				DeviceMacs: []string{"00:11:22:33:44:55", "00:11:22:33:44:66"},
			},
		},
		{
			name: "normalizes mac case and separators",
			model: &apGroupResourceModel{
				Name:       types.StringValue("Mixed"),
				DeviceMacs: mixedSet,
			},
			want: &unifi.APGroup{
				Name:       "Mixed",
				DeviceMacs: []string{"aa:bb:cc:dd:ee:ff"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &apGroupResource{}
			got, err := r.modelToAPIAPGroup(ctx, tt.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("modelToAPIAPGroup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.Name != tt.want.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.want.Name)
			}
			if !reflect.DeepEqual(got.DeviceMacs, tt.want.DeviceMacs) {
				t.Errorf("DeviceMacs = %v, want %v", got.DeviceMacs, tt.want.DeviceMacs)
			}
		})
	}
}

func Test_apGroupResource_apGroupToModel(t *testing.T) {
	tests := []struct {
		name string
		api  *unifi.APGroup
		want diag.Diagnostics
	}{
		{
			name: "basic API to model",
			api: &unifi.APGroup{
				ID:         "ap1",
				Name:       "Test",
				DeviceMacs: []string{"00:11:22:33:44:55"},
			},
			want: nil,
		},
		{
			name: "empty macs produces empty set",
			api: &unifi.APGroup{
				ID:         "ap2",
				Name:       "Empty",
				DeviceMacs: []string{},
			},
			want: nil,
		},
		{
			name: "nil macs produces empty set (not null)",
			api: &unifi.APGroup{
				ID:         "ap4",
				Name:       "NilMacs",
				DeviceMacs: nil,
			},
			want: nil,
		},
		{
			name: "empty name produces null",
			api: &unifi.APGroup{
				ID:         "ap3",
				Name:       "",
				DeviceMacs: []string{},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &apGroupResource{}
			model := &apGroupResourceModel{}
			got := r.apGroupToModel(context.Background(), tt.api, model, "default")
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("apGroupToModel() = %v, want %v", got, tt.want)
			}
			if tt.api.ID != "" && model.ID.ValueString() != tt.api.ID {
				t.Errorf("ID = %q, want %q", model.ID.ValueString(), tt.api.ID)
			}
			if tt.api.Name == "" && !model.Name.IsNull() {
				t.Error("expected Name to be null for empty API name")
			}
			if len(tt.api.DeviceMacs) == 0 {
				if model.DeviceMacs.IsNull() {
					t.Error(
						"expected DeviceMacs to be an empty (non-null) set for empty DeviceMacs",
					)
				} else if n := len(model.DeviceMacs.Elements()); n != 0 {
					t.Errorf("expected empty DeviceMacs set, got %d elements", n)
				}
			} else if n := len(model.DeviceMacs.Elements()); n != len(tt.api.DeviceMacs) {
				t.Errorf("DeviceMacs elements = %d, want %d", n, len(tt.api.DeviceMacs))
			}
		})
	}
}

func Test_apGroupResource_ListResourceConfigSchema(t *testing.T) {
	r := &apGroupResource{}
	resp := &fwlist.ListResourceSchemaResponse{}
	r.ListResourceConfigSchema(context.Background(), fwlist.ListResourceSchemaRequest{}, resp)
	if _, ok := resp.Schema.Attributes["site"]; !ok {
		t.Error("ListResourceConfigSchema missing 'site' attribute")
	}
}
