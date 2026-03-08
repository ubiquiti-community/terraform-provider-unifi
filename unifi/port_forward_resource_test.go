package unifi

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPortForward_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPortForwardConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_port_forward.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test",
						"name",
						"tfacc-port-forward",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test",
						"forward.ip",
						"192.168.1.100",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test",
						"forward.port",
						"8080",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test",
						"wan.port",
						"80",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test",
						"protocol",
						"tcp_udp",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test",
						"syslog_logging",
						"false",
					),
				),
			},
			{
				ResourceName:      "unifi_port_forward.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"enabled",
				},
			},
		},
	})
}

func TestAccPortForward_wan(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPortForwardConfig_wan(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_port_forward.test_wan", "id"),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_wan",
						"name",
						"tfacc-wan-forward",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_wan",
						"wan.interface",
						"wan",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_wan",
						"wan.port",
						"443",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_wan",
						"forward.ip",
						"192.168.1.50",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_wan",
						"forward.port",
						"443",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_wan",
						"protocol",
						"tcp",
					),
				),
			},
			{
				ResourceName:      "unifi_port_forward.test_wan",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"enabled",
				},
			},
		},
	})
}

func TestAccPortForward_sourceLimitingIP(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPortForwardConfig_sourceLimitingIP(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_port_forward.test_src", "id"),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_src",
						"name",
						"tfacc-src-limited",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_src",
						"source_limiting.ip",
						"10.0.0.0/24",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_src",
						"source_limiting.enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_src",
						"source_limiting.type",
						"ip",
					),
				),
			},
			{
				ResourceName:      "unifi_port_forward.test_src",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"enabled",
				},
			},
		},
	})
}

func TestAccPortForward_sourceLimitingFirewallGroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPortForwardConfig_sourceLimitingFirewallGroup(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_port_forward.test_fwg", "id"),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_fwg",
						"name",
						"tfacc-fwg-limited",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_fwg",
						"source_limiting.enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_fwg",
						"source_limiting.type",
						"firewall_group",
					),
					resource.TestCheckResourceAttrSet(
						"unifi_port_forward.test_fwg",
						"source_limiting.firewall_group_id",
					),
				),
			},
			{
				ResourceName:      "unifi_port_forward.test_fwg",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"enabled",
				},
			},
		},
	})
}

func TestAccPortForward_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPortForwardConfig_update_before(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_update",
						"name",
						"tfacc-update-forward",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_update",
						"forward.ip",
						"192.168.1.10",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_update",
						"forward.port",
						"80",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_update",
						"wan.port",
						"8080",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_update",
						"protocol",
						"tcp",
					),
				),
			},
			{
				Config: testAccPortForwardConfig_update_after(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_update",
						"name",
						"tfacc-update-forward-renamed",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_update",
						"forward.ip",
						"192.168.1.20",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_update",
						"forward.port",
						"443",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_update",
						"wan.port",
						"9443",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_update",
						"protocol",
						"tcp_udp",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_update",
						"syslog_logging",
						"true",
					),
				),
			},
		},
	})
}

func TestAccPortForward_syslogLogging(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPortForwardConfig_syslogLogging(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_port_forward.test_log", "id"),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_log",
						"name",
						"tfacc-logging",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_forward.test_log",
						"syslog_logging",
						"true",
					),
				),
			},
		},
	})
}

func TestAccPortForward_protocols(t *testing.T) {
	for _, proto := range []string{"tcp", "udp", "tcp_udp"} {
		t.Run(proto, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { preCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: testAccPortForwardConfig_protocol(proto),
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(
								"unifi_port_forward.test_proto",
								"protocol",
								proto,
							),
						),
					},
				},
			})
		})
	}
}

func testAccPortForwardConfig_basic() string {
	return `
resource "unifi_port_forward" "test" {
  name = "tfacc-port-forward"

  wan = {
    port = "80"
  }

  forward = {
    ip   = "192.168.1.100"
    port = "8080"
  }
}
`
}

func testAccPortForwardConfig_wan() string {
	return `
resource "unifi_port_forward" "test_wan" {
  name     = "tfacc-wan-forward"
  protocol = "tcp"

  wan = {
    interface = "wan"
    port      = "443"
  }

  forward = {
    ip   = "192.168.1.50"
    port = "443"
  }
}
`
}

func testAccPortForwardConfig_sourceLimitingIP() string {
	return `
resource "unifi_port_forward" "test_src" {
  name = "tfacc-src-limited"

  wan = {
    port = "22"
  }

  forward = {
    ip   = "192.168.1.10"
    port = "22"
  }

  source_limiting = {
    ip      = "10.0.0.0/24"
    enabled = true
  }
}
`
}

func testAccPortForwardConfig_sourceLimitingFirewallGroup() string {
	return `
resource "unifi_firewall_group" "test_src_group" {
  name = "tfacc-pf-src-group"
  type = "address-group"
  members = [
    "10.0.0.1",
    "10.0.0.2",
  ]
}

resource "unifi_port_forward" "test_fwg" {
  name = "tfacc-fwg-limited"

  wan = {
    port = "2222"
  }

  forward = {
    ip   = "192.168.1.10"
    port = "22"
  }

  source_limiting = {
    firewall_group_id = unifi_firewall_group.test_src_group.id
    enabled           = true
  }
}
`
}

func testAccPortForwardConfig_update_before() string {
	return `
resource "unifi_port_forward" "test_update" {
  name     = "tfacc-update-forward"
  protocol = "tcp"

  wan = {
    port = "8080"
  }

  forward = {
    ip   = "192.168.1.10"
    port = "80"
  }
}
`
}

func testAccPortForwardConfig_update_after() string {
	return `
resource "unifi_port_forward" "test_update" {
  name          = "tfacc-update-forward-renamed"
  protocol      = "tcp_udp"
  syslog_logging = true

  wan = {
    port = "9443"
  }

  forward = {
    ip   = "192.168.1.20"
    port = "443"
  }
}
`
}

func testAccPortForwardConfig_syslogLogging() string {
	return `
resource "unifi_port_forward" "test_log" {
  name           = "tfacc-logging"
  syslog_logging = true

  wan = {
    port = "3000"
  }

  forward = {
    ip   = "192.168.1.100"
    port = "3000"
  }
}
`
}

func testAccPortForwardConfig_protocol(proto string) string {
	return fmt.Sprintf(`
resource "unifi_port_forward" "test_proto" {
  name     = "tfacc-proto-%s"
  protocol = "%s"

  wan = {
    port = "5000"
  }

  forward = {
    ip   = "192.168.1.100"
    port = "5000"
  }
}
`, proto, proto)
}
