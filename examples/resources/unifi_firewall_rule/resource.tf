# unifi_firewall_rule manages an individual firewall rule on the gateway.
#
# action  must be one of: drop, accept, reject.
# ruleset is the rule chain, e.g. LAN_IN, WAN_IN, GUEST_IN, WANv6_IN, etc.
# rule_index must fall in an interface-specific block:
#   2000-2999 (LAN), 3000-3999 (WAN), 4000-4999 (GUEST),
#   or the high-range equivalents 20000-29999 / 30000-39999 / 40000-49999.

# Drop all traffic destined to the gateway from the LAN.
resource "unifi_firewall_rule" "drop_to_gateway" {
  name    = "drop to gateway"
  action  = "drop"
  ruleset = "LAN_IN"

  rule_index = 2011
  protocol   = "all"

  dst_address = "192.168.1.1"
}

# Accept established/related return traffic on the WAN inbound chain.
resource "unifi_firewall_rule" "wan_in_established" {
  name    = "allow established/related"
  action  = "accept"
  ruleset = "WAN_IN"

  rule_index = 3001
  protocol   = "all"

  state_established = true
  state_related     = true
}

# Use firewall groups: allow the trusted admin hosts to reach the web ports.
resource "unifi_firewall_group" "admin_hosts" {
  name    = "admin-hosts"
  type    = "address-group"
  members = ["192.168.1.10", "192.168.1.11"]
}

resource "unifi_firewall_group" "web_ports" {
  name    = "web-ports"
  type    = "port-group"
  members = ["80", "443"]
}

resource "unifi_firewall_rule" "allow_admin_web" {
  name    = "allow admin to web"
  action  = "accept"
  ruleset = "LAN_IN"

  rule_index = 2012
  protocol   = "tcp"

  src_firewall_group_ids = [unifi_firewall_group.admin_hosts.id]
  dst_firewall_group_ids = [unifi_firewall_group.web_ports.id]
}

# Source/destination address and port with a single host source (ADDRv4).
resource "unifi_firewall_rule" "allow_ssh_from_host" {
  name    = "allow ssh from jump host"
  action  = "accept"
  ruleset = "LAN_IN"

  rule_index = 2013
  protocol   = "tcp"

  src_address      = "192.168.1.5"
  src_network_type = "ADDRv4"
  dst_address      = "192.168.20.0/24"
  dst_port         = "22"

  logging = true
}

# Reject UDP traffic to a guest network with logging enabled.
resource "unifi_firewall_rule" "reject_guest_udp" {
  name    = "reject guest udp"
  action  = "reject"
  ruleset = "GUEST_IN"

  rule_index = 4001
  protocol   = "udp"

  dst_port = "53"
  logging  = true
}

# IPv6 example on the WANv6 inbound chain, dropping all inbound IPv6.
resource "unifi_firewall_rule" "drop_wan_v6" {
  name    = "drop inbound ipv6"
  action  = "drop"
  ruleset = "WANv6_IN"

  rule_index  = 3002
  protocol_v6 = "all"
}
