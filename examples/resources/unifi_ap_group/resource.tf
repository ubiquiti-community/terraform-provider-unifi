# unifi_ap_group manages a group of access points. AP groups can be referenced
# from wireless networks (unifi_wlan via ap_group_ids) to control which access
# points broadcast a given SSID.
#
# Members are the MAC addresses of the access points in the group. MAC addresses
# are compared case- and separator-insensitively, so "AA:BB:CC:DD:EE:FF" and
# "aa-bb-cc-dd-ee-ff" are treated as the same value.
resource "unifi_ap_group" "example" {
  name = "example"

  device_macs = [
    "00:11:22:33:44:55",
  ]
}
