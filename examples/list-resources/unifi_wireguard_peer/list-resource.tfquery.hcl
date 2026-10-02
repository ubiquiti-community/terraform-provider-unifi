# List all WireGuard peers of a WireGuard server network (network_id is required)
list "unifi_wireguard_peer" "all" {
  provider = unifi

  config {
    network_id = "6606e3e415f6df0721014c52"
  }
}

# List WireGuard peers of a server network in a specific site
list "unifi_wireguard_peer" "site_peers" {
  provider = unifi

  config {
    site       = "my-site"
    network_id = "6606e3e415f6df0721014c52"
  }
}

# List a WireGuard peer by name
list "unifi_wireguard_peer" "by_name" {
  provider = unifi

  config {
    network_id = "6606e3e415f6df0721014c52"

    filter {
      name  = "name"
      value = "laptop"
    }
  }
}
