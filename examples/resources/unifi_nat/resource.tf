# Forward TCP 8443 arriving on the WAN network to an internal host.
resource "unifi_nat" "dnat" {
  description  = "Forward 8443 to the web server"
  type         = "DNAT"
  protocol     = "tcp"
  in_interface = var.wan_network_id
  ip_address   = "192.168.1.20"
  port         = 443

  destination_filter = {
    filter_type = "ADDRESS_AND_PORT"
    port        = 8443
  }
}

# Masquerade traffic from a LAN network leaving on the WAN network.
resource "unifi_nat" "masquerade" {
  description   = "Masquerade LAN to WAN"
  type          = "MASQUERADE"
  out_interface = var.wan_network_id

  source_filter = {
    filter_type     = "NETWORK_CONF"
    network_conf_id = var.lan_network_id
  }
}
