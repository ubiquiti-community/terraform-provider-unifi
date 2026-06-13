# Basic DHCP WAN on a tagged VLAN.
resource "unifi_wan" "primary" {
  name    = "Internet 1"
  type    = "dhcp"   # one of: dhcp, static, pppoe, disabled
  type_v6 = "dhcpv6" # IPv6 WAN type: dhcpv6, slaac, ...
  enabled = true

  # WAN VLAN tagging.
  vlan = {
    enabled = true
    id      = 10
  }

  # Manual upstream DNS instead of the ISP-provided servers.
  dns = {
    preference = "manual" # auto or manual
    primary    = "1.1.1.1"
    secondary  = "1.0.0.1"
  }
}

# Fully-featured secondary WAN used as a weighted load-balance member, with
# Smart Queues (SQM), QoS marking, IGMP proxy and advertised line capacity.
resource "unifi_wan" "secondary" {
  name    = "Internet 2"
  type    = "dhcp"
  enabled = true

  vlan = {
    enabled = true
    id      = 20
  }

  # Smart Queue Management rates are in kbps.
  smartq = {
    enabled   = true
    up_rate   = 500000
    down_rate = 500000
  }

  # Egress QoS / 802.1p priority (0-7).
  egress_qos = {
    enabled  = true
    priority = 1
  }

  # Participate in WAN load balancing. type: failover-only | weighted.
  load_balance = {
    type              = "weighted"
    weight            = 75 # 1-100
    failover_priority = 2  # 1-10
  }

  # Multicast/IGMP proxy: downstream is none | lan | guest.
  igmp_proxy = {
    downstream = "lan"
    upstream   = true
  }

  # Advertise the line's real capacity to the controller (kbps).
  provider_capabilities = {
    download_kilobits_per_second = 1000000
    upload_kilobits_per_second   = 100000
  }
}
