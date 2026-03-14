# Route traffic to specific IP addresses and subnets with ports
resource "unifi_traffic_route" "ip_route" {
  description         = "Route web traffic to private subnets"
  enabled             = true
  kill_switch_enabled = false

  destination = {
    ip = [
      {
        address = "10.0.0.0/8"
        ports   = ["80", "443"]
      },
      {
        address = "192.168.1.0/24"
        ports   = ["8080-8090"]
      },
    ]
  }
}

# Route traffic to an IP range
resource "unifi_traffic_route" "ip_range_route" {
  description = "Route traffic to an IP range"
  enabled     = true

  destination = {
    ip = [{ address = "192.168.10.1-192.168.10.255" }]
  }
}

# Route traffic matching specific domains
resource "unifi_traffic_route" "domain_route" {
  description = "Route traffic for specific domains"
  enabled     = true

  destination = {
    domain = ["example.com", "api.example.com"]
  }
}

# Route traffic matching specific regions
resource "unifi_traffic_route" "region_route" {
  description = "Route traffic for US and Canada"
  enabled     = true

  destination = {
    region = ["US", "CA"]
  }
}

# Route all internet traffic through a specific network/VPN
resource "unifi_traffic_route" "all_internet" {
  description = "Route all internet traffic through VPN"
  enabled     = true
  network_id  = unifi_network.vpn.id
}

# Route traffic from specific source clients
resource "unifi_traffic_route" "client_route" {
  description         = "Route specific client traffic"
  enabled             = true
  kill_switch_enabled = true

  destination = {
    ip = [{ address = "172.16.0.0/12" }]
  }

  source = {
    clients = [{ mac = "aa:bb:cc:dd:ee:ff" }]
  }
}

# Route traffic from specific source networks
resource "unifi_traffic_route" "network_route" {
  description = "Route traffic from IoT network"
  enabled     = true

  destination = {
    domain = ["updates.example.com"]
  }

  source = {
    networks = [{ id = unifi_network.iot.id }]
  }
}
