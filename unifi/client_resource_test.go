package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// TestClientToModel_DefaultsWhenAPIOmitsFields proves the fix for the spurious
// in-place diff on every create/import: when the controller omits blocked / groups /
// qos_rate (as UniFi OS 5.x / Network App 10.x does for fixed-IP-only clients),
// Read must store the documented default (blocked=false) rather than null, and leave
// groups / qos_rate null so the UseStateForUnknown plan modifiers can keep the plan
// clean. For this minimal client clientToModel makes no API calls, so no live
// controller (and no mock) is needed.
func TestClientToModel_DefaultsWhenAPIOmitsFields(t *testing.T) {
	r := &clientResource{}

	client := &unifi.Client{
		ID:                     "61d1...",
		MAC:                    "02:00:00:de:ad:01",
		Name:                   "tf-test",
		FixedIP:                "192.168.40.251",
		Blocked:                nil, // controller omitted "blocked"
		UserGroupID:            "",  // no qos_rate / usergroup
		NetworkMembersGroupIDs: nil, // no groups
	}

	var model clientResourceModel
	diags := r.clientToModel(context.Background(), client, &model, "default")
	if diags.HasError() {
		t.Fatalf("clientToModel returned errors: %v", diags)
	}

	if model.Blocked.IsNull() || model.Blocked.IsUnknown() {
		t.Errorf("blocked: want concrete value, got null/unknown (%#v)", model.Blocked)
	}
	if model.Blocked.ValueBool() != false {
		t.Errorf("blocked: want false, got %v", model.Blocked.ValueBool())
	}
	if !model.Groups.IsNull() {
		t.Errorf("groups: want null, got %#v", model.Groups)
	}
	if !model.QOSRate.IsNull() {
		t.Errorf("qos_rate: want null, got %#v", model.QOSRate)
	}
}

// TestClientToModel_PreservesBlockedTrue ensures a blocked client still round-trips.
func TestClientToModel_PreservesBlockedTrue(t *testing.T) {
	r := &clientResource{}
	blocked := true
	client := &unifi.Client{MAC: "02:00:00:de:ad:02", Blocked: &blocked}

	var model clientResourceModel
	if diags := r.clientToModel(context.Background(), client, &model, "default"); diags.HasError() {
		t.Fatalf("clientToModel returned errors: %v", diags)
	}
	if model.Blocked.ValueBool() != true {
		t.Errorf("blocked: want true, got %v", model.Blocked.ValueBool())
	}
}

func TestAccClientFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_client.test", "name", "tfacc-client"),
					resource.TestCheckResourceAttr("unifi_client.test", "mac", "01:23:45:67:89:ab"),
					resource.TestCheckResourceAttr("unifi_client.test", "blocked", "false"),
				),
			},
			{
				ResourceName:    "unifi_client.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func testAccClientFrameworkConfig_basic() string {
	return `
resource "unifi_client" "test" {
	name = "tfacc-client"
	mac  = "01:23:45:67:89:ab"
}
`
}

func TestAccClientFramework_blocked(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientFrameworkConfig_blocked(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_client.test",
						"name",
						"tfacc-blocked-client",
					),
					resource.TestCheckResourceAttr("unifi_client.test", "blocked", "true"),
					resource.TestCheckResourceAttr(
						"unifi_client.test",
						"note",
						"Blocked for testing",
					),
				),
			},
		},
	})
}

func testAccClientFrameworkConfig_blocked() string {
	return `
resource "unifi_client" "test" {
	name    = "tfacc-blocked-client"
	mac     = "01:23:45:67:89:ac"
	blocked = true
	note    = "Blocked for testing"
}
`
}

func TestAccClientFramework_fixedIP(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientFrameworkConfig_fixedIP(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_client.test",
						"name",
						"tfacc-fixed-ip-client",
					),
					resource.TestCheckResourceAttr(
						"unifi_client.test",
						"fixed_ip",
						"192.168.2.100",
					),
				),
			},
		},
	})
}

func testAccClientFrameworkConfig_fixedIP() string {
	return `
resource "unifi_network" "test" {
	name    = "Test"
	subnet  = "192.168.2.1/24"
	vlan    = 2

	dhcp_server = {
		enabled    = true
		start = "192.168.2.6"
		stop  = "192.168.2.254"
	}
}

resource "unifi_client" "test" {
	name       = "tfacc-fixed-ip-client"
	mac        = "01:23:45:67:89:ad"
	fixed_ip   = "192.168.2.100"
	network_id = unifi_network.test.id
}
`
}

func TestAccClientFramework_groups(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create client with one group
			{
				Config: testAccClientFrameworkConfig_groups_one(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_client.test",
						"name",
						"tfacc-groups-client",
					),
					resource.TestCheckResourceAttr("unifi_client.test", "mac", "01:23:45:67:89:ae"),
					resource.TestCheckResourceAttr("unifi_client.test", "groups.#", "1"),
					resource.TestCheckResourceAttr(
						"unifi_client.test",
						"groups.0",
						"tfacc-group-a",
					),
				),
			},
			// Step 2: Import the client and verify groups survive
			{
				ResourceName:            "unifi_client.test",
				ImportState:             true,
				ImportStateKind:         resource.ImportBlockWithResourceIdentity,
				ImportStateVerifyIgnore: []string{"allow_existing", "skip_forget_on_destroy"},
			},
			// Step 3: Add another group
			{
				Config: testAccClientFrameworkConfig_groups_two(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_client.test",
						"name",
						"tfacc-groups-client",
					),
					resource.TestCheckResourceAttr("unifi_client.test", "groups.#", "2"),
					resource.TestCheckResourceAttr(
						"unifi_client.test",
						"groups.0",
						"tfacc-group-a",
					),
					resource.TestCheckResourceAttr(
						"unifi_client.test",
						"groups.1",
						"tfacc-group-b",
					),
				),
			},
		},
	})
}

func testAccClientFrameworkConfig_groups_one() string {
	return `
resource "unifi_client" "test" {
	name   = "tfacc-groups-client"
	mac    = "01:23:45:67:89:ae"
	groups = ["tfacc-group-a"]
}
`
}

func testAccClientFrameworkConfig_groups_two() string {
	return `
resource "unifi_client" "test" {
	name   = "tfacc-groups-client"
	mac    = "01:23:45:67:89:ae"
	groups = ["tfacc-group-a", "tfacc-group-b"]
}
`
}
