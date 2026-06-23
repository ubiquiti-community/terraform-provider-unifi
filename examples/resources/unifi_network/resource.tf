resource "unifi_network" "vlan" {
  name   = "wifi-vlan"
  subnet = "10.0.0.1/24"
  vlan   = 10

  dhcp_server = {
    enabled = true
    start   = "10.0.0.6"
    stop    = "10.0.0.254"
  }
}

# Dual-stack network with IPv6 static subnet, RA, and DHCPv6
resource "unifi_network" "dual_stack" {
  name   = "dual-stack-vlan"
  subnet = "10.0.1.1/24"
  vlan   = 11

  dhcp_server = {
    enabled = true
    start   = "10.0.1.6"
    stop    = "10.0.1.254"
  }

  ipv6_interface_type        = "static"
  ipv6_static_subnet         = "fd00:1::1/64"
  ipv6_ra                    = true
  ipv6_ra_priority           = "high"
  ipv6_ra_valid_lifetime     = "24h"
  ipv6_ra_preferred_lifetime = "4h"

  dhcp_v6_server = {
    enabled  = true
    dns_auto = true
    start    = "::2"
    stop     = "::7d1"
    lease    = 86400
  }
}

# Network with IPv6 Prefix Delegation from WAN
resource "unifi_network" "ipv6_pd" {
  name   = "ipv6-pd-vlan"
  subnet = "10.0.2.1/24"
  vlan   = 12

  ipv6_interface_type           = "pd"
  ipv6_pd_interface             = "wan"
  ipv6_pd_prefixid              = "1"
  ipv6_pd_auto_prefixid_enabled = false
  # ipv6_pd_start/stop are required for a prefix-delegation network — the
  # controller rejects it with api.err.InvalidIpv6Addr otherwise.
  ipv6_pd_start = "::2"
  ipv6_pd_stop  = "::7d1"
  ipv6_ra       = true
}

# Third-party gateway (VLAN-only) network
resource "unifi_network" "third_party" {
  name                = "third-party-vlan"
  subnet              = "192.168.20.1/24"
  vlan                = 20
  third_party_gateway = true

  dhcp_guarding = {
    enabled = true
    servers = ["192.168.20.1"]
  }
}
