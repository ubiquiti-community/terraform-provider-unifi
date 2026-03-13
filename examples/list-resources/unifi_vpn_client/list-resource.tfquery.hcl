# List all VPN client connections in the default site
list "unifi_vpn_client" "all" {
  provider = unifi
}

# List VPN clients in a specific site
list "unifi_vpn_client" "site_vpn_clients" {
  provider = unifi

  config {
    site = "my-site"
  }
}

# List VPN clients filtered by name
list "unifi_vpn_client" "filtered" {
  provider = unifi

  config {
    filter {
      name  = "name"
      value = "my-wireguard-vpn"
    }
  }
}
