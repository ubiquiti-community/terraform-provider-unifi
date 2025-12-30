resource "unifi_firewall_group" "can_print" {
  name = "can-print"
  type = "address-group"

  members = ["192.168.1.25"]
}
