# Look up a RADIUS profile by name. The built-in profile is
# typically named "Default".
data "unifi_radius_profile" "default" {
  name = "Default"
}

# The profile ID, e.g. for referencing from a unifi_wlan resource.
output "radius_profile_id" {
  value = data.unifi_radius_profile.default.id
}

# Whether RADIUS accounting is enabled (computed).
output "radius_profile_accounting_enabled" {
  value = data.unifi_radius_profile.default.accounting_enabled
}

# Whether dynamic VLAN assignment is enabled and its WLAN mode (computed).
output "radius_profile_vlan_enabled" {
  value = data.unifi_radius_profile.default.vlan_enabled
}

output "radius_profile_vlan_wlan_mode" {
  value = data.unifi_radius_profile.default.vlan_wlan_mode
}

# Look up a profile on a specific site.
data "unifi_radius_profile" "site_specific" {
  site = "production"
  name = "Corporate"
}
