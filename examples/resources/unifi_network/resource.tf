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
