resource "unifi_network" "my_vlan" {
  name   = "wifi-vlan"
  subnet = "10.0.0.1/24"
  vlan   = 10

  dhcp_server = {
    enabled = true
    start   = "10.0.0.6"
    stop    = "10.0.0.254"
  }
}

resource "unifi_client" "test" {
  mac  = "01:23:45:67:89:AB"
  name = "some client"
  note = "my note"

  fixed_ip   = "10.0.0.50"
  network_id = unifi_network.my_vlan.id
}
