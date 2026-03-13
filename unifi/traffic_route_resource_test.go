package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTrafficRoute_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTrafficRouteConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"description",
						"tfacc-basic-route",
					),
					resource.TestCheckResourceAttr("unifi_traffic_route.test", "enabled", "true"),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip_addresses.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip_addresses.0.ip_or_subnet",
						"192.168.1.2",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"kill_switch_enabled",
						"false",
					),
				),
			},
			{
				ResourceName:    "unifi_traffic_route.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func testAccTrafficRouteConfig_basic() string {
	return `
data "unifi_network" "default" {
	name = "Default"
}

resource "unifi_traffic_route" "test" {
	description         = "tfacc-basic-route"
	enabled             = true
	next_hop				    = "192.168.1.1"
	network_id			    = data.unifi_network.default.id
	destination = {
		ip_addresses = [{ ip_or_subnet = "192.168.1.2" }]
	}
	kill_switch_enabled = false
}
`
}

func TestAccTrafficRoute_ipAddresses(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTrafficRouteConfig_ipAddresses(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"description",
						"tfacc-ip-route",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip_addresses.#",
						"2",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip_addresses.0.ip_or_subnet",
						"10.0.0.0/8",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip_addresses.0.ports.#",
						"2",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip_addresses.0.ports.0",
						"80",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip_addresses.0.ports.1",
						"443",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip_addresses.1.ip_or_subnet",
						"192.168.1.0/24",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip_addresses.1.port_ranges.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip_addresses.1.port_ranges.0.start",
						"8080",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip_addresses.1.port_ranges.0.stop",
						"8090",
					),
				),
			},
			{
				ResourceName:    "unifi_traffic_route.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func testAccTrafficRouteConfig_ipAddresses() string {
	return `
resource "unifi_traffic_route" "test" {
	description     = "tfacc-ip-route"
	enabled         = true

	destination = {
		ip_addresses = [
			{
				ip_or_subnet = "10.0.0.0/8"
				ports        = [80, 443]
			},
			{
				ip_or_subnet = "192.168.1.0/24"
				port_ranges  = [{ start = 8080, stop = 8090 }]
			},
		]
	}
}
`
}

func TestAccTrafficRoute_ipRanges(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTrafficRouteConfig_ipRanges(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"description",
						"tfacc-iprange-route",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip_ranges.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip_ranges.0.start",
						"10.0.0.1",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip_ranges.0.stop",
						"10.0.0.100",
					),
				),
			},
			{
				ResourceName:    "unifi_traffic_route.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func testAccTrafficRouteConfig_ipRanges() string {
	return `
resource "unifi_traffic_route" "test" {
	description     = "tfacc-iprange-route"
	enabled         = true

	destination = {
		ip_ranges = [
			{
				start   = "10.0.0.1"
				stop    = "10.0.0.100"
			},
		]
	}
}
`
}

func TestAccTrafficRoute_sourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTrafficRouteConfig_sourceDefault(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"description",
						"tfacc-source-default-route",
					),
					resource.TestCheckNoResourceAttr(
						"unifi_traffic_route.test",
						"source.networks.#",
					),
					resource.TestCheckNoResourceAttr(
						"unifi_traffic_route.test",
						"source.clients.#",
					),
				),
			},
		},
	})
}

func testAccTrafficRouteConfig_sourceDefault() string {
	return `
resource "unifi_traffic_route" "test" {
	description     = "tfacc-source-default-route"
	enabled         = true
	destination = {
		domains = [{ "domain" = "test.example.com" }]
	}
}
`
}

func TestAccTrafficRoute_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Initial creation
			{
				Config: testAccTrafficRouteConfig_updateStep1(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"description",
						"tfacc-update-route",
					),
					resource.TestCheckResourceAttr("unifi_traffic_route.test", "enabled", "true"),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.domains.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.domains.0.domain",
						"before.example.com",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"kill_switch_enabled",
						"false",
					),
				),
			},
			// Step 2: Update description, domains, and enable kill switch
			{
				Config: testAccTrafficRouteConfig_updateStep2(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"description",
						"tfacc-update-route-modified",
					),
					resource.TestCheckResourceAttr("unifi_traffic_route.test", "enabled", "true"),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.domains.#",
						"2",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.domains.0.domain",
						"after1.example.com",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.domains.1.domain",
						"after2.example.com",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"kill_switch_enabled",
						"true",
					),
				),
			},
			// Step 3: Disable the route
			{
				Config: testAccTrafficRouteConfig_updateStep3(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_traffic_route.test", "enabled", "false"),
				),
			},
		},
	})
}

func testAccTrafficRouteConfig_updateStep1() string {
	return `
resource "unifi_traffic_route" "test" {
	description     = "tfacc-update-route"
	enabled         = true
	destination = {
		domains = [{ "domain" = "before.example.com" }]
	}
}
`
}

func testAccTrafficRouteConfig_updateStep2() string {
	return `
resource "unifi_traffic_route" "test" {
	description        = "tfacc-update-route-modified"
	enabled            = true
	destination = {
		domains = [{ "domain" = "after1.example.com" }, { "domain" = "after2.example.com" }]
	}
	kill_switch_enabled = true
}
`
}

func testAccTrafficRouteConfig_updateStep3() string {
	return `
resource "unifi_traffic_route" "test" {
	description     = "tfacc-update-route-modified"
	enabled         = false
	destination = {
		domains = [{ "domain" = "after1.example.com" }, { "domain" = "after2.example.com" }]
	}
}
`
}

func TestAccTrafficRoute_regions(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTrafficRouteConfig_regions(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"description",
						"tfacc-region-route",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.regions.#",
						"2",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.regions.0",
						"US",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.regions.1",
						"CA",
					),
				),
			},
			{
				ResourceName:    "unifi_traffic_route.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func testAccTrafficRouteConfig_regions() string {
	return `
resource "unifi_traffic_route" "test" {
	description     = "tfacc-region-route"
	enabled         = true
	destination = {
		regions = ["US", "CA"]
	}
}
`
}

func TestAccTrafficRoute_fullConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTrafficRouteConfig_full(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"description",
						"tfacc-full-route",
					),
					resource.TestCheckResourceAttr("unifi_traffic_route.test", "enabled", "true"),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"kill_switch_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip_addresses.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip_addresses.0.ip_or_subnet",
						"172.16.0.0/12",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip_ranges.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip_ranges.0.start",
						"192.168.0.1",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"destination.ip_ranges.0.stop",
						"192.168.0.50",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"source.clients.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_traffic_route.test",
						"source.clients.0.mac",
						"aa:bb:cc:dd:ee:ff",
					),
				),
			},
			{
				ResourceName:    "unifi_traffic_route.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func testAccTrafficRouteConfig_full() string {
	return `
resource "unifi_traffic_route" "test" {
	description         = "tfacc-full-route"
	enabled             = true
	kill_switch_enabled = true

	destination = {
		ip_addresses = [{ ip_or_subnet = "172.16.0.0/12" }]

		ip_ranges = [
			{
				start   = "192.168.0.1"
				stop    = "192.168.0.50"
			},
		]
	}

	source = { clients = [{ mac = "aa:bb:cc:dd:ee:ff" }] }
}
`
}
