# Look up an existing access point group by name. The group ID and the list of
# member device MAC addresses are computed, making this useful for referencing a
# group from other resources (e.g. WLAN ap_group_ids) or auditing membership.
data "unifi_ap_group" "default" {
  name = "Default"
}

output "ap_group_id" {
  description = "The ID of the AP group, for use in other resources."
  value       = data.unifi_ap_group.default.id
}

output "ap_group_device_macs" {
  description = "MAC addresses of the access points in the group."
  value       = data.unifi_ap_group.default.device_macs
}
