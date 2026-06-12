# Device Supervisor (UniFi Network 10.2+): watch a device's heartbeat and
# power-cycle its upstream PoE source if it goes silent. The controller resolves
# the upstream PoE port automatically from the device MAC.
resource "unifi_power_supervisor" "ap_lobby" {
  device_mac = "94:2a:6f:d6:ce:fd"

  # All timings are in seconds (these are the defaults).
  heartbeat_interval = 60  # probe every minute
  silence_threshold  = 900 # power-cycle after 15 min of silence
  power_off_duration = 120 # keep the PoE port off for 2 min
}
