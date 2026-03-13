# List all networks in the default site
list "unifi_network" "all" {
  provider = unifi
}

# List networks in a specific site
list "unifi_network" "site_networks" {
  provider = unifi

  config {
    site = "my-site"
  }
}

# List networks filtered by name
list "unifi_network" "filtered" {
  provider = unifi

  config {
    filter {
      name  = "name"
      value = "wifi-vlan"
    }
  }
}
