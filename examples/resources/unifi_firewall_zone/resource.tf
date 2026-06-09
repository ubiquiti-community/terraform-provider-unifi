# A zone-based firewall zone (UniFi OS 8.x+) grouping one or more networks.
# Reference its id from unifi_firewall_policy source/destination zone_id.
resource "unifi_firewall_zone" "dmz" {
  name = "DMZ"
  network_ids = [
    unifi_network.dmz.id,
  ]
}
