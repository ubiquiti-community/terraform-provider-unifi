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

# An access port for end devices: STP edge (forwards as soon as the link is up) with BPDU Guard, which shuts the
# port down if a switch is plugged in behind it.
resource "unifi_port_profile" "desk" {
  name = "Desk"

  native_networkconf_id = unifi_network.vlan.id
  forward               = "native"
  tagged_vlan_mgmt      = "block_all"

  stp_port_mode          = true
  stp_edge_state         = "enabled"
  stp_bpdu_guard_enabled = true
  link_debounce = {
    auto  = false
    delay = "500ms"
  }
}

# An AV port: a Pro AV QoS preset, PTP (needs PTP enabled on the switch), and the port as a multicast router port on
# its network (which needs IGMP snooping).
resource "unifi_port_profile" "av" {
  name = "AV"

  native_networkconf_id = unifi_network.vlan.id
  forward               = "native"
  tagged_vlan_mgmt      = "block_all"

  precision_time_protocol_enabled  = true
  multicast_router_mode            = "CUSTOM"
  multicast_router_networkconf_ids = [unifi_network.vlan.id]
  qos_profile = {
    mode = "dante_audio"
  }
}

# Custom QoS: traffic matching each policy is put in an egress queue (required) and can be re-marked.
resource "unifi_port_profile" "voice" {
  name = "Voice"

  native_networkconf_id = unifi_network.vlan.id

  qos_profile = {
    mode = "custom"
    policies = [
      {
        matching = { protocol = "udp", dscp_code = 46 }
        marking  = { queue = 6 }
      },
      {
        matching = { protocol = "tcp", dst_port = 5061 }
        marking  = { queue = 5, dscp_code = 26 }
      },
    ]
  }
}
