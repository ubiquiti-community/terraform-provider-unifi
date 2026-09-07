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

  ipv6 = {
    interface_type = "static"
    static_subnet  = "fd00:1::1/64"
    ra = {
      enabled            = true
      priority           = "high"
      valid_lifetime     = "24h"
      preferred_lifetime = "4h"
    }
  }

  dhcp_v6_server = {
    enabled = true
    dns = {
      auto = true
    }
    start = "::2"
    stop  = "::7d1"
    lease = 86400
  }
}

# Network with IPv6 Prefix Delegation from WAN
resource "unifi_network" "ipv6_pd" {
  name   = "ipv6-pd-vlan"
  subnet = "10.0.2.1/24"
  vlan   = 12

  ipv6 = {
    interface_type = "pd"
    pd = {
      interface             = "wan"
      prefixid              = "1"
      auto_prefixid_enabled = false
      # start/stop are required for a prefix-delegation network — the
      # controller rejects it with api.err.InvalidIpv6Addr otherwise.
      start = "::2"
      stop  = "::7d1"
    }
    ra = {
      enabled = true
    }
  }
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
