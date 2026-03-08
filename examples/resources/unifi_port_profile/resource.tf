variable "vlan_id" {
  default = 10
}

resource "unifi_network" "vlan" {
  name   = "wifi-vlan"
  subnet = "10.0.0.1/24"
  vlan   = var.vlan_id

  dhcp_server = {
    enabled = true
    start   = "10.0.0.6"
    stop    = "10.0.0.254"
  }
}

resource "unifi_port_profile" "poe_disabled" {
  name = "POE Disabled"

  native_networkconf_id = unifi_network.vlan.id
  poe_mode              = "off"
}
