variable "vlan_id" {
  default = 10
}

data "unifi_ap_group" "default" {
}

data "unifi_client_qos_rate" "default" {
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

resource "unifi_wlan" "wifi" {
  name       = "myssid"
  passphrase = "12345678"
  security   = "wpapsk"

  # enable WPA2/WPA3 support
  wpa3_support    = true
  wpa3_transition = true
  pmf_mode        = "optional"

  network_id    = unifi_network.vlan.id
  ap_group_ids  = [data.unifi_ap_group.default.id]
  user_group_id = data.unifi_client_qos_rate.default.id
}
