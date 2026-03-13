# List all VPN servers in the default site
list "unifi_vpn_server" "all" {
  provider = unifi
}

# List VPN servers in a specific site
list "unifi_vpn_server" "site_vpn_servers" {
  provider = unifi

  config {
    site = "my-site"
  }
}

# List VPN servers filtered by name
list "unifi_vpn_server" "filtered" {
  provider = unifi

  config {
    filter {
      name  = "name"
      value = "my-wireguard-server"
    }
  }
}
