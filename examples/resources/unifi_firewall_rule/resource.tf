resource "unifi_firewall_rule" "drop_all" {
  name    = "drop all"
  action  = "drop"
  ruleset = "LAN_IN"

  rule_index = 2011

  protocol = "all"

  dst_address = "192.168.1.1"
}
