resource "unifi_client_group" "wifi" {
  name = "wifi"

  qos_rate_max_down = 2000 # 2mbps
  qos_rate_max_up   = 10   # 10kbps
}


resource "unifi_client" "test" {
  mac  = "01:23:45:67:89:AB"
  name = "some client"
  note = "my note"

  fixed_ip   = "10.0.0.50"
  network_id = unifi_network.my_vlan.id
  group_id   = unifi_client_group.wifi.id
}
