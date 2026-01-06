resource "unifi_wan" "default" {
  name         = "Internet 1"
  type         = "dhcp"
  type_v6      = "dhcpv6"
  vlan_enabled = true
  vlan         = 10
  enabled      = true

  dns_preference = "manual"
  dns1           = "1.1.1.1"
  dns2           = "1.0.0.1"

  smartq_enabled   = true
  smartq_up_rate   = 500000
  smartq_down_rate = 500000

  egress_qos_enabled = true
  egress_qos         = 1
  dhcp_cos           = 0
  dhcpv6_cos         = 0

  provider_capabilities = {
    download_kilobits_per_second = 1000000
    upload_kilobits_per_second   = 100000
  }

  load_balance_type   = "weighted"
  load_balance_weight = 75
  failover_priority   = 2
}
