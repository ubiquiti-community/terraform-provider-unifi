# Zone-based firewall zones (UniFi OS 8.x+) group one or more networks.
# Reference a zone's id from unifi_firewall_policy source/destination zone_id.

# Networks to place into the zones below.
resource "unifi_network" "dmz" {
  name   = "dmz"
  subnet = "10.0.20.1/24"
  vlan   = 20

  dhcp_server = {
    enabled = true
    start   = "10.0.20.6"
    stop    = "10.0.20.254"
  }
}

resource "unifi_network" "iot" {
  name   = "iot"
  subnet = "10.0.30.1/24"
  vlan   = 30

  dhcp_server = {
    enabled = true
    start   = "10.0.30.6"
    stop    = "10.0.30.254"
  }
}

# A DMZ zone containing the DMZ network.
resource "unifi_firewall_zone" "dmz" {
  name = "DMZ"
  network_ids = [
    unifi_network.dmz.id,
  ]
}

# A second zone for IoT devices.
resource "unifi_firewall_zone" "iot" {
  name = "IoT"
  network_ids = [
    unifi_network.iot.id,
  ]
}
