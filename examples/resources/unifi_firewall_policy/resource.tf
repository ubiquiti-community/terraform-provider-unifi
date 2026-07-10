# Networks referenced by the zones and policies below.
resource "unifi_network" "lan" {
  name   = "LAN"
  subnet = "10.0.0.1/24"
  vlan   = 10

  dhcp_server = {
    enabled = true
    start   = "10.0.0.6"
    stop    = "10.0.0.254"
  }
}

resource "unifi_network" "dmz" {
  name   = "DMZ"
  subnet = "10.0.20.1/24"
  vlan   = 20

  dhcp_server = {
    enabled = true
    start   = "10.0.20.6"
    stop    = "10.0.20.254"
  }
}

resource "unifi_network" "iot" {
  name   = "IoT"
  subnet = "10.0.30.1/24"
  vlan   = 30

  dhcp_server = {
    enabled = true
    start   = "10.0.30.6"
    stop    = "10.0.30.254"
  }
}

# Zone-based firewall zones (UniFi OS 8.x+) grouping the networks above.
resource "unifi_firewall_zone" "lan" {
  name        = "LAN"
  network_ids = [unifi_network.lan.id]
}

resource "unifi_firewall_zone" "dmz" {
  name        = "DMZ"
  network_ids = [unifi_network.dmz.id]
}

resource "unifi_firewall_zone" "iot" {
  name        = "IoT"
  network_ids = [unifi_network.iot.id]
}

# Allow web traffic from the LAN zone to a web server in the DMZ on tcp/443.
# create_allow_respond lets UniFi auto-create the established/related return rule.
resource "unifi_firewall_policy" "lan_to_dmz_https" {
  name                 = "Allow LAN to DMZ HTTPS"
  action               = "ALLOW"
  protocol             = "tcp"
  logging              = true
  create_allow_respond = true

  source = {
    zone_id         = unifi_firewall_zone.lan.id
    matching_target = "NETWORK"
    network_ids     = [unifi_network.lan.id]
  }

  destination = {
    zone_id            = unifi_firewall_zone.dmz.id
    matching_target    = "IP"
    ips                = ["10.0.20.10"]
    port               = 443
    port_matching_type = "SPECIFIC"
  }
}

# Block all traffic from IoT devices to the LAN, with logging for visibility.
resource "unifi_firewall_policy" "block_iot_to_lan" {
  name        = "Block IoT to LAN"
  action      = "BLOCK"
  protocol    = "all"
  description = "Keep IoT devices isolated from trusted clients."
  logging     = true

  source = {
    zone_id         = unifi_firewall_zone.iot.id
    matching_target = "ANY"
  }

  destination = {
    zone_id         = unifi_firewall_zone.lan.id
    matching_target = "ANY"
  }
}

# Reject DNS (udp/53) from a specific IoT device to anywhere, forcing it to use
# the local resolver. REJECT sends an ICMP unreachable instead of silently dropping.
resource "unifi_firewall_policy" "iot_block_external_dns" {
  name     = "Reject IoT external DNS"
  action   = "REJECT"
  protocol = "udp"

  source = {
    zone_id         = unifi_firewall_zone.iot.id
    matching_target = "CLIENT"
    client_macs     = ["00:11:22:33:44:55"]
  }

  destination = {
    zone_id            = unifi_firewall_zone.iot.id
    matching_target    = "ANY"
    port               = 53
    port_matching_type = "SPECIFIC"
  }
}

# Allow internal ICMP (ping) between LAN and DMZ. For icmp/icmpv6 the controller
# rejects create_allow_respond = true, so add an explicit reverse policy if you
# need replies in the other direction.
resource "unifi_firewall_policy" "lan_dmz_ping" {
  name     = "Allow LAN to DMZ ping"
  action   = "ALLOW"
  protocol = "icmp"
  enabled  = true

  source = {
    zone_id         = unifi_firewall_zone.lan.id
    matching_target = "ANY"
  }

  destination = {
    zone_id         = unifi_firewall_zone.dmz.id
    matching_target = "ANY"
  }
}

# Block access to specific web domains (FQDN matching) from the LAN zone.
resource "unifi_firewall_policy" "block_web_domains" {
  name     = "Block social media from LAN"
  action   = "BLOCK"
  protocol = "all"

  # Apply this policy during weekday evening hours. Use ALWAYS to make a
  # policy continuously active, EVERY_DAY for a daily time range,
  # ONE_TIME_ONLY with date and an explicit time range, or CUSTOM with
  # date_start/date_end. Active modes require an explicit time_all_day value.
  schedule = {
    mode             = "EVERY_WEEK"
    repeat_on_days   = ["mon", "tue", "wed", "thu", "fri"]
    time_all_day     = false
    time_range_start = "18:00"
    time_range_end   = "23:00"
  }

  source = {
    zone_id         = unifi_firewall_zone.lan.id
    matching_target = "ANY"
  }

  destination = {
    zone_id         = unifi_firewall_zone.lan.id
    matching_target = "WEB"
    web_domains     = ["facebook.com", "tiktok.com"]
  }
}
