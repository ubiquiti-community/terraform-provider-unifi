# unifi_firewall_group manages reusable groups of addresses or ports that can be
# referenced from firewall rules (unifi_firewall_rule) and port forwards.
#
# The "type" attribute must be one of: address-group, port-group, ipv6-address-group.

# An address-group of IPv4 hosts/subnets, e.g. trusted admin workstations.
resource "unifi_firewall_group" "admin_hosts" {
  name = "admin-hosts"
  type = "address-group"

  members = [
    "192.168.1.10",
    "192.168.1.11",
    "192.168.10.0/24",
  ]
}

# A port-group of TCP/UDP ports, e.g. common web ports to allow or block.
resource "unifi_firewall_group" "web_ports" {
  name = "web-ports"
  type = "port-group"

  members = [
    "80",
    "443",
    "8080",
  ]
}

# An ipv6-address-group of IPv6 hosts/prefixes.
resource "unifi_firewall_group" "ipv6_servers" {
  name = "ipv6-servers"
  type = "ipv6-address-group"

  members = [
    "2001:db8:1::10",
    "2001:db8:2::/64",
  ]
}

# Example: reference the groups above from a firewall rule. The destination
# address group and destination port group are matched together.
#
# resource "unifi_firewall_rule" "block_web" {
#   name    = "block web from admins"
#   action  = "drop"
#   ruleset = "LAN_IN"
#
#   rule_index = 2010
#   protocol   = "tcp"
#
#   src_firewall_group_ids = [unifi_firewall_group.admin_hosts.id]
#   dst_firewall_group_ids = [unifi_firewall_group.web_ports.id]
# }
