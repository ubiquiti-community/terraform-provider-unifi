# Look up a zone-based firewall zone by its display name
# (UniFi Network 8.x+). Useful for wiring zone IDs into
# unifi_firewall_policy resources.
data "unifi_firewall_zone" "internal" {
  name = "Internal"
}

# The zone ID, e.g. for use as a source/destination zone in a policy.
output "internal_zone_id" {
  value = data.unifi_firewall_zone.internal.id
}

# The internal zone key (e.g. "lan", "wan") and the networks in the zone.
output "internal_zone_key" {
  value = data.unifi_firewall_zone.internal.zone_key
}

output "internal_zone_network_ids" {
  value = data.unifi_firewall_zone.internal.network_ids
}

# Look up a zone on a specific site.
data "unifi_firewall_zone" "external" {
  site = "production"
  name = "External"
}
