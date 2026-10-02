# Device Supervisor (UniFi Network 10.2+): watch a device's heartbeat and
# power-cycle its upstream PoE source if it goes silent. The controller resolves
# the upstream PoE port automatically from the device MAC.
resource "unifi_power_supervisor" "ap_lobby" {
  device_mac = "94:2a:6f:d6:ce:fd"

  # All timings are Go duration strings (these are the defaults).
  heartbeat_interval = "1m"  # probe every minute
  silence_threshold  = "15m" # power-cycle after 15 min of silence
  power_off_duration = "2m"  # keep the PoE port off for 2 min
}
