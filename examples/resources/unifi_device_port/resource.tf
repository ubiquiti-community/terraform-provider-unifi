# One port of a switch, managed on its own. The other ports of the switch are
# left as the controller has them, and the port can be imported directly with
# "<device_mac>/<index>".
resource "unifi_device_port" "nas" {
  device_mac = "aa:bb:cc:dd:ee:ff"
  index      = 8

  name                  = "nas"
  native_networkconf_id = unifi_network.servers.id
  poe_mode              = "off"
  stp_edge_state        = "enabled"
}
