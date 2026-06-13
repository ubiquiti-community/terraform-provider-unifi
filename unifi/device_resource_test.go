package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// TestMergePortOverridesByIndex guards #266: declaring a subset of port_override
// blocks must not wipe the device's other ports. The UniFi PUT replaces the whole
// port_overrides array, so the provider merges the declared ports (by port_idx)
// onto the device's current overrides before sending.
func TestMergePortOverridesByIndex(t *testing.T) {
	current := []unifi.DevicePortOverrides{
		{PortIDX: ptrInt64(3), NATiveNetworkID: "vlan-a"},
		{PortIDX: ptrInt64(4), NATiveNetworkID: "vlan-b"},
		{PortIDX: ptrInt64(5), NATiveNetworkID: "vlan-c"},
	}

	t.Run("subset replaces only its port, keeps the rest", func(t *testing.T) {
		declared := []unifi.DevicePortOverrides{
			{PortIDX: ptrInt64(5), NATiveNetworkID: "vlan-z"},
		}
		got := mergePortOverridesByIndex(current, declared)
		byIdx := indexOverrides(got)
		if len(got) != 3 {
			t.Fatalf("merged length = %d, want 3 (ports 3,4 must survive): %+v", len(got), got)
		}
		if byIdx[3].NATiveNetworkID != "vlan-a" || byIdx[4].NATiveNetworkID != "vlan-b" {
			t.Errorf("undeclared ports were altered: %+v", got)
		}
		if byIdx[5].NATiveNetworkID != "vlan-z" {
			t.Errorf("declared port 5 = %q, want vlan-z", byIdx[5].NATiveNetworkID)
		}
	})

	t.Run("declared new port is appended", func(t *testing.T) {
		declared := []unifi.DevicePortOverrides{
			{PortIDX: ptrInt64(7), NATiveNetworkID: "vlan-new"},
		}
		got := mergePortOverridesByIndex(current, declared)
		byIdx := indexOverrides(got)
		if len(got) != 4 {
			t.Fatalf("merged length = %d, want 4: %+v", len(got), got)
		}
		if byIdx[7].NATiveNetworkID != "vlan-new" {
			t.Errorf("new port 7 not appended: %+v", got)
		}
	})

	t.Run("no declared overrides returns current unchanged", func(t *testing.T) {
		got := mergePortOverridesByIndex(current, nil)
		if len(got) != 3 {
			t.Errorf("merged length = %d, want 3", len(got))
		}
	})
}

func indexOverrides(pos []unifi.DevicePortOverrides) map[int64]unifi.DevicePortOverrides {
	m := make(map[int64]unifi.DevicePortOverrides, len(pos))
	for _, po := range pos {
		if po.PortIDX != nil {
			m[*po.PortIDX] = po
		}
	}
	return m
}

func TestAccDeviceFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDeviceFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_device.test", "id"),
					resource.TestCheckResourceAttr("unifi_device.test", "name", "Test Device"),
					resource.TestCheckResourceAttr("unifi_device.test", "adopted", "true"),
				),
			},
			{
				ResourceName:            "unifi_device.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"allow_adoption", "forget_on_destroy"},
			},
		},
	})
}

func testAccDeviceFrameworkConfig_basic() string {
	return `
resource "unifi_device" "test" {
	mac  = "00:27:22:00:00:02"
	name = "Test Device"
	allow_adoption = true
	forget_on_destroy = false
}
`
}
